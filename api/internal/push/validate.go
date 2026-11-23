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

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// ValidateEndpoint refuses an endpoint that could reach anything but a public host on 443; errors never echo it.
func ValidateEndpoint(ctx context.Context, r Resolver, endpoint string) error {
	u, err := validateSyntax(endpoint)
	if err != nil {
		return err
	}
	_, err = resolvePublic(ctx, r, u.Hostname())
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
	if name == "localhost" || strings.HasSuffix(name, ".localhost") {
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

func publicAddr(a netip.Addr) bool {
	a = a.Unmap()
	return a.IsValid() && !a.IsUnspecified() && !a.IsLoopback() && !a.IsPrivate() &&
		!a.IsLinkLocalUnicast() && !a.IsMulticast() && !cgnat.Contains(a)
}
