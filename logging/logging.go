package logging

import (
	"log"
	"time"
	"wish"

	"github.com/gliderlabs/ssh"
)

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
