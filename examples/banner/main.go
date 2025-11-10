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

	_ "embed"

	"charm.land/wish/v2"
	"charm.land/wish/v2/elapsed"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/ssh"
)

const (
	host = "localhost"
	port = "23234"
)

//go:embed banner.txt
var banner string

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		// A banner is always shown, even before authentication.
		wish.WithBannerHandler(func(ctx ssh.Context) string {
			return fmt.Sprintf(banner, ctx.User())
		}),
		wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
			return password == "asd123"
		}),
		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(sess ssh.Session) {
					wish.Println(sess, fmt.Sprintf("Hello, %s!", sess.User()))
					next(sess)
				}
			},
			logging.Middleware(),
			// This middleware prints the session duration before disconnecting.
			elapsed.Middleware(),
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
