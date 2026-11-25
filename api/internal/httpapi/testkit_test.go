package httpapi

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/ntfy"
	"github.com/davcs86/happybabytracker/api/internal/push"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/testlog"
)

const (
	testInternalToken = "internal-token-0123456789abcdef0123456789abcdef"
	testAdminUser     = "root-admin"
	testAdminPass     = "admin-pass-0123456789"
	testParentPass    = "parent pass 12345"
)

var t0 = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

// One real hash per test binary; every seeded parent shares it.
var testHash = sync.OnceValue(func() string {
	h, err := auth.HashPassword(testParentPass)
	if err != nil {
		panic(err)
	}
	return h
})

type kit struct {
	t        *testing.T
	Store    *store.Store
	Cfg      config.Config
	Limiter  *auth.Limiter
	Log      *bytes.Buffer
	Handler  http.Handler
	now      time.Time
	Push     push.Sender
	Recorder *push.Recorder
	Resolver push.Resolver
}

// testVAPID is one P-256 pair per test binary; the kit runs with Web Push on and the fake sender.
var testVAPID = sync.OnceValue(func() *ecdsa.PrivateKey {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return k
})

// migratedDB is a store migrated once per test binary; a kit opens a copy, so no kit pays for the migrations.
var migratedDB = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "happybabytracker-kit-")
	if err != nil {
		return "", err
	}
	s, err := store.Open(dir)
	if err != nil {
		return "", err
	}
	return dir, s.Close()
})

func TestMain(m *testing.M) {
	code := m.Run()
	if dir, err := migratedDB(); err == nil {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}

func newKit(t *testing.T) *kit {
	t.Helper()
	tmpl, err := migratedDB()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	files, err := os.ReadDir(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(tmpl, f.Name()))
		if err == nil {
			err = os.WriteFile(filepath.Join(dir, f.Name()), b, 0o600)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return newKitWith(t, s)
}

func newKitWith(t *testing.T, s *store.Store) *kit {
	t.Helper()
	t.Cleanup(func() { s.Close() })
	k := &kit{
		t:     t,
		Store: s,
		Cfg: config.Config{
			AdminUsername: testAdminUser, AdminPassword: testAdminPass,
			InternalAPIToken: testInternalToken, PublicOrigin: "https://example.test",
		},
		Log: &bytes.Buffer{},
		now: t0,
	}
	k.Cfg.VAPIDKey, k.Cfg.VAPIDSubject, k.Cfg.AllowFakePush = testVAPID(), "https://example.test", true
	k.Recorder = &push.Recorder{}
	k.Push = k.Recorder
	k.Cfg.Ntfy, k.Cfg.NtfyEnabled = ntfy.Config{URL: testNtfyURL, TopicPrefix: "hbt-test"}, true
	k.Limiter = auth.NewLimiter(constants.LoginAttemptsPerMinute, k.clock)
	k.Handler = NewHandler(k.deps())
	return k
}

const testNtfyURL = "http://ntfy:2586"

// newKitNoNtfy is the kit without NTFY_URL: ntfy off, Web Push unchanged.
func newKitNoNtfy(t *testing.T) *kit {
	t.Helper()
	k := newKit(t)
	k.Cfg.Ntfy, k.Cfg.NtfyEnabled = ntfy.Config{}, false
	k.Handler = NewHandler(k.deps())
	return k
}

// newKitNoPush is the kit with neither VAPID variable set: Web Push off, no fake, no test route.
func newKitNoPush(t *testing.T) *kit {
	t.Helper()
	k := newKit(t)
	k.Cfg.VAPIDKey, k.Cfg.VAPIDSubject, k.Cfg.AllowFakePush = nil, "", false
	k.Recorder, k.Push = nil, nil
	k.Handler = NewHandler(k.deps())
	return k
}

func (k *kit) clock() time.Time { return k.now }

func (k *kit) deps() Deps {
	d := Deps{
		Store: k.Store, Cfg: k.Cfg, Limiter: k.Limiter, Now: k.clock, LastTick: k.clock,
		Log:  slog.New(slog.NewJSONHandler(k.Log, nil)),
		Push: k.Push, Resolver: k.Resolver, PushRecorder: k.Recorder,
	}
	if k.Cfg.VAPIDKey != nil {
		d.VAPIDPublicKey = push.PublicKey(k.Cfg.VAPIDKey)
	}
	return d
}

// addParent creates a parent with the shared test password; the username must already be normalised.
func (k *kit) addParent(username string) model.Parent {
	k.t.Helper()
	p, err := k.Store.CreateParent(context.Background(), username, "Name of "+username, k.now)
	if err != nil {
		k.t.Fatal(err)
	}
	if err := k.Store.SetPassword(context.Background(), p.ID, testHash(), ""); err != nil {
		k.t.Fatal(err)
	}
	return p
}

// asAdmin seeds an admin session and returns its token.
func (k *kit) asAdmin() string {
	k.t.Helper()
	return k.seedSession("admin", "", constants.AdminSessionTTL)
}

// asParent seeds a session for the parent, creating the parent first when it does not exist.
func (k *kit) asParent(username string) string {
	k.t.Helper()
	p, _, err := k.Store.ParentByUsername(context.Background(), username)
	if errors.Is(err, store.ErrNotFound) {
		p = k.addParent(username)
	} else if err != nil {
		k.t.Fatal(err)
	}
	return k.seedSession("parent", p.ID, constants.ParentSessionTTL)
}

func (k *kit) seedSession(kind, parentID string, ttl time.Duration) string {
	k.t.Helper()
	plain, hash, err := auth.NewToken()
	if err != nil {
		k.t.Fatal(err)
	}
	if _, err := k.Store.CreateSession(context.Background(), kind, parentID, hash, k.now, k.now.Add(ttl)); err != nil {
		k.t.Fatal(err)
	}
	return plain
}

// newReq returns a request carrying the internal token and, when body is not nil, a JSON body.
func (k *kit) newReq(method, path string, body any) *http.Request {
	k.t.Helper()
	var rd *bytes.Reader
	switch b := body.(type) {
	case nil:
		rd = bytes.NewReader(nil)
	case string:
		rd = bytes.NewReader([]byte(b))
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			k.t.Fatal(err)
		}
		rd = bytes.NewReader(raw)
	}
	r := httptest.NewRequest(method, path, rd)
	r.Header.Set("Authorization", "Bearer "+testInternalToken)
	return r
}

func (k *kit) serve(r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	k.Handler.ServeHTTP(rec, r)
	return rec
}

// do sends an authenticated request, as the session token's owner when token is not empty.
func (k *kit) do(method, path, token string, body any) *httptest.ResponseRecorder {
	k.t.Helper()
	r := k.newReq(method, path, body)
	if token != "" {
		r.Header.Set("X-Session-Token", token)
	}
	return k.serve(r)
}

// login posts credentials from the given client address (empty sends no address header).
func (k *kit) login(addr, username, password string) *httptest.ResponseRecorder {
	k.t.Helper()
	r := k.newReq(http.MethodPost, "/v1/session", map[string]string{"username": username, "password": password})
	if addr != "" {
		r.Header.Set("X-Client-Address", addr)
	}
	return k.serve(r)
}

func bodyMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("body %q is not a JSON object: %v", rec.Body.String(), err)
	}
	return m
}

