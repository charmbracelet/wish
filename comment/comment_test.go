package comment

import (
	"testing"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
	gossh "golang.org/x/crypto/ssh"
)

func TestMiddleware(t *testing.T) {
	t.Run("recover session", func(t *testing.T) {
		b, err := setup(t).Output("")
		requireNoError(t, err)
		if string(b) != "test\n" {
			t.Errorf("expected comment to be 'test', got %q", string(b))
		}
	})
}

func setup(tb testing.TB) *gossh.Session {
	tb.Helper()
	return testsession.New(tb, &ssh.Server{
		Handler: Middleware("test")(func(s ssh.Session) {}),
	}, nil)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expected no error, got %q", err.Error())
	}
}
