package main

// An example git server. This will list all available repos if you ssh
// directly to the server. To test `ssh -p 23233 localhost` once it's running.

import (
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/charmbracelet/wish"
	gm "github.com/charmbracelet/wish/git"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
)

const port = 23233
const host = "localhost"
const repoDir = ".repos"

func main() {
	s, err := wish.NewServer(
		fmt.Sprintf("%s:%d", host, port),
		".ssh/git_server_ed25519",
		gm.Middleware(repoDir, "", ""),
		gitListMiddleware,
		lm.Middleware(),
	)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Starting SSH server on %s:%d", host, port)
	err = s.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
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
				log.Println(err)
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
