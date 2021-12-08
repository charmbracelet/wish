package ttl_test

import (
	"testing"
	"time"

	"github.com/charmbracelet/wish/testsession"
	"github.com/charmbracelet/wish/ttl"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const expectedTimeoutErr = "Process exited with status 15"
const msg1 = "message 1"
const msg2 = "message 2"

func TestMiddleware(t *testing.T) {
	t.Run("timedOut", func(t *testing.T) {
		sess := setup(t, time.Millisecond)
		bts, err := sess.CombinedOutput("")
		t.Log(string(bts))
		if err == nil || err.Error() != expectedTimeoutErr {
			t.Errorf("expected error %q, got %v", expectedTimeoutErr, err)
		}
		if string(bts) != msg1 {
			t.Errorf("expected output %q, got %q", msg1, string(bts))
		}
	})

	t.Run("do not timeout", func(t *testing.T) {
		sess := setup(t, 15*time.Millisecond)
		bts, err := sess.CombinedOutput("")
		t.Log(string(bts))
		if err != nil {
			t.Errorf("expected no errors, got %v", err)
		}
		if string(bts) != msg1+msg2 {
			t.Errorf("expected output %q, got %q", msg1, string(bts))
		}
	})
}

func setup(t *testing.T, d time.Duration) *gossh.Session {
	session, _, cleanup := testsession.New(t, &ssh.Server{
		Handler: ttl.Middleware(d)(func(s ssh.Session) {
			s.Write([]byte(msg1))
			time.Sleep(time.Millisecond * 10)
			s.Write([]byte(msg2))
		}),
	}, nil)
	t.Cleanup(cleanup)
	return session
}
