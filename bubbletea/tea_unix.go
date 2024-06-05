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
	if !ok {
		return []tea.ProgramOption{
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithEnvironment(environ),
		}
	}

	if pty.Term != "" {
		environ = append(environ, "TERM="+pty.Term)
	}

	// XXX: This is a hack to make the output colorized in PTY sessions.
	environ = append(environ, "CLICOLOR_FORCE=1")
	if s.EmulatedPty() {
		return []tea.ProgramOption{
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithEnvironment(environ),
		}
	}

	return []tea.ProgramOption{
		tea.WithInput(pty.Slave),
		tea.WithOutput(pty.Slave),
		tea.WithEnvironment(environ),
	}
}
