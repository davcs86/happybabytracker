package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondError(t *testing.T) {
	cases := []struct {
		field []string
		want  string
	}{
		{nil, `{"error":"bad_request"}`},
		{[]string{"name"}, `{"error":"bad_request","field":"name"}`},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		respondError(rec, http.StatusBadRequest, "bad_request", c.field...)
		if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != c.want {
			t.Errorf("%v: %d %q, want %q", c.field, rec.Code, rec.Body.String(), c.want)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
	}
}
