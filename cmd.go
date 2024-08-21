package wish

import (
	"context"
	"io"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
)

// CommandContext is like Command but includes a context.
//
// If the current session does not have a PTY, it sets them to the session
// itself.
func CommandContext(ctx context.Context, s ssh.Session, name string, args ...string) *Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	return &Cmd{s, cmd}
}

// Command sets stdin, stdout, and stderr to the current session's PTY.
//
// If the current session does not have a PTY, it sets them to the session
// itself.
//
// This will use the session's context as the context for exec.Command.
func Command(s ssh.Session, name string, args ...string) *Cmd {
	return CommandContext(s.Context(), s, name, args...)
}

// Cmd wraps a *exec.Cmd and a ssh.Pty so a command can be properly run.
type Cmd struct {
	sess ssh.Session
	cmd  *exec.Cmd
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

// Run runs the program and waits for it to finish.
func (c *Cmd) Run() error {
	ppty, winCh, ok := c.sess.Pty()
	if !ok {
		c.cmd.Stdin, c.cmd.Stdout, c.cmd.Stderr = c.sess, c.sess, c.sess.Stderr()
		return c.cmd.Run()
	}
	return c.doRun(ppty, winCh)
}

var _ tea.ExecCommand = &Cmd{}

// SetStderr conforms with tea.ExecCommand.
func (*Cmd) SetStderr(io.Writer) {}

// SetStdin conforms with tea.ExecCommand.
func (*Cmd) SetStdin(io.Reader) {}

// SetStdout conforms with tea.ExecCommand.
func (*Cmd) SetStdout(io.Writer) {}