func wantErr(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	want := `{"error":"` + code + `"}`
	if rec.Code != status || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("got %d %q, want %d %s", rec.Code, rec.Body.String(), status, want)
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func sameKeys(t *testing.T, got map[string]any, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("keys = %v, want %v", keys(got), want)
	}
	for _, w := range want {
		if _, ok := got[w]; !ok {
			t.Fatalf("keys = %v, want %v", keys(got), want)
		}
	}
}

var allowedLogFields = []string{"time", "level", "msg", "route", "status", "principal_kind", "request_id", "duration_ms", constants.LogFieldCause, constants.LogFieldSlide,
	constants.LogFieldOutcomeGrowth, constants.LogFieldOutcomeVaccinations, constants.LogFieldOutcomeMedications}

// logLeaks is testlog.Leaks with this package's allowlist.
func logLeaks(line string, banned ...string) []string {
	return testlog.Leaks(line, allowedLogFields, banned...)
}

// wantCleanLog checks that log holds want lines and that none of them leaks.
func wantCleanLog(t *testing.T, log string, want int, banned ...string) {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) != want {
		t.Fatalf("%d log lines, want %d:\n%s", len(lines), want, log)
	}
	for _, line := range lines {
		for _, leak := range logLeaks(line, banned...) {
			t.Errorf("%s: %s", leak, line)
		}
	}
}

func TestLogLeaksCatchesAValueButNotARandomOrMeasuredOne(t *testing.T) {
	for line, leaks := range map[string]int{
		`{"msg":"request","route":"x 4321"}`:                   1,
		`{"msg":"request","extra":"y"}`:                        1,
		`{"msg":"request","request_id":"a4321","time":"4321"}`: 0,
		`{"msg":"request","duration_ms":4321,"status":200}`:    0,
	} {
		if got := logLeaks(line, "4321"); len(got) != leaks {
			t.Errorf("%s: %v, want %d leaks", line, got, leaks)
		}
	}
}
