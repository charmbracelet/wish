package wish

import (
	"testing"
	"time"

	"github.com/gliderlabs/ssh"
)

func TestWithIdleTimeout(t *testing.T) {
	s := ssh.Server{}
	requireNoError(t, WithIdleTimeout(time.Second)(&s))
	requireEqual(t, time.Second, s.IdleTimeout)
}

func TestWithMaxTimeout(t *testing.T) {
	s := ssh.Server{}
	requireNoError(t, WithMaxTimeout(time.Second)(&s))
	requireEqual(t, time.Second, s.MaxTimeout)
}

func requireEqual(tb testing.TB, a, b interface{}) {
	tb.Helper()
	if a != b {
		tb.Errorf("expected %v, got %v", a, b)
	}
}

func requireNoError(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Errorf("expected no error, got %v", err)
	}
}
