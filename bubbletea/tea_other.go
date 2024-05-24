//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd && !solaris
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd,!solaris

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

	if !ok {
		return []tea.ProgramOption{
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithEnvironment(environ),
		}
	}

	return []tea.ProgramOption{
		tea.WithInput(s),
		tea.WithOutput(s),
		tea.WithEnvironment(append(environ, "CLICOLOR_FORCE=1")),
	}
}
