package auth

import "testing"

func TestNormalizeUsername(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{" Ana ", "ana", true},
		{"ANA.b-c_9", "ana.b-c_9", true},
		{"\tana\n", "ana", true},
		{"ab", "", false},
		{"a b", "", false},
		{"Ünï", "", false},
		{"", "", false},
		{"   ", "", false},
		{"ana ", "", false},
		{"Kbc", "", false},
		{"ana@home", "", false},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "", false},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
	}
	for _, c := range cases {
		got, ok := NormalizeUsername(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizeUsername(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestNormalizeUsernameRejectsKelvinSign(t *testing.T) {
	if got, ok := NormalizeUsername("Kbc"); ok || got != "" {
		t.Fatalf("Kelvin sign + bc = (%q, %v), want (\"\", false)", got, ok)
	}
}
