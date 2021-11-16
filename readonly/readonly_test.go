package readonly_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/charmbracelet/wish/readonly"
	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const out = "hello world"

func TestMiddleware(t *testing.T) {
	requireEmpty := func(tb testing.TB, s string) {
		tb.Helper()
		if s != "" {
			tb.Errorf("expected output to be empty, got %q", s)
		}
	}

	requireOutput := func(tb testing.TB, s string) {
		tb.Helper()
		if out != s {
			t.Errorf("expected %q, got %q", out, s)
		}
	}

	t.Run("no allowed cmds no cmd", func(t *testing.T) {
		var b bytes.Buffer
		if err := setup(t, &b).Run(""); err != nil {
			t.Error(err)
		}
		requireOutput(t, b.String())
	})

	t.Run("no allowed cmds with cmd", func(t *testing.T) {
		var b bytes.Buffer
		if err := setup(t, &b).Run("echo"); err == nil {
			t.Errorf("should have errored")
		}
		requireEmpty(t, b.String())
	})

	t.Run("allowed cmds no cmd", func(t *testing.T) {
		var b bytes.Buffer
		if err := setup(t, &b, "echo").Run(""); err != nil {
			t.Error(err)
		}
		requireOutput(t, b.String())
	})

	t.Run("allowed cmds with allowed cmd", func(t *testing.T) {
		var b bytes.Buffer
		if err := setup(t, &b, "echo").Run("echo"); err != nil {
			t.Error(err)
		}
		requireOutput(t, b.String())
	})

	t.Run("allowed cmds with disallowed cmd", func(t *testing.T) {
		var b bytes.Buffer
		if err := setup(t, &b, "echo").Run("cat"); err == nil {
			t.Error(err)
		}
		requireEmpty(t, b.String())
	})

	t.Run("allowed cmds with allowed cmd followed disallowed cmd", func(t *testing.T) {
		var b bytes.Buffer
		if err := setup(t, &b, "echo").Run("cat echo"); err == nil {
			t.Error(err)
		}
		requireEmpty(t, b.String())
	})
}

func setup(t *testing.T, w io.Writer, allowedCmds ...string) *gossh.Session {
	session, _, cleanup := testsession.New(t, &ssh.Server{
		Handler: readonly.Middleware(allowedCmds...)(func(s ssh.Session) {
			s.Write([]byte(out))
		}),
	}, nil)
	t.Cleanup(cleanup)
	session.Stdout = w
	return session
}
