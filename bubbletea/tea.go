// Package bubbletea provides middleware for serving bubbletea apps over SSH.
package bubbletea

import (
	"context"

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

// Middleware takes a Handler and hooks the input and output for the
// ssh.Session into the tea.Program.
//
// It also captures window resize events and sends them to the tea.Program
// as tea.WindowSizeMsgs.
func Middleware(bth Handler) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			tty, windowChanges, ok := s.Pty()
			if !ok {
				wish.Fatalln(s, "no active terminal, skipping")
				return
			}

			renderer := lipgloss.NewRenderer(tty.Slave, termenv.WithColorCache(true))

			m, opts := bth(s, renderer)
			if m == nil {
				log.Error("no model returned by the handler")
				return
			}

			p := tea.NewProgram(m, append(
				opts,
				tea.WithInput(tty.Slave),
				tea.WithOutput(tty.Slave),
			)...)
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

			}
			sh(s)
		}
	}
}
