//go:build darwin
// +build darwin

package wish

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/creack/pty"
	"github.com/muesli/cancelreader"
	"golang.org/x/term"
)

// on macOS, the slave pty is killed once exec finishes.
// since we're using it for the ssh session, this would render
// the pty and the session unusable.
// so, we need to create another pty, and run the Cmd on it instead.
func (c *Cmd) doRun(ppty ssh.Pty) error {
	ptmx, err := pty.Start(c.cmd)
	if err != nil {
		return fmt.Errorf("cmd: %w", err)
	}
	defer func() {
		if err := ptmx.Close(); err != nil {
			log.Warn("could not close pty", "err", err)
		}
	}()

	// setup resizes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(ppty.Master, ptmx); err != nil {
				log.Warn("error resizing pty", "err", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH // initial size
	defer func() {
		signal.Stop(ch)
		close(ch)
	}()

	// put the ssh session's pty in raw mode
	oldState, err := term.MakeRaw(int(ppty.Slave.Fd()))
	if err != nil {
		return fmt.Errorf("cmd: %w", err)
	}
	defer func() {
		if err := term.Restore(int(ppty.Slave.Fd()), oldState); err != nil {
			log.Error("could not restore terminal", "err", err)
		}
	}()

	// we'll need to be able to cancel the reader, otherwise the copy
	// from ptmx will eat the next keypress after the exec exits.
	stdin, err := cancelreader.NewReader(ppty.Slave)
	if err != nil {
		return fmt.Errorf("cmd: %w", err)
	}
	defer func() { stdin.Cancel() }()

	// sync io
	go func() {
		if _, err := io.Copy(ptmx, stdin); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, cancelreader.ErrCanceled) {
				// safe to ignore
				return
			}
			log.Warn("failed to copy", "err", err)
		}
	}()
	if _, err := io.Copy(ppty.Slave, ptmx); err != nil {
		if !errors.Is(err, io.EOF) && !errors.Is(err, syscall.EIO) {
			return fmt.Errorf("cmd: copy: %w", err)
		}
		log.Warn("failed to copy", "err", err)
	}

	return c.cmd.Wait()
}
