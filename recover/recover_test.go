package recover

import (
	"testing"

	"charm.land/wish/v2/testsession"
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("recover session", func(t *testing.T) {
		_, err := setup(t).Output("")
		requireNoError(t, err)
	})
}

func setup(tb testing.TB) *gossh.Session {
	tb.Helper()
	return testsession.New(tb, &ssh.Server{
		Handler: Middleware(func(h ssh.Handler) ssh.Handler {
			return func(s ssh.Session) {
				panic("hello")
			}
		})(func(s ssh.Session) {}),
	}, nil)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expected no error, got %q", err.Error())
	}
}
