package recover

import (
	stdlog "log"
	"runtime/debug"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Middleware is a wish middleware that recovers from panics and log to stderr.
func Middleware(mw ...wish.Middleware) wish.Middleware {
	return MiddlewareWithLogger(nil, mw...)
}

// MiddlewareWithLogger is a wish middleware that recovers from panics and log to
// the provided logger.
func MiddlewareWithLogger(logger *stdlog.Logger, mw ...wish.Middleware) wish.Middleware {
	if logger == nil {
		logger = log.StandardLog()
	}
	h := func(ssh.Session) {}
	for _, m := range mw {
		h = m(h)
	}
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Printf("panic: %v\n%s", r, string(debug.Stack()))
					}
				}()
				h(s)
			}()
			sh(s)
		}
	}
}
