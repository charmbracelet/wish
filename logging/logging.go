package logging

import (
	"log"
	"time"

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

// Logger is the interface that wraps the basic Log method.
type Logger interface {
	Printf(format string, v ...interface{})
}

// MiddlewareWithLogger provides basic connection logging. Connects are logged with the
// remote address, invoked command, TERM setting, window dimensions and if the
// auth was public key based. Disconnect will log the remote address and
// connection duration.
func MiddlewareWithLogger(l Logger) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			ct := time.Now()
			hpk := s.PublicKey() != nil
			pty, _, _ := s.Pty()
			l.Printf("%s connect %s %v %v %s %v %v\n", s.User(), s.RemoteAddr().String(), hpk, s.Command(), pty.Term, pty.Window.Width, pty.Window.Height)
			sh(s)
			l.Printf("%s disconnect %s\n", s.RemoteAddr().String(), time.Since(ct))
		}
	}
}
