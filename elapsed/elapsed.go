package timer

import (
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// MiddlewareWithFormat returns a middleware that logs the elapsed time of the
// session. It accepts a format string to print the elapsed time.
//
// This must be called as the last middleware in the chain.
func MiddlewareWithFormat(format string) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			now := time.Now()
			sh(s)
			wish.Printf(s, format, time.Since(now))
		}
	}
}

// Middleware returns a middleware that logs the elapsed time of the session.
//
// This must be called as the last middleware in the chain.
func Middleware() wish.Middleware {
	return MiddlewareWithFormat("elapsed time: %v\n")
}
