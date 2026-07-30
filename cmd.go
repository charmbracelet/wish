// Package wish provides utilities for building SSH servers.
package wish

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
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

// SetEnv sets the underlying exec.Cmd env.
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
	if c.hasCustomStdio() {
		c.cmd.Stdin, c.cmd.Stdout, c.cmd.Stderr = c.stdin, c.stdout, c.stderr
		return c.cmd.Run() //nolint:wrapcheck
	}
	ppty, winCh, ok := c.sess.Pty()
	if !ok {
		c.cmd.Stdin, c.cmd.Stdout, c.cmd.Stderr = c.sess, c.sess, c.sess.Stderr()
		if c.stdin != nil {
			c.cmd.Stdin = c.stdin
		}
		if c.stdout != nil {
			c.cmd.Stdout = c.stdout
		}
		if c.stderr != nil {
			c.cmd.Stderr = c.stderr
		}
		if err := c.cmd.Run(); err != nil {
			return fmt.Errorf("run command: %w", err)
		}
		return nil
	}
	return c.doRun(ppty, winCh)
}

// hasCustomStdio reports whether all three stdio handles were set, e.g. by
// tea.Exec. The gate is all-or-nothing: a partial set keeps the default
// session/PTY wiring, since ppty.Start cannot mix custom handles with the
// PTY slave.
func (c *Cmd) hasCustomStdio() bool {
	return c.stdin != nil && c.stdout != nil && c.stderr != nil
}

var _ tea.ExecCommand = &Cmd{}

// SetStderr conforms with tea.ExecCommand. It must be called before Run;
// Cmd is not safe for concurrent use.
func (c *Cmd) SetStderr(w io.Writer) {
	c.stderr = w
}

// SetStdin conforms with tea.ExecCommand. It must be called before Run;
// Cmd is not safe for concurrent use.
func (c *Cmd) SetStdin(r io.Reader) {
	c.stdin = r
}

// SetStdout conforms with tea.ExecCommand. It must be called before Run;
// Cmd is not safe for concurrent use.
func (c *Cmd) SetStdout(w io.Writer) {
	c.stdout = w
}
