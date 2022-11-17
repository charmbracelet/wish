package recover

import (
	"testing"

	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/testsession"
	"golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("recover session", func(t *testing.T) {
		_, err := setup(t).Output("")
		requireNoError(t, err)
	})
}

func setup(tb testing.TB) *ssh.Session {
	tb.Helper()
	return testsession.New(tb, &wish.Server{
		Handler: Middleware(func(h wish.Handler) wish.Handler {
			return func(s wish.Session) {
				panic("hello")
			}
		})(func(s wish.Session) {}),
	}, nil)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expected no error, got %q", err.Error())
	}
}
