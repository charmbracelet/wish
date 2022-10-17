// go:genarate mockgen -package mocks -destination mocks/session.go github.com/gliderlabs/ssh Session
package wish

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
)

func TestNewServer(t *testing.T) {
	_, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewServerWithOptions(t *testing.T) {
	_, err := NewServer(
		WithMaxTimeout(time.Second),
		WithAddress(":2222"),
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestErrorActive(t *testing.T) {
	eerr := errors.New("foo err")
	sess := testsession.New(t, &ssh.Server{
		Handler: func(s ssh.Session) {
			_, _, active := s.Pty()
			if !active {
				t.Error("expected active pty, got inactive")
			}
			Error(s, eerr)
		},
	}, nil)
	var out bytes.Buffer
	sess.Stderr = &out
	if err := sess.RequestPty("xterm", 80, 40, nil); err != nil {
		t.Errorf("failed to request pty: %v", err)
	}
	if err := sess.Run(""); err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if s := strings.TrimSpace(out.String()); s != eerr.Error() {
		t.Errorf("expected %s, got %s", s, eerr)
	}
}

func TestFatal(t *testing.T) {
	err := errors.New("foo err")
	sess := testsession.New(t, &ssh.Server{
		Handler: func(s ssh.Session) {
			Fatal(s, err)
		},
	}, nil)
	var out bytes.Buffer
	sess.Stderr = &out
	if err := sess.Run(""); err == nil {
		t.Error("expected an error, got nil")
	}
	if s := strings.TrimSpace(out.String()); s != err.Error() {
		t.Errorf("expected %s, got %s", s, err)
	}
}
