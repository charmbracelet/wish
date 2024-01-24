//go:build darwin
// +build darwin

package wish

import (
	"errors"
	"io"

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
func (c *Cmd) doRun(ppty ssh.Pty, winCh <-chan ssh.Window) error {
	done := make(chan struct{}, 1)
	go func() { <-done; close(done) }()
	ptmx, err := pty.Start(c.cmd)
	if err != nil {
		return err
	}
	defer func() {
		<-done
		if err := ptmx.Close(); err != nil {
			log.Warn("could not close pty", "err", err)
		}
	}()

	// setup resizes
	go func() {
		for {
			select {
			case <-done:
				return
			case w := <-winCh:
				log.Infof("resize %d %d", w.Height, w.Width)
				if err := pty.Setsize(ptmx, &pty.Winsize{
					Rows: uint16(w.Height),
					Cols: uint16(w.Width),
				}); err != nil {
					log.Warn("could not set term size", "err", err)
				}
			}
		}
	}()
	if err := pty.InheritSize(ppty.Slave, ptmx); err != nil {
		return err
	}

	// put the ssh session's pty in raw mode
	oldState, err := term.MakeRaw(int(ppty.Slave.Fd()))
	if err != nil {
		return err
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
		return err
	}
	defer func() {
		stdin.Cancel()
	}()

	// sync io
	go func() {
		if _, err := io.Copy(ptmx, stdin); err != nil {
			done <- struct{}{}
			if errors.Is(err, io.EOF) || errors.Is(err, cancelreader.ErrCanceled) {
				// safe to ignore
				return
			}
			log.Warn("failed to copy", "err", err)
		}
		done <- struct{}{}
	}()
	if _, err := io.Copy(ppty.Slave, ptmx); err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	return c.cmd.Wait()
}
