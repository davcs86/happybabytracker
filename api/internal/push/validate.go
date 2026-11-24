package push

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Resolver is satisfied by *net.Resolver; tests pass a fake so no lookup touches the network.
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

var (
	errEndpointNotAllowed = errors.New("push endpoint is not allowed")
	errResolve            = errors.New("push endpoint host did not resolve")
)

// nonGlobal lists the special-use ranges the netip predicates miss, including ones that embed or reach IPv4.
var nonGlobal = prefixes(
	"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24",
	"203.0.113.0/24", "240.0.0.0/4",
	"::/96", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/32", "2001:db8::/32", "2002::/16", "fec0::/10",
	"3fff::/20", "2001:2::/48",
)

func prefixes(ss ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(ss))
	for i, s := range ss {
		out[i] = netip.MustParsePrefix(s)
	}
	return out
}

// ValidateEndpoint refuses an endpoint that could reach anything but a public host on 443; errors never echo it.
func ValidateEndpoint(ctx context.Context, r Resolver, endpoint string) error {
	u, err := validateSyntax(endpoint)
	if err != nil {
		return err
	}
	_, err = resolvePublic(ctx, r, u.Hostname())
	return err
}

// ValidateEndpointSyntax is the HBT_ALLOW_FAKE_PUSH registration check: no DNS; errors never echo the endpoint.
func ValidateEndpointSyntax(endpoint string) error {
	_, err := validateSyntax(endpoint)
	return err
}

func validateSyntax(endpoint string) (*url.URL, error) {
	if len(endpoint) > constants.PushEndpointMaxLen {
		return nil, errEndpointNotAllowed
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil {
		return nil, errEndpointNotAllowed
	}
	host := u.Hostname()
	if host == "" || (u.Host != host && u.Port() != "443") {
		return nil, errEndpointNotAllowed
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return nil, errEndpointNotAllowed
	}
	name := strings.ToLower(strings.TrimSuffix(host, "."))
	if name == "localhost" || strings.HasSuffix(name, ".localhost") || numericLabel(name) {
		return nil, errEndpointNotAllowed
	}
	return u, nil
}

// resolvePublic requires every address of host to be public, so a mixed answer cannot be raced.
func resolvePublic(ctx context.Context, r Resolver, host string) ([]netip.Addr, error) {
	addrs, err := r.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addrs) == 0 {
		return nil, errResolve
	}
	for _, a := range addrs {
		if !publicAddr(a) {
			return nil, errEndpointNotAllowed
		}
	}
	return addrs, nil
}

// numericLabel catches legacy IPv4 forms (2130706433, 127.1, 0x7f.1) that some resolvers parse: no TLD is numeric.
func numericLabel(name string) bool {
	last := name[strings.LastIndexByte(name, '.')+1:]
	return strings.HasPrefix(last, "0x") || strings.Trim(last, "0123456789") == ""
}

func publicAddr(a netip.Addr) bool {
	a = a.Unmap()
	if !a.IsValid() || a.IsUnspecified() || a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() || a.IsMulticast() {
		return false
	}
	for _, p := range nonGlobal {
		if p.Contains(a) {
			return false
		}
	}
	return true
}
