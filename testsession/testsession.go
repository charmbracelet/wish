// Package testsession provides utilities to test SSH sessions.
//
// more or less copied from gliderlabs/ssh tests
package testsession

import (
	"net"
	"testing"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// New starts a local SSH server with the given config and returns a client session.
// It automatically closes everything afterwards.
func New(tb testing.TB, srv *ssh.Server, cfg *gossh.ClientConfig) *gossh.Session {
	tb.Helper()
	l := newLocalListener(tb)
	go func() {
		if err := srv.Serve(l); err != nil && err != ssh.ErrServerClosed {
			tb.Fatalf("failed to serve: %v", err)
		}
	}()
	tb.Cleanup(func() {
		srv.Close() // nolint: errcheck
	})
	return newClientSession(tb, l.Addr().String(), cfg)
}

func newLocalListener(tb testing.TB) net.Listener {
	tb.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			tb.Fatalf("failed to listen on a port: %v", err)
		}
	}
	return l
}

func newClientSession(tb testing.TB, addr string, config *gossh.ClientConfig) *gossh.Session {
	tb.Helper()
	if config == nil {
		config = &gossh.ClientConfig{
			User: "testuser",
			Auth: []gossh.AuthMethod{
				gossh.Password("testpass"),
			},
		}
	}
	if config.HostKeyCallback == nil {
		config.HostKeyCallback = gossh.InsecureIgnoreHostKey() // nolint: gosec
	}
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		tb.Fatal(err)
	}
	session, err := client.NewSession()
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		session.Close() // nolint: errcheck
		client.Close()  // nolint: errcheck
	})
	return session
}
