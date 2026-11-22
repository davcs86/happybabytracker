package growth

import (
	"runtime"
	"strings"
	"testing"
)

func TestSourceInfoComplete(t *testing.T) {
	if len(Sources) != 2 || Sources[0] != SourceWHO || Sources[1] != SourceCDC {
		t.Fatalf("Sources = %v", Sources)
	}
	for _, s := range Sources {
		info, ok := SourceInfo(s)
		if !ok {
			t.Fatalf("%s: no info", s)
		}
		if info.ID != s {
			t.Errorf("%s: ID %q", s, info.ID)
		}
		for name, v := range map[string]string{
			"Name": info.Name, "Attribution": info.Attribution, "Licence": info.Licence, "Version": info.Version,
			"OriginURL": info.OriginURL, "Retrieved": info.Retrieved, "Notice": info.Notice,
		} {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s: empty %s", s, name)
			}
		}
		if !strings.HasPrefix(info.OriginURL, "https://") {
			t.Errorf("%s: origin %q", s, info.OriginURL)
		}
		if info.Retrieved != "2026-11-22" {
			t.Errorf("%s: retrieved %q", s, info.Retrieved)
		}
	}
	who, _ := SourceInfo(SourceWHO)
	if !strings.Contains(who.Notice, "CC BY-NC-SA 3.0 IGO") || !strings.Contains(who.LicenceText, "legalcode") {
		t.Error("WHO notice or licence text missing")
	}
	cdc, _ := SourceInfo(SourceCDC)
	if !strings.Contains(cdc.Notice, "Source: CDC") || cdc.LicenceText != "" {
		t.Errorf("CDC notice %q, licence text %q", cdc.Notice, cdc.LicenceText)
	}
	if _, ok := SourceInfo(Source("uk-who")); ok {
		t.Error("uk-who has info")
	}
}

func TestHeapAfterLoad(t *testing.T) {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	t.Logf("HeapInuse before %d B, after loading both sources %d B (delta %d B)",
		before.HeapInuse, after.HeapInuse, int64(after.HeapInuse)-int64(before.HeapInuse))
	if after.HeapInuse > 2<<20 {
		t.Errorf("HeapInuse %d B after loading, want < 2 MiB", after.HeapInuse)
	}
}
