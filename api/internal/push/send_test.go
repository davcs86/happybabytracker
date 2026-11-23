package push

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

var testNow = time.Unix(1_800_000_000, 0)

type subFixture struct {
	sub  Subscription
	ua   *ecdh.PrivateKey
	auth []byte
}

func newSub(t *testing.T, endpoint string) subFixture {
	t.Helper()
	ua, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	rand.Read(auth)
	return subFixture{
		sub: Subscription{ID: "dev-1", Endpoint: endpoint, P256dh: b64.EncodeToString(ua.PublicKey().Bytes()), Auth: b64.EncodeToString(auth)},
		ua:  ua, auth: auth,
	}
}

func newTestSigner(t *testing.T) (*Signer, string) {
	t.Helper()
	priv, pub := newKey(t)
	k, err := ParsePrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSigner(k, testSubject)
	if err != nil {
		t.Fatal(err)
	}
	return s, pub
}

type captured struct {
	header http.Header
	body   []byte
	path   string
}

// pushServer is a TLS push service; every connection reaches it whatever the endpoint's host
// (the cert covers example.com), so the tests stand in for the dial check's bypass.
type pushServer struct {
	srv   *httptest.Server
	mu    sync.Mutex
	got   []captured
	reply func(w http.ResponseWriter, r *http.Request)
}

func newPushServer(t *testing.T, reply func(w http.ResponseWriter, r *http.Request)) *pushServer {
	t.Helper()
	p := &pushServer{reply: reply}
	p.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		p.mu.Lock()
		p.got = append(p.got, captured{header: r.Header.Clone(), body: b, path: r.URL.Path})
		p.mu.Unlock()
		p.reply(w, r)
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *pushServer) transport() http.RoundTripper {
	tr := p.srv.Client().Transport.(*http.Transport).Clone()
	addr := p.srv.Listener.Addr().String()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	return tr
}

func (p *pushServer) requests() []captured {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]captured(nil), p.got...)
}

func status(code int, hdr ...string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i+1 < len(hdr); i += 2 {
			w.Header().Set(hdr[i], hdr[i+1])
		}
		w.WriteHeader(code)
	}
}

const testEndpoint = "https://example.com/wpush/v2/secret-path"

var testPayload = []byte(`{"v":1,"category":"medication","kid":"k1","title":"Medication due for Mia","path":"/kids/k1/medication","tag":"reminder-r1"}`)

func testMessage() Message {
	return Message{Payload: testPayload, TTL: 1200 * time.Second, Topic: "0123456789abcdef0123456789abcdef"}
}

func verifyVAPID(t *testing.T, h, pub string) map[string]any {
	t.Helper()
	rest, ok := strings.CutPrefix(h, "vapid t=")
	jwt, kv, ok2 := strings.Cut(rest, ", k=")
	if !ok || !ok2 || kv != pub {
		t.Fatalf("authorization shape %q", h)
	}
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("jwt has %d parts", len(parts))
	}
	sig := mustDec(t, parts[2])
	ecpub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), mustDec(t, pub))
	if err != nil || len(sig) != 64 {
		t.Fatalf("key %v sig len %d", err, len(sig))
	}
	d := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(ecpub, d[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		t.Fatal("VAPID signature does not verify")
	}
	var claims map[string]any
	if err := json.Unmarshal(mustDec(t, parts[1]), &claims); err != nil {
		t.Fatal(err)
	}
	return claims
}

func TestSendHeadersAndBody(t *testing.T) {
	p := newPushServer(t, status(http.StatusCreated))
	sig, pub := newTestSigner(t)
	f := newSub(t, testEndpoint)
	s := NewSender(sig, func() time.Time { return testNow }, withTransport(p.transport()))
	res := s.Send(context.Background(), f.sub, testMessage())
	if res != (Result{Outcome: Sent, Status: 201}) {
		t.Fatalf("result %+v", res)
	}
	reqs := p.requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests", len(reqs))
	}
	h := reqs[0].header
	for k, v := range map[string]string{
		"Ttl": "1200", "Urgency": "high", "Topic": "0123456789abcdef0123456789abcdef",
		"Content-Encoding": "aes128gcm", "Content-Type": "application/octet-stream",
	} {
		if got := h.Get(k); got != v {
			t.Errorf("header %s = %q, want %q", k, got, v)
		}
	}
	if reqs[0].path != "/wpush/v2/secret-path" {
		t.Errorf("path %q", reqs[0].path)
	}
	claims := verifyVAPID(t, h.Get("Authorization"), pub)
	if claims["aud"] != "https://example.com" || claims["sub"] != testSubject ||
		int64(claims["exp"].(float64)) != testNow.Add(constants.PushJWTLifetime).Unix() {
		t.Errorf("claims %v", claims)
	}
	if got := decrypt(t, reqs[0].body, f.ua, f.auth); !bytes.Equal(got, testPayload) {
		t.Errorf("body decrypts to %q", got)
	}
}

