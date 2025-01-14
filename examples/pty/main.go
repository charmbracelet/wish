//go:build !windows

package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/v2"
	"github.com/charmbracelet/wish/v2/activeterm"
	"github.com/charmbracelet/wish/v2/logging"
)

const (
	host = "localhost"
	port = "23234"
)

func main() {
	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),

		// Wish can allocate a PTY per user session.
		ssh.AllocatePty(),

		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(sess ssh.Session) {
					pty, _, _ := sess.Pty()

					wish.Printf(sess, "Hello, world!\r\n")
					wish.Printf(sess, "Term: %s\r\n", pty.Term)
					wish.Printf(sess, "PTY: %s\r\n", pty.Slave.Name())
					wish.Printf(sess, "FD: %d\r\n", pty.Slave.Fd())
					next(sess)
				}
			},

			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("Starting SSH server", "host", host, "port", port)
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	log.Info("Stopping SSH server")
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}
