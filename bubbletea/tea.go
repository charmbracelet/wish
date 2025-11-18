// Package bubbletea provides middleware for serving bubbletea apps over SSH.
package bubbletea

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"github.com/charmbracelet/ssh"
)

// BubbleTeaHandler is the function Bubble Tea apps implement to hook into the
// SSH Middleware. This will create a new tea.Program for every connection and
// start it with the tea.ProgramOptions returned.
//
// Deprecated: use Handler instead.
type BubbleTeaHandler = Handler // nolint: revive

// Handler is the function Bubble Tea apps implement to hook into the
// SSH Middleware. This will create a new tea.Program for every connection and
// start it with the tea.ProgramOptions returned.
type Handler func(sess ssh.Session) (tea.Model, []tea.ProgramOption)

// ProgramHandler is the function Bubble Tea apps implement to hook into the SSH
// Middleware. This should return a new tea.Program. This handler is different
// from the default handler in that it returns a tea.Program instead of
// (tea.Model, tea.ProgramOptions).
//
// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly.
type ProgramHandler func(sess ssh.Session) *tea.Program

// Middleware takes a Handler and hooks the input and output for the
// ssh.Session into the tea.Program.
//
// It also captures window resize events and sends them to the tea.Program
// as tea.WindowSizeMsgs.
func Middleware(handler Handler) wish.Middleware {
	return MiddlewareWithProgramHandler(newDefaultProgramHandler(handler))
}

// MiddlewareWithProgramHandler allows you to specify the ProgramHandler to be
// able to access the underlying tea.Program.
//
// This is useful for creating custom middlewares that need access to
// tea.Program for instance to use p.Send() to send messages to tea.Program.
//
// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly. The recommended way
// of doing so is by using MakeOptions.
func MiddlewareWithProgramHandler(handler ProgramHandler) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			program := handler(sess)
			if program == nil {
				next(sess)
				return
			}
			_, windowChanges, ok := sess.Pty()
			if !ok {
				wish.Fatalln(sess, "no active terminal, skipping")
				return
			}
			ctx, cancel := context.WithCancel(sess.Context())
			go func() {
				for {
					select {
					case <-ctx.Done():
						program.Quit()
						return
					case w := <-windowChanges:
						program.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
					}
				}
			}()
			if _, err := program.Run(); err != nil {
				log.Error("app exit with error", "error", err)
			}
			// p.Kill() will force kill the program if it's still running,
			// and restore the terminal to its original state in case of a
			// tui crash
			program.Kill()
			cancel()
			next(sess)
		}
	}
}

// MakeOptions returns the tea.WithInput and tea.WithOutput program options
// taking into account possible Emulated or Allocated PTYs.
func MakeOptions(sess ssh.Session) []tea.ProgramOption {
	return append(makeOpts(sess), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
		if _, ok := msg.(tea.SuspendMsg); ok {
			return tea.ResumeMsg{}
		}
		return msg
	}))
}

func newDefaultProgramHandler(handler Handler) ProgramHandler {
	return func(s ssh.Session) *tea.Program {
		m, opts := handler(s)
		if m == nil {
			return nil
		}
		return tea.NewProgram(m, append(opts, MakeOptions(s)...)...)
	}
}
