package main

import (
	"errors"
	"net"

	"github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/v2"
	"github.com/charmbracelet/wish/v2/logging"
)

const (
	host = "localhost"
	port = "23234"
)

func main() {
	srv, err := wish.NewServer(
		// The address the server will listen to.
		wish.WithAddress(net.JoinHostPort(host, port)),

		// The SSH server need its own keys, this will create a keypair in the
		// given path if it doesn't exist yet.
		// By default, it will create an ED25519 key.
		wish.WithHostKeyPath(".ssh/id_ed25519"),

		// Middlewares do something on a ssh.Session, and then call the next
		// middleware in the stack.
		wish.WithMiddleware(
			func(next ssh.Handler) ssh.Handler {
				return func(sess ssh.Session) {
					wish.Println(sess, "Hello, world!")
					next(sess)
				}
			},

			// The last item in the chain is the first to be called.
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	log.Info("Starting SSH server", "host", host, "port", port)
	if err = srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		// We ignore ErrServerClosed because it is expected.
		log.Error("Could not start server", "error", err)
	}
}
