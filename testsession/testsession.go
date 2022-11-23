// Package testsession provides utilities to test SSH sessions.
//
// more or less copied from charmbracelet/ssh tests
package testsession

import (
	"net"
	"testing"

	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// New starts a local SSH server with the given config and returns a client session.
// It automatically closes everything afterwards.
func New(tb testing.TB, srv *ssh.Server, cfg *gossh.ClientConfig) *gossh.Session {
	tb.Helper()
	sess, err := NewClientSession(tb, Listen(tb, srv), cfg)
	if err != nil {
		tb.Fatal(err)
	}
	return sess
}

// Listen starts a test server.
func Listen(tb testing.TB, srv *ssh.Server) string {
	tb.Helper()
	l := newLocalListener(tb)
	go func() {
		if err := srv.Serve(l); err != nil && err != ssh.ErrServerClosed {
			tb.Fatalf("failed to serve: %v", err)
		}
	}()
	tb.Cleanup(func() {
		_ = srv.Close()
	})
	return l.Addr().String()
}

func newLocalListener(tb testing.TB) net.Listener {
	tb.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			tb.Fatalf("failed to listen on a port: %v", err)
		}
	}

	tb.Cleanup(func() { _ = l.Close() })
	return l
}

// NewClientSession creates a new client session to the given address.
func NewClientSession(tb testing.TB, addr string, config *gossh.ClientConfig) (*gossh.Session, error) {
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
		return nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	tb.Cleanup(func() {
		_ = session.Close()
		_ = client.Close()
	})
	return session, nil
}
