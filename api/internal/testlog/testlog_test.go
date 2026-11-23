package testlog

import "testing"

func TestLeaksUsesTheCallersAllowlist(t *testing.T) {
	allowed := []string{"time", "level", "msg", "request_id", "duration_ms", "channel"}
	for line, leaks := range map[string]int{
		`{"msg":"delivery","channel":"push"}`:                   0,
		`{"msg":"delivery","channel":"x 4321"}`:                 1,
		`{"msg":"delivery","route":"push"}`:                     1,
		`{"msg":"delivery","request_id":"a4321","time":"4321"}`: 0,
		`{"msg":"delivery","duration_ms":4321}`:                 0,
		`not json`:                                              1,
	} {
		if got := Leaks(line, allowed, "4321"); len(got) != leaks {
			t.Errorf("%s: %v, want %d leaks", line, got, leaks)
		}
	}
	if got := Leaks(`{"msg":"delivery","channel":"push"}`, []string{"msg"}); len(got) != 1 {
		t.Errorf("a field outside a narrower allowlist: %v, want 1 leak", got)
	}
}
