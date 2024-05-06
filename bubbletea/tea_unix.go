//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	"image/color"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/x/exp/term"
	"github.com/charmbracelet/x/exp/term/ansi"
	"github.com/charmbracelet/x/exp/term/input"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
)

func makeOpts(s ssh.Session) []tea.ProgramOption {
	pty, _, ok := s.Pty()
	if !ok || s.EmulatedPty() {
		return []tea.ProgramOption{
			tea.WithInput(s),
			tea.WithOutput(s),
		}
	}

	return []tea.ProgramOption{
		tea.WithInput(pty.Slave),
		tea.WithOutput(pty.Slave),
	}
}

func newRenderer(s ssh.Session) *lipgloss.Renderer {
	pty, _, ok := s.Pty()
	env := sshEnviron(append(s.Environ(), "TERM="+pty.Term))
	var r *lipgloss.Renderer
	var bg color.Color
	if ok && pty.Slave != nil {
		r = lipgloss.NewRenderer(
			pty.Slave,
			termenv.WithEnvironment(env),
			termenv.WithColorCache(true),
		)
		bg = term.BackgroundColor(pty.Slave, pty.Slave)
	} else {
		r = lipgloss.NewRenderer(
			s,
			termenv.WithEnvironment(env),
			termenv.WithUnsafe(),
			termenv.WithColorCache(true),
		)
		bg = queryBackgroundColor(s)
	}
	c, ok := colorful.MakeColor(bg)
	if ok {
		_, _, l := c.Hsl()
		r.SetHasDarkBackground(l < 0.5)
	}
	return r
}

// copied from x/exp/term.
func queryBackgroundColor(s ssh.Session) (bg color.Color) {
	_ = term.QueryTerminal(s, s, func(events []input.Event) bool {
		for _, e := range events {
			switch e := e.(type) {
			case input.BackgroundColorEvent:
				bg = e.Color
				continue // we need to consume the next DA1 event
			case input.PrimaryDeviceAttributesEvent:
				return false
			}
		}
		return true
	}, ansi.RequestBackgroundColor+ansi.RequestPrimaryDeviceAttributes)
	return
}
