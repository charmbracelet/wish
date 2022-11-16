// Package activeterm provides a middleware to block inactive PTYs.
package activeterm

import (
	"fmt"

	"github.com/charmbracelet/wish"
)

// Middleware will exit 1 connections trying with no active terminals.
func Middleware() wish.Middleware {
	return func(sh wish.Handler) wish.Handler {
		return func(s wish.Session) {
			_, _, active := s.Pty()
			if !active {
				fmt.Fprintln(s, "Requires an active PTY")
				s.Exit(1) // nolint: errcheck
				return    // unreachable
			}
			sh(s)
		}
	}
}
