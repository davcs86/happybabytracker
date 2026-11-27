package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

const (
	cliAdmin = "root-admin"
	cliPass  = "cli-password-AAAAAAAA"
	cliPass2 = "cli-password-BBBBBBBB"
)

type cliResult struct {
	code           int
	stdout, stderr string
}

func (r cliResult) messages(t *testing.T) []string {
	t.Helper()
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(r.stdout+"\n"+r.stderr), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("output line %q: %v", line, err)
		}
		got = append(got, m["msg"].(string))
	}
	return got
}

func (r cliResult) has(t *testing.T, msg string) bool {
	t.Helper()
	for _, m := range r.messages(t) {
		if m == msg {
			return true
		}
	}
	return false
}

func cli(t *testing.T, dir, stdin string, args ...string) cliResult {
	t.Helper()
	var out, errOut bytes.Buffer
	env := map[string]string{"DATA_DIR": dir}
	code := runAdmin(context.Background(), args, func(k string) string { return env[k] }, strings.NewReader(stdin), &out, &errOut, now)
	return cliResult{code, out.String(), errOut.String()}
}

func openDir(t *testing.T, dir string) *store.Store {
	t.Helper()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func rawDB(t *testing.T, dir string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "happybabytracker.db")+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func adminCount(t *testing.T, dir string) int {
	t.Helper()
	var n int
	if err := rawDB(t, dir).QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func seedAdmin(t *testing.T, dir string) {
	t.Helper()
	if r := cli(t, dir, cliPass+"\n", "create", cliAdmin); r.code != 0 {
		t.Fatalf("create: %+v", r)
	}
}

func sessionFor(t *testing.T, s *store.Store, kind, principal, token string) {
	t.Helper()
	if _, err := s.CreateSession(context.Background(), kind, principal, []byte(token), now, now.Add(constants.AdminSessionTTL), ""); err != nil {
		t.Fatal(err)
	}
}

func alive(t *testing.T, s *store.Store, token string) bool {
	t.Helper()
	_, err := s.SessionByToken(context.Background(), []byte(token), now)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
	return err == nil
}

func TestCreateOnAFreshVolume(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	r := cli(t, dir, cliPass+"\n", "create", cliAdmin)
	if r.code != 0 || !r.has(t, "admin created") {
		t.Fatalf("create = %+v", r)
	}
	if n := adminCount(t, dir); n != 1 {
		t.Fatalf("admins = %d, want 1", n)
	}
	if !adminVerifies(t, openDir(t, dir), cliAdmin, cliPass) {
		t.Fatal("the password does not verify")
	}
}

func TestCreateRefusedWhenAnAdminExists(t *testing.T) {
	dir := t.TempDir()
	seedAdmin(t, dir)
	var before string
	if err := rawDB(t, dir).QueryRow(`SELECT id || username || secret_hash FROM admins JOIN admin_identities ON admin_id = id`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	r := cli(t, dir, cliPass2+"\n", "create", "other-admin")
	if r.code != 1 || !r.has(t, "an admin already exists") {
		t.Fatalf("create = %+v", r)
	}
	var after string
	if err := rawDB(t, dir).QueryRow(`SELECT id || username || secret_hash FROM admins JOIN admin_identities ON admin_id = id`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before || adminCount(t, dir) != 1 {
		t.Fatal("the admin row changed")
	}
}

func TestCreateRefusesAParentsName(t *testing.T) {
	dir := t.TempDir()
	s := openDir(t, dir)
	if _, err := s.CreateParent(context.Background(), "david", "David", now); err != nil {
		t.Fatal(err)
	}
	r := cli(t, dir, cliPass+"\n", "create", "david")
	if r.code != 1 || !r.has(t, "that username belongs to a parent") {
		t.Fatalf("create = %+v", r)
	}
	if n := adminCount(t, dir); n != 0 {
		t.Fatalf("admins = %d, want 0", n)
	}
}

func TestResetPassword(t *testing.T) {
	dir := t.TempDir()
	seedAdmin(t, dir)
	s := openDir(t, dir)
	ctx := context.Background()
	a, _, err := s.AdminByUsername(ctx, cliAdmin)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.CreateParent(ctx, "ana", "Ana", now)
	if err != nil {
		t.Fatal(err)
	}
	sessionFor(t, s, "admin", a.ID, "admin-1")
	sessionFor(t, s, "admin", a.ID, "admin-2")
	sessionFor(t, s, "parent", p.ID, "parent-1")

	r := cli(t, dir, cliPass2+"\n", "reset-password", cliAdmin)
	if r.code != 0 || !r.has(t, "admin password reset; its sessions were ended") {
		t.Fatalf("reset-password = %+v", r)
	}
	if alive(t, s, "admin-1") || alive(t, s, "admin-2") {
		t.Error("an admin session survived")
	}
	if !alive(t, s, "parent-1") {
		t.Error("the parent session was ended")
	}
	if !adminVerifies(t, s, cliAdmin, cliPass2) {
		t.Error("the new password does not verify")
	}
	if adminVerifies(t, s, cliAdmin, cliPass) {
		t.Error("the old password still verifies")
	}
	var changed int64
	if err := rawDB(t, dir).QueryRow(`SELECT secret_changed_at FROM admin_identities WHERE admin_id = ? AND provider = 'local'`, a.ID).Scan(&changed); err != nil || changed != now.Unix() {
		t.Errorf("secret_changed_at = %d, %v; want %d", changed, err, now.Unix())
	}
}

func TestResetPasswordUnknownAdmin(t *testing.T) {
	dir := t.TempDir()
	seedAdmin(t, dir)
	r := cli(t, dir, cliPass2+"\n", "reset-password", "other-admin")
	if r.code != 1 || !r.has(t, "no admin with that username") {
		t.Fatalf("reset-password = %+v", r)
	}
	if !adminVerifies(t, openDir(t, dir), cliAdmin, cliPass) {
		t.Error("the stored password changed")
	}
}

func TestUnlinkSSO(t *testing.T) {
	dir := t.TempDir()
	seedAdmin(t, dir)
	s := openDir(t, dir)
	a, _, err := s.AdminByUsername(context.Background(), cliAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rawDB(t, dir).Exec(`INSERT INTO admin_identities(admin_id, provider, subject) VALUES (?, 'https://issuer.example', 'sub-1')`, a.ID); err != nil {
		t.Fatal(err)
	}
	sessionFor(t, s, "admin", a.ID, "admin-1")

	r := cli(t, dir, "", "unlink-sso", cliAdmin)
	if r.code != 0 || !r.has(t, "admin single sign-on unlinked; its sessions were ended") {
		t.Fatalf("unlink-sso = %+v", r)
	}
	got, _, err := s.AdminByUsername(context.Background(), cliAdmin)
	if err != nil || got.SSOLinked {
		t.Errorf("admin = %+v, %v; want unlinked", got, err)
	}
	if alive(t, s, "admin-1") {
		t.Error("the admin session survived")
	}
	if !adminVerifies(t, s, cliAdmin, cliPass) {
		t.Error("the local row is gone")
	}

	r = cli(t, dir, "", "unlink-sso", "other-admin")
	if r.code != 1 || !r.has(t, "no admin with that username") {
		t.Fatalf("unlink-sso unknown = %+v", r)
	}
}

func TestUsage(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no verb", nil},
		{"unknown verb", []string{"delete", cliAdmin}},
		{"create without a username", []string{"create"}},
		{"reset without a username", []string{"reset-password"}},
		{"unlink without a username", []string{"unlink-sso"}},
		{"extra argument", []string{"create", cliAdmin, "extra"}},
		{"upper case username", []string{"reset-password", "Root-Admin"}},
		{"short username", []string{"unlink-sso", "qz"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "data")
			r := cli(t, dir, cliPass+"\n", tc.args...)
			if r.code != 2 {
				t.Fatalf("exit = %d, want 2 (%+v)", r.code, r)
			}
			if len(r.messages(t)) != 1 {
				t.Errorf("output %+v, want one usage line", r)
			}
		})
	}
}

// countHashes replaces the KDF seam for the test and returns how many times it ran.
func countHashes(t *testing.T) *int {
	t.Helper()
	n := new(int)
	orig := hashAdminPassword
	hashAdminPassword = func(pw string) (string, error) { *n++; return orig(pw) }
	t.Cleanup(func() { hashAdminPassword = orig })
	return n
}

func TestPasswordOnStdin(t *testing.T) {
	pw := "pw-0123456789abcd"
	maxIn := strings.Repeat("x", constants.CLIPasswordMaxBytes)
	for _, tc := range []struct {
		name, stdin, stored string
		refused, noHash     bool
	}{
		{name: "newline", stdin: pw + "\n", stored: pw},
		{name: "crlf", stdin: pw + "\r\n", stored: pw},
		{name: "no newline", stdin: pw, stored: pw},
		{name: "spaces kept", stdin: " " + pw + " \n", stored: " " + pw + " "},
		{name: "inner cr kept", stdin: "pw\r" + pw + "\n", stored: "pw\r" + pw},
		{name: "only one newline stripped", stdin: pw + "\n\n", stored: pw + "\n"},
		{name: "empty", stdin: "", refused: true},
		{name: "15 multibyte runes", stdin: strings.Repeat("é", 15) + "\n", refused: true},
		{name: "257 runes", stdin: strings.Repeat("y", 257), refused: true},
		{name: "over the byte cap", stdin: maxIn + "x", refused: true, noHash: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			n := countHashes(t)
			r := cli(t, dir, tc.stdin, "create", cliAdmin)
			if tc.refused {
				if r.code != 1 || !r.has(t, "the password must be 16 to 256 characters") {
					t.Fatalf("create = %+v, want the length refusal", r)
				}
				if adminCount(t, dir) != 0 {
					t.Fatal("an admin was stored")
				}
				if tc.noHash && *n != 0 {
					t.Fatalf("hashAdminPassword ran %d times", *n)
				}
				return
			}
			if r.code != 0 {
				t.Fatalf("create = %+v", r)
			}
			if !adminVerifies(t, openDir(t, dir), cliAdmin, tc.stored) {
				t.Fatalf("the stored password is not %q", tc.stored)
			}
		})
	}
}

func TestCLIOutputCarriesNoName(t *testing.T) {
	dir := t.TempDir()
	s := openDir(t, dir)
	if _, err := s.CreateParent(context.Background(), "david-parent", "David", now); err != nil {
		t.Fatal(err)
	}
	runs := []struct {
		stdin string
		args  []string
	}{
		{cliPass + "\n", []string{"create", "david-parent"}},
		{"short\n", []string{"create", cliAdmin}},
		{cliPass + "\n", []string{"create", cliAdmin}},
		{cliPass2 + "\n", []string{"create", "other-admin"}},
		{cliPass2 + "\n", []string{"reset-password", cliAdmin}},
		{cliPass2 + "\n", []string{"reset-password", "other-admin"}},
		{"", []string{"unlink-sso", cliAdmin}},
		{"", []string{"unlink-sso", "other-admin"}},
		{"", []string{"rename", "secret-verb-arg"}},
		{"", []string{"create", "Bad-Name"}},
	}
	for _, run := range runs {
		r := cli(t, dir, run.stdin, run.args...)
		all := r.stdout + r.stderr
		if all == "" {
			t.Errorf("%v printed nothing", run.args)
		}
		for _, v := range []string{cliAdmin, "other-admin", "david-parent", "secret-verb-arg", "Bad-Name", cliPass, cliPass2, "short"} {
			if strings.Contains(all, v) {
				t.Errorf("%v printed %q: %s", run.args, v, all)
			}
		}
	}
}

func TestCreateRacesTheImport(t *testing.T) {
	const importName = "env-admin"
	for round := 0; round < 20; round++ {
		dir := t.TempDir()
		imp := openDir(t, dir)
		cfg := config.Config{DataDir: dir, AdminUsername: importName, AdminPassword: "admin-password-AAAAAAAA"}
		var (
			wg        sync.WaitGroup
			start     = make(chan struct{})
			r         cliResult
			importErr error
			importLog bytes.Buffer
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			var out, errOut bytes.Buffer
			env := map[string]string{"DATA_DIR": dir}
			code := runAdmin(context.Background(), []string{"create", cliAdmin}, func(k string) string { return env[k] },
				strings.NewReader(cliPass+"\n"), &out, &errOut, now)
			r = cliResult{code, out.String(), errOut.String()}
		}()
		go func() {
			defer wg.Done()
			<-start
			importErr = importAdmin(context.Background(), imp, cfg, slogTo(&importLog), now)
		}()
		close(start)
		wg.Wait()

		if n := adminCount(t, dir); n != 1 {
			t.Fatalf("round %d: %d admins, want 1", round, n)
		}
		_, _, err := imp.AdminByUsername(context.Background(), importName)
		importWon := err == nil
		if importWon {
			if importErr != nil {
				t.Fatalf("round %d: import won with %v", round, importErr)
			}
			if r.code != 1 || !r.has(t, "an admin already exists") {
				t.Fatalf("round %d: import won but create = %+v", round, r)
			}
			continue
		}
		if r.code != 0 {
			t.Fatalf("round %d: no import row and create = %+v", round, r)
		}
		if importErr != nil {
			t.Fatalf("round %d: create won and the import returned %v", round, importErr)
		}
		if importLog.Len() == 0 {
			t.Fatalf("round %d: create won and the import logged nothing", round)
		}
		if n := len(ignoredLines(t, importLog.String())); n != 1 {
			t.Fatalf("round %d: create won and the import logged %d warnings, want 1", round, n)
		}
	}
}

func slogTo(w io.Writer) *slog.Logger { return slog.New(slog.NewJSONHandler(w, nil)) }
