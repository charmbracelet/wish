package main

import (
	"bufio"
	"fmt"
	"io"
	"log"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/v2"
	"github.com/charmbracelet/wish/v2/logging"
)

// This is a simple example of a wish server that reads input from the user and
// prints it back to them. It's like a very simple `cat` command over SSH. It
// handles PTYs by allocating ones when requested using [ssh.AllocatePty] and
// using the appropriate input and output streams.

func middleware(sh ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		var input io.Reader = s
		var output io.Writer = s
		pty, _, active := s.Pty()
		if active {
			// When the session request a PTY, like SSHing into server without
			// arguments or using -t, the we need to use a real PTY to read
			// input from the user. Otherwise, terminal settings like echo and
			// line editing won't work.
			log.Printf("session requested pty")

			// TODO: use pty.Read and pty.Write instead.
			// TODO: use platform-independent pty streams.
			input = pty.Slave
			output = pty.Slave
		}

		sc := bufio.NewScanner(input)
		for sc.Scan() {
			fmt.Fprintln(output, sc.Text())
		}

		sh(s)
	}
}

func main() {
	s, err := wish.NewServer(
		// We need to allocate PTYs to read input from the user. Otherwise, we
		// won't be able to read input from the user.
		ssh.AllocatePty(),
		wish.WithHostKeyPath("id_cat"),
		wish.WithAddress(":2022"),
		wish.WithMiddleware(
			logging.MiddlewareWithLogger(log.Default()),
			middleware,
		),
	)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	log.Printf("listening on %q", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}
