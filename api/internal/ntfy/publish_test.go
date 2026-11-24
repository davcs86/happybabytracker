package ntfy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/push"
)

const (
	zqTopic = "zqprefix-AbCdEfGhIjKlMnOpQrStUv"
	zqToken = "tk_zqsecretzqsecretzqsecret0123"
)

type got struct {
	method, path, auth, ctype string
	hasAuth                   bool
	raw                       []byte
}

// serve answers every request with status (and header, when set) and records the last one.
func serve(t *testing.T, status int, header http.Header) (*httptest.Server, *got) {
	t.Helper()
	g := &got{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.method, g.path, g.ctype = r.Method, r.URL.Path, r.Header.Get("Content-Type")
		_, g.hasAuth = r.Header["Authorization"]
		g.auth = r.Header.Get("Authorization")
		g.raw, _ = io.ReadAll(r.Body)
		for k, v := range header {
			w.Header()[k] = v
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, g
}

func pub(srv *httptest.Server, token string, opts ...option) Publisher {
	return NewPublisher(Config{URL: srv.URL, TopicPrefix: "zqprefix", Token: token}, opts...)
}

func TestPublishBody(t *testing.T) {
	srv, g := serve(t, 200, nil)
	res := pub(srv, "").Publish(context.Background(), zqTopic, "Medication due for Mia", "Medication due for Mia", "https://hbt.example.org/kids/k1/medication")
	if res.Outcome != push.Sent || res.Status != 200 || res.ConfigFault {
		t.Fatalf("result %+v, want Sent 200", res)
	}
	if g.method != "POST" || g.path != "/" || g.ctype != "application/json" {
		t.Fatalf("request %s %s %q, want POST / application/json", g.method, g.path, g.ctype)
	}
	var m map[string]any
	if err := json.Unmarshal(g.raw, &m); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"topic": zqTopic, "title": "Medication due for Mia", "message": "Medication due for Mia",
		"click": "https://hbt.example.org/kids/k1/medication",
	}
	if len(m) != len(want) {
		t.Fatalf("body %s has %d keys, want %d", g.raw, len(m), len(want))
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
}

func TestPublishOmitsAnEmptyClick(t *testing.T) {
	srv, g := serve(t, 200, nil)
	pub(srv, "").Publish(context.Background(), zqTopic, "t", "m", "")
	if strings.Contains(string(g.raw), "click") {
		t.Fatalf("body %s carries click", g.raw)
	}
}

func TestPublishAuthorizationOnlyWithAToken(t *testing.T) {
	srv, g := serve(t, 200, nil)
	pub(srv, "").Publish(context.Background(), zqTopic, "t", "m", "")
	if g.hasAuth {
		t.Fatalf("Authorization %q sent without a token", g.auth)
	}
	pub(srv, zqToken).Publish(context.Background(), zqTopic, "t", "m", "")
	if g.auth != "Bearer "+zqToken {
		t.Fatalf("Authorization %q, want the bearer token", g.auth)
	}
	if strings.Contains(string(g.raw), zqToken) {
		t.Fatal("the token is in the body")
	}
}

func TestPublishStatuses(t *testing.T) {
	cases := []struct {
		status int
		want   Result
	}{
		{200, Result{Outcome: push.Sent, Status: 200}},
		{204, Result{Outcome: push.Sent, Status: 204}},
		{400, Result{Outcome: push.Failed, Status: 400}},
		{401, Result{Outcome: push.Failed, Status: 401, ConfigFault: true}},
		{403, Result{Outcome: push.Failed, Status: 403, ConfigFault: true}},
		{404, Result{Outcome: push.Failed, Status: 404}},
		{413, Result{Outcome: push.Failed, Status: 413}},
		{429, Result{Outcome: push.Retry, Status: 429}},
		{500, Result{Outcome: push.Retry, Status: 500}},
		{502, Result{Outcome: push.Retry, Status: 502}},
		{503, Result{Outcome: push.Retry, Status: 503}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			srv, _ := serve(t, tc.status, nil)
			if res := pub(srv, zqToken).Publish(context.Background(), zqTopic, "t", "m", ""); res != tc.want {
				t.Fatalf("result %+v, want %+v", res, tc.want)
			}
		})
	}
}

func TestPublishRetryAfter(t *testing.T) {
	srv, _ := serve(t, 429, http.Header{"Retry-After": {"90"}})
	if res := pub(srv, "").Publish(context.Background(), zqTopic, "t", "m", ""); res.Outcome != push.Retry || res.RetryAfter != 90*time.Second {
		t.Fatalf("result %+v, want Retry after 90s", res)
	}
	srv, _ = serve(t, 503, http.Header{"Retry-After": {"86400"}})
	if res := pub(srv, "").Publish(context.Background(), zqTopic, "t", "m", ""); res.RetryAfter != constants.ReminderExpiry {
		t.Fatalf("RetryAfter %v, want it capped at %v", res.RetryAfter, constants.ReminderExpiry)
	}
	srv, _ = serve(t, 429, http.Header{"Retry-After": {"zq"}})
	if res := pub(srv, "").Publish(context.Background(), zqTopic, "t", "m", ""); res.RetryAfter != 0 {
		t.Fatalf("RetryAfter %v for an unreadable header, want 0", res.RetryAfter)
	}
}

func TestPublishDoesNotFollowRedirects(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer target.Close()
	for _, code := range []int{301, 302, 307, 308} {
		srv, _ := serve(t, code, http.Header{"Location": {target.URL}})
		res := pub(srv, zqToken).Publish(context.Background(), zqTopic, "t", "m", "")
		if res.Outcome != push.Failed || res.Status != code || res.ConfigFault {
			t.Fatalf("%d: result %+v, want Failed %d", code, res, code)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("a redirect target was requested %d times", hits.Load())
	}
}

func TestPublishNetworkFaultsRetryAndLeakNothing(t *testing.T) {
	srv, _ := serve(t, 200, nil)
	p := pub(srv, zqToken)
	srv.Close()
	res := p.Publish(context.Background(), zqTopic, "t", "m", "")
	if res.Outcome != push.Retry || res.Status != 0 {
		t.Fatalf("closed server: result %+v, want Retry with no status", res)
	}
	if s := fmt.Sprintf("%+v %#v", res, res); strings.Contains(s, "zqprefix") || strings.Contains(s, "zqsecret") {
		t.Fatalf("result %s carries the topic or the token", s)
	}
}

func TestPublishTimeoutRetries(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-time.After(10 * time.Second):
		}
	}))
	defer srv.Close()
	defer close(release)
	start := time.Now()
	res := pub(srv, "", withTimeout(100*time.Millisecond)).Publish(context.Background(), zqTopic, "t", "m", "")
	if res.Outcome != push.Retry {
		t.Fatalf("result %+v, want Retry", res)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("took %v, want about the 100ms timeout", d)
	}
}

