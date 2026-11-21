package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
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

func TestDecodeJSON(t *testing.T) {
	type in struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	cases := []struct {
		name   string
		body   string
		ok     bool
		status int
		want   string
	}{
		{"valid", `{"name":"ana","n":3}`, true, 200, ""},
		{"valid with trailing newline", "{\"name\":\"ana\"}\n", true, 200, ""},
		{"unknown field", `{"name":"ana","extra":1}`, false, 422, `{"error":"unknown_field","field":"extra"}`},
		{"malformed", `{"name":`, false, 400, `{"error":"bad_request"}`},
		{"empty", ``, false, 400, `{"error":"bad_request"}`},
		{"wrong type", `{"name":5}`, false, 400, `{"error":"bad_request"}`},
		{"not an object", `[1]`, false, 400, `{"error":"bad_request"}`},
		{"trailing value", `{"name":"a"}{"name":"b"}`, false, 400, `{"error":"bad_request"}`},
		{"trailing garbage", `{"name":"a"} x`, false, 400, `{"error":"bad_request"}`},
		{"over the cap", `{"name":"` + strings.Repeat("a", constants.MaxBodyBytes) + `"}`, false, 413, `{"error":"body_too_large"}`},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(c.body))
		var v in
		got := decodeJSON(rec, r, &v)
		if got != c.ok {
			t.Errorf("%s: ok = %v, want %v", c.name, got, c.ok)
		}
		if c.ok {
			if rec.Body.Len() != 0 || v.Name != "ana" {
				t.Errorf("%s: wrote %q, decoded %+v", c.name, rec.Body.String(), v)
			}
			continue
		}
		if rec.Code != c.status || strings.TrimSpace(rec.Body.String()) != c.want {
			t.Errorf("%s: %d %q, want %d %s", c.name, rec.Code, rec.Body.String(), c.status, c.want)
		}
	}
}
