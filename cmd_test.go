package wish

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
)

func TestCommandNoPty(t *testing.T) {
	tmp := t.TempDir()
	sess := testsession.New(t, &ssh.Server{
		Handler: func(s ssh.Session) {
			if err := Command(s, "echo", "hello").Run(); err != nil {
				Fatal(s, err)
			}

			cmd := Command(s, "env")
			cmd.SetEnv([]string{"HELLO=hello"})
			if len(cmd.Environ()) != 1 {
				Fatal(s, "unexpected cmd environ:", cmd.Environ())
			}
			if err := cmd.Run(); err != nil {
				Fatal(s, err)
			}

			cmd = Command(s, "pwd")
			cmd.SetDir(tmp)
			// these should do nothing...
			cmd.SetStderr(nil)
			cmd.SetStdin(nil)
			cmd.SetStdout(nil)
			if err := cmd.Run(); err != nil {
				Fatal(s, err)
			}
		},
	}, nil)
	var out bytes.Buffer
	sess.Stdout = &out
	if err := sess.Run(""); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expect := strings.Join([]string{
		"hello",
		"HELLO=hello",
		tmp,
	}, "\n") + "\n"
	if s := out.String(); s != expect {
		t.Errorf("expected output to be %q, got %q", expect, s)
	}
}

func TestCommandPty(t *testing.T) {
	tmp := t.TempDir()
	srv := &ssh.Server{
		Handler: func(s ssh.Session) {
			if err := Command(s, "echo", "hello").Run(); err != nil {
				Fatal(s, err)
			}

			cmd := Command(s, "env")
			cmd.SetEnv([]string{"HELLO=hello"})
			if err := cmd.Run(); err != nil {
				Fatal(s, err)
			}

			cmd = Command(s, "pwd")
			cmd.SetDir(tmp)
			// these should do nothing...
			cmd.SetStderr(nil)
			cmd.SetStdin(nil)
			cmd.SetStdout(nil)
			if err := cmd.Run(); err != nil {
				Fatal(s, err)
			}
		},
	}
	if err := ssh.AllocatePty()(srv); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	sess := testsession.New(t, srv, nil)
	if err := sess.RequestPty("xterm", 500, 200, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var out bytes.Buffer
	sess.Stdout = &out
	if err := sess.Run(""); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expect := strings.Join([]string{
		"hello",
		"HELLO=hello",
		tmp,
	}, "\r\n") + "\r\n"
	if s := out.String(); s != expect {
		t.Errorf("expected output to be %q, got %q", expect, s)
	}
}

func TestCommandPtyError(t *testing.T) {
	srv := &ssh.Server{
		Handler: func(s ssh.Session) {
			if err := Command(s, "nopenopenope").Run(); err != nil {
				Fatal(s, err)
			}
		},
	}
	if err := ssh.AllocatePty()(srv); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	sess := testsession.New(t, srv, nil)
	if err := sess.RequestPty("xterm", 500, 200, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var out bytes.Buffer
	sess.Stderr = &out
	if err := sess.Run(""); err == nil {
		t.Errorf("expected an error, got nil")
	}
	expect := `exec: "nopenopenope"`
	if s := out.String(); !strings.Contains(s, expect) {
		t.Errorf("expected output to contain %q, got %q", expect, s)
	}
}
