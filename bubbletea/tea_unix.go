//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/ssh"
)

func makeOpts(s ssh.Session) []tea.ProgramOption {
	pty, _, ok := s.Pty()
	if !ok || s.EmulatedPty() {
		envs := append(s.Environ(), "TERM="+pty.Term)
		return []tea.ProgramOption{
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithEnvironment(envs),
			tea.WithColorProfile(colorprofile.Env(envs)),
		}
	}

	return []tea.ProgramOption{
		tea.WithInput(pty.Slave),
		tea.WithOutput(pty.Slave),
		tea.WithEnvironment(s.Environ()),
	}
}
