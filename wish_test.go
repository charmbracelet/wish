// go:generate mockgen -package mocks -destination mocks/session.go github.com/charmbracelet/ssh Session
package wish

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
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

func TestError(t *testing.T) {
	eerr := errors.New("foo err")
	sess := testsession.New(t, &ssh.Server{
		Handler: func(s ssh.Session) {
			Error(s, eerr)
		},
	}, nil)
	var out bytes.Buffer
	sess.Stderr = &out
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
