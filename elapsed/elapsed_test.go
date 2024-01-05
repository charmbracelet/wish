package elapsed

import (
	"testing"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
	gossh "golang.org/x/crypto/ssh"
)

var waitDuration = time.Second

func TestMiddleware(t *testing.T) {
	t.Run("recover session", func(t *testing.T) {
		b, err := setup(t).Output("")
		requireNoError(t, err)
		dur, err := time.ParseDuration(string(b))
		requireNoError(t, err)
		if dur < waitDuration {
			t.Errorf("expected elapsed time to be at least 1s, got %v", dur)
		}
	})
}

func setup(tb testing.TB) *gossh.Session {
	tb.Helper()
	return testsession.New(tb, &ssh.Server{
		Handler: MiddlewareWithFormat("%v")(func(s ssh.Session) {
			time.Sleep(waitDuration)
		}),
	}, nil)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expected no error, got %q", err.Error())
	}
}