func TestPublishDefaultTimeout(t *testing.T) {
	p := NewPublisher(Config{URL: "http://127.0.0.1:1", TopicPrefix: "zqprefix"}).(*publisher)
	if p.client.Timeout != constants.NtfyTimeout {
		t.Fatalf("timeout %v, want %v", p.client.Timeout, constants.NtfyTimeout)
	}
}

func TestPublishHonoursACancelledContext(t *testing.T) {
	srv, _ := serve(t, 200, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res := pub(srv, "").Publish(ctx, zqTopic, "t", "m", ""); res.Outcome != push.Retry {
		t.Fatalf("result %+v, want Retry", res)
	}
}

func TestPublishReadsAtMostTheCapOfTheResponse(t *testing.T) {
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(strings.Repeat("x", 4*constants.NtfyResponseMaxBytes)))
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
		close(done)
	}))
	defer srv.Close()
	start := time.Now()
	res := pub(srv, "").Publish(context.Background(), zqTopic, "t", "m", "")
	if res.Outcome != push.Sent {
		t.Fatalf("result %+v, want Sent", res)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("took %v: the body was read to the end", d)
	}
	<-done
}

func TestPublishRefusesWithoutSending(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()
	p := pub(srv, "")
	max := strings.Repeat("a", constants.NtfyMessageMaxBytes)
	if res := p.Publish(context.Background(), zqTopic, "t", max, ""); res.Outcome != push.Sent {
		t.Fatalf("message at the limit: %+v, want Sent", res)
	}
	for name, c := range map[string][4]string{
		"message over the limit": {zqTopic, "t", max + "a", ""},
		"multibyte over":         {zqTopic, "t", strings.Repeat("é", constants.NtfyMessageMaxBytes/2+1), ""},
		"empty topic":            {"", "t", "m", ""},
	} {
		before := hits.Load()
		res := p.Publish(context.Background(), c[0], c[1], c[2], c[3])
		if res.Outcome != push.Failed || res.ConfigFault || hits.Load() != before {
			t.Fatalf("%s: result %+v, requests %d, want Failed and no request", name, res, hits.Load()-before)
		}
	}
}

func TestPublisherTransport(t *testing.T) {
	p := NewPublisher(Config{URL: "http://ntfy:2586", TopicPrefix: "zqprefix"}).(*publisher)
	tr, ok := p.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is %T", p.client.Transport)
	}
	if tr.Proxy != nil {
		t.Fatal("the transport honours a proxy")
	}
	if tr.MaxConnsPerHost != constants.DeliveryConcurrency || tr.ResponseHeaderTimeout != constants.NtfyTimeout ||
		tr.TLSHandshakeTimeout != constants.NtfyTimeout || tr.MaxResponseHeaderBytes != constants.PushResponseHeaderMaxBytes {
		t.Fatalf("transport limits %+v", tr)
	}
}

func TestNewPublisherNeedsAURL(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewPublisher accepted an empty URL")
		}
	}()
	NewPublisher(Config{})
}
