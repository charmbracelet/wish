package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/logging"
	"github.com/charmbracelet/ssh"
	"github.com/spf13/cobra"
)

const (
	host = "localhost"
	port = "23235"
)

func cmd() *cobra.Command {
	var reverse bool
	cmd := &cobra.Command{
		Use:  "echo [string]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := args[0]
			if reverse {
				ss := make([]byte, 0, len(s))
				for i := len(s) - 1; i >= 0; i-- {
					ss = append(ss, s[i])
				}
				s = string(ss)
			}
			cmd.Println(s)
			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&reverse, "reverse", "r", false, "Reverse string on echo")
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
					rootCmd := cmd()
					rootCmd.SetArgs(sess.Command())
					rootCmd.SetIn(sess)
					rootCmd.SetOut(sess)
					rootCmd.SetErr(sess.Stderr())
					rootCmd.CompletionOptions.DisableDefaultCmd = true
					if err := rootCmd.Execute(); err != nil {
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
