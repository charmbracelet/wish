package recover

import (
	"log"
	"runtime/debug"

	"github.com/charmbracelet/wish"
)

// Middleware is a wish middleware that recovers from panics and log to stderr.
func Middleware(mw ...wish.Middleware) wish.Middleware {
	return MiddlewareWithLogger(nil, mw...)
}

// MiddlewareWithLogger is a wish middleware that recovers from panics and log to
// the provided logger.
func MiddlewareWithLogger(logger *log.Logger, mw ...wish.Middleware) wish.Middleware {
	if logger == nil {
		logger = log.Default()
	}
	h := func(wish.Session) {}
	for _, m := range mw {
		h = m(h)
	}
	return func(sh wish.Handler) wish.Handler {
		return func(s wish.Session) {
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
