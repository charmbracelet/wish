package main

// An example git server. This will list all available repos if you ssh
// directly to the server. To test `ssh -p 23233 localhost` once it's running.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/git"
	"github.com/charmbracelet/wish/logging"
)

const (
	port    = "23233"
	host    = "localhost"
	repoDir = ".repos"
)

type app struct {
	access git.AccessLevel
}

func (a app) AuthRepo(string, ssh.PublicKey) git.AccessLevel {
	return a.access
}

func (a app) Push(repo string, _ ssh.PublicKey) {
	log.Info("push", "repo", repo)
}

func (a app) Fetch(repo string, _ ssh.PublicKey) {
	log.Info("fetch", "repo", repo)
}

func main() {
	// A simple GitHooks implementation to allow global read write access.
	a := app{git.ReadWriteAccess}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		// Accept any public key.
		ssh.PublicKeyAuth(func(ssh.Context, ssh.PublicKey) bool { return true }),
		// Do not accept password auth.
		ssh.PasswordAuth(func(ssh.Context, string) bool { return false }),
		wish.WithMiddleware(
			// Setup the git middleware.
			git.Middleware(repoDir, a),
			// Adds a middleware to list all available repositories to the user.
			gitListMiddleware,
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

// Normally we would use a Bubble Tea program for the TUI but for simplicity,
// we'll just write a list of the pushed repos to the terminal and exit the ssh
// session.
func gitListMiddleware(next ssh.Handler) ssh.Handler {
	return func(sess ssh.Session) {
		// Git will have a command included so only run this if there are no
		// commands passed to ssh.
		if len(sess.Command()) != 0 {
			next(sess)
			return
		}

		dest, err := os.ReadDir(repoDir)
		if err != nil && err != fs.ErrNotExist {
			log.Error("Invalid repository", "error", err)
		}
		if len(dest) > 0 {
			fmt.Fprintf(sess, "\n### Repo Menu ###\n\n")
		}
		for _, dir := range dest {
			wish.Println(sess, fmt.Sprintf("• %s - ", dir.Name()))
			wish.Println(sess, fmt.Sprintf("git clone ssh://%s/%s", net.JoinHostPort(host, port), dir.Name()))
		}
		wish.Printf(sess, "\n\n### Add some repos! ###\n\n")
		wish.Printf(sess, "> cd some_repo\n")
		wish.Printf(sess, "> git remote add wish_test ssh://%s/some_repo\n", net.JoinHostPort(host, port))
		wish.Printf(sess, "> git push wish_test\n\n\n")
		next(sess)
	}
}
