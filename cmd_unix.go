//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package wish

import "github.com/charmbracelet/ssh"

func (c *Cmd) doRun(ppty ssh.Pty, _ <-chan ssh.Window) error {
	if err := ppty.Start(c.cmd); err != nil {
		return err
	}
	return c.cmd.Wait()
}
