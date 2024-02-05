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
	return MiddlewareWithLogger(log.StandardLog())
}

// Logger is the interface that wraps the basic Log method.
type Logger interface {
	Printf(format string, v ...interface{})
}

// MiddlewareWithLogger provides basic connection logging. Connects are logged with the
// remote address, invoked command, TERM setting, window dimensions and if the
// auth was public key based. Disconnect will log the remote address and
// connection duration.
func MiddlewareWithLogger(logger Logger) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			ct := time.Now()
			hpk := sess.PublicKey() != nil
			pty, _, _ := sess.Pty()
			logger.Printf(
				"%s connect %s %v %v %s %v %v",
				sess.User(),
				sess.RemoteAddr().String(),
				hpk,
				sess.Command(),
				pty.Term,
				pty.Window.Width,
				pty.Window.Height,
			)
			next(sess)
			logger.Printf(
				"%s disconnect %s\n",
				sess.RemoteAddr().String(),
				time.Since(ct),
			)
		}
	}
}
