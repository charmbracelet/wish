// Package bubbletea provides middleware for serving bubbletea apps over SSH.
package bubbletea

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/muesli/termenv"
)

// ProgramHandler is the function Bubble Tea apps implement to hook into the SSH
// Middleware. This should return a new tea.Program. This handler is different
// from the default handler in that it returns a tea.Program instead of
// (tea.Model, tea.ProgramOptions).
//
// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly.
type ProgramHandler func(ssh.Session) *tea.Program

// Handler is the function Bubble Tea apps implement to hook into the
// SSH Middleware. This will create a new tea.Program for every connection and
// start it with the tea.ProgramOptions returned.
type Handler func(ssh.Session) (tea.Model, []tea.ProgramOption)

// MakeRenderer returns a lipgloss renderer for the current session.
// This function handle PTYs as well, and should be used to style your application.
func MakeRenderer(s ssh.Session) *lipgloss.Renderer {
	return newRenderer(s)
}

// MakePTYAwareOpts returns tea.WithInput and tea.WithOutput taking into
// account Emulated and Allocated PTYs.
func MakePTYAwareOpts(s ssh.Session) []tea.ProgramOption {
	return makeOpts(s)
}

// Middleware takes a Handler and hooks the input and output for the
// ssh.Session into the tea.Program.
//
// It also captures window resize events and sends them to the tea.Program
// as tea.WindowSizeMsgs.
func Middleware(bth Handler) wish.Middleware {
	h := func(s ssh.Session) *tea.Program {
		m, opts := bth(s)
		if m == nil {
			return nil
		}
		return tea.NewProgram(m, append(opts, makeOpts(s)...)...)
	}
	return MiddlewareWithProgramHandler(h)
}

// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly.
func MiddlewareWithProgramHandler(bth ProgramHandler) wish.Middleware {
	return func(h ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			_, windowChanges, ok := s.Pty()
			if !ok {
				wish.Fatalln(s, "no active terminal, skipping")
				return
			}
			p := bth(s)
			if p == nil {
				h(s)
				return
			}
			ctx, cancel := context.WithCancel(s.Context())
			go func() {
				for {
					select {
					case <-ctx.Done():
						p.Quit()
						return
					case w := <-windowChanges:
						p.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
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
			h(s)
		}
	}
}

type sshEnviron []string

var _ termenv.Environ = sshEnviron(nil)

// Environ implements termenv.Environ.
func (e sshEnviron) Environ() []string {
	return e
}

// Getenv implements termenv.Environ.
func (e sshEnviron) Getenv(k string) string {
	for _, v := range e {
		if strings.HasPrefix(v, k+"=") {
			return v[len(k)+1:]
		}
	}
	return ""
}
