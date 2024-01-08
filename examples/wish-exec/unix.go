//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
)

func newRenderer(s ssh.Session) *lipgloss.Renderer {
	pty, _, _ := s.Pty()
	return lipgloss.NewRenderer(pty.Slave, termenv.WithColorCache(true))
}
