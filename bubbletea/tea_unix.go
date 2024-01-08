//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
)

// MakeRenderer returns a new lipgloss.Renderer for the given ssh.Session.
// It will use the pty if one is available, otherwise it will use the session
// writer.
func MakeRenderer(s ssh.Session) *lipgloss.Renderer {
	var f io.Writer = s
	pty, _, ok := s.Pty()
	if ok {
		f = pty.Slave
	}
	return lipgloss.NewRenderer(f, termenv.WithColorCache(true))
}

func makeIOOpts(s ssh.Session) []tea.ProgramOption {
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
