package main

// An example git server. This will list all available repos if you ssh
// directly to the server. To test `ssh -p 23233 localhost` once it's running.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"
)

const (
	port    = 23233
	host    = "localhost"
	repoDir = ".repos"
)

type app struct {
	access gm.AccessLevel
}

func (a app) AuthRepo(repo string, pk ssh.PublicKey) gm.AccessLevel {
	return a.access
}

func (a app) Push(repo string, pk ssh.PublicKey) {
	log.Info("push", "repo", repo)
}

func (a app) Fetch(repo string, pk ssh.PublicKey) {
	log.Info("fetch", "repo", repo)
}

func passHandler(ctx ssh.Context, password string) bool {
	return false
}

func pkHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func main() {
	// A simple GitHooks implementation to allow global read write access.
	a := app{gm.ReadWriteAccess}

	s, err := wish.NewServer(
		ssh.PublicKeyAuth(pkHandler),
		ssh.PasswordAuth(passHandler),
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/git_server_ed25519"),
		wish.WithMiddleware(
			gm.Middleware(repoDir, a),
			gitListMiddleware,
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Error("could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}

// Normally we would use a Bubble Tea program for the TUI but for simplicity,
// we'll just write a list of the pushed repos to the terminal and exit the ssh
// session.
func gitListMiddleware(h ssh.Handler) ssh.Handler {
	return func(s ssh.Session) {
		// Git will have a command included so only run this if there are no
		// commands passed to ssh.
		if len(s.Command()) == 0 {
			des, err := os.ReadDir(repoDir)
			if err != nil && err != fs.ErrNotExist {
				log.Error("invalid repository", "error", err)
			}
			if len(des) > 0 {
				fmt.Fprintf(s, "\n### Repo Menu ###\n\n")
			}
			for _, de := range des {
				fmt.Fprintf(s, "• %s - ", de.Name())
				fmt.Fprintf(s, "git clone ssh://%s:%d/%s\n", host, port, de.Name())
			}
			fmt.Fprintf(s, "\n\n### Add some repos! ###\n\n")
			fmt.Fprintf(s, "> cd some_repo\n")
			fmt.Fprintf(s, "> git remote add wish_test ssh://%s:%d/some_repo\n", host, port)
			fmt.Fprintf(s, "> git push wish_test\n\n\n")
		}
		h(s)
	}
}
