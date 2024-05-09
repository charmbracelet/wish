
# Truecolor Over SSH With Wish

## Noteworthy environment variables (for debugging)

`TERM` - provides information about your terminal emulator's capabilities.

`COLORTERM` - provides information about your terminal emulator's color
capabilities. Used primarily to specify if your emulator has `truecolor`
support.

`NO_COLOR` - turns colors on and off. `NO_COLOR=1` for non-colored text
outputs, `NO_COLOR=0` for colored text outputs.

`CLICOLOR` - turns colors on and off. `CLICOLOR=1` for colored text outputs,
`CLICOLOR=0` for non-colored text outputs.

`CLICOLOR_FORCE` - overrides `CLICOLOR`.

NO_COLOR vs CLICOLOR: if NO_COLOR is set or CLICOLOR=0 then the output should
not be colored. Otherwise, the output can include ansi sequences.

## Color profiles? What, like it's hard?

In an SSH session, the client only sends the `TERM` environment variable, which
can only detect `256 color` support. If you run `echo $COLORTERM` in your shell
you'll likely see `truecolor` as the result, which is what you want for ultra
colorful terminal outputs. If you don't see that as a result, it might be time
to try a new [terminal emulator][supported-emulators]. 

Unfortunately, there is no standard way for terminals to detect `truecolor`
support in an SSH session, hence why it defaults to `256 color`. Most terminals
express their `truecolor` support with the `COLORTERM` environment variable,
but this doesn't get sent when connecting over SSH. One workaround is to make
SSH send this environment variable using `SendEnv COLORTERM` in the [`ssh
config`][truecolor-ssh]. By default, the OpenSSH client (what you're using when
you run `ssh`) will only send the `TERM` to the remote, so other variables must
be configured. In the future, we hope to solve this problem by querying the
terminal for support if `COLORTERM` is not detected.

You can learn more about [checking for `COLORTERM`][colorterm-issue].

Because of this, the color options are limited and your experience running the
app locally will differ to how it presents over SSH. You're probably wondering
*how much* of a difference this makes. Well, `256 color` support uses a palette
with 256 colors available. By contrast, `truecolor` supports a whopping 16
**million** different colors.

[Learn more about color standards for terminal emulators][termstandard]

## Solving this in Wish

### What is Wish

Wish is an SSH server that allows you to make your apps accessible over SSH. It
uses SSH middleware to handle connections, so you can serve specific actions to
the user, then call the next middleware.

Wish uses the SSH protocol to authenticate users, then allows the developer to
specify how to handle these connections. We've used wish to serve both TUIs
(textual UIs) *and* CLIs. If you've hosted your own [Soft Serve][soft] git
server, then you'll have seen this first hand. Soft Serve uses a middleware to
serve the TUI and another middleware for its CLI, allowing users to interact
with the server through either interface. In this case, the CLI is useful for
any server administration tasks, while the TUI provides a great interface to
view your repositories. Note that both of these options are accessible through
the same port (pretty neat).

Similar to a website, this process runs on the server, freeing up your computer's
resources for other things. What's great about this is it also gives you a
consistent state no matter where you connect from (as long as you've got your
authorized SSH keys with you).

### Workarounds

Alright, back to the `truecolor` issue. To force `truecolor` support in Wish to
give you all those **precious** color choices, you'll want to use the
`bm.MiddlewareWithColorProfile(handler, termenv.TrueColor)` as a middleware for
your `wish` server. This will force `truecolor` support. For example:

TODO: ayman confirm this example is correct
```go
import (
// ...
    tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

// ...

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
            bubbletea.MiddlewareWithColorProfile(teaHandler, termenv.TrueColor) // Force truecolor.
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			logging.Middleware(),
		),
	)		),
	)
// ...
```

Alternatively, you can use a custom renderer for each session. 

```go
// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// tea.WithAltScreen) on a session by session basis.
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// This should never fail, as we are using the activeterm middleware.
	pty, _, _ := s.Pty()

	// When running a Bubble Tea app over SSH, you shouldn't use the default
	// lipgloss.NewStyle function.
	// That function will use the color profile from the os.Stdin, which is the
	// server, not the client.
	// We provide a MakeRenderer function in the bubbletea middleware package,
	// so you can easily get the correct renderer for the current session, and
	// use it to create the styles.
	// The recommended way to use these styles is to then pass them down to
	// your Bubble Tea model.
	renderer := bubbletea.MakeRenderer(s)
	txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("10"))
	quitStyle := renderer.NewStyle().Foreground(lipgloss.Color("8"))

	m := model{
		term:      pty.Term,
		width:     pty.Window.Width,
		height:    pty.Window.Height,
		txtStyle:  txtStyle,
		quitStyle: quitStyle,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}
```

[see more][examples-bubbletea]

It's up to you to decide which solution works better for you. In the case of
forcing `truecolor` this may cause issues for users accessing your Wish app
through a terminal emulator that is not `truecolor` compatible (e.g. Apple's
Terminal app).

TODO: are there any performance differences between these two options?

### Roadmap for improvement

**Is there a best practice?**

> Not right now. Both are viable options. There's still work to be done to
> support it properly with Bubble Tea. We're working on a solution to make it
> easier to pass a custom renderer to `tea` Programs, but the launch date is
> still TBD.

[termstandard]: https://github.com/termstandard/colors
[supported-emulators]: https://github.com/termstandard/colors?tab=readme-ov-file#terminal-emulators
[truecolor-ssh]: https://fixnum.org/2023-03-22-helix-truecolor-ssh-screen/
[colorterm-issue]: https://github.com/termstandard/colors?tab=readme-ov-file#truecolor-detection
[examples-bubbletea]: https://github.com/charmbracelet/wish/blob/main/examples/bubbletea/main.go#L35
[soft]: https://github.com/charmbracelet/soft-serve
