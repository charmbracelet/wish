package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const (
	host = "localhost"
	port = "23234"
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
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

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// Here we create a model with an embedded huh.Form.
	// We then need to handle its usage in View, Update, and Init.
	m := &model{}

	renderer := bubbletea.MakeRenderer(s)
	log.Info("Renderer", "profile", renderer.Output().Profile)

	// we need to setup and set the theme using the session's renderer
	theme := huh.ThemeCatppuccin(huh.WithRenderer(renderer))

	m.txtStyle = renderer.NewStyle().Foreground(lipgloss.Color("10"))
	m.quitStyle = renderer.NewStyle().Foreground(lipgloss.Color("8"))
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Value(&m.fullname).
				Title("Name:").
				Description("What is your full name?"),
		),
	).WithTheme(theme)

	return m, nil
}

type model struct {
	form      *huh.Form
	fullname  string
	txtStyle  lipgloss.Style
	quitStyle lipgloss.Style
}

func (m *model) Init() tea.Cmd {
	return m.form.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.form != nil {
		if m.form.State == huh.StateCompleted || m.form.State == huh.StateAborted {
			m.form = nil
			return m, nil
		}
		f, cmd := m.form.Update(msg)
		m.form = f.(*huh.Form)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string {
	if m.form != nil {
		return m.form.View()
	}
	s := fmt.Sprintf("Your full name is %s", m.fullname)
	return m.txtStyle.Render(s) + "\n\n" + m.quitStyle.Render("Press 'q' to quit\n")
}
