//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
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
	if !ok || pty.Slave == nil {
		return lipgloss.NewRenderer(s, termenv.WithEnvironment(env), termenv.WithUnsafe(), termenv.WithColorCache(true))
	}
	return lipgloss.NewRenderer(pty.Slave, termenv.WithEnvironment(env), termenv.WithColorCache(true))
}
