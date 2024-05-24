//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
)

func makeOpts(s ssh.Session) []tea.ProgramOption {
	pty, _, ok := s.Pty()
	environ := s.Environ()
	if pty.Term != "" {
		environ = append(environ, "TERM="+pty.Term)
	}

	opts := []tea.ProgramOption{
		tea.WithInput(s),
		tea.WithOutput(s),
		tea.WithEnvironment(environ),
	}

	if !ok {
		return opts
	}

	if s.EmulatedPty() {
		return append(opts, tea.WithEnvironment(append(environ, "CLICOLOR_FORCE=1")))
	}

	return []tea.ProgramOption{
		tea.WithInput(pty.Slave),
		tea.WithOutput(pty.Slave),
		tea.WithEnvironment(environ),
	}
}
