package main

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/push"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

var now = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

func baseEnv(dir string) map[string]string {
	return map[string]string{
		"ADMIN_USERNAME": "root-admin", "ADMIN_PASSWORD": "admin-password-AAAAAAAA", "DATA_DIR": dir,
		"INTERNAL_API_TOKEN": "internal-token-0123456789abcdef0123456789abcdef", "PUBLIC_ORIGIN": "http://localhost:9212",
	}
}

// pushLine returns the value of "push" on the one start line that carries it.
func pushLine(t *testing.T, out string) string {
	t.Helper()
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		if v, ok := m["push"]; ok {
			got = append(got, fmt.Sprint(v))
		}
	}
	if len(got) != 1 {
		t.Fatalf("push log lines %v in %q, want exactly one", got, out)
	}
	return got[0]
}

func TestStartWithoutVAPIDLogsPushOff(t *testing.T) {
	var out bytes.Buffer
	env := baseEnv(t.TempDir())
	a, err := start(context.Background(), func(k string) string { return env[k] }, &out)
	if err != nil {
		t.Fatal(err)
	}
	defer a.store.Close()
	if a.handler == nil {
		t.Fatal("no handler")
	}
	if got := pushLine(t, out.String()); got != "off" {
		t.Fatalf("push = %q, want off", got)
	}
	if strings.Contains(out.String(), "listening") {
		t.Fatal("start listened")
	}
}

// testVAPIDKey is a fresh P-256 private key as VAPID_PRIVATE_KEY carries it, never the edge's.
func testVAPIDKey(t *testing.T) string {
	t.Helper()
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(k.Bytes())
}

func TestStartWithVAPIDLogsPushOn(t *testing.T) {
	key := testVAPIDKey(t)
	var out bytes.Buffer
	env := baseEnv(t.TempDir())
	env["VAPID_PRIVATE_KEY"], env["VAPID_SUBJECT"] = key, "https://happybaby.private.example.org"
	a, err := start(context.Background(), func(k string) string { return env[k] }, &out)
	if err != nil {
		t.Fatal(err)
	}
	defer a.store.Close()
	if got := pushLine(t, out.String()); got != "on" {
		t.Fatalf("push = %q, want on", got)
	}
	if strings.Contains(out.String(), key) || strings.Contains(out.String(), "happybaby.private") {
		t.Fatal("start logged a VAPID value")
	}
}

func TestStartRefusesHalfSetVAPID(t *testing.T) {
	env := baseEnv(t.TempDir())
	env["VAPID_SUBJECT"] = "https://happybaby.private.example.org"
	_, err := start(context.Background(), func(k string) string { return env[k] }, io.Discard)
	if want := "VAPID_PRIVATE_KEY is required when VAPID_SUBJECT is set"; err == nil || err.Error() != want {
		t.Fatalf("start: %v, want %q", err, want)
	}
}

// channelsLine returns the one start line that logs the channels.
func channelsLine(t *testing.T, out string) map[string]any {
	t.Helper()
	var got []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		if m["msg"] == "channels" {
			got = append(got, m)
		}
	}
	if len(got) != 1 {
		t.Fatalf("%d channels lines in %q, want exactly one", len(got), out)
	}
	return got[0]
}

func TestStartLogsNtfyOffAndOn(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  map[string]string
		want string
	}{
		{"off", nil, "off"},
		{"on", map[string]string{"NTFY_URL": "http://zqntfy:2586", "NTFY_TOPIC_PREFIX": "zqprefix", "NTFY_TOKEN": "tk_zqsecret", "NTFY_PUBLIC_URL": "https://zqpublic.example.org"}, "on"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			env := baseEnv(t.TempDir())
			for k, v := range tc.set {
				env[k] = v
			}
			a, err := start(context.Background(), func(k string) string { return env[k] }, &out)
			if err != nil {
				t.Fatal(err)
			}
			defer a.store.Close()
			line := channelsLine(t, out.String())
			if line["ntfy"] != tc.want || line["push"] != "off" {
				t.Fatalf("channels line %v, want ntfy %q and push off", line, tc.want)
			}
			if strings.Contains(out.String(), "zq") {
				t.Fatalf("start logged an ntfy value: %s", out.String())
			}
		})
	}
}

