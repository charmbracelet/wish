package logging

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Middleware provides basic connection logging. Connects are logged with the
// remote address, invoked command, TERM setting, window dimensions and if the
// auth was public key based. Disconnect will log the remote address and
// connection duration.
//
// The logger is set to the std default logger.
func Middleware() wish.Middleware {
	return MiddlewareWithLogger(log.Default())
}

// MiddlewareWithLogger provides basic connection logging. Connects are logged with the
// remote address, invoked command, TERM setting, window dimensions and if the
// auth was public key based. Disconnect will log the remote address and
// connection duration.
func MiddlewareWithLogger(l log.Logger) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			ct := time.Now()
			hpk := s.PublicKey() != nil
			pty, _, _ := s.Pty()
			l.Info(
				"connect",
				"user", s.User(),
				"remoteaddr", s.RemoteAddr().String(),
				"publickey", hpk,
				"command", s.Command(),
				"term", pty.Term,
				"width", pty.Window.Width,
				"height", pty.Window.Height,
			)
			sh(s)
			l.Info("disconnect", "remoteaddr", s.RemoteAddr().String(), "duration", time.Since(ct))
		}
	}
}
