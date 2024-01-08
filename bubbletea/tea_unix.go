//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
)

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
