//go:build windows
// +build windows

package wish

import (
	"fmt"
	"time"

	"github.com/charmbracelet/ssh"
)

func (c *Cmd) doRun(ppty ssh.Pty, _ <-chan ssh.Window) error {
	// If custom stdio was set (e.g., by tea.Exec), use it instead of PTY.
	// This ensures proper sequencing of alt screen escape sequences when using
	// tea.Exec with bubbletea.Middleware.
	// On Windows, we only use the custom path if all stdio are set, since
	// there's no ppty.Slave to fall back to for individual handles.
	if c.stdin != nil && c.stdout != nil && c.stderr != nil {
		c.cmd.Stdin = c.stdin
		c.cmd.Stdout = c.stdout
		c.cmd.Stderr = c.stderr
		return c.cmd.Run()
	}
	// Original behavior: use PTY slave for all stdio via ppty.Start()
	if err := ppty.Start(c.cmd); err != nil {
		return err
	}

	start := time.Now()
	for c.cmd.ProcessState == nil {
		if time.Since(start) > time.Second*10 {
			return fmt.Errorf("could not start process")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !c.cmd.ProcessState.Success() {
		return fmt.Errorf("process failed: exit %d", c.cmd.ProcessState.ExitCode())
	}
	return nil
}
