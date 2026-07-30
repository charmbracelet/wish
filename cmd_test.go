package wish

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
	"time"

	"charm.land/ssh"
	"charm.land/wish/v2/testsession"
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

// TestCommandSetStdio verifies that Run uses the custom handles when all
// three are set. The cat round trip proves both stdin and stdout flow
// through: output can only appear in the buffer if both are wired.
func TestCommandSetStdio(t *testing.T) {
	srv := &ssh.Server{
		Handler: func(s ssh.Session) {
			cmd := Command(s, "cat")
			if runtime.GOOS == "windows" {
				cmd = Command(s, "findstr", "roundtrip")
			}
			var out, errOut bytes.Buffer
			cmd.SetStdin(strings.NewReader("roundtrip\n"))
			cmd.SetStdout(&out)
			cmd.SetStderr(&errOut)
			if err := cmd.Run(); err != nil {
				Fatal(s, err)
			}
			if !strings.Contains(out.String(), "roundtrip") {
				Fatalf(s, "expected stdin to round trip through custom stdout, got %q", out.String())
			}
			_, _ = s.Write([]byte("SUCCESS"))
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
	sess.Stdout = &stdout
	if err := sess.Run(""); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectContains(t, stdout.String(), "SUCCESS")
}

// TestCommandSetStdioPartial verifies that a partial set keeps the default
// PTY wiring: the gate is all-or-nothing.
func TestCommandSetStdioPartial(t *testing.T) {
	srv := &ssh.Server{
		Handler: func(s ssh.Session) {
			cmd := Command(s, "echo", "partial")
			if runtime.GOOS == "windows" {
				cmd = Command(s, "cmd", "/C", "echo", "partial")
			}
			// Only stdout is set, so the command must still run on the PTY
			// and the session must see the output.
			var sink bytes.Buffer
			cmd.SetStdout(&sink)
			if err := cmd.Run(); err != nil {
				Fatal(s, err)
			}
			if sink.Len() != 0 {
				Fatalf(s, "expected PTY fallback with partial stdio set, but output went to custom stdout: %q", sink.String())
			}
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
	sess.Stdout = &stdout
	if err := sess.Run(""); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectContains(t, stdout.String(), "partial")
}

func runEcho(s ssh.Session, str string) {
	cmd := Command(s, "echo", str)
	if runtime.GOOS == "windows" {
		cmd = Command(s, "cmd", "/C", "echo", str)
	}
	// With all handles nil, the gate stays closed and the command runs on
	// the session/PTY as before.
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
