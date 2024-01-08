//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd && !solaris
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd,!solaris

package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
)

// MakeRenderer returns a new lipgloss.Renderer for the given ssh.Session.
// It will use the pty if one is available, otherwise it will use the session
// writer.
func MakeRenderer(s ssh.Session) *lipgloss.Renderer {
	return lipgloss.NewRenderer(s, termenv.WithUnsafe(), termenv.WithColorCache(true))
}

func makeIOOpts(s ssh.Session) []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithInput(s),
		tea.WithOutput(s),
	}
}
