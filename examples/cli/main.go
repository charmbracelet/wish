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

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	"github.com/urfave/cli/v3"
)

const (
	host = "localhost"
	port = "23235"
)

func cmd(sess ssh.Session) *cli.Command {
	var reverse bool
	cmd := &cli.Command{
		Usage: "echo [string] [--reverse]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "reverse",
				Aliases:     []string{"r"},
				Usage:       "Reverse string on echo",
				Destination: &reverse,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			s := command.Args().First()
			if s == "" {
				return errors.New("no string provided")
			}
			if reverse {
				ss := make([]byte, 0, len(s))
				for i := len(s) - 1; i >= 0; i-- {
					ss = append(ss, s[i])
				}
				s = string(ss)
			}
			if _, err := fmt.Fprintln(sess, s); err != nil {
				return fmt.Errorf("failed to write to session: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(sess ssh.Session) {
					// Here we wire our command's args and IO to the user
					// session's
					rootCmd := cmd(sess)
					args := []string{"echo"}
					args = append(args, sess.Command()...)
					if err := rootCmd.Run(context.Background(), args); err != nil {
						_, _ = fmt.Fprintln(sess.Stderr(), err)
						_ = sess.Exit(1)
						return
					}

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
