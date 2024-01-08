//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !openbsd && !solaris
// +build !linux,!darwin,!freebsd,!dragonfly,!netbsd,!openbsd,!solaris

// TODO: support Windows
package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
)

func newRenderer(s ssh.Session) *lipgloss.Renderer {
	return lipgloss.NewRenderer(s, termenv.WithColorCache(true))
}
