package fonts

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
)

const (
	regularSHA256 = "f3961a9cde016d41a4879aecda1474d3a36d6bf54fa0e4643de029cc2248b0e8"
	boldSHA256    = "87cb2d84472a7d66da659ee47b6cdb9552326e8c128245231f191b6ac72529d9"
)

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestFontHashes(t *testing.T) {
	if got := sum(Regular()); got != regularSHA256 {
		t.Errorf("Regular sha256 = %s, want %s", got, regularSHA256)
	}
	if got := sum(Bold()); got != boldSHA256 {
		t.Errorf("Bold sha256 = %s, want %s", got, boldSHA256)
	}
	r := Regular()
	r[0] ^= 0xff
	if sum(Regular()) != regularSHA256 {
		t.Error("a caller's change reached the embedded Regular bytes")
	}
}

func TestLicenceFiles(t *testing.T) {
	lic, err := os.ReadFile("LICENSE")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(lic), "Copyright 2022 The Noto Project Authors") {
		t.Error("LICENSE does not start with the Noto copyright line")
	}
	if !strings.Contains(string(lic), "SIL Open Font License, Version 1.1") {
		t.Error("LICENSE does not name the SIL Open Font License, Version 1.1")
	}
	notice, err := os.ReadFile("NOTICE")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"NotoSans-Regular.ttf", "NotoSans-Bold.ttf", regularSHA256, boldSHA256, "2.015",
		"These fonts are not covered by the MIT licence of this repository; they are licensed under the SIL Open Font License 1.1 (see LICENSE). Subsets of them are embedded in every exported PDF; the OFL does not apply to those documents."} {
		if !strings.Contains(string(notice), want) {
			t.Errorf("NOTICE lacks %q", want)
		}
	}
}

func TestLoadFailure(t *testing.T) {
	t.Cleanup(func() { resetLoad(load) })
	boom := errors.New("boom")
	calls := 0
	resetLoad(func() error { calls++; return boom })
	for i := range 3 {
		err := Load()
		if !errors.Is(err, ErrFontsUnavailable) || !errors.Is(err, boom) {
			t.Fatalf("call %d: Load() = %v, want ErrFontsUnavailable wrapping boom", i, err)
		}
	}
	if calls != 1 {
		t.Errorf("loader ran %d times, want 1", calls)
	}
	if Covers('a') {
		t.Error("Covers('a') = true after a failed load")
	}
	if out, sub := Encode("ab"); out != "??" || !sub {
		t.Errorf("Encode after a failed load = %q, %v", out, sub)
	}

	resetLoad(func() error { panic("bad table") })
	if err := Load(); !errors.Is(err, ErrFontsUnavailable) || !errors.Is(Load(), ErrFontsUnavailable) {
		t.Errorf("a panicking loader: Load() = %v", err)
	}
}

func TestLoadSucceeds(t *testing.T) {
	t.Cleanup(func() { resetLoad(load) })
	resetLoad(load)
	if err := Load(); err != nil {
		t.Fatalf("Load() = %v", err)
	}
}

func resetLoad(l func() error) {
	loader, once, loadErr = l, sync.Once{}, nil
	regularSet, boldSet = nil, nil
}
