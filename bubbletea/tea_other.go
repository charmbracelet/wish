//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd && !solaris
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd,!solaris

package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
)

func makeIOOpts(s ssh.Session) []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithInput(s),
		tea.WithOutput(s),
	}
}

func newRenderer(s ssh.Session) *lipgloss.Renderer {
	return lipgloss.NewRenderer(s, termenv.WithColorCache(true))
}
