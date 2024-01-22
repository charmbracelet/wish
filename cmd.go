package wish

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"

	"github.com/charmbracelet/ssh"
)

// Command sets stdin, stdout, and stderr to the current session's PTY slave.
//
// If the current session does not have a PTY, it sets them to the session
// itself.
func Command(s ssh.Session, name string, args ...string) *Cmd {
	c := exec.Command(name, args...)
	pty, _, ok := s.Pty()
	if !ok {
		c.Stdin, c.Stdout, c.Stderr = s, s, s
		return &Cmd{cmd: c}
	}

	return &Cmd{c, &pty}
}

// Cmd wraps a *exec.Cmd and a ssh.Pty so a command can be properly run.
type Cmd struct {
	cmd *exec.Cmd
	pty *ssh.Pty
}

// SetDir set the underlying exec.Cmd env.
func (c *Cmd) SetEnv(env []string) {
	c.cmd.Env = env
}

// Environ returns the underlying exec.Cmd environment.
func (c *Cmd) Environ() []string {
	return c.cmd.Environ()
}

// SetDir set the underlying exec.Cmd dir.
func (c *Cmd) SetDir(dir string) {
	c.cmd.Dir = dir
}

// SetStderr conforms with tea.ExecCommand.
func (*Cmd) SetStderr(io.Writer) {}

// SetStdin conforms with tea.ExecCommand.
func (*Cmd) SetStdin(io.Reader) {}

// SetStdout conforms with tea.ExecCommand.
func (*Cmd) SetStdout(io.Writer) {}

// Run runs the program and waits for it to finish.
func (c *Cmd) Run() error {
	if c.pty == nil {
		return c.cmd.Run()
	}

	if err := c.pty.Start(c.cmd); err != nil {
		return err
	}
	start := time.Now()
	if runtime.GOOS == "windows" {
		for c.cmd.ProcessState == nil {
			if time.Since(start) > time.Second*10 {
				return fmt.Errorf("could not start process")
			}
			time.Sleep(100 * time.Millisecond)
		}

		if !c.cmd.ProcessState.Success() {
			return fmt.Errorf("process failed: exit %d", c.cmd.ProcessState.ExitCode())
		}
	} else {
		if err := c.cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}
