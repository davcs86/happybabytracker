package ntfy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/push"
)

// Result carries no text, so no topic or token can leave through it. Outcome shares push's meanings; ntfy
// never answers Gone. ConfigFault marks a 401 or 403, which no retry fixes.
type Result struct {
	Outcome     push.Outcome
	Status      int
	RetryAfter  time.Duration
	ConfigFault bool
}

type Publisher interface {
	Publish(ctx context.Context, topic, title, message, click string) Result
}

type publisher struct {
	url, token string
	now        func() time.Time
	timeout    time.Duration
	client     *http.Client
}

type option func(*publisher)

func withTimeout(d time.Duration) option { return func(p *publisher) { p.timeout = d } }

// NewPublisher panics on an empty URL. The URL is the operator's own, possibly internal (http://ntfy:2586), so the
// push package's public-address checks do not apply.
func NewPublisher(cfg Config, opts ...option) Publisher {
	if cfg.URL == "" {
		panic("ntfy: NewPublisher needs a URL")
	}
	p := &publisher{url: cfg.URL, token: cfg.Token, now: time.Now, timeout: constants.NtfyTimeout}
	for _, o := range opts {
		o(p)
	}
	p.client = &http.Client{
		Timeout: p.timeout,
		Transport: &http.Transport{
			DialContext:            (&net.Dialer{Timeout: p.timeout}).DialContext,
			TLSHandshakeTimeout:    p.timeout,
			ResponseHeaderTimeout:  p.timeout,
			ExpectContinueTimeout:  p.timeout,
			IdleConnTimeout:        constants.PushIdleConnTimeout,
			MaxIdleConnsPerHost:    1,
			MaxConnsPerHost:        constants.DeliveryConcurrency,
			MaxResponseHeaderBytes: constants.PushResponseHeaderMaxBytes,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return p
}

type body struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	Title   string `json:"title"`
	Click   string `json:"click,omitempty"`
}

func (p *publisher) Publish(ctx context.Context, topic, title, message, click string) Result {
	if topic == "" || len(message) > constants.NtfyMessageMaxBytes {
		return Result{Outcome: push.Failed}
	}
	b, err := json.Marshal(body{Topic: topic, Message: message, Title: title, Click: click})
	if err != nil {
		return Result{Outcome: push.Failed}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(b))
	if err != nil {
		return Result{Outcome: push.Failed}
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return Result{Outcome: push.Retry}
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, constants.NtfyResponseMaxBytes)
	return classify(resp, p.now())
}

func classify(resp *http.Response, now time.Time) Result {
	r := Result{Status: resp.StatusCode}
	switch c := resp.StatusCode; {
	case c >= 200 && c < 300:
		r.Outcome = push.Sent
	case c == http.StatusUnauthorized || c == http.StatusForbidden:
		r.Outcome, r.ConfigFault = push.Failed, true
	case c == http.StatusTooManyRequests || c >= 500:
		r.Outcome = push.Retry
		r.RetryAfter = push.ParseRetryAfter(resp.Header.Get("Retry-After"), now)
	default:
		r.Outcome = push.Failed
	}
	return r
}
