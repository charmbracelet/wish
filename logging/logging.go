package logging

import (
	"log"
	"time"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Middleware provides basic connection logging. Connects are logged with the
// remote address, invoked command, TERM setting, window dimensions and if the
// auth was public key based. Disconnect will log the remote address and
// connection duration.
func Middleware() wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			ct := time.Now()
			hpk := s.PublicKey() != nil
			pty, _, _ := s.Pty()
			log.Printf("%s connect %v %v %s %v %v\n", s.RemoteAddr().String(), hpk, s.Command(), pty.Term, pty.Window.Width, pty.Window.Height)
			sh(s)
			log.Printf("%s disconnect %s\n", s.RemoteAddr().String(), time.Since(ct))
		}
	}
}