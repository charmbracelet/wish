package logging_test

import (
	"testing"

	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("inactive term", func(t *testing.T) {
		if err := setup(t).Run(""); err != nil {
			t.Error(err)
		}
	})
}

func setup(t *testing.T) *gossh.Session {
	session, _, cleanup := testsession.New(t, &ssh.Server{
		Handler: logging.Middleware()(func(s ssh.Session) {
			s.Write([]byte("hello"))
		}),
	}, nil)
	t.Cleanup(cleanup)
	return session
}
