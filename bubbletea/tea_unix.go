//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	"image/color"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/input"
	"github.com/charmbracelet/x/term"
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
	if !ok || pty.Term == "" || pty.Term == "dumb" {
		return lipgloss.NewRenderer(s, termenv.WithProfile(termenv.Ascii))
	}
	env := sshEnviron(append(s.Environ(), "TERM="+pty.Term))
	var r *lipgloss.Renderer
	var bg color.Color
	if ok && pty.Slave != nil {
		r = lipgloss.NewRenderer(
			pty.Slave,
			termenv.WithEnvironment(env),
			termenv.WithColorCache(true),
		)
		state, err := term.MakeRaw(pty.Slave.Fd())
		if err == nil {
			bg, _ = queryBackgroundColor(pty.Slave, pty.Slave)
			_ = term.Restore(pty.Slave.Fd(), state)
		}
	} else {
		r = lipgloss.NewRenderer(
			s,
			termenv.WithEnvironment(env),
			termenv.WithUnsafe(),
			termenv.WithColorCache(true),
		)
		bg = querySessionBackgroundColor(s)
	}
	if bg != nil {
		c, ok := colorful.MakeColor(bg)
		if ok {
			_, _, l := c.Hsl()
			r.SetHasDarkBackground(l < 0.5)
		}
	}
	return r
}

// copied from x/term@v0.1.3.
func querySessionBackgroundColor(s ssh.Session) (bg color.Color) {
	_ = queryTerminal(s, s, time.Second, func(events []input.Event) bool {
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
