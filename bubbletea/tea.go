// Package bubbletea provides middleware for serving bubbletea apps over SSH.
package bubbletea

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/creack/pty"
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
type Handler func(ssh.Session) (tea.Model, []tea.ProgramOption)

// ColoredHandler is the function Bubble Tea apps implement to hook into the
// SSH Middleware. This will create a new tea.Program for every connection and
// start it with the tea.ProgramOptions returned. It receives the current
// ssh.Session and a pre-configured lipgloss.Renderer as args.
type ColoredHandler func(ssh.Session, *lipgloss.Renderer) (tea.Model, []tea.ProgramOption)

// ProgramHandler is the function Bubble Tea apps implement to hook into the SSH
// Middleware. This should return a new tea.Program. This handler is different
// from the default handler in that it returns a tea.Program instead of
// (tea.Model, tea.ProgramOptions).
//
// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly.
type ProgramHandler func(ssh.Session) *tea.Program

// ColoredMiddleware takes a ColoredHandler and hooks the input and output for
// the ssh.Session into the tea.Program. It also captures window resize events
// and sends them to the tea.Program as tea.WindowSizeMsgs.
func ColoredMiddleware(bth ColoredHandler) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			out := outputFromSession(s)
			renderer := lipgloss.NewRenderer(s)
			renderer.SetOutput(out)
			m, opts := bth(s, renderer)
			if m == nil {
				wish.Fatalln(s, "no model returned")
			}
			opts = append(opts, tea.WithInput(s), tea.WithOutput(out))
			p := tea.NewProgram(m, opts...)
			programHandler(s, p)
			sh(s)
		}
	}
}

// Middleware takes a Handler and hooks the input and output for the
// ssh.Session into the tea.Program. It also captures window resize events and
// sends them to the tea.Program as tea.WindowSizeMsgs. By default a 256 color
// profile will be used when rendering with Lip Gloss.
func Middleware(bth Handler) wish.Middleware {
	switch h := any(bth).(type) {
	case Handler:
		return MiddlewareWithColorProfile(h, termenv.ANSI256)
	case ColoredHandler:
		return func(sh ssh.Handler) ssh.Handler {
			return func(s ssh.Session) {
				out := outputFromSession(s)
				renderer := lipgloss.NewRenderer(s)
				renderer.SetOutput(out)
				m, opts := h(s, renderer)
				if m == nil {
					wish.Fatalln(s, "no model returned")
				}
				opts = append(opts, tea.WithInput(s), tea.WithOutput(s))
				p := tea.NewProgram(m, opts...)
				programHandler(s, p)
				sh(s)
			}
		}
	default:
		panic("will never happen")
	}
}

// MiddlewareWithColorProfile allows you to specify the number of colors
// returned by the server when using Lip Gloss. The number of colors supported
// by an SSH client's terminal cannot be detected by the server but this will
// allow for manually setting the color profile on all SSH connections.
func MiddlewareWithColorProfile(bth Handler, cp termenv.Profile) wish.Middleware {
	h := func(s ssh.Session) *tea.Program {
		m, opts := bth(s)
		if m == nil {
			return nil
		}
		opts = append(opts, tea.WithInput(s), tea.WithOutput(s))
		return tea.NewProgram(m, opts...)
	}
	return MiddlewareWithProgramHandler(h, cp)
}

// MiddlewareWithProgramHandler allows you to specify the ProgramHandler to be
// able to access the underlying tea.Program. This is useful for creating custom
// middlewars that need access to tea.Program for instance to use p.Send() to
// send messages to tea.Program.
//
// Make sure to set the tea.WithInput and tea.WithOutput to the ssh.Session
// otherwise the program will not function properly.
func MiddlewareWithProgramHandler(bth ProgramHandler, cp termenv.Profile) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		lipgloss.SetColorProfile(cp)
		return func(s ssh.Session) {
			p := bth(s)
			programHandler(s, p)
			sh(s)
		}
	}
}

func programHandler(s ssh.Session, p *tea.Program) {
	if p != nil {
		_, windowChanges, _ := s.Pty()
		go func() {
			for {
				select {
				case <-s.Context().Done():
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
	}
}

// Bridge Wish and Termenv so we can query for a user's terminal capabilities.
type sshOutput struct {
	ssh.Session
	tty *os.File
}

func (s *sshOutput) Write(p []byte) (int, error) {
	return s.Session.Write(p)
}

func (s *sshOutput) Read(p []byte) (int, error) {
	return s.Session.Read(p)
}

func (s *sshOutput) Fd() uintptr {
	return s.tty.Fd()
}

type sshEnviron struct {
	environ []string
}

func (s *sshEnviron) Getenv(key string) string {
	for _, v := range s.environ {
		k, v, ok := strings.Cut(v, "=")
		if ok && k == key {
			return v
		}
	}
	return ""
}

func (s *sshEnviron) Environ() []string {
	return s.environ
}

func outputFromSession(sess ssh.Session) *termenv.Output {
	sshPty, _, _ := sess.Pty()
	_, tty, err := pty.Open()
	if err != nil {
		log.Fatal(err)
	}
	o := &sshOutput{
		Session: sess,
		tty:     tty,
	}
	environ := sess.Environ()
	environ = append(environ, fmt.Sprintf("TERM=%s", sshPty.Term))
	e := &sshEnviron{environ: environ}
	// We need to use unsafe mode here because the ssh session is not running
	// locally and we already know that the session is a TTY.
	return termenv.NewOutput(o, termenv.WithUnsafe(), termenv.WithEnvironment(e), termenv.WithColorCache(true))
}
