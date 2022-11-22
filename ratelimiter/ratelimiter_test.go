package ratelimiter

import (
	"testing"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

func TestRateLimiterNoLimit(t *testing.T) {
	s := &ssh.Server{
		Handler: Middleware(NewRateLimiter(rate.Limit(0), 0, 5))(func(s ssh.Session) {
			s.Write([]byte("hello"))
		}),
	}

	sess := testsession.New(t, s, nil)
	if err := sess.Run(""); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestRateLimiterZeroedMaxEntried(t *testing.T) {
	s := &ssh.Server{
		Handler: Middleware(NewRateLimiter(rate.Limit(1), 1, 0))(func(s ssh.Session) {
			s.Write([]byte("hello"))
		}),
	}

	sess := testsession.New(t, s, nil)
	if err := sess.Run(""); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRateLimiter(t *testing.T) {
	s := &ssh.Server{
		Handler: Middleware(NewRateLimiter(rate.Limit(10), 4, 1))(func(s ssh.Session) {
			// noop
		}),
	}

	addr := testsession.Listen(t, s)

	g := errgroup.Group{}
	for i := 0; i < 10; i++ {
		g.Go(func() error {
			sess, err := testsession.NewClientSession(t, addr, nil)
			if err != nil {
				t.Fatalf("expected no errors, got %v", err)
			}
			if err := sess.Run(""); err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err == nil {
		t.Fatal("expected error, got nil")
	}

	// after some time, it should reset and pass again
	time.Sleep(100 * time.Millisecond)
	sess, err := testsession.NewClientSession(t, addr, nil)
	if err != nil {
		t.Fatalf("expected no errors, got %v", err)
	}
	if err := sess.Run(""); err != nil {
		t.Fatalf("expected no errors, got %v", err)
	}
}
