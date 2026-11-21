package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// The primary result codes worth naming; an extended code takes its primary code's name.
var sqliteNames = map[int]string{
	sqlite3.SQLITE_BUSY:       "SQLITE_BUSY",
	sqlite3.SQLITE_LOCKED:     "SQLITE_LOCKED",
	sqlite3.SQLITE_READONLY:   "SQLITE_READONLY",
	sqlite3.SQLITE_IOERR:      "SQLITE_IOERR",
	sqlite3.SQLITE_FULL:       "SQLITE_FULL",
	sqlite3.SQLITE_CANTOPEN:   "SQLITE_CANTOPEN",
	sqlite3.SQLITE_CORRUPT:    "SQLITE_CORRUPT",
	sqlite3.SQLITE_CONSTRAINT: "SQLITE_CONSTRAINT",
}

const primaryCodeMask = 0xff

func sqliteCause(code int) string {
	name, ok := sqliteNames[code&primaryCodeMask]
	if !ok {
		name = constants.CauseSQLite
	}
	return fmt.Sprintf("%s(%d)", name, code)
}

// storeCause names a store failure without its text, which can carry SQL, paths or values.
func storeCause(err error) string {
	if err == nil {
		return ""
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		return sqliteCause(se.Code())
	}
	return fmt.Sprintf("%s%T", constants.CauseOtherPrefix, err)
}

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
