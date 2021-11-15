package activeterm

import (
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Middleware will exit 1 connections trying with no active terminals.
func Middleware() wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			_, _, active := s.Pty()
			if !active {
				s.Exit(1)
				return
			}
			sh(s)
		}
	}
}