func TestSendTTLBounds(t *testing.T) {
	p := newPushServer(t, status(http.StatusCreated))
	sig, _ := newTestSigner(t)
	f := newSub(t, testEndpoint)
	s := NewSender(sig, time.Now, withTransport(p.transport()))
	for _, c := range []struct {
		ttl  time.Duration
		want string
	}{
		{0, "1"}, {-time.Minute, "1"}, {500 * time.Millisecond, "1"}, {1500 * time.Millisecond, "1"},
		{constants.PushTTLMax, "3600"}, {2 * constants.PushTTLMax, "3600"},
	} {
		m := testMessage()
		m.TTL = c.ttl
		if res := s.Send(context.Background(), f.sub, m); res.Outcome != Sent {
			t.Fatalf("ttl %v: %+v", c.ttl, res)
		}
		reqs := p.requests()
		if got := reqs[len(reqs)-1].header.Get("TTL"); got != c.want {
			t.Errorf("ttl %v sent as %q, want %q", c.ttl, got, c.want)
		}
	}
	m := testMessage()
	m.Topic = ""
	s.Send(context.Background(), f.sub, m)
	reqs := p.requests()
	if _, ok := reqs[len(reqs)-1].header["Topic"]; ok {
		t.Error("empty topic sent as a header")
	}
}

func TestSendFreshKeyAndSaltPerMessage(t *testing.T) {
	p := newPushServer(t, status(http.StatusCreated))
	sig, _ := newTestSigner(t)
	f := newSub(t, testEndpoint)
	s := NewSender(sig, time.Now, withTransport(p.transport()))
	for range 2 {
		if res := s.Send(context.Background(), f.sub, testMessage()); res.Outcome != Sent {
			t.Fatalf("%+v", res)
		}
	}
	reqs := p.requests()
	a, b := reqs[0].body, reqs[1].body
	if bytes.Equal(a[:16], b[:16]) {
		t.Error("salt reused between two messages")
	}
	if bytes.Equal(a[21:86], b[21:86]) {
		t.Error("ephemeral key (keyid) reused between two messages")
	}
	for _, body := range [][]byte{a, b} {
		if got := decrypt(t, body, f.ua, f.auth); !bytes.Equal(got, testPayload) {
			t.Errorf("decrypts to %q", got)
		}
	}
	x, err := encryptFresh(testPayload, f.ua.PublicKey().Bytes(), f.auth)
	y, err2 := encryptFresh(testPayload, f.ua.PublicKey().Bytes(), f.auth)
	if err != nil || err2 != nil || bytes.Equal(x[:16], y[:16]) || bytes.Equal(x[21:86], y[21:86]) {
		t.Errorf("encryptFresh reuses its salt or key (%v, %v)", err, err2)
	}
}

