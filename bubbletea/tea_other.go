//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd && !solaris
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd,!solaris

package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
)

func makeOpts(s ssh.Session) []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithInput(s),
		tea.WithOutput(s),
	}
}

func newRenderer(s ssh.Session) *lipgloss.Renderer {
	pty, _, _ := s.Pty()
	env := sshEnviron(append(s.Environ(), "TERM="+pty.Term))
	return lipgloss.NewRenderer(s, termenv.WithEnvironment(env), termenv.WithUnsafe(), termenv.WithColorCache(true))
}
