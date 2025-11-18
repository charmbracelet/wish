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
)

const (
	host = "localhost"
	port = "23234"
)

// example usage: ssh -N -R 23236:localhost:23235 -p 23234 localhost

func main() {
	// Create a new SSH ForwardedTCPHandler.
	forwardHandler := &ssh.ForwardedTCPHandler{}
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		func(s *ssh.Server) error {
			// Set the Reverse TCP Handler up:
			s.ReversePortForwardingCallback = func(_ ssh.Context, bindHost string, bindPort uint32) bool {
				log.Info("reverse port forwarding allowed", "host", bindHost, "port", bindPort)
				return true
			}
			s.RequestHandlers = map[string]ssh.RequestHandler{
				"tcpip-forward":        forwardHandler.HandleSSHRequest,
				"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
			}
			return nil
		},
		wish.WithMiddleware(
			func(h ssh.Handler) ssh.Handler {
				return func(s ssh.Session) {
					wish.Println(s, "Remote port forwarding available!")
					wish.Println(s, "Try it with:")
					wish.Println(s, "  ssh -N -R 23236:localhost:23235 -p 23234 localhost")
					h(s)
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