func TestSendClassifiesStatuses(t *testing.T) {
	for _, c := range []struct {
		name  string
		reply func(w http.ResponseWriter, r *http.Request)
		want  Result
	}{
		{"201", status(201), Result{Outcome: Sent, Status: 201}},
		{"404", status(404), Result{Outcome: Gone, Status: 404}},
		{"410", status(410), Result{Outcome: Gone, Status: 410}},
		{"429 seconds", status(429, "Retry-After", "120"), Result{Outcome: Retry, Status: 429, RetryAfter: 2 * time.Minute}},
		{"429 date", status(429, "Retry-After", testNow.Add(5*time.Minute).UTC().Format(http.TimeFormat)), Result{Outcome: Retry, Status: 429, RetryAfter: 5 * time.Minute}},
		{"429 capped", status(429, "Retry-After", "999999999999999999999"), Result{Outcome: Retry, Status: 429, RetryAfter: constants.ReminderExpiry}},
		{"429 far date", status(429, "Retry-After", testNow.Add(2*time.Hour).UTC().Format(http.TimeFormat)), Result{Outcome: Retry, Status: 429, RetryAfter: constants.ReminderExpiry}},
		{"429 past date", status(429, "Retry-After", testNow.Add(-time.Hour).UTC().Format(http.TimeFormat)), Result{Outcome: Retry, Status: 429}},
		{"429 negative", status(429, "Retry-After", "-5"), Result{Outcome: Retry, Status: 429}},
		{"429 garbage", status(429, "Retry-After", "soon"), Result{Outcome: Retry, Status: 429}},
		{"429 none", status(429), Result{Outcome: Retry, Status: 429}},
		{"500", status(500), Result{Outcome: Retry, Status: 500}},
		{"503 retry-after", status(503, "Retry-After", "60"), Result{Outcome: Retry, Status: 503, RetryAfter: time.Minute}},
		{"302", status(302, "Location", "https://example.com/elsewhere"), Result{Outcome: Failed, Status: 302}},
		{"400", status(400), Result{Outcome: Failed, Status: 400}},
		{"401", status(401), Result{Outcome: Failed, Status: 401}},
		{"403", status(403), Result{Outcome: Failed, Status: 403}},
		{"413", status(413), Result{Outcome: Failed, Status: 413}},
		{"big body", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(400)
			w.Write(bytes.Repeat([]byte("x"), 10*constants.PushResponseMaxBytes))
		}, Result{Outcome: Failed, Status: 400}},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := newPushServer(t, c.reply)
			sig, _ := newTestSigner(t)
			f := newSub(t, testEndpoint)
			s := NewSender(sig, func() time.Time { return testNow }, withTransport(p.transport()))
			if got := s.Send(context.Background(), f.sub, testMessage()); got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
			if n := len(p.requests()); n != 1 {
				t.Fatalf("%d requests (a redirect followed?)", n)
			}
		})
	}
}

func TestSendTimeoutAndNetworkRetry(t *testing.T) {
	release := make(chan struct{})
	p := newPushServer(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
		w.WriteHeader(201)
	})
	defer close(release)
	sig, _ := newTestSigner(t)
	f := newSub(t, testEndpoint)
	s := NewSender(sig, time.Now, withTransport(p.transport()), withTimeout(100*time.Millisecond))
	start := time.Now()
	if got := s.Send(context.Background(), f.sub, testMessage()); got != (Result{Outcome: Retry}) {
		t.Fatalf("slow response: %+v", got)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("timeout took %v", d)
	}

	dead := httptest.NewTLSServer(http.NotFoundHandler())
	addr := dead.Listener.Addr().String()
	dead.Close()
	tr := dead.Client().Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	s = NewSender(sig, time.Now, withTransport(tr))
	if got := s.Send(context.Background(), f.sub, testMessage()); got != (Result{Outcome: Retry}) {
		t.Fatalf("refused connection: %+v", got)
	}
}

func TestSendDefaultTimeoutIsPushTimeout(t *testing.T) {
	sig, _ := newTestSigner(t)
	if s := NewSender(sig, time.Now).(*sender); s.timeout != constants.PushTimeout {
		t.Fatalf("timeout %v", s.timeout)
	}
}

