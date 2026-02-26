//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd && !solaris
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd,!solaris

package bubbletea

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/ssh"
)

func makeOpts(s ssh.Session) []tea.ProgramOption {
	pty, _, ok := s.Pty()
	envs := s.Environ()
	if ok {
		envs = append(envs, "TERM="+pty.Term)
	}
	//nolint:godox
	// TODO: Support Windows PTYs
	return []tea.ProgramOption{
		tea.WithInput(s),
		tea.WithOutput(s),
		tea.WithEnvironment(envs),
		tea.WithWindowSize(pty.Window.Width, pty.Window.Height),
	}
}
