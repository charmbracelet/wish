//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd && !solaris
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd,!solaris

package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/ssh"
)

func makeOpts(s ssh.Session) []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithInput(s),
		tea.WithOutput(s),
	}
}
