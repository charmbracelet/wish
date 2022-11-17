package comment

import "github.com/charmbracelet/wish"

// Middleware prints a comment at the end of the session.
func Middleware(comment string) wish.Middleware {
	return func(sh wish.Handler) wish.Handler {
		return func(s wish.Session) {
			sh(s)
			wish.Println(s, comment)
		}
	}
}
