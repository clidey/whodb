// Package sourcedial lets an edition route every connection a database
// connector opens to a user-configured destination through one dialer. A
// check in that dialer then covers every address the driver actually
// connects to, including hosts it resolves again, discovers from the server
// or is redirected to.
//
// Nothing is registered by default, and connectors then keep their driver's
// own dialing, so behavior only changes when an edition calls SetDialer.
package sourcedial

import (
	"context"
	"crypto/tls"
	"net"
	"sync/atomic"
)

// DialFunc opens a connection, like net.Dialer.DialContext.
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

var registered atomic.Pointer[DialFunc]

// SetDialer routes source connections opened afterwards through dial. Call it
// once during startup, before connectors open connections.
func SetDialer(dial DialFunc) {
	registered.Store(&dial)
}

// Dialer returns the registered dialer, or nil when none is registered.
func Dialer() DialFunc {
	if dial := registered.Load(); dial != nil {
		return *dial
	}
	return nil
}

// ContextDialer adapts a DialFunc for drivers that take a dialer value with a
// DialContext method (MongoDB, Cassandra, SQL Server, Oracle).
type ContextDialer DialFunc

// DialContext opens a connection through the adapted DialFunc.
func (d ContextDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d(ctx, network, addr)
}

// DialTLS dials through dial and completes a TLS handshake, for drivers whose
// custom-dialer hook replaces their own TLS setup (Redis, the ClickHouse
// native protocol). Like tls.Dial, an empty ServerName defaults to the host
// in addr.
func DialTLS(ctx context.Context, dial DialFunc, network, addr string, config *tls.Config) (net.Conn, error) {
	conn, err := dial(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	if config.ServerName == "" {
		if host, _, splitErr := net.SplitHostPort(addr); splitErr == nil {
			config = config.Clone()
			config.ServerName = host
		}
	}
	tlsConn := tls.Client(conn, config)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return tlsConn, nil
}
