// Package ttl provides a middleware to put a TTL to a session.
package ttl

import (
	"log"
	"time"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Middleware that sends a SIGTERM to connections after the given time.
func Middleware(ttl time.Duration) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			go func() {
				time.Sleep(ttl)
				log.Println("reached ttl, closing session")
				if err := s.CloseWrite(); err != nil {
					log.Printf("failed to close session: %v", err)
				}
				s.Exit(15)
			}()
			sh(s)
		}
	}
}
