package auth

import (
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/config"
)

const (
	adminToken = "internal-token-0123456789abcdef0123"
	adminUser  = "ana"
	adminPass  = "correct horse battery"
)

func TestAdminFingerprintChanges(t *testing.T) {
	base := AdminFingerprint(adminToken, adminUser, adminPass)
	if base == AdminFingerprint(adminToken, "bob", adminPass) {
		t.Fatal("fingerprint ignores the username")
	}
	if base == AdminFingerprint(adminToken, adminUser, adminPass+"x") {
		t.Fatal("fingerprint ignores the password")
	}
	if base == AdminFingerprint(adminToken+"x", adminUser, adminPass) {
		t.Fatal("fingerprint ignores the key")
	}
	if base != AdminFingerprint(adminToken, adminUser, adminPass) {
		t.Fatal("fingerprint is not deterministic")
	}
}

func TestAdminFingerprintValue(t *testing.T) {
	const want = "6f92bd4d1dac0bb2c1d4ae8919feaaa4d06176964ca5626bb5e49dbfe1245783"
	if got := AdminFingerprint(adminToken, adminUser, adminPass); got != want {
		t.Fatalf("fingerprint = %s, want %s", got, want)
	}
}

func TestAdminFingerprintSeparatesFields(t *testing.T) {
	if AdminFingerprint(adminToken, "ab", "c") == AdminFingerprint(adminToken, "a", "bc") {
		t.Fatal("username/password boundary is ambiguous")
	}
}

func TestCheckAdmin(t *testing.T) {
	cfg := config.Config{AdminUsername: adminUser, AdminPassword: adminPass}
	cases := []struct {
		name, user, pass string
		want             bool
	}{
		{"right pair", adminUser, adminPass, true},
		{"right username, wrong password", adminUser, "wrong password!!", false},
		{"wrong username, right password", "bob", adminPass, false},
		{"both wrong", "bob", "wrong password!!", false},
		{"empty both", "", "", false},
		{"password prefix", adminUser, adminPass[:len(adminPass)-1], false},
		{"password suffix", adminUser, adminPass + "x", false},
		{"username not normalised here", "ANA", adminPass, false},
	}
	for _, c := range cases {
		if got := CheckAdmin(cfg, c.user, c.pass); got != c.want {
			t.Errorf("%s: CheckAdmin = %v, want %v", c.name, got, c.want)
		}
	}
}
