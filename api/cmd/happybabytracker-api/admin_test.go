package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

const collisionMsg = "ADMIN_USERNAME collides with an existing parent account"

const ignoredMsg = "admin credentials in the environment are ignored"

func startWith(t *testing.T, env map[string]string) (*app, string) {
	t.Helper()
	var out bytes.Buffer
	a, err := start(context.Background(), func(k string) string { return env[k] }, &out)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return a, out.String()
}

func adminVerifies(t *testing.T, s *store.Store, username, password string) bool {
	t.Helper()
	_, hash, err := s.AdminByUsername(context.Background(), username)
	if err != nil {
		t.Fatal(err)
	}
	return auth.VerifyPassword(hash, password)
}

// ignoredLines returns the lines carrying the ignored-credentials message.
func ignoredLines(t *testing.T, out string) []string {
	t.Helper()
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		if m["msg"] == ignoredMsg {
			if m["level"] != "WARN" {
				t.Errorf("level = %v, want WARN", m["level"])
			}
			got = append(got, line)
		}
	}
	return got
}

func TestImportCreatesTheAdminOnce(t *testing.T) {
	a, out := startWith(t, baseEnv(t.TempDir()))
	defer a.store.Close()
	if ok, err := a.store.HasAdmin(context.Background()); err != nil || !ok {
		t.Fatalf("HasAdmin = %v, %v; want true", ok, err)
	}
	if !adminVerifies(t, a.store, "root-admin", "admin-password-AAAAAAAA") {
		t.Fatal("the imported admin password does not verify")
	}
	if n := len(ignoredLines(t, out)); n != 0 {
		t.Fatalf("%d ignored warnings on the first start, want 0", n)
	}
}

func TestImportIsIgnoredOnceAnAdminExists(t *testing.T) {
	dir := t.TempDir()
	a, _ := startWith(t, baseEnv(dir))
	if err := a.store.Close(); err != nil {
		t.Fatal(err)
	}

	env := baseEnv(dir)
	env["ADMIN_PASSWORD"] = "admin-password-BBBBBBBB"
	a, out := startWith(t, env)
	if !adminVerifies(t, a.store, "root-admin", "admin-password-AAAAAAAA") {
		t.Error("the stored admin password changed")
	}
	lines := ignoredLines(t, out)
	if len(lines) != 1 {
		t.Fatalf("ignored warnings %q, want exactly one", lines)
	}
	for _, v := range []string{"root-admin", "admin-password-BBBBBBBB", "admin-password-AAAAAAAA"} {
		if strings.Contains(out, v) {
			t.Errorf("start logged %q", v)
		}
	}
	if err := a.store.Close(); err != nil {
		t.Fatal(err)
	}

	env = baseEnv(dir)
	delete(env, "ADMIN_USERNAME")
	delete(env, "ADMIN_PASSWORD")
	a, out = startWith(t, env)
	defer a.store.Close()
	if n := len(ignoredLines(t, out)); n != 0 {
		t.Fatalf("%d ignored warnings with both variables unset, want 0", n)
	}
}

func TestImportRequiresTheVariablesWhileEmpty(t *testing.T) {
	for _, tc := range []struct {
		name, variable, val string
	}{
		{"username unset", "ADMIN_USERNAME", "\x00unset"},
		{"username upper case", "ADMIN_USERNAME", "David"},
		{"username padded", "ADMIN_USERNAME", " david "},
		{"username too short", "ADMIN_USERNAME", "qz"},
		{"password unset", "ADMIN_PASSWORD", "\x00unset"},
		{"password 15 runes", "ADMIN_PASSWORD", "wxyzwxyzwxyzwxy"},
		{"password 15 multibyte runes", "ADMIN_PASSWORD", strings.Repeat("é", 15)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := baseEnv(t.TempDir())
			if tc.val == "\x00unset" {
				delete(env, tc.variable)
			} else {
				env[tc.variable] = tc.val
			}
			_, err := start(context.Background(), func(k string) string { return env[k] }, &bytes.Buffer{})
			if err == nil {
				t.Fatal("start accepted the value")
			}
			if !strings.Contains(err.Error(), tc.variable) {
				t.Errorf("error %q does not name %s", err, tc.variable)
			}
			if tc.val != "\x00unset" && strings.Contains(err.Error(), strings.TrimSpace(tc.val)) {
				t.Errorf("error %q echoes the value", err)
			}
		})
	}
}

func TestImportRefusesAParentsName(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateParent(ctx, "david", "David", now); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	env := baseEnv(dir)
	env["ADMIN_USERNAME"] = "david"
	err = run(ctx, func(k string) string { return env[k] }, &bytes.Buffer{})
	if err == nil || err.Error() != collisionMsg {
		t.Fatalf("run: %v, want %q", err, collisionMsg)
	}
	if strings.Contains(err.Error(), "david") {
		t.Fatalf("error echoes the username: %q", err)
	}
	s, err = store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if ok, err := s.HasAdmin(ctx); err != nil || ok {
		t.Fatalf("HasAdmin = %v, %v; want false", ok, err)
	}
}
