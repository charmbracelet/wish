package recover

import (
	"testing"

	"charm.land/ssh"
	"charm.land/wish/v2/testsession"
	gossh "golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("recover session", func(t *testing.T) {
		_, err := setup(t).Output("")
		requireNoError(t, err)
	})

	// The next handler in the chain used to run outside the recover, so a
	// panic there escaped and killed the whole server process instead of
	// just the session. This test fails by crashing if that regresses.
	t.Run("recover next handler", func(t *testing.T) {
		sess := testsession.New(t, &ssh.Server{
			Handler: Middleware(func(h ssh.Handler) ssh.Handler {
				return func(s ssh.Session) { h(s) }
			})(func(s ssh.Session) {
				panic("panic in next handler")
			}),
		}, nil)
		_, err := sess.Output("")
		requireNoError(t, err)
	})

	// Both guarded sections must run: a panic in the wrapped chain should not
	// stop the next handler from being called.
	t.Run("next handler runs after chain panics", func(t *testing.T) {
		called := make(chan struct{}, 1)
		sess := testsession.New(t, &ssh.Server{
			Handler: Middleware(func(h ssh.Handler) ssh.Handler {
				return func(s ssh.Session) { panic("panic in chain") }
			})(func(s ssh.Session) {
				called <- struct{}{}
			}),
		}, nil)
		_, err := sess.Output("")
		requireNoError(t, err)

		select {
		case <-called:
		default:
			t.Error("next handler was not called after the chain panicked")
		}
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
