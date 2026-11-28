package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/config"
)

// setMode switches the kit's AUTH_MODE and rebuilds its handler.
func (k *kit) setMode(m config.AuthMode) {
	k.t.Helper()
	k.Cfg.AuthMode = m
	k.Handler = NewHandler(k.deps())
}

func TestAuthModeRoute(t *testing.T) {
	k := newKit(t)
	for _, m := range []config.AuthMode{config.AuthLocal, config.AuthOIDC, config.AuthBoth} {
		k.setMode(m)
		rec := k.do(http.MethodGet, "/v1/auth", "", nil)
		if want := `{"mode":"` + string(m) + `"}`; rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
			t.Errorf("%s: %d %q, want 200 %s", m, rec.Code, rec.Body.String(), want)
		}
	}
	r := k.newReq(http.MethodGet, "/v1/auth", nil)
	r.Header.Del("Authorization")
	if rec := k.serve(r); rec.Code != http.StatusUnauthorized {
		t.Errorf("without the internal token: %d, want 401", rec.Code)
	}
}
