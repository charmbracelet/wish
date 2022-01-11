package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
)

const host = "localhost"
const port = 23234

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			lm.Middleware(),
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

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// teaw.WithAltScreen) on a session by session basis
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, active := s.Pty()
	if !active {
		fmt.Println("no active terminal, skipping")
		return nil, nil
	}
	m := model{
		term:   pty.Term,
		width:  pty.Window.Width,
		height: pty.Window.Height,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	term   string
	width  int
	height int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "Your term is %s\n"
	s += "Your window size is x: %d y: %d\n\n"
	s += "Press 'q' to quit\n"
	return fmt.Sprintf(s, m.term, m.width, m.height)
}
