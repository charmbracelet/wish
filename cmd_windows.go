//go:build windows
// +build windows

package wish

import (
	"charm.land/ssh"
	"github.com/charmbracelet/x/xpty"
)

func (c *Cmd) doRun(ppty ssh.Pty, _ <-chan ssh.Window) error {
	if err := ppty.Start(c.cmd); err != nil {
		return err //nolint:wrapcheck
	}
	// cmd.Wait() doesn't work with ConPTY; xpty.WaitProcess waits on the
	// process directly and honors the session context for cancellation.
	return xpty.WaitProcess(c.sess.Context(), c.cmd) //nolint:wrapcheck
}
