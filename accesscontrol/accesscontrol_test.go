package accesscontrol_test

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/accesscontrol"
	"github.com/charmbracelet/wish/testsession"
	gossh "golang.org/x/crypto/ssh"
)

const out = "hello world"

func TestMiddleware(t *testing.T) {
	requireDenied := func(tb testing.TB, s, cmd string) {
		tb.Helper()
		expected := fmt.Sprintf("Command is not allowed: %s\n", cmd)
		if s != expected {
			t.Errorf("expected %q, got %q", expected, s)
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
		requireDenied(t, string(out), "echo")
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
		requireDenied(t, string(out), "cat")
	})

	t.Run("allowed cmds with allowed cmd followed disallowed cmd", func(t *testing.T) {
		out, err := setup(t, "echo").Output("cat echo")
		if err == nil {
			t.Error(err)
		}
		requireDenied(t, string(out), "cat")
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
