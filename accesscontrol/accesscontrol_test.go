package accesscontrol_test

import (
	"testing"

	"github.com/charmbracelet/wish/accesscontrol"
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
		out, err := setup(t).Output("")
		if err != nil {
			t.Error(err)
		}
		requireOutput(t, string(out))
	})

	t.Run("no allowed cmds with cmd", func(t *testing.T) {
		out, err := setup(t).Output("echo")
		if err == nil {
			t.Errorf("should have errored")
		}
		requireEmpty(t, string(out))
	})

	t.Run("allowed cmds no cmd", func(t *testing.T) {
		out, err := setup(t, "echo").Output("")
		if err != nil {
			t.Error(err)
		}
		requireOutput(t, string(out))
	})

	t.Run("allowed cmds with allowed cmd", func(t *testing.T) {
		out, err := setup(t, "echo").Output("echo")
		if err != nil {
			t.Error(err)
		}
		requireOutput(t, string(out))
	})

	t.Run("allowed cmds with disallowed cmd", func(t *testing.T) {
		out, err := setup(t, "echo").Output("cat")
		if err == nil {
			t.Error(err)
		}
		requireEmpty(t, string(out))
	})

	t.Run("allowed cmds with allowed cmd followed disallowed cmd", func(t *testing.T) {
		out, err := setup(t, "echo").Output("cat echo")
		if err == nil {
			t.Error(err)
		}
		requireEmpty(t, string(out))
	})
}

func setup(tb testing.TB, allowedCmds ...string) *gossh.Session {
	tb.Helper()
	return testsession.New(tb, &ssh.Server{
		Handler: accesscontrol.Middleware(allowedCmds...)(func(s ssh.Session) {
			s.Write([]byte(out))
		}),
	}, nil)
}
