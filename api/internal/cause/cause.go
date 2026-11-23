// Package cause names a failure for a log line without its text, which can carry SQL, paths or values.
package cause

import (
	"errors"
	"fmt"

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

// SQLite names a SQLite result code.
func SQLite(code int) string {
	name, ok := sqliteNames[code&primaryCodeMask]
	if !ok {
		name = constants.CauseSQLite
	}
	return fmt.Sprintf("%s(%d)", name, code)
}

// Of names err: its SQLite code, else its type; "" for nil.
func Of(err error) string {
	if err == nil {
		return ""
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		return SQLite(se.Code())
	}
	return fmt.Sprintf("%s%T", constants.CauseOtherPrefix, err)
}
