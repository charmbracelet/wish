package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

type BubbleTeaHandler func(ssh.Session) (tea.Model, []tea.ProgramOption)

func Middleware(bth BubbleTeaHandler) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			m, opts := bth(s)
			if m != nil {
				opts = append(opts, tea.WithInput(s), tea.WithOutput(s))
				p := tea.NewProgram(m, opts...)
				_, windowChanges, _ := s.Pty()
				go func() {
					for {
						w := <-windowChanges
						if p != nil {
							p.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
						}
					}
				}()
				_ = p.Start()
			}
			sh(s)
		}
	}
}
