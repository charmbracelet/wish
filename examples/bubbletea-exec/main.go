package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/v2"
	"github.com/charmbracelet/wish/v2/activeterm"
	"github.com/charmbracelet/wish/v2/bubbletea"
	"github.com/charmbracelet/wish/v2/logging"
	"github.com/charmbracelet/x/editor"
)

const (
	host = "localhost"
	port = "23234"
)

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),

		// Allocate a pty.
		// This creates a pseudoconsole on windows, compatibility is limited in
		// that case, see the open issues for more details.
		ssh.AllocatePty(),
		wish.WithMiddleware(
			// run our Bubble Tea handler
			bubbletea.Middleware(teaHandler),

			// ensure the user has requested a tty
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
	// Set up the model with the current session and styles.
	// We'll use the session to call wish.Command, which makes it compatible
	// with tea.Command.
	m := model{
		sess:     s,
		style:    lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		errStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
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

type cmdFinishedMsg struct{ err error }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			// Open file.txt in the default editor.
			edit, err := editor.Cmd("wish", "file.txt")
			if err != nil {
				m.err = err
				return m, nil
			}
			// Creates a wish.Cmd from the exec.Cmd
			wishCmd := wish.Command(m.sess, edit.Path, edit.Args...)
			// Runs the cmd through Bubble Tea.
			// Bubble Tea should handle the IO to the program, and get it back
			// once the program quits.
			cmd := tea.Exec(wishCmd, func(err error) tea.Msg {
				if err != nil {
					log.Error("editor finished", "error", err)
				}
				return cmdFinishedMsg{err: err}
			})
			return m, cmd
		case "s":
			// We can also execute a shell and give it over to the user.
			// Note that this session won't have control, so it can't run tasks
			// in background, suspend, etc.
			c := wish.Command(m.sess, "htop")
			if runtime.GOOS == "windows" {
				c = wish.Command(m.sess, "powershell")
			}
			cmd := tea.Exec(c, func(err error) tea.Msg {
				if err != nil {
					log.Error("shell finished", "error", err)
				}
				return cmdFinishedMsg{err: err}
			})
			return m, cmd
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case cmdFinishedMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return m.errStyle.Render(m.err.Error() + "\n")
	}
	return m.style.Render("Press 'e' to edit, 's' to hop into a shell, or 'q' to quit...\n")
}
