// Package security provides server-side egress controls for outbound requests
// to user-controlled URLs (AI provider endpoints, etc.), defending against SSRF
// to cloud metadata services and internal cluster addresses.
package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/env"
)

// egressEnforced reports whether outbound egress restrictions should apply.
// Controlled by the explicit WHODB_BLOCK_INTERNAL_AI_ENDPOINTS flag (off by
// default): a self-hosted WhoDB legitimately connects to local model servers
// (Ollama on localhost, an in-cluster LLM, a private-network endpoint), so
// blocking internal addresses is opt-in for multi-tenant/hosted deployments.
func egressEnforced() bool {
	return env.BlockInternalAIEndpoints
}

// EgressRestricted reports whether outbound egress restrictions are enabled.
// Call sites use this to also drop client-supplied endpoint overrides.
func EgressRestricted() bool {
	return egressEnforced()
}

// EnforceOutboundURL validates the URL only when egress protection is enabled
// (WHODB_BLOCK_INTERNAL_AI_ENDPOINTS=true). Otherwise it is a no-op so local AI
// model endpoints keep working. Use this from call sites that handle
// user-supplied endpoints.
func EnforceOutboundURL(raw string) error {
	if !egressEnforced() {
		return nil
	}
	return ValidateOutboundURL(raw)
}

// ValidateOutboundURL rejects URLs that target loopback, link-local (including
// the cloud metadata service 169.254.169.254), private, or otherwise non-public
// addresses. Only http/https schemes are allowed. Hostnames are resolved so a
// name pointing at an internal IP is also rejected.
func ValidateOutboundURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("URL scheme %q is not allowed", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("URL has no host")
	}

	// If the host is a literal IP, check it directly. Otherwise resolve and
	// check every returned address (the dialer re-checks the actual dialed IP
	// to defend against DNS rebinding).
	if ip := net.ParseIP(host); ip != nil {
		return checkIP(ip)
	}
	ips, err := resolveHost(context.Background(), host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if err := checkIP(ip); err != nil {
			return err
		}
	}
	return nil
}

// resolveHost resolves a hostname to IPs using a context-aware resolver with a
// bounded timeout.
func resolveHost(ctx context.Context, host string) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve host %q: %w", host, err)
	}
	ips := make([]net.IP, len(addrs))
	for i, a := range addrs {
		ips[i] = a.IP
	}
	return ips, nil
}

// checkIP rejects non-public IP ranges.
func checkIP(ip net.IP) error {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() {
		return fmt.Errorf("destination address %s is not allowed", ip)
	}
	// Explicitly block the cloud metadata service address (covered by
	// IsLinkLocalUnicast, but called out for clarity) and IPv4-mapped variants.
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return fmt.Errorf("destination address %s is not allowed", ip)
	}
	return nil
}

// SafeDialContext returns a DialContext that re-validates the resolved IP at
// connection time, defeating DNS-rebinding attacks where a hostname resolves to
// a public IP during validation and an internal IP at dial time. In
// self-hosted/local mode it delegates to a plain dialer so local model
// endpoints keep working.
func SafeDialContext(timeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	enforce := egressEnforced()
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !enforce {
			return dialer.DialContext(ctx, network, addr)
		}
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := resolveHost(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if err := checkIP(ip); err != nil {
				return nil, err
			}
		}
		// Dial the first validated address explicitly so we connect to exactly
		// what we checked.
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}

// SafeHTTPTransport returns an http.Transport cloned from http.DefaultTransport
// (preserving proxy support, HTTP/2, and connection-pool defaults) with the
// SSRF-aware dialer installed. Use this instead of constructing a bare
// http.Transport, which would silently drop those defaults.
func SafeHTTPTransport(timeout time.Duration) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = SafeDialContext(timeout)
	return transport
}
