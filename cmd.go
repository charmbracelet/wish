package wish

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/creack/pty"
	"github.com/muesli/cancelreader"
	"golang.org/x/term"
)

// CommandContext is like Command but includes a context.
//
// If the current session does not have a PTY, it sets them to the session
// itself.
func CommandContext(ctx context.Context, s ssh.Session, name string, args ...string) *Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	return &Cmd{s, cmd}
}

// Command sets stdin, stdout, and stderr to the current session's PTY slave.
//
// If the current session does not have a PTY, it sets them to the session
// itself.
//
// This will call CommandContext using the session's Context.
func Command(s ssh.Session, name string, args ...string) *Cmd {
	return CommandContext(s.Context(), s, name, args...)
}

// Cmd wraps a *exec.Cmd and a ssh.Pty so a command can be properly run.
type Cmd struct {
	sess ssh.Session
	cmd  *exec.Cmd
}

// SetDir set the underlying exec.Cmd env.
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
	ppty, _, ok := c.sess.Pty()
	if !ok {
		c.cmd.Stdin, c.cmd.Stdout, c.cmd.Stderr = c.sess, c.sess, c.sess
		return c.cmd.Run()
	}

	// especially on macOS, the slave pty is killed once exec finishes.
	// since we're using it for the ssh session, this would render
	// the pty and the session unusable.
	// so, we need to create another pty, and run the Cmd on it instead.
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

	// TODO: check if this works on windows.
	if runtime.GOOS == "windows" {
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

	return c.cmd.Wait()
}

var _ tea.ExecCommand = &Cmd{}

// SetStderr conforms with tea.ExecCommand.
func (*Cmd) SetStderr(io.Writer) {}

// SetStdin conforms with tea.ExecCommand.
func (*Cmd) SetStdin(io.Reader) {}

// SetStdout conforms with tea.ExecCommand.
func (*Cmd) SetStdout(io.Writer) {}
