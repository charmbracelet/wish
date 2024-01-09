// Package bubbletea provides middleware for serving bubbletea apps over SSH.
package bubbletea

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
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
type Handler func(ssh.Session) (tea.Model, []tea.ProgramOption)

// MakeRenderer returns a lipgloss renderer for the current session.
// This function handle PTYs as well, and should be used to style your application.
func MakeRenderer(s ssh.Session) *lipgloss.Renderer {
	return newRenderer(s)
}

// Middleware takes a Handler and hooks the input and output for the
// ssh.Session into the tea.Program.
//
// It also captures window resize events and sends them to the tea.Program
// as tea.WindowSizeMsgs.
func Middleware(bth Handler) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			_, windowChanges, ok := s.Pty()
			if !ok {
				wish.Fatalln(s, "no active terminal, skipping")
				return
			}

			m, opts := bth(s)
			if m == nil {
				log.Error("no model returned by the handler")
				return
			}

			p := tea.NewProgram(m, append(opts, makeIOOpts(s)...)...)
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
			}
			sh(s)
		}
	}
}
