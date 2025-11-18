package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"
)

const (
	host = "localhost"
	port = "23234"
)

var users = map[string]string{
	"Carlos": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILxWe2rXKoiO6W14LYPVfJKzRfJ1f3Jhzxrgjc/D4tU7",
	// You can add add your name and public key here :)
}

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		// This will allow anyone to log in, as long as they have given an
		// ed25519 public key.
		// You can test this by doing something like:
		//		ssh -i ~/.ssh/id_ed25519 -p 23234 localhost
		//		ssh -i ~/.ssh/id_rsa -p 23234 localhost
		//		ssh -o PreferredAuthentications=password -p 23234 localhost
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return key.Type() == "ssh-ed25519"
		}),
		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(sess ssh.Session) {
					// if the current session's user public key is one of the
					// known users, we greet them and return.
					for name, pubkey := range users {
						parsed, _, _, _, _ := ssh.ParseAuthorizedKey(
							[]byte(pubkey),
						)
						if ssh.KeysEqual(sess.PublicKey(), parsed) {
							wish.Println(sess, fmt.Sprintf("Hey %s!", name))
							next(sess)
							return
						}
					}
					wish.Println(sess, "Hey, I don't know who you are!")
					next(sess)
				}
			},
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}
