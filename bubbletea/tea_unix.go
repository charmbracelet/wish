//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package bubbletea

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/ssh"
)

func makeOpts(s ssh.Session) []tea.ProgramOption {
	pty, _, ok := s.Pty()
	envs := s.Environ()

	if !ok {
		return []tea.ProgramOption{
			tea.WithInput(s),
			tea.WithOutput(s),
			tea.WithEnvironment(envs),
		}
	}

	// Make sure we have $TERM in the environment when we have a PTY session.
	envs = append(envs, "TERM="+pty.Term)
	if s.EmulatedPty() {
		return []tea.ProgramOption{
			tea.WithInput(s),
			tea.WithOutput(s),
			// Force color profile to be set based on environment variables. We
			// do this because we don't have a real PTY attached, hence
			// [ssh.Session.EmulatedPty]. This is sort of a hack, but it's the
			// best we can do ;)
			tea.WithColorProfile(colorprofile.Env(envs)),
			tea.WithEnvironment(envs),
			tea.WithWindowSize(pty.Window.Width, pty.Window.Height),
		}
	}

	//nolint:godox
	// TODO: Add $SSH_PTY and other environment variables to the environment
	// when we have a real PTY attached.

	return []tea.ProgramOption{
		tea.WithInput(pty.Slave),
		tea.WithOutput(pty.Slave),
		tea.WithEnvironment(envs),
		tea.WithWindowSize(pty.Window.Width, pty.Window.Height),
	}
}
