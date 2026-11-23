package main

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

var now = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

func cfgWith(user, pass string) config.Config {
	return config.Config{
		AdminUsername: user, AdminPassword: pass,
		InternalAPIToken: "internal-token-0123456789abcdef0123456789abcdef",
	}
}

// restart closes and reopens the database, as a process restart does.
func restart(t *testing.T, s *store.Store, dir string) *store.Store {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func has(t *testing.T, s *store.Store, hash []byte) bool {
	t.Helper()
	_, err := s.SessionByToken(context.Background(), hash, now)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
	return err == nil
}

func TestAdminFingerprintRotation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.Close() }()
	exp := now.Add(time.Hour)

	a := cfgWith("root-admin", "admin-password-AAAAAAAA")
	if err := syncAdminFingerprint(ctx, s, a); err != nil {
		t.Fatal(err)
	}
	p, err := s.CreateParent(ctx, "ana", "Ana", now)
	if err != nil {
		t.Fatal(err)
	}
	mustSession := func(kind, parentID, hash string) {
		t.Helper()
		if _, err := s.CreateSession(ctx, kind, parentID, []byte(hash), now, exp); err != nil {
			t.Fatal(err)
		}
	}
	mustSession("admin", "", "admin-1")
	mustSession("parent", p.ID, "parent-1")

	// Same credentials: nothing changes.
	s = restart(t, s, dir)
	if err := syncAdminFingerprint(ctx, s, a); err != nil {
		t.Fatal(err)
	}
	if !has(t, s, []byte("admin-1")) {
		t.Fatal("restart with the same credentials ended the admin session")
	}

	// New password: admin sessions go, parent sessions stay.
	b := cfgWith("root-admin", "admin-password-BBBBBBBB")
	s = restart(t, s, dir)
	if err := syncAdminFingerprint(ctx, s, b); err != nil {
		t.Fatal(err)
	}
	if has(t, s, []byte("admin-1")) {
		t.Fatal("admin session survived a password change")
	}
	if !has(t, s, []byte("parent-1")) {
		t.Fatal("a parent session was removed by an admin credential change")
	}

	// Restart again with the new password: sessions made since are kept.
	mustSession("admin", "", "admin-2")
	s = restart(t, s, dir)
	if err := syncAdminFingerprint(ctx, s, b); err != nil {
		t.Fatal(err)
	}
	if !has(t, s, []byte("admin-2")) {
		t.Fatal("restart with unchanged credentials ended the admin session")
	}

	// A changed username rotates too.
	c := cfgWith("other-admin", "admin-password-BBBBBBBB")
	s = restart(t, s, dir)
	if err := syncAdminFingerprint(ctx, s, c); err != nil {
		t.Fatal(err)
	}
	if has(t, s, []byte("admin-2")) {
		t.Fatal("admin session survived a username change")
	}

	stored, ok, err := s.MetaGet(ctx, "admin_fingerprint")
	if want := auth.AdminFingerprint(c.InternalAPIToken, c.AdminUsername, c.AdminPassword); err != nil || !ok || stored != want {
		t.Fatalf("stored fingerprint = %q (%v, %v), want %q", stored, ok, err, want)
	}
}

func TestFirstStartRecordsTheFingerprint(t *testing.T) {
	ctx := context.Background()
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateSession(ctx, "admin", "", []byte("early"), now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	cfg := cfgWith("root-admin", "admin-password-AAAAAAAA")
	if err := syncAdminFingerprint(ctx, s, cfg); err != nil {
		t.Fatal(err)
	}
	if got, ok, err := s.MetaGet(ctx, "admin_fingerprint"); err != nil || !ok || got == "" {
		t.Fatalf("fingerprint not stored: %q %v %v", got, ok, err)
	}
	// With no record, a session cannot be tied to the current credentials.
	if has(t, s, []byte("early")) {
		t.Fatal("an admin session of unknown origin survived the first fingerprint")
	}
}

const collisionMsg = "ADMIN_USERNAME collides with an existing parent account"

func TestAdminUsernameCollidingWithAParentIsRefused(t *testing.T) {
	ctx := context.Background()
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateParent(ctx, "david", "David", now); err != nil {
		t.Fatal(err)
	}

	err = checkAdminUsername(ctx, s, cfgWith("david", "admin-password-AAAAAAAA"))
	if err == nil || err.Error() != collisionMsg {
		t.Fatalf("collision: %v, want %q", err, collisionMsg)
	}
	if strings.Contains(err.Error(), "david") {
		t.Fatalf("error echoes the username: %q", err)
	}

	// login normalises case and whitespace, so a config value that only differs that way collides too.
	for _, v := range []string{"David", " david ", "DAVID\t"} {
		if err := checkAdminUsername(ctx, s, cfgWith(v, "admin-password-AAAAAAAA")); err == nil || err.Error() != collisionMsg {
			t.Errorf("config value %q: %v, want %q", v, err, collisionMsg)
		}
	}

	if err := checkAdminUsername(ctx, s, cfgWith("root-admin", "admin-password-AAAAAAAA")); err != nil {
		t.Fatalf("no collision: %v", err)
	}
}

func TestConfigNeverHoldsAnUnnormalisedAdminUsername(t *testing.T) {
	for _, v := range []string{"David", " david ", "DAVID"} {
		env := map[string]string{
			"ADMIN_USERNAME": v, "ADMIN_PASSWORD": "admin-password-AAAAAAAA",
			"INTERNAL_API_TOKEN": "internal-token-0123456789abcdef0123456789abcdef", "PUBLIC_ORIGIN": "http://localhost:9212",
		}
		if _, err := config.Load(func(k string) string { return env[k] }); err == nil {
			t.Errorf("config.Load accepted %q", v)
		}
	}
}

func TestRunRefusesToStartOnACollision(t *testing.T) {
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
	env := map[string]string{
		"ADMIN_USERNAME": "david", "ADMIN_PASSWORD": "admin-password-AAAAAAAA", "DATA_DIR": dir,
		"INTERNAL_API_TOKEN": "internal-token-0123456789abcdef0123456789abcdef", "PUBLIC_ORIGIN": "http://localhost:9212",
	}
	err = run(ctx, func(k string) string { return env[k] }, io.Discard)
	if err == nil || err.Error() != collisionMsg {
		t.Fatalf("run: %v, want %q", err, collisionMsg)
	}

	// Refused before anything was changed: no fingerprint was recorded.
	s, err = store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, ok, err := s.MetaGet(ctx, "admin_fingerprint"); err != nil || ok {
		t.Fatalf("fingerprint recorded despite the refusal: %v %v", ok, err)
	}
}

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

func TestStartWithVAPIDLogsPushOn(t *testing.T) {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key := base64.RawURLEncoding.EncodeToString(k.Bytes())
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
