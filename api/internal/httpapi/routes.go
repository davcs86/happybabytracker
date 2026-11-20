package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// Access is the one class a route declares; the middleware enforces it.
type Access int

const (
	Open      Access = iota // no token
	Internal                // internal token only
	Session                 // any valid session
	Admin                   // admin session
	Parent                  // parent session
	Caregiver               // parent with any role on {kid}
	Guardian                // parent with the guardian role on {kid}
)

type Route struct {
	Method, Pattern string
	Access          Access
	Handler         http.HandlerFunc
}

type Deps struct {
	Store   *store.Store
	Cfg     config.Config
	Limiter *auth.Limiter
	Now     func() time.Time
	Log     *slog.Logger
}

// Routes is the only place routes are declared.
func Routes(d Deps) []Route {
	return []Route{
		{http.MethodGet, "/healthz", Open, Health(d.Store)},
		{http.MethodPost, "/v1/session", Internal, d.login},
		{http.MethodGet, "/v1/session", Session, d.currentSession},
		{http.MethodDelete, "/v1/session", Session, d.logout},
	}
}

func NewHandler(d Deps) http.Handler {
	return newMux(d, Routes(d))
}

// newMux sends every request, matched or not, through the middleware.
func newMux(d Deps, routes []Route) http.Handler {
	mux := http.NewServeMux()
	for _, rt := range routes {
		mux.Handle(rt.Method+" "+rt.Pattern, d.guard(rt.Method+" "+rt.Pattern, rt.Access, rt.Handler))
	}
	mux.Handle("/", d.guard("unmatched", Internal, func(w http.ResponseWriter, _ *http.Request) {
		respondError(w, http.StatusNotFound, "not_found")
	}))
	return mux
}
