//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package wish

import "github.com/charmbracelet/ssh"

func (c *Cmd) doRun(ppty ssh.Pty, _ <-chan ssh.Window) error {
	// If ALL custom stdio are set (e.g., by tea.Exec), use them instead of PTY.
	// This ensures proper sequencing of alt screen escape sequences when using
	// tea.Exec with bubbletea.Middleware.
	// We require all three to be set to avoid deadlocks with partial PTY usage.
	//
	// NOTE: We only check for this if we are NOT on Windows, as Windows
	// handling is slightly different (see cmd_windows.go).
	if c.stdin != nil && c.stdout != nil && c.stderr != nil {
		c.cmd.Stdin = c.stdin
		c.cmd.Stdout = c.stdout
		c.cmd.Stderr = c.stderr
		return c.cmd.Run()
	}
	// Original behavior: use PTY for all stdio via ppty.Start()
	if err := ppty.Start(c.cmd); err != nil {
		return err
	}
	return c.cmd.Wait()
}