func TestStartRefusesNtfySettingsWithoutURL(t *testing.T) {
	env := baseEnv(t.TempDir())
	env["NTFY_TOKEN"] = "tk_zqsecret"
	_, err := start(context.Background(), func(k string) string { return env[k] }, io.Discard)
	if want := "NTFY_TOKEN requires NTFY_URL"; err == nil || err.Error() != want {
		t.Fatalf("start: %v, want %q", err, want)
	}
}

const testInternal = "internal-token-0123456789abcdef0123456789abcdef"

// call serves one request through the app's handler with the internal token and, when set, a session.
func call(t *testing.T, a *app, method, path, session string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rd io.Reader = http.NoBody
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(raw)
	}
	r := httptest.NewRequest(method, path, rd)
	r.Header.Set("Authorization", "Bearer "+testInternal)
	if session != "" {
		r.Header.Set("X-Session-Token", session)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, r)
	return rec
}

func parentSession(t *testing.T, s *store.Store) string {
	t.Helper()
	p, err := s.CreateParent(context.Background(), "ana", "Ana", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSession(context.Background(), "parent", p.ID, hash, time.Now(), time.Now().Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	return plain
}

func TestStartRunsTheWorkerWithNeitherChannel(t *testing.T) {
	env := baseEnv(t.TempDir())
	a, err := start(context.Background(), func(k string) string { return env[k] }, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer a.store.Close()
	if err := a.worker.Tick(context.Background()); err != nil {
		t.Fatalf("tick with no channel: %v", err)
	}
	if a.worker.LastTick().IsZero() {
		t.Fatal("the tick was not recorded")
	}
	if rec := call(t, a, http.MethodGet, "/v1/_test/pushes", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("test route without the fake: %d", rec.Code)
	}
	for _, path := range []string{"/healthz", "/healthz/reminders"} {
		if rec := call(t, a, http.MethodGet, path, "", nil); rec.Code != http.StatusOK {
			t.Fatalf("%s with no channel: %d", path, rec.Code)
		}
	}
}

func TestStartWithFakePushMarksSendingRowsUnconfirmed(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.CreateParent(ctx, "ana", "Ana", now)
	if err != nil {
		t.Fatal(err)
	}
	kid, err := s.CreateKid(ctx, model.Kid{Name: "Mia", BirthDate: "2026-01-02", Sex: "female", TimeZone: "UTC", GrowthSource: "who",
		ReminderOverdueMinutes: constants.DefaultReminderOverdueMinutes}, p.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateMedication(ctx, kid.ID, p.ID, model.Medication{Name: "Amoxicillin", Dose: "2.5 ml", Active: true, StartAt: now,
		Schedule: schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 1}}, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "happybabytracker.db")+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// A delivery a crashed process left mid-send.
	if _, err := db.Exec(`INSERT INTO reminder_deliveries(id, reminder_id, channel, target_id, parent_id, state, attempts, next_attempt_at)
		SELECT 'left-sending', id, 'push', 'dev', ?, 'sending', 1, 0 FROM reminders LIMIT 1`, p.ID); err != nil {
		t.Fatal(err)
	}

	env := baseEnv(dir)
	env["VAPID_PRIVATE_KEY"], env["VAPID_SUBJECT"], env["HBT_ALLOW_FAKE_PUSH"] = testVAPIDKey(t), "https://happybaby.private.example.org", "1"
	a, err := start(ctx, func(k string) string { return env[k] }, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer a.store.Close()
	var state string
	if err := db.QueryRow(`SELECT state FROM reminder_deliveries WHERE id = 'left-sending'`).Scan(&state); err != nil || state != "unconfirmed" {
		t.Fatalf("left-sending row is %q (%v), want unconfirmed before the first tick", state, err)
	}
	if err := a.worker.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	rec := call(t, a, http.MethodGet, "/v1/_test/pushes", "", nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"pushes":[]}` {
		t.Fatalf("test route with the fake: %d %s", rec.Code, rec.Body.String())
	}
}

func TestStartWithRealPushRegistersADevice(t *testing.T) {
	if _, err := net.DefaultResolver.LookupHost(context.Background(), "fcm.googleapis.com"); err != nil {
		t.Skipf("registration resolves a real push service name and this host cannot: %v", err)
	}
	key := testVAPIDKey(t)
	env := baseEnv(t.TempDir())
	env["VAPID_PRIVATE_KEY"], env["VAPID_SUBJECT"] = key, "https://happybaby.private.example.org"
	a, err := start(context.Background(), func(k string) string { return env[k] }, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer a.store.Close()
	session := parentSession(t, a.store)

	k, err := push.ParsePrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	rec := call(t, a, http.MethodGet, "/v1/push", session, nil)
	if want := `{"enabled":true,"public_key":"` + push.PublicKey(k) + `"}`; rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("GET /v1/push: %d %s, want %s", rec.Code, rec.Body.String(), want)
	}

	ua, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	authSecret := make([]byte, 16)
	if _, err := rand.Read(authSecret); err != nil {
		t.Fatal(err)
	}
	// A real push service name: registration resolves it through the process's resolver.
	body := map[string]any{
		"endpoint": "https://fcm.googleapis.com/fcm/send/main-test-device",
		"keys": map[string]string{"p256dh": base64.RawURLEncoding.EncodeToString(ua.PublicKey().Bytes()),
			"auth": base64.RawURLEncoding.EncodeToString(authSecret)},
		"label": "Phone",
	}
	if rec := call(t, a, http.MethodPut, "/v1/me/push-devices", session, body); rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, a, http.MethodGet, "/v1/_test/pushes", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("test route with real push: %d", rec.Code)
	}
}

func TestStartWiresProviderSearchOnlyUnderTheFlag(t *testing.T) {
	for _, flag := range []string{"", "1"} {
		env := baseEnv(t.TempDir())
		env["NPI_SEARCH"] = flag
		a, err := start(context.Background(), func(k string) string { return env[k] }, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		session := parentSession(t, a.store)
		rec := call(t, a, http.MethodGet, "/v1/providers", session, nil)
		if want := fmt.Sprintf(`{"search":{"enabled":%v}}`, flag == "1"); strings.TrimSpace(rec.Body.String()) != want {
			t.Errorf("NPI_SEARCH %q: GET /v1/providers %d %s, want %s", flag, rec.Code, rec.Body.String(), want)
		}
		// An invalid kind is refused before any registry call, so the test never leaves the process.
		rec = call(t, a, http.MethodPost, "/v1/providers/search", session, map[string]string{"kind": "x"})
		if want := map[string]int{"": http.StatusNotFound, "1": http.StatusUnprocessableEntity}[flag]; rec.Code != want {
			t.Errorf("NPI_SEARCH %q: search %d %s, want %d", flag, rec.Code, rec.Body.String(), want)
		}
		a.store.Close()
	}
}

func TestStartWiresAuthMode(t *testing.T) {
	for val, want := range map[string]string{"": "local", "oidc": "oidc", "both": "both"} {
		env := baseEnv(t.TempDir())
		env["AUTH_MODE"] = val
		a, err := start(context.Background(), func(k string) string { return env[k] }, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		if rec := call(t, a, http.MethodGet, "/v1/auth", "", nil); strings.TrimSpace(rec.Body.String()) != `{"mode":"`+want+`"}` {
			t.Errorf("AUTH_MODE %q: GET /v1/auth %d %s, want mode %s", val, rec.Code, rec.Body.String(), want)
		}
		a.store.Close()
	}
	env := baseEnv(t.TempDir())
	env["AUTH_MODE"] = "bogus"
	if _, err := start(context.Background(), func(k string) string { return env[k] }, io.Discard); err == nil || err.Error() != "AUTH_MODE must be local, oidc or both" {
		t.Fatalf("start: %v, want the AUTH_MODE refusal", err)
	}
}
