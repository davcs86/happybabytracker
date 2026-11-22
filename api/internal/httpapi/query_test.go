package httpapi

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

var queryNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func queryReq(rawQuery string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/v1/kids/k/feeds?"+rawQuery, nil)
}

func TestParseListQuery(t *testing.T) {
	from := "2026-05-01T00:00:00Z"
	to := "2026-05-02T00:00:00+02:00"
	okCursor := encodeCursor(cursor{At: 1780000000, ID: "abc-123"})

	t.Run("defaults", func(t *testing.T) {
		q, bad := parseListQuery(queryReq(""), queryNow)
		if bad != "" {
			t.Fatalf("invalid %q", bad)
		}
		if !q.From.Equal(time.Unix(0, 0)) {
			t.Errorf("From = %v, want the epoch", q.From)
		}
		if want := queryNow.Add(constants.MaxFutureSkew); !q.To.Equal(want) {
			t.Errorf("To = %v, want %v", q.To, want)
		}
		if q.Limit != constants.TrackerPageDefault || q.After != nil {
			t.Errorf("Limit = %d, After = %v", q.Limit, q.After)
		}
	})

	t.Run("all given", func(t *testing.T) {
		v := url.Values{"from": {from}, "to": {to}, "limit": {"100"}, "cursor": {okCursor}}
		q, bad := parseListQuery(queryReq(v.Encode()), queryNow)
		if bad != "" {
			t.Fatalf("invalid %q", bad)
		}
		if !q.From.Equal(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) || !q.To.Equal(time.Date(2026, 5, 1, 22, 0, 0, 0, time.UTC)) {
			t.Errorf("range = [%v, %v)", q.From, q.To)
		}
		if q.Limit != 100 {
			t.Errorf("Limit = %d", q.Limit)
		}
		if q.After == nil || *q.After != (cursor{At: 1780000000, ID: "abc-123"}) {
			t.Errorf("After = %v", q.After)
		}
	})

	t.Run("limit bounds", func(t *testing.T) {
		q, bad := parseListQuery(queryReq("limit=1"), queryNow)
		if bad != "" || q.Limit != 1 {
			t.Fatalf("limit=1: %d, %q", q.Limit, bad)
		}
	})

	t.Run("equal from and to is an empty range, not an error", func(t *testing.T) {
		if _, bad := parseListQuery(queryReq("from="+from+"&to="+from), queryNow); bad != "" {
			t.Fatalf("invalid %q", bad)
		}
	})

	invalid := []struct{ name, query, field string }{
		{"limit zero", "limit=0", "limit"},
		{"limit above max", "limit=101", "limit"},
		{"limit text", "limit=abc", "limit"},
		{"limit negative", "limit=-1", "limit"},
		{"limit signed", "limit=%2B5", "limit"},
		{"limit empty", "limit=", "limit"},
		{"limit huge", "limit=99999999999999999999", "limit"},
		{"from after to", "from=" + url.QueryEscape("2026-05-02T00:00:00Z") + "&to=" + url.QueryEscape("2026-05-01T00:00:00Z"), "from"},
		{"from after default to", "from=" + url.QueryEscape("2027-01-01T00:00:00Z"), "from"},
		{"from not RFC3339", "from=2026-05-01", "from"},
		{"from empty", "from=", "from"},
		{"to not RFC3339", "to=yesterday", "to"},
		{"to date only", "to=2026-05-01", "to"},
		{"cursor garbage", "cursor=!!!", "cursor"},
		{"cursor empty", "cursor=", "cursor"},
		{"first invalid wins: from before to", "from=x&to=y&limit=0&cursor=!", "from"},
		{"first invalid wins: to before limit", "to=y&limit=0&cursor=!", "to"},
		{"first invalid wins: limit before cursor", "limit=0&cursor=!", "limit"},
	}
	for _, c := range invalid {
		t.Run(c.name, func(t *testing.T) {
			_, bad := parseListQuery(queryReq(c.query), queryNow)
			if bad != c.field {
				t.Fatalf("invalid field = %q, want %q", bad, c.field)
			}
		})
	}
}

func TestCursorRoundTrip(t *testing.T) {
	for _, c := range []cursor{
		{At: 0, ID: "a"},
		{At: 1780000000, ID: "0198-abcd-EF"},
		{At: 1<<63 - 1, ID: strings.Repeat("Z", 64)},
	} {
		s := encodeCursor(c)
		if strings.ContainsAny(s, "=+/") {
			t.Errorf("%q is not unpadded base64url", s)
		}
		got, err := decodeCursor(s)
		if err != nil {
			t.Fatalf("decode %q: %v", s, err)
		}
		if got != c {
			t.Errorf("got %v, want %v", got, c)
		}
	}
}

func TestBadCursor(t *testing.T) {
	enc := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	bad := map[string]string{
		"empty":            "",
		"not base64":       "!!!not base64!!!",
		"padded":           base64.URLEncoding.EncodeToString([]byte("12:a")),
		"std alphabet":     "Pz8_Pw==",
		"no separator":     enc("12345"),
		"empty id":         enc("12:"),
		"empty at":         enc(":abc"),
		"negative at":      enc("-1:abc"),
		"signed at":        enc("+1:abc"),
		"leading zero at":  enc("01:abc"),
		"text at":          enc("x:abc"),
		"overflow at":      enc("9223372036854775808:abc"),
		"unsafe id":        enc("1:a/b"),
		"id with space":    enc("1:a b"),
		"id with colon":    enc("1:a:b"),
		"id with newline":  enc("1:a\nb"),
		"id too long":      enc("1:" + strings.Repeat("a", 65)),
		"id with unicode":  enc("1:é"),
		"id with dot":      enc("1:a.b"),
		"id with under":    enc("1:a_b"),
		"extra whitespace": enc(" 1:abc"),
	}
	for name, s := range bad {
		t.Run(name, func(t *testing.T) {
			if c, err := decodeCursor(s); err == nil {
				t.Fatalf("decodeCursor(%q) accepted: %v", s, c)
			}
		})
	}
}
