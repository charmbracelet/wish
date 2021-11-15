package readonly

import (
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Middleware will exit 1 connections trying to execute commands.
func Middleware() wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			if s.RawCommand() != "" {
				s.Exit(1)
				return
			}
			sh(s)
		}
	}
}
