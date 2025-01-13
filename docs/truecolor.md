# Truecolor Over SSH With Wish

If you're having issues with colors displaying properly in your Wish apps, you
may have to force `truecolor` support with
`bm.MiddlewareWithColorProfile(handler, termenv.TrueColor)` when defining the
middleware for your `Wish` server. This will force `truecolor` support. For
example:

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
	)),
	)
// ...
```

[see more][examples-bubbletea]

This _may_ cause issues for users accessing your Wish app through a terminal
emulator that is not `truecolor` compatible (e.g. Apple's Terminal app).

## Lipgloss Renderer aka "no colors when run in a server"

[Lipgloss][], which we use to create styles, will have its "runtime renderer",
which is based on the current system environment.

This is perfectly fine for CLI apps running locally.

On the case of Wish apps, though, the runtime is the server, but we want the
profile to match the one of the client.

To do this, we can a create a custom renderer from the session, and use it to
create styles, for example:

```go
import (
  // ...
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/wish/bubbletea"
)

// ...

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	renderer := bubbletea.MakeRenderer(s)

  // ...

  m := &model{}
  m.initStyles(renderer)
  return m, nil
}

func (m *model) initStyles(r *lipgloss.Renderer) {
	m.txtStyle = r.NewStyle().Foreground(lipgloss.Color("10"))
}
```

This is one strategy you can use, but you can adapt it to whatever makes more
sense for your app.

## Color profiles? What, like it's hard?

We'll dive into the root of the problem for those who want to learn more. In an
SSH session, the client only sends the `TERM` environment variable, which can
only detect `256 color` support. If you run `echo $COLORTERM` in your shell
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
_how much_ of a difference this makes. Well, `256 color` support uses a palette
with 256 colors available. By contrast, `truecolor` supports a whopping 16
**million** different colors.

[Learn more about color standards for terminal emulators][termstandard]

## What is Wish

Wish is an SSH server that allows you to make your apps accessible over SSH. It
uses SSH middleware to handle connections, so you can serve specific actions to
the user, then call the next middleware.

Wish uses the SSH protocol to authenticate users, then allows the developer to
specify how to handle these connections. We've used Wish to serve both TUIs
(textual UIs) _and_ CLIs. If you've hosted your own [Soft Serve][soft] git
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

## Noteworthy environment variables (for debugging)

These environment variables may help you should you encounter unexpected
behavior when working with terminal styling.

`TERM` - provides information about your terminal emulator's capabilities. This
is the only environment variable out of this list that is sent in an SSH
session. The rest are included for debugging purposes only.

`COLORTERM` - provides information about your terminal emulator's color
capabilities. Used primarily to specify if your emulator has `truecolor`
support.

`NO_COLOR` - turns colors on and off. `NO_COLOR=1` for non-colored text
outputs, `NO_COLOR=0` for colored text outputs.

`CLICOLOR` - turns colors on and off. `CLICOLOR=1` for colored text outputs,
`CLICOLOR=0` for non-colored text outputs.

`CLICOLOR_FORCE` - overrides `CLICOLOR`.

`NO_COLOR` vs `CLICOLOR`: if `NO_COLOR` is set or `CLICOLOR=0` then the output should
not be colored. Otherwise, the output can include ansi sequences.

[termstandard]: https://github.com/termstandard/colors
[supported-emulators]: https://github.com/termstandard/colors?tab=readme-ov-file#terminal-emulators
[truecolor-ssh]: https://fixnum.org/2023-03-22-helix-truecolor-ssh-screen/
[colorterm-issue]: https://github.com/termstandard/colors?tab=readme-ov-file#truecolor-detection
[examples-bubbletea]: https://github.com/charmbracelet/wish/blob/main/examples/bubbletea/main.go#L35
[soft]: https://github.com/charmbracelet/soft-serve
[termenv]: https://github.com/muesli/termenv/blob/345783024a348cbb893bf6f08f1d7ab79d2e22ff/termenv_unix.go#L53
[lipgloss]: https://github.com/charmbracelet/lipgloss
