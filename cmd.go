package wish

import (
	"context"
	"io"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/ssh"
)

// CommandContext is like Command but includes a context.
//
// If the current session does not have a PTY, it sets them to the session
// itself.
func CommandContext(ctx context.Context, s ssh.Session, name string, args ...string) *Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	return &Cmd{sess: s, cmd: cmd}
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
	sess   ssh.Session
	cmd    *exec.Cmd
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
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
func (c *Cmd) SetStderr(w io.Writer) {
	c.stderr = w
}

// SetStdin conforms with tea.ExecCommand.
func (c *Cmd) SetStdin(r io.Reader) {
	c.stdin = r
}

// SetStdout conforms with tea.ExecCommand.
func (c *Cmd) SetStdout(w io.Writer) {
	c.stdout = w
}
