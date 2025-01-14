package logging_test

import (
	"testing"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/v2"
	"github.com/charmbracelet/wish/v2/logging"
	"github.com/charmbracelet/wish/v2/testsession"
	gossh "golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("inactive term", func(t *testing.T) {
		if err := setup(t, logging.Middleware()).Run(""); err != nil {
			t.Error(err)
		}
	})
}

func TestStructuredMiddleware(t *testing.T) {
	t.Run("inactive term", func(t *testing.T) {
		if err := setup(t, logging.StructuredMiddleware()).Run(""); err != nil {
			t.Error(err)
		}
	})
}

func setup(tb testing.TB, middleware wish.Middleware) *gossh.Session {
	tb.Helper()
	return testsession.New(tb, &ssh.Server{
		Handler: middleware(func(s ssh.Session) {
			s.Write([]byte("hello"))
		}),
	}, nil)
}
