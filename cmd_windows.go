//go:build windows
// +build windows

package wish

import (
	"fmt"
	"time"

	"github.com/charmbracelet/ssh"
)

func (c *Cmd) doRun(ppty ssh.Pty, _ <-chan ssh.Window) error {
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
