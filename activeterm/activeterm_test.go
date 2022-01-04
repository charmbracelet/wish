package activeterm_test

import (
	"testing"

	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("inactive term", func(t *testing.T) {
		if err := setup(t).Run(""); err == nil {
			t.Errorf("tests should be an inactive pty")
		}
	})
}

func setup(tb testing.TB) *gossh.Session {
	tb.Helper()
	return testsession.New(tb, &ssh.Server{
		Handler: activeterm.Middleware()(func(s ssh.Session) {
			s.Write([]byte("hello"))
		}),
	}, nil)
}
