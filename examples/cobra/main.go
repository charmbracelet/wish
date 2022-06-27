package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
)

const (
	host = "localhost"
	port = 23235
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
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			func(h ssh.Handler) ssh.Handler {
				return func(s ssh.Session) {
					rootCmd := cmd()
					rootCmd.SetArgs(s.Command())
					rootCmd.SetIn(s)
					rootCmd.SetOut(s)
					rootCmd.SetErr(s.Stderr())
					rootCmd.Execute()
					h(s)
				}
			},
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Starting SSH server on %s:%d", host, port)
	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}
