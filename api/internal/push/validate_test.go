package push

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// fakeResolver answers from a map; a missing name is a lookup error. It never touches the network.
type fakeResolver struct {
	mu    sync.Mutex
	addrs map[string][]string
	asked []string
}

func (f *fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, host)
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
	for _, bad := range []string{
		"127.0.0.1", "127.255.0.1", "::1", "10.0.0.1", "172.16.0.1", "172.31.255.255", "192.168.1.1",
		"169.254.169.254", "fe80::1", "fc00::1", "fd12:3456::1", "100.64.0.1", "100.127.255.255",
		"0.0.0.0", "::", "224.0.0.1", "239.255.255.250", "ff02::1", "ff05::2",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", "::ffff:100.64.0.1",
	} {
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
			"push.example.org": {"8.8.8.8", "2001:4860:4860::8888", "100.63.255.255", "100.128.0.1", "::ffff:8.8.4.4"},
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
