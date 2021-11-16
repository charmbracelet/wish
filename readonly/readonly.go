package readonly

import (
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Middleware will exit 1 connections trying to execute commands that are not allowed.
func Middleware(cmds ...string) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			if len(s.Command()) == 0 {
				sh(s)
				return
			}
			for _, cmd := range cmds {
				if s.Command()[0] == cmd {
					sh(s)
					return
				}
			}
			s.Exit(1)
		}
	}
}
