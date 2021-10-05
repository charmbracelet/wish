package bubbletea

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// BubbleTeaHander is the function Bubble Tea apps implement to hook into the
// SSH Middleware. This will create a new tea.Program for every connection and
// start it with the tea.ProgramOptions returned.
type BubbleTeaHandler func(ssh.Session) (tea.Model, []tea.ProgramOption)

// Middleware takes a BubbleTeaHandler and hooks the input and output for the
// ssh.Session into the tea.Program. It also captures window resize events and
// sends them to the tea.Program as tea.WindowSizeMsgs.
func Middleware(bth BubbleTeaHandler) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			errc := make(chan error, 1)
			m, opts := bth(s)
			if m != nil {
				opts = append(opts, tea.WithInput(s), tea.WithOutput(s))
				p := tea.NewProgram(m, opts...)
				_, windowChanges, _ := s.Pty()
				go func() {
					for {
						select {
						case <-s.Context().Done():
							if p != nil {
								p.Send(tea.Quit())
							}
						case w := <-windowChanges:
							if p != nil {
								p.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
							}
						case err := <-errc:
							if err != nil {
								log.Print(err)
							}
						}
					}
				}()
				errc <- p.Start()
			}
			sh(s)
		}
	}
}
