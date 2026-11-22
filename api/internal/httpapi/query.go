package httpapi

import (
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// cursor names the last entry of a page; the next page is strictly older.
type cursor struct {
	At int64
	ID string
}

// listQuery is the parsed range, page size and position of a list request; the range is [From, To).
type listQuery struct {
	From, To time.Time
	Limit    int
	After    *cursor
}

var (
	errBadCursor    = errors.New("malformed cursor")
	cursorIDPattern = regexp.MustCompile(constants.CursorIDPattern)
)

func encodeCursor(c cursor) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(c.At, 10) + ":" + c.ID))
}

func decodeCursor(s string) (cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, errBadCursor
	}
	at, id, ok := strings.Cut(string(raw), ":")
	n, err := strconv.ParseInt(at, 10, 64)
	if !ok || err != nil || n < 0 || strconv.FormatInt(n, 10) != at || !cursorIDPattern.MatchString(id) {
		return cursor{}, errBadCursor
	}
	return cursor{At: n, ID: id}, nil
}

// parseListQuery reads from, to, limit and cursor; the second result names the first invalid one, in that order, or "".
func parseListQuery(r *http.Request, now time.Time) (listQuery, string) {
	v := r.URL.Query()
	q := listQuery{From: time.Unix(0, 0).UTC(), To: now.Add(constants.MaxFutureSkew), Limit: constants.TrackerPageDefault}
	var err error
	if v.Has("from") {
		if q.From, err = time.Parse(time.RFC3339, v.Get("from")); err != nil {
			return listQuery{}, "from"
		}
	}
	if v.Has("to") {
		if q.To, err = time.Parse(time.RFC3339, v.Get("to")); err != nil {
			return listQuery{}, "to"
		}
	}
	if q.From.After(q.To) {
		return listQuery{}, "from"
	}
	if v.Has("limit") {
		n, err := strconv.ParseUint(v.Get("limit"), 10, 31)
		if err != nil || n < 1 || n > constants.TrackerPageMax {
			return listQuery{}, "limit"
		}
		q.Limit = int(n)
	}
	if v.Has("cursor") {
		c, err := decodeCursor(v.Get("cursor"))
		if err != nil {
			return listQuery{}, "cursor"
		}
		q.After = &c
	}
	return q, ""
}
