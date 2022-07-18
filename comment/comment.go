package comment

import (
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Middleware prints a comment at the end of the session.
func Middleware(comment string) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			sh(s)
			wish.Println(s, comment)
		}
	}
}
