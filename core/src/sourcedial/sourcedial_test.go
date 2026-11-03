package sourcedial

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDialerIsNilUntilRegistered(t *testing.T) {
	t.Cleanup(func() { registered.Store(nil) })
	registered.Store(nil)
	if Dialer() != nil {
		t.Fatal("Dialer() is set before SetDialer")
	}
	calls := 0
	SetDialer(func(ctx context.Context, network, addr string) (net.Conn, error) {
		calls++
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)
	conn, err := ContextDialer(Dialer()).DialContext(context.Background(), "tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	if calls != 1 {
		t.Fatalf("registered dialer called %d times, want 1", calls)
	}
}

func TestDialTLSHandshakesWithTheHostAsServerName(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(server.Close)
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	pool := server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	plain := (&net.Dialer{}).DialContext

	// The test certificate is issued for example.com, so the handshake only
	// succeeds when ServerName comes from the address.
	dialExample := func(ctx context.Context, network, _ string) (net.Conn, error) {
		return plain(ctx, network, net.JoinHostPort("127.0.0.1", port))
	}
	conn, err := DialTLS(context.Background(), dialExample, "tcp", net.JoinHostPort("example.com", port), &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatalf("DialTLS: %v", err)
	}
	_ = conn.Close()

	if _, err := DialTLS(context.Background(), dialExample, "tcp", net.JoinHostPort("wrong.invalid", port), &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}); err == nil {
		t.Fatal("DialTLS accepted a certificate for a different host")
	}
}
