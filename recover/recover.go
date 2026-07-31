// Package recover provides a middleware that recovers from panics.
package recover

import (
	"runtime/debug"

	"charm.land/log/v2"
	"charm.land/ssh"
	"charm.land/wish/v2"
)

// Middleware is a wish middleware that recovers from panics and log to stderr.
func Middleware(mw ...wish.Middleware) wish.Middleware {
	return MiddlewareWithLogger(nil, mw...)
}

// Logger is the interface that wraps the basic Log method.
type Logger interface {
	Printf(format string, v ...any)
}

// MiddlewareWithLogger is a wish middleware that recovers from panics and log to
// the provided logger.
func MiddlewareWithLogger(logger Logger, mw ...wish.Middleware) wish.Middleware {
	if logger == nil {
		logger = log.StandardLog()
	}
	h := func(ssh.Session) {}
	for _, m := range mw {
		h = m(h)
	}
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			guard(logger, func() { h(s) })
			guard(logger, func() { sh(s) })
		}
	}
}

// guard runs fn and recovers any panic it raises, logging it with a stack
// trace.
//
// Both the wrapped middleware chain and the next handler are guarded. A panic
// in either runs on the connection's goroutine, and Go has no process-wide
// panic handler, so letting one escape would terminate the whole server
// process rather than just the offending session.
func guard(logger Logger, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logger.Printf(
				"panic: %v\n%s",
				r,
				string(debug.Stack()),
			)
		}
	}()
	fn()
}
