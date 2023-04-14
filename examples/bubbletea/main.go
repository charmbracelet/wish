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
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bm.ColoredMiddleware(teaHandler),
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// tea.WithAltScreen) on a session by session basis.
func teaHandler(s ssh.Session, r *lipgloss.Renderer) (tea.Model, []tea.ProgramOption) {
	pty, _, active := s.Pty()
	if !active {
		wish.Fatalln(s, "no active terminal, skipping")
		return nil, nil
	}

	m := model{
		term:     pty.Term,
		width:    pty.Window.Width,
		height:   pty.Window.Height,
		renderer: r,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	term     string
	width    int
	height   int
	renderer *lipgloss.Renderer
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
	str := fmt.Sprintf(
		"Your term is %s\nYour window size is x: %d y: %d\n\nPress 'q' to quit\n",
		m.term,
		m.width,
		m.height,
	)
	styles := makeStyles(m.renderer)
	str += fmt.Sprintf(
		"\n%s %s %s %s %s",
		styles.bold,
		styles.faint,
		styles.italic,
		styles.underline,
		styles.strikethrough,
	)

	str += fmt.Sprintf(
		"\n%s %s %s %s %s %s %s",
		styles.red,
		styles.green,
		styles.yellow,
		styles.blue,
		styles.magenta,
		styles.cyan,
		styles.gray,
	)

	str += fmt.Sprintf(
		"\n%s %s %s %s %s %s %s\n\n",
		styles.red,
		styles.green,
		styles.yellow,
		styles.blue,
		styles.magenta,
		styles.cyan,
		styles.gray,
	)

	str += fmt.Sprintf(
		"%s %t %s\n\n",
		styles.bold.Copy().UnsetString().Render("Has dark background?"),
		m.renderer.HasDarkBackground(),
		m.renderer.Output().BackgroundColor(),
	)

	return m.renderer.Place(
		m.width,
		lipgloss.Height(str),
		lipgloss.Center,
		lipgloss.Center,
		str,
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceForeground(lipgloss.AdaptiveColor{
			Light: "250",
			Dark:  "236",
		}),
	)
}

// Create new styles against a given renderer.
func makeStyles(r *lipgloss.Renderer) styles {
	return styles{
		bold:          r.NewStyle().SetString("bold").Bold(true),
		faint:         r.NewStyle().SetString("faint").Faint(true),
		italic:        r.NewStyle().SetString("italic").Italic(true),
		underline:     r.NewStyle().SetString("underline").Underline(true),
		strikethrough: r.NewStyle().SetString("strikethrough").Strikethrough(true),
		red:           r.NewStyle().SetString("red").Foreground(lipgloss.Color("#E88388")),
		green:         r.NewStyle().SetString("green").Foreground(lipgloss.Color("#A8CC8C")),
		yellow:        r.NewStyle().SetString("yellow").Foreground(lipgloss.Color("#DBAB79")),
		blue:          r.NewStyle().SetString("blue").Foreground(lipgloss.Color("#71BEF2")),
		magenta:       r.NewStyle().SetString("magenta").Foreground(lipgloss.Color("#D290E4")),
		cyan:          r.NewStyle().SetString("cyan").Foreground(lipgloss.Color("#66C2CD")),
		gray:          r.NewStyle().SetString("gray").Foreground(lipgloss.Color("#B9BFCA")),
	}
}

type styles struct {
	bold          lipgloss.Style
	faint         lipgloss.Style
	italic        lipgloss.Style
	underline     lipgloss.Style
	strikethrough lipgloss.Style
	red           lipgloss.Style
	green         lipgloss.Style
	yellow        lipgloss.Style
	blue          lipgloss.Style
	magenta       lipgloss.Style
	cyan          lipgloss.Style
	gray          lipgloss.Style
}
