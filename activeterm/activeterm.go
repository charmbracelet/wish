// Package activeterm provides a middleware to block inactive PTYs.
package activeterm

import (
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Middleware will exit 1 connections trying with no active terminals.
func Middleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			_, _, active := sess.Pty()
			if active {
				next(sess)
				return
			}
			wish.Println(sess, "Requires an active PTY")
			_ = sess.Exit(1)
		}
	}
}
