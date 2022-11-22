// Package accesscontrol provides a middleware that allows you to restrict the commands the user can execute.
package accesscontrol

import (
	"fmt"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Middleware will exit 1 connections trying to execute commands that are not allowed.
// If no allowed commands are provided, no commands will be allowed.
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
			fmt.Fprintln(s, "Command is not allowed: "+s.Command()[0])
			s.Exit(1) // nolint: errcheck
		}
	}
}