func TestSendRefusesBeforeSending(t *testing.T) {
	var hits atomic.Int32
	p := newPushServer(t, func(w http.ResponseWriter, _ *http.Request) { hits.Add(1); w.WriteHeader(201) })
	sig, _ := newTestSigner(t)
	s := NewSender(sig, time.Now, withTransport(p.transport()))
	good := newSub(t, testEndpoint)
	compressed := good.sub
	compressed.P256dh = b64.EncodeToString(good.ua.PublicKey().Bytes()[:33])
	shortAuth := good.sub
	shortAuth.Auth = b64.EncodeToString(good.auth[:15])
	for name, c := range map[string]struct {
		sub Subscription
		m   Message
	}{
		"http endpoint":  {Subscription{ID: "d", Endpoint: "http://example.com/secret-path", P256dh: good.sub.P256dh, Auth: good.sub.Auth}, testMessage()},
		"ip endpoint":    {Subscription{ID: "d", Endpoint: "https://127.0.0.1/secret-path", P256dh: good.sub.P256dh, Auth: good.sub.Auth}, testMessage()},
		"port endpoint":  {Subscription{ID: "d", Endpoint: "https://example.com:8443/secret-path", P256dh: good.sub.P256dh, Auth: good.sub.Auth}, testMessage()},
		"compressed key": {compressed, testMessage()},
		"auth 15":        {shortAuth, testMessage()},
		"payload large":  {good.sub, Message{Payload: make([]byte, MaxPlaintext+1), TTL: time.Minute}},
		"topic long":     {good.sub, Message{Payload: testPayload, TTL: time.Minute, Topic: strings.Repeat("a", 33)}},
		"topic chars":    {good.sub, Message{Payload: testPayload, TTL: time.Minute, Topic: "a b"}},
		"topic newline":  {good.sub, Message{Payload: testPayload, TTL: time.Minute, Topic: "a\r\nX-Evil: 1"}},
	} {
		if got := s.Send(context.Background(), c.sub, c.m); got != (Result{Outcome: Failed}) {
			t.Errorf("%s: %+v", name, got)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("%d requests reached the push service", hits.Load())
	}
}

func TestDialCheckRefusesPrivateAddressThroughDNS(t *testing.T) {
	sig, _ := newTestSigner(t)
	f := newSub(t, "https://push.example.org/secret-path")
	for _, set := range [][]string{{"127.0.0.1"}, {"8.8.8.8", "127.0.0.1"}, {"::1"}, {"10.0.0.1"}, {"100.64.0.1"}} {
		r := &fakeResolver{addrs: map[string][]string{"push.example.org": set}}
		s := NewSender(sig, time.Now, withResolver(r), withTimeout(2*time.Second))
		if got := s.Send(context.Background(), f.sub, testMessage()); got != (Result{Outcome: Failed}) {
			t.Errorf("resolving to %v: %+v", set, got)
		}
		if len(r.asked) != 1 {
			t.Errorf("resolving to %v: resolver asked %d times", set, len(r.asked))
		}
	}
}

// With the pre-dial check skipped, only the real transport's Control stands between the sender and loopback:
// a dial without it reaches 127.0.0.1:443 and is refused by the kernel, which reads as Retry, not Failed.
// IPv6 loopback is left out: a host without IPv6 fails the socket before Control runs.
func TestControlRefusesWithoutResolveCheck(t *testing.T) {
	sig, _ := newTestSigner(t)
	f := newSub(t, "https://push.example.org/secret-path")
	for _, set := range [][]string{{"127.0.0.1"}, {"::ffff:127.0.0.1"}, {"127.0.0.2", "127.0.0.1"}} {
		r := &fakeResolver{addrs: map[string][]string{"push.example.org": set}}
		s := NewSender(sig, time.Now, withResolver(r), withoutResolveCheck(), withTimeout(2*time.Second))
		if got := s.Send(context.Background(), f.sub, testMessage()); got != (Result{Outcome: Failed}) {
			t.Errorf("resolving to %v: %+v (Control not on the transport?)", set, got)
		}
	}
}

// The transport detaches the dial from the request's context, so the dial must carry the timeout itself.
func TestDialCarriesThePushTimeout(t *testing.T) {
	sig, _ := newTestSigner(t)
	f := newSub(t, "https://push.example.org/secret-path")
	r := &fakeResolver{addrs: map[string][]string{"push.example.org": {"127.0.0.1"}}}
	const d = 1500 * time.Millisecond
	start := time.Now()
	NewSender(sig, time.Now, withResolver(r), withTimeout(d)).Send(context.Background(), f.sub, testMessage())
	end := time.Now()
	if len(r.deadlines) != 1 || r.deadlines[0].Before(start.Add(d)) || r.deadlines[0].After(end.Add(d)) {
		t.Fatalf("dial deadlines %v, want one %v after the dial started", r.deadlines, d)
	}
}

func TestTransportLimits(t *testing.T) {
	sig, _ := newTestSigner(t)
	tr := NewSender(sig, time.Now).(*sender).rt.(*http.Transport)
	if tr.Proxy != nil || tr.IdleConnTimeout != constants.PushIdleConnTimeout || tr.MaxIdleConnsPerHost != 1 ||
		tr.MaxResponseHeaderBytes != constants.PushResponseHeaderMaxBytes {
		t.Fatalf("transport proxy=%v idle=%v perHost=%d headerMax=%d", tr.Proxy != nil, tr.IdleConnTimeout,
			tr.MaxIdleConnsPerHost, tr.MaxResponseHeaderBytes)
	}
	for size, want := range map[int]Result{
		constants.PushResponseHeaderMaxBytes / 2: {Outcome: Sent, Status: 201},
		constants.PushResponseHeaderMaxBytes + 1: {Outcome: Retry},
	} {
		p := newPushServer(t, status(201, "X-Big", strings.Repeat("a", size)))
		real := newTransport(&checkedDialer{r: publicResolver(), d: newDialer(), timeout: time.Second})
		real.TLSClientConfig = p.srv.Client().Transport.(*http.Transport).TLSClientConfig
		real.DialContext = p.transport().(*http.Transport).DialContext
		f := newSub(t, testEndpoint)
		if got := NewSender(sig, time.Now, withTransport(real)).Send(context.Background(), f.sub, testMessage()); got != want {
			t.Errorf("%d-byte header: %+v, want %+v", size, got, want)
		}
	}
}

func TestNewSenderRefusesNilWiring(t *testing.T) {
	sig, _ := newTestSigner(t)
	for name, fn := range map[string]func(){
		"nil signer": func() { NewSender(nil, time.Now) },
		"nil clock":  func() { NewSender(sig, nil) },
	} {
		func() {
			defer func() {
				if r := recover(); r != "push: NewSender needs a signer and a clock" {
					t.Errorf("%s: recovered %v", name, r)
				}
			}()
			fn()
		}()
	}
}

func TestDialerControlRefusesTheConnectedAddress(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	var accepted atomic.Int32
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			c.Close()
		}
	}()
	if c, err := newDialer().DialContext(context.Background(), "tcp", ln.Addr().String()); err == nil {
		c.Close()
		t.Fatal("dialer connected to loopback")
	} else if !errors.Is(err, errBlocked) {
		t.Fatalf("dial error %v is not errBlocked", err)
	}
	for addr, ok := range map[string]bool{
		"8.8.8.8:443": true, "[2001:4860:4860::8888]:443": true,
		"8.8.8.8:80": false, "127.0.0.1:443": false, "[::1]:443": false, "10.1.2.3:443": false,
		"169.254.169.254:443": false, "100.64.0.1:443": false, "[fc00::1]:443": false, "[::ffff:127.0.0.1]:443": false,
		"0.0.0.0:443": false, "224.0.0.1:443": false, "garbage": false, "[2606:4700:4700::1111]:443": true,
	} {
		if err := dialControl("tcp", addr, nil); (err == nil) != ok {
			t.Errorf("dialControl(%s) = %v", addr, err)
		}
	}
	for _, a := range nonGlobalExamples {
		if err := dialControl("tcp", net.JoinHostPort(a, "443"), nil); err == nil {
			t.Errorf("dialControl(%s) accepted", a)
		}
	}
	if accepted.Load() != 0 {
		t.Fatal("loopback listener accepted a connection")
	}
}

func TestResultCarriesNoText(t *testing.T) {
	// Only integer fields, so no endpoint, key or payload can ride on a Result.
	rt := reflect.TypeFor[Result]()
	for i := range rt.NumField() {
		switch rt.Field(i).Type.Kind() {
		case reflect.Int, reflect.Int64:
		default:
			t.Errorf("Result.%s is a %s", rt.Field(i).Name, rt.Field(i).Type)
		}
	}
	for _, e := range []error{errBlocked, errResolve, errEndpointNotAllowed, errTopic, errNoEphemeral} {
		if strings.Contains(e.Error(), "example") || strings.Contains(e.Error(), "secret-path") {
			t.Errorf("sentinel %q", e)
		}
	}
}
