package httpapi

import (
	"net/http"

	"github.com/davcs86/happybabytracker/api/internal/cause"
	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// The shared classifier under the names this package's callers and tests use.
var (
	sqliteCause = cause.SQLite
	storeCause  = cause.Of
)

// recorder finds the request's log holder behind any wrappers; nil when w did not come from guard.
func recorder(w http.ResponseWriter) *statusRecorder {
	for w != nil {
		if rec, ok := w.(*statusRecorder); ok {
			return rec
		}
		u, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return nil
		}
		w = u.Unwrap()
	}
	return nil
}

// noteCause hands the cause to the request's one log line; the first one wins.
func noteCause(w http.ResponseWriter, err error) {
	if rec := recorder(w); rec != nil && rec.cause == "" {
		rec.cause = storeCause(err)
	}
}

// noteSlideFailed marks the request's log line; the request itself is served.
func noteSlideFailed(w http.ResponseWriter, err error) {
	noteCause(w, err)
	if rec := recorder(w); rec != nil {
		rec.slide = constants.SlideFailed
	}
}
