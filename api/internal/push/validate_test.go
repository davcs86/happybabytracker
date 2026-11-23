package push

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// fakeResolver answers from a map; a missing name is a lookup error. It never touches the network.
type fakeResolver struct {
	mu        sync.Mutex
	addrs     map[string][]string
	asked     []string
	deadlines []time.Time
}

func (f *fakeResolver) LookupNetIP(ctx context.Context, _, host string) ([]netip.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, host)
	dl, _ := ctx.Deadline()
	f.deadlines = append(f.deadlines, dl)
	as, ok := f.addrs[host]
	if !ok {
		return nil, errors.New("no such host")
	}
	out := make([]netip.Addr, 0, len(as))
	for _, a := range as {
		out = append(out, netip.MustParseAddr(a))
	}
	return out, nil
}

func publicResolver() *fakeResolver {
	return &fakeResolver{addrs: map[string][]string{"push.example.org": {"8.8.8.8", "2001:4860:4860::8888"}}}
}

func wantRefused(t *testing.T, err error, endpoint string) {
	t.Helper()
	if err == nil {
		t.Errorf("%.80q accepted", endpoint)
		return
	}
	if endpoint != "" && strings.Contains(err.Error(), endpoint) {
		t.Errorf("error %q echoes the endpoint", err)
	}
	if strings.Contains(err.Error(), "example") || strings.Contains(err.Error(), "secret-path") {
		t.Errorf("error %q echoes part of the endpoint", err)
	}
}

// nonGlobalExamples has one address in each nonGlobal prefix (and the reviewer's probes).
var nonGlobalExamples = []string{
	"0.1.2.3", "100.64.0.1", "192.0.0.1", "192.0.2.1", "198.18.0.1", "198.19.255.255", "198.51.100.1", "203.0.113.1",
	"240.0.0.1", "255.255.255.255",
	"::7f00:1", "::a00:1", "64:ff9b::a00:1", "64:ff9b:1::a00:1", "100::1", "2001::1", "2001:0:ffff::1",
	"2001:db8::1", "2002:a00:1::1", "fec0::1", "feff::1",
}

func TestValidateEndpointRefusesSyntax(t *testing.T) {
	long := "https://push.example.org/" + strings.Repeat("a", constants.PushEndpointMaxLen-len("https://push.example.org/")+1)
	if len(long) != 2049 {
		t.Fatalf("long endpoint is %d bytes", len(long))
	}
	for _, e := range []string{
		"http://push.example.org/secret-path",
		"HTTP://push.example.org/secret-path",
		"wss://push.example.org/secret-path",
		"https://user:pw@push.example.org/secret-path",
		"https://user@push.example.org/secret-path",
		"https://push.example.org:8443/secret-path",
		"https://push.example.org:80/secret-path",
		"https://push.example.org:/secret-path",
		"https://127.0.0.1/secret-path",
		"https://[::1]/secret-path",
		"https://[::ffff:127.0.0.1]/secret-path",
		"https://localhost/secret-path",
		"https://LOCALHOST/secret-path",
		"https://localhost./secret-path",
		"https://a.localhost/secret-path",
		"https://10.0.0.1/secret-path",
		"https://169.254.169.254/secret-path",
		"https://[fc00::1]/secret-path",
		"https://[fe80::1%25eth0]/secret-path",
		"https://100.64.0.1/secret-path",
		"https://8.8.8.8/secret-path",
		"https://2130706433/secret-path",
		"https://0x7f.1/secret-path",
		"https://127.1/secret-path",
		"https://0177.0.0.1/secret-path",
		"https://push.0x7f/secret-path",
		"https://push.example.123/secret-path",
		"https://push.example.org.1./secret-path",
		"https:///secret-path",
		"https://:443/secret-path",
		"https:push.example.org/secret-path",
		"//push.example.org/secret-path",
		"::",
		"",
		long,
	} {
		wantRefused(t, ValidateEndpoint(context.Background(), publicResolver(), e), e)
	}
}

func TestValidateEndpointRefusesResolvedAddresses(t *testing.T) {
	for _, bad := range append([]string{
		"127.0.0.1", "127.255.0.1", "::1", "10.0.0.1", "172.16.0.1", "172.31.255.255", "192.168.1.1",
		"169.254.169.254", "fe80::1", "fc00::1", "fd12:3456::1", "100.64.0.1", "100.127.255.255",
		"0.0.0.0", "::", "224.0.0.1", "239.255.255.250", "ff02::1", "ff05::2",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", "::ffff:100.64.0.1",
	}, nonGlobalExamples...) {
		for _, set := range [][]string{{bad}, {"8.8.8.8", bad}, {bad, "2001:4860:4860::8888"}} {
			r := &fakeResolver{addrs: map[string][]string{"push.example.org": set}}
			e := "https://push.example.org/secret-path"
			if err := ValidateEndpoint(context.Background(), r, e); err == nil {
				t.Errorf("resolving to %v accepted", set)
			} else {
				wantRefused(t, err, e)
			}
		}
	}
	for name, r := range map[string]*fakeResolver{
		"lookup error": {addrs: map[string][]string{}},
		"no address":   {addrs: map[string][]string{"push.example.org": {}}},
	} {
		e := "https://push.example.org/secret-path"
		if err := ValidateEndpoint(context.Background(), r, e); err == nil {
			t.Errorf("%s accepted", name)
		} else {
			wantRefused(t, err, e)
		}
	}
}

func TestValidateEndpointAccepts(t *testing.T) {
	exact := "https://push.example.org/" + strings.Repeat("a", constants.PushEndpointMaxLen-len("https://push.example.org/"))
	for _, e := range []string{
		"https://push.example.org/wpush/v2/abc",
		"https://push.example.org:443/wpush/v2/abc",
		"https://PUSH.example.org/wpush/v2/abc",
		"https://push.example.org/fcm/send/abc?x=1",
		exact,
	} {
		r := &fakeResolver{addrs: map[string][]string{
			"push.example.org": {"8.8.8.8", "2001:4860:4860::8888", "2606:4700:4700::1111", "1.1.1.1", "100.63.255.255",
				"100.128.0.1", "::ffff:8.8.4.4", "198.17.255.255", "198.20.0.1", "2001:1::1", "2003::1", "64:ff9b:2::1"},
			"PUSH.example.org": {"8.8.8.8"},
		}}
		if err := ValidateEndpoint(context.Background(), r, e); err != nil {
			t.Errorf("%.60q refused: %v", e, err)
		}
		if len(r.asked) != 1 {
			t.Errorf("%.60q: resolver asked %d times", e, len(r.asked))
		}
	}
}

func TestValidateEndpointSyntax(t *testing.T) {
	for _, e := range []string{"https://push.example.org/wpush/v2/abc", "https://push.example.org:443/x"} {
		if err := ValidateEndpointSyntax(e); err != nil {
			t.Errorf("%q refused: %v", e, err)
		}
	}
	for _, e := range []string{"http://push.example.org/secret-path", "https://10.0.0.1/secret-path", "https://127.1/secret-path",
		"https://2130706433/secret-path", "https://0x7f.1/secret-path", "https://0177.0.0.1/secret-path", "https://push.0X7F/secret-path",
		"https://push.example.org:8443/secret-path", "https://localhost/secret-path", ""} {
		wantRefused(t, ValidateEndpointSyntax(e), e)
	}
}
