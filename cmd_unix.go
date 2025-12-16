//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package wish

import "github.com/charmbracelet/ssh"

func (c *Cmd) doRun(ppty ssh.Pty, _ <-chan ssh.Window) error {
	// If custom stdio was set (e.g., by tea.Exec), use it instead of PTY slave.
	// This ensures proper sequencing of alt screen escape sequences when using
	// tea.Exec with bubbletea.Middleware.
	if c.stdin != nil || c.stdout != nil || c.stderr != nil {
		if c.stdin != nil {
			c.cmd.Stdin = c.stdin
		} else {
			c.cmd.Stdin = ppty.Slave
		}
		if c.stdout != nil {
			c.cmd.Stdout = c.stdout
		} else {
			c.cmd.Stdout = ppty.Slave
		}
		if c.stderr != nil {
			c.cmd.Stderr = c.stderr
		} else {
			c.cmd.Stderr = ppty.Slave
		}
		return c.cmd.Run()
	}
	// Original behavior: use PTY slave for all stdio via ppty.Start()
	if err := ppty.Start(c.cmd); err != nil {
		return err
	}
	return c.cmd.Wait()
}
