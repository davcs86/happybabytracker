package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

const (
	headerSession = "X-Session-Token"
	bearerPrefix  = "Bearer "

	kindAdmin  = "admin"
	kindParent = "parent"
	kindNone   = "none"
)

type ctxKey int

const (
	principalKey ctxKey = iota
	kidRoleKey
)

// PrincipalFrom is the zero Principal on a route that resolves no session.
func PrincipalFrom(ctx context.Context) model.Principal {
	p, _ := ctx.Value(principalKey).(model.Principal)
	return p
}

// KidRoleFrom is the zero Role until a Caregiver or Guardian route sets one.
func KidRoleFrom(ctx context.Context) model.Role {
	r, _ := ctx.Value(kidRoleKey).(model.Role)
	return r
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// guard wraps h: body cap, internal token, principal, access class, then one log line.
func (d Deps) guard(route string, access Access, h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		r.Body = http.MaxBytesReader(rec, r.Body, constants.MaxBodyBytes)

		ctx, p, ok := d.admit(rec, r, access)
		kind := kindNone
		if p.Kind != "" {
			kind = p.Kind
		}
		if ok {
			h(rec, r.WithContext(ctx))
		}
		d.Log.Info("request",
			"route", route,
			"status", rec.status,
			"principal_kind", kind,
			"request_id", model.NewID(),
			"duration_ms", time.Since(start).Milliseconds())
	})
}

// admit enforces the access class, writing the refusal itself; the principal is zero when none was resolved.
func (d Deps) admit(w http.ResponseWriter, r *http.Request, access Access) (context.Context, model.Principal, bool) {
	ctx := r.Context()
	if access != Open && !d.internalTokenOK(r) {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return ctx, model.Principal{}, false
	}
	switch access {
	case Open, Internal:
		return ctx, model.Principal{}, true
	case Session, Admin, Parent, Caregiver, Guardian:
	default: // an unknown class is a bug, never an open route
		respondError(w, http.StatusInternalServerError, "internal")
		return ctx, model.Principal{}, false
	}

	token := r.Header.Get(headerSession)
	if token == "" {
		respondError(w, http.StatusUnauthorized, "no_session")
		return ctx, model.Principal{}, false
	}
	now := d.Now()
	p, err := d.Store.SessionByToken(ctx, auth.HashToken(token), now)
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusUnauthorized, "no_session")
		return ctx, model.Principal{}, false
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal")
		return ctx, model.Principal{}, false
	}
	perKid := access == Caregiver || access == Guardian
	if (access == Admin && p.Kind != kindAdmin) || ((access == Parent || perKid) && p.Kind != kindParent) {
		respondError(w, http.StatusForbidden, "forbidden")
		return ctx, p, false
	}
	if perKid {
		// Unknown kid and no grant must stay byte-identical: both are this one 404.
		role, err := d.Store.RoleFor(ctx, r.PathValue("kid"), p.ParentID)
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "not_found")
			return ctx, p, false
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal")
			return ctx, p, false
		}
		if access == Guardian && role != model.RoleGuardian {
			respondError(w, http.StatusForbidden, "forbidden")
			return ctx, p, false
		}
		ctx = context.WithValue(ctx, kidRoleKey, role)
	}

	if p.Kind == kindParent && p.ExpiresAt.Sub(now) < constants.ParentSessionTTL-constants.SessionTouchInterval {
		expires := now.Add(constants.ParentSessionTTL)
		if err := d.Store.TouchSession(ctx, p.SessionID, now, expires); err != nil {
			respondError(w, http.StatusInternalServerError, "internal")
			return ctx, p, false
		}
		p.ExpiresAt = expires
	}
	return context.WithValue(ctx, principalKey, p), p, true
}

// internalTokenOK compares digests so neither the token's length nor its content leaks through timing.
func (d Deps) internalTokenOK(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	if d.Cfg.InternalAPIToken == "" || len(h) < len(bearerPrefix) || !strings.EqualFold(h[:len(bearerPrefix)], bearerPrefix) {
		return false
	}
	got := sha256.Sum256([]byte(h[len(bearerPrefix):]))
	want := sha256.Sum256([]byte(d.Cfg.InternalAPIToken))
	return subtle.ConstantTimeCompare(got[:], want[:]) == 1
}
