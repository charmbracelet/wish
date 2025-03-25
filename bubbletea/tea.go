// Package bubbletea provides middleware for serving bubbletea apps over SSH.
package bubbletea

import (
	"context"
	"fmt"
	"strings"

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
	return MiddlewareWithProgramHandler(newDefaultProgramHandler(handler), termenv.Ascii)
}

// MiddlewareWithColorProfile allows you to specify the minimum number of colors
// this program needs to work properly.
//
// If the client's color profile has less colors than p, p will be forced.
// Use with caution.
func MiddlewareWithColorProfile(handler Handler, profile termenv.Profile) wish.Middleware {
	return MiddlewareWithProgramHandler(newDefaultProgramHandler(handler), profile)
}

// MiddlewareWithProgramHandler allows you to specify the ProgramHandler to be
// able to access the underlying tea.Program, and the minimum supported color
// profile.
//
// This is useful for creating custom middlewares that need access to
// tea.Program for instance to use p.Send() to send messages to tea.Program.
//
// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly. The recommended way
// of doing so is by using MakeOptions.
//
// If the client's color profile has less colors than p, p will be forced.
// Use with caution.
func MiddlewareWithProgramHandler(handler ProgramHandler, profile termenv.Profile) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			sess.Context().SetValue(minColorProfileKey, profile)
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

var minColorProfileKey struct{}

var profileNames = [4]string{"TrueColor", "ANSI256", "ANSI", "Ascii"}

// MakeRenderer returns a lipgloss renderer for the current session.
// This function handle PTYs as well, and should be used to style your application.
func MakeRenderer(sess ssh.Session) *lipgloss.Renderer {
	cp, ok := sess.Context().Value(minColorProfileKey).(termenv.Profile)
	if !ok {
		cp = termenv.Ascii
	}

	r := newRenderer(sess)

	// We only force the color profile if the requested session is a PTY.
	_, _, ok = sess.Pty()
	if !ok {
		return r
	}

	if r.ColorProfile() > cp {
		_, _ = fmt.Fprintf(sess.Stderr(), "Warning: Client's terminal is %q, forcing %q\r\n",
			profileNames[r.ColorProfile()], profileNames[cp])
		r.SetColorProfile(cp)
	}
	return r
}

// MakeOptions returns the tea.WithInput and tea.WithOutput program options
// taking into account possible Emulated or Allocated PTYs.
func MakeOptions(sess ssh.Session) []tea.ProgramOption {
	return makeOpts(sess)
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

func newDefaultProgramHandler(handler Handler) ProgramHandler {
	return func(s ssh.Session) *tea.Program {
		m, opts := handler(s)
		if m == nil {
			return nil
		}
		return tea.NewProgram(m, append(opts, makeOpts(s)...)...)
	}
}
