package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/store"
)

func TestHealth(t *testing.T) {
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	h := Health(s)
	get := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		return rec
	}

	rec := get()
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("open store: %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if rec := get(); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed store: %d, want 503", rec.Code)
	}
}
