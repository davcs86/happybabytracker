package push

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"syscall"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Subscription is one device; ID is the device id and is the only field a caller may report.
type Subscription struct{ ID, Endpoint, P256dh, Auth string }

type Message struct {
	Payload []byte
	TTL     time.Duration
	Topic   string
}

// Outcome starts at 1 so a zero Result never reads as Sent.
type Outcome int

const (
	Sent Outcome = iota + 1
	Gone
	Retry
	Failed
)

// Result carries no text, so no endpoint, key or payload can leave through it.
type Result struct {
	Outcome    Outcome
	Status     int
	RetryAfter time.Duration
}

type Sender interface {
	Send(ctx context.Context, s Subscription, m Message) Result
}

const urgency = "high"

var (
	errBlocked     = errors.New("push dial refused a non-public address")
	errDial        = errors.New("push dial failed")
	errTopic       = errors.New("push topic is not at most 32 of [A-Za-z0-9_-]")
	errNoEphemeral = errors.New("push ephemeral key is missing")
)

type sender struct {
	sig      *Signer
	now      func() time.Time
	timeout  time.Duration
	resolver Resolver
	rt       http.RoundTripper
	client   *http.Client
}

type option func(*sender)

// withTransport replaces the dialing transport, and with it the dial-time address check: tests only.
func withTransport(rt http.RoundTripper) option { return func(s *sender) { s.rt = rt } }
func withTimeout(d time.Duration) option        { return func(s *sender) { s.timeout = d } }
func withResolver(r Resolver) option            { return func(s *sender) { s.resolver = r } }

func NewSender(sig *Signer, now func() time.Time, opts ...option) Sender {
	s := &sender{sig: sig, now: now, timeout: constants.PushTimeout, resolver: net.DefaultResolver}
	for _, o := range opts {
		o(s)
	}
	if s.rt == nil {
		s.rt = newTransport(s.resolver)
	}
	s.client = &http.Client{
		Transport:     s.rt,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return s
}

func (s *sender) Send(ctx context.Context, sub Subscription, m Message) Result {
	body, ttl, err := prepare(sub, m)
	if err != nil {
		return Result{Outcome: Failed}
	}
	authz, err := s.sig.Authorization(sub.Endpoint, s.now())
	if err != nil {
		return Result{Outcome: Failed}
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{Outcome: Failed}
	}
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", strconv.FormatInt(ttl, 10))
	req.Header.Set("Urgency", urgency)
	if m.Topic != "" {
		req.Header.Set("Topic", m.Topic)
	}
	req.Header.Set("Authorization", authz)
	resp, err := s.client.Do(req)
	if err != nil {
		if errors.Is(err, errBlocked) {
			return Result{Outcome: Failed}
		}
		return Result{Outcome: Retry}
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, constants.PushResponseMaxBytes)
	return classify(resp, s.now())
}

func classify(resp *http.Response, now time.Time) Result {
	r := Result{Status: resp.StatusCode}
	switch c := resp.StatusCode; {
	case c >= 200 && c < 300:
		r.Outcome = Sent
	case c == http.StatusNotFound || c == http.StatusGone:
		r.Outcome = Gone
	case c == http.StatusTooManyRequests || c >= 500:
		r.Outcome = Retry
		r.RetryAfter = retryAfter(resp.Header.Get("Retry-After"), now)
	default:
		r.Outcome = Failed
	}
	return r
}

// retryAfter reads delta-seconds or an HTTP date; capped at ReminderExpiry, past which no retry is sent.
func retryAfter(v string, now time.Time) time.Duration {
	if v == "" {
		return 0
	}
	var d time.Duration
	if n, err := strconv.ParseUint(v, 10, 64); err == nil {
		d = time.Duration(min(n, uint64(constants.ReminderExpiry/time.Second))) * time.Second
	} else if errors.Is(err, strconv.ErrRange) {
		d = constants.ReminderExpiry
	} else if t, err := http.ParseTime(v); err == nil {
		d = t.Sub(now)
	}
	return max(0, min(d, constants.ReminderExpiry))
}

// prepare runs the checks shared with the Recorder and encrypts under a fresh key and salt.
func prepare(sub Subscription, m Message) (body []byte, ttl int64, err error) {
	if _, err := validateSyntax(sub.Endpoint); err != nil {
		return nil, 0, err
	}
	uaPub, auth, err := ParseKeys(sub.P256dh, sub.Auth)
	if err != nil {
		return nil, 0, err
	}
	if !validTopic(m.Topic) {
		return nil, 0, errTopic
	}
	body, err = encryptFresh(m.Payload, uaPub, auth)
	if err != nil {
		return nil, 0, err
	}
	return body, ttlSeconds(m.TTL), nil
}

func ttlSeconds(d time.Duration) int64 {
	return int64(max(time.Second, min(d, constants.PushTTLMax)) / time.Second)
}

func validTopic(t string) bool {
	if len(t) > 32 {
		return false
	}
	for _, c := range []byte(t) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// encryptFresh never reuses a key or salt: reuse for one subscription would repeat the AES-GCM nonce.
func encryptFresh(plaintext, uaPub, auth []byte) ([]byte, error) {
	as, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return Encrypt(plaintext, uaPub, auth, as, salt)
}

func newDialer() *net.Dialer { return &net.Dialer{Control: dialControl} }

// newTransport has no proxy: every connection is dialled here and checked.
func newTransport(r Resolver) *http.Transport {
	d := newDialer()
	return &http.Transport{
		ForceAttemptHTTP2: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, errBlocked
			}
			addrs, err := resolvePublic(ctx, r, host)
			if errors.Is(err, errEndpointNotAllowed) {
				return nil, errBlocked
			} else if err != nil {
				return nil, errDial
			}
			for _, a := range addrs {
				c, err := d.DialContext(ctx, network, net.JoinHostPort(a.Unmap().String(), port))
				if err == nil {
					return c, nil
				}
				if errors.Is(err, errBlocked) {
					return nil, errBlocked
				}
			}
			return nil, errDial
		},
	}
}

// dialControl checks the address actually being connected, after resolution.
func dialControl(_, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil || ap.Port() != 443 || !publicAddr(ap.Addr()) {
		return errBlocked
	}
	return nil
}
