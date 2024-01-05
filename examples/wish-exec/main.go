package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
)

const (
	host = "localhost"
	port = 23235
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		ssh.AllocatePty(),
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
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
			done <- nil
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

func teaHandler(s ssh.Session, renderer *lipgloss.Renderer) (tea.Model, []tea.ProgramOption) {
	m := model{
		sess:     s,
		style:    renderer.NewStyle().Foreground(lipgloss.Color("8")),
		errStyle: renderer.NewStyle().Foreground(lipgloss.Color("3")),
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type model struct {
	err      error
	sess     ssh.Session
	style    lipgloss.Style
	errStyle lipgloss.Style
}

func (m model) Init() tea.Cmd {
	return nil
}

type VimFinishedMsg struct{ err error }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			pty, _, _ := m.sess.Pty()
			c := pty.Command("vim", "file.txt")
			cmd := tea.Exec(bm.Command(c), func(err error) tea.Msg {
				if err != nil {
					log.Error("vim finished", "error", err)
				}
				return VimFinishedMsg{err: err}
			})
			return m, cmd
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case VimFinishedMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return m.errStyle.Render(m.err.Error() + "\n")
	}
	return m.style.Render("Press 'e' to edit or 'q' to quit...\n")
}