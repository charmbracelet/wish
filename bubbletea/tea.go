// Package bubbletea provides middleware for serving bubbletea apps over SSH.
package bubbletea

import (
	"context"
	"io"

	"github.com/aymanbagabas/go-pty"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/muesli/termenv"
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
type Handler func(ssh.Session, *lipgloss.Renderer) (tea.Model, []tea.ProgramOption)

// ProgramHandler is the function Bubble Tea apps implement to hook into the SSH
// Middleware. This should return a new tea.Program. This handler is different
// from the default handler in that it returns a tea.Program instead of
// (tea.Model, tea.ProgramOptions).
//
// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly.
type ProgramHandler func(ssh.Session) *tea.Program

// MiddlewareWithProgramHandler allows you to specify the ProgramHandler to be
// able to access the underlying tea.Program. This is useful for creating custom
// middlewars that need access to tea.Program for instance to use p.Send() to
// send messages to tea.Program.
//
// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly.
func Middleware(bth Handler) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			opts := []tea.ProgramOption{tea.WithInput(s), tea.WithOutput(s)}

			tty, windowChanges, ok := s.Pty()
			if !ok {
				wish.Fatalln(s, "no active terminal, skipping")
				return
			}

			upty, ok := tty.Pty.(pty.UnixPty)
			if ok {
				f := upty.Slave()
				opts = []tea.ProgramOption{tea.WithInput(f), tea.WithOutput(f)}
			}

			renderer := lipgloss.NewRenderer(tty, termenv.WithColorCache(true))

			m, hopts := bth(s, renderer)
			if m == nil {
				log.Error("no model returned by the handler")
				return
			}

			opts = append(opts, hopts...)

			p := tea.NewProgram(m, opts...)
			if p != nil {
				ctx, cancel := context.WithCancel(s.Context())
				go func() {
					for {
						select {
						case <-ctx.Done():
							if p != nil {
								p.Quit()
								return
							}
						case w := <-windowChanges:
							if p != nil {
								p.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
							}
						}
					}
				}()
				if _, err := p.Run(); err != nil {
					log.Error("app exit with error", "error", err)
				}
				// p.Kill() will force kill the program if it's still running,
				// and restore the terminal to its original state in case of a
				// tui crash
				p.Kill()
				cancel()
				if err := tty.Close(); err != nil {
					log.Error("could not close pty", "error", err)
					return
				}

				sh(s)
			}
		}
	}
}

// Command makes a *pty.Cmd executable with tea.Exec.
func Command(c *pty.Cmd) tea.ExecCommand { return &ptyCommand{c} }

type ptyCommand struct{ *pty.Cmd }

func (*ptyCommand) SetStderr(io.Writer) {} // noop
func (*ptyCommand) SetStdin(io.Reader)  {} // noop
func (*ptyCommand) SetStdout(io.Writer) {} // noop
