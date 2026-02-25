package wish

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
)

func TestCommandNoPty(t *testing.T) {
	tmp := t.TempDir()
	sess := testsession.New(t, &ssh.Server{
		Handler: func(s ssh.Session) {
			runEcho(s, "hello")
			runEnv(s, []string{"HELLO=world"})
			runPwd(s, tmp)
		},
	}, nil)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr
	if err := sess.Run(""); err != nil {
		t.Errorf("expected no error, got %v: %s", err, stderr.String())
	}
	out := stdout.String()
	expectContains(t, out, "hello")
	expectContains(t, out, "HELLO=world")
	expectContains(t, out, tmp)
}

func TestCommandPty(t *testing.T) {
	tmp := t.TempDir()
	srv := &ssh.Server{
		Handler: func(s ssh.Session) {
			runEcho(s, "hello")
			runEnv(s, []string{"HELLO=world"})
			runPwd(s, tmp)
			// for some reason sometimes on macos github action runners,
			// it cuts parts of the output.
			time.Sleep(100 * time.Millisecond)
		},
	}
	if err := ssh.AllocatePty()(srv); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	sess := testsession.New(t, srv, nil)
	if err := sess.RequestPty("xterm", 500, 200, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr
	if err := sess.Run(""); err != nil {
		t.Errorf("expected no error, got %v: %s", err, stderr.String())
	}
	out := stdout.String()
	expectContains(t, out, "hello")
	expectContains(t, out, "HELLO=world")
	expectContains(t, out, tmp)
}

func TestCommandPtyError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
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

	var stderr bytes.Buffer
	sess.Stderr = &stderr
	if err := sess.Run(""); err == nil {
		t.Errorf("expected an error, got nil")
	}
	expect := `exec: "nopenopenope"`
	if s := stderr.String(); !strings.Contains(s, expect) {
		t.Errorf("expected output to contain %q, got %q", expect, s)
	}
}

// TestCommandSetStdio verifies that SetStdin, SetStdout, SetStderr methods
// properly store custom I/O handles. This is important for tea.Exec integration
// where Bubble Tea sets these to share I/O with the child process.
func TestCommandSetStdio(t *testing.T) {
	srv := &ssh.Server{
		Handler: func(s ssh.Session) {
			cmd := Command(s, "echo", "custom")
			var buf bytes.Buffer
			// Set all stdio handles (required for custom stdio path)
			cmd.SetStdin(strings.NewReader(""))
			cmd.SetStdout(&buf)
			cmd.SetStderr(&buf)
			if err := cmd.Run(); err != nil {
				Fatal(s, err)
			}
			// Verify that output went to our custom buffer
			if !strings.Contains(buf.String(), "custom") {
				Fatal(s, "expected output in custom buffer")
			}
			// Write success marker to session
			_, _ = s.Write([]byte("SUCCESS"))
		},
	}
	if err := ssh.AllocatePty()(srv); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	sess := testsession.New(t, srv, nil)
	if err := sess.RequestPty("xterm", 500, 200, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var stdout bytes.Buffer
	sess.Stdout = &stdout
	if err := sess.Run(""); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectContains(t, stdout.String(), "SUCCESS")
}

func runEcho(s ssh.Session, str string) {
	cmd := Command(s, "echo", str)
	if runtime.GOOS == "windows" {
		cmd = Command(s, "cmd", "/C", "echo", str)
	}
	// Setting nil should be safe and not change behavior
	cmd.SetStderr(nil)
	cmd.SetStdin(nil)
	cmd.SetStdout(nil)
	if err := cmd.Run(); err != nil {
		Fatal(s, err)
	}
}

func runEnv(s ssh.Session, env []string) {
	cmd := Command(s, "env")
	if runtime.GOOS == "windows" {
		cmd = Command(s, "cmd", "/C", "set")
	}
	cmd.SetEnv(env)
	if err := cmd.Run(); err != nil {
		Fatal(s, err)
	}
	if len(cmd.Environ()) == 0 {
		Fatal(s, "cmd.Environ() should not be empty")
	}
}

func runPwd(s ssh.Session, dir string) {
	cmd := Command(s, "pwd")
	if runtime.GOOS == "windows" {
		cmd = Command(s, "cmd", "/C", "cd")
	}
	cmd.SetDir(dir)
	if err := cmd.Run(); err != nil {
		Fatal(s, err)
	}
}

func expectContains(tb testing.TB, s, substr string) {
	if !strings.Contains(s, substr) {
		tb.Errorf("expected output %q to contain %q", s, substr)
	}
}
