package activeterm_test

import (
	"testing"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/testsession"
	gossh "golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("inactive term", func(t *testing.T) {
		out, err := setup(t).Output("")
		if err == nil {
			t.Errorf("tests should be an inactive pty")
		}
		if string(out) != "Requires an active PTY\n" {
			t.Errorf("invalid output: %q", string(out))
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
