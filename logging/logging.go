package logging

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Middleware provides basic connection logging.
// Connects are logged with the remote address, invoked command, TERM setting,
// window dimensions, client version, and if the auth was public key based.
// Disconnect will log the remote address and connection duration.
//
// It will use charmbracelet/log.StandardLog() by default.
func Middleware() wish.Middleware {
	return MiddlewareWithLogger(log.StandardLog())
}

// Logger is the interface that wraps the basic Log method.
type Logger interface {
	Printf(format string, v ...interface{})
}

// MiddlewareWithLogger provides basic connection logging.
// Connects are logged with the remote address, invoked command, TERM setting,
// window dimensions, client version, and if the auth was public key based.
// Disconnect will log the remote address and connection duration.
func MiddlewareWithLogger(logger Logger) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			ct := time.Now()
			hpk := sess.PublicKey() != nil
			pty, _, _ := sess.Pty()
			logger.Printf(
				"%s connect %s %v %v %s %v %v %v",
				sess.User(),
				sess.RemoteAddr().String(),
				hpk,
				sess.Command(),
				pty.Term,
				pty.Window.Width,
				pty.Window.Height,
				sess.Context().ClientVersion(),
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

// StructuredMiddleware provides basic connection logging in a structured form.
// Connects are logged with the remote address, invoked command, TERM setting,
// window dimensions, client version, and if the auth was public key based.
// Disconnect will log the remote address and connection duration.
//
// It will use the charmbracelet/log.Default() and Info level by default.
func StructuredMiddleware() wish.Middleware {
	return StructuredMiddlewareWithLogger(log.Default(), log.InfoLevel)
}

// StructuredMiddlewareWithLogger provides basic connection logging in a structured form.
// Connects are logged with the remote address, invoked command, TERM setting,
// window dimensions, client version, and if the auth was public key based.
// Disconnect will log the remote address and connection duration.
func StructuredMiddlewareWithLogger(logger *log.Logger, level log.Level) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			ct := time.Now()
			hpk := sess.PublicKey() != nil
			pty, _, _ := sess.Pty()
			logger.Log(
				level,
				"connect",
				"user", sess.User(),
				"remote-addr", sess.RemoteAddr().String(),
				"public-key", hpk,
				"command", sess.Command(),
				"term", pty.Term,
				"width", pty.Window.Width,
				"height", pty.Window.Height,
				"client-version", sess.Context().ClientVersion(),
			)
			next(sess)
			logger.Log(
				level,
				"disconnect",
				"user", sess.User(),
				"remote-addr", sess.RemoteAddr().String(),
				"duration", time.Since(ct),
			)
		}
	}
}
