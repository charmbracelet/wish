//go:build dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build dragonfly freebsd linux netbsd openbsd solaris

package wish

import "github.com/charmbracelet/ssh"

func (c *Cmd) doRun(ppty ssh.Pty) error {
	if err := ppty.Start(c.cmd); err != nil {
		return err
	}
	return c.cmd.Wait()
}
