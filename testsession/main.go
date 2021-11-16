// more or less copied from gliderlabs/ssh tests
package testsession

import (
	"fmt"
	"net"
	"testing"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func New(tb testing.TB, srv *ssh.Server, cfg *gossh.ClientConfig) (*gossh.Session, *gossh.Client, func()) {
	tb.Helper()
	l := newLocalListener()
	go srv.Serve(l)
	return newClientSession(tb, l.Addr().String(), cfg)
}

func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("failed to listen on a port: %v", err))
		}
	}
	return l
}

func newClientSession(tb testing.TB, addr string, config *gossh.ClientConfig) (*gossh.Session, *gossh.Client, func()) {
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
		config.HostKeyCallback = gossh.InsecureIgnoreHostKey()
	}
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		tb.Fatal(err)
	}
	session, err := client.NewSession()
	if err != nil {
		tb.Fatal(err)
	}
	return session, client, func() {
		session.Close()
		client.Close()
	}
}
