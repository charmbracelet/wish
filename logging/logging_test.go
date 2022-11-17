package logging_test

import (
	"testing"

	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/testsession"
	"golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("inactive term", func(t *testing.T) {
		if err := setup(t).Run(""); err != nil {
			t.Error(err)
		}
	})
}

func setup(tb testing.TB) *ssh.Session {
	tb.Helper()
	return testsession.New(tb, &wish.Server{
		Handler: logging.Middleware()(func(s wish.Session) {
			s.Write([]byte("hello"))
		}),
	}, nil)
}
