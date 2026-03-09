# Upgrade Guide: Wish v2

This guide walks you through upgrading from Wish v1 to v2. Most changes are straightforward—mainly import paths and adopting Bubble Tea v2's declarative view pattern.

## Quick Start

The fastest way to upgrade:

1. Update import paths to `charm.land/wish/v2`
2. Update Bubble Tea to v2 with declarative views
3. Remove color profile detection code
4. Update program options

That's it for most apps!

## Import Paths

All Charm libraries now use the `charm.land` vanity domain:

```go
// Before
import (
    "github.com/charmbracelet/wish"
    "github.com/charmbracelet/wish/bubbletea"
    "github.com/charmbracelet/wish/logging"
    "github.com/charmbracelet/wish/activeterm"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/log"
)

// After
import (
    "charm.land/wish/v2"
    "charm.land/wish/v2/bubbletea"
    "charm.land/wish/v2/logging"
    "charm.land/wish/v2/activeterm"
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
    "charm.land/log/v2"
)
```

**All middleware packages** follow the same pattern:

- `charm.land/wish/v2/accesscontrol`
- `charm.land/wish/v2/comment`
- `charm.land/wish/v2/elapsed`
- `charm.land/wish/v2/git`
- `charm.land/wish/v2/ratelimiter`
- `charm.land/wish/v2/recover`
- `charm.land/wish/v2/scp`

## Bubble Tea Handler Changes

### Remove Color Profile Detection

The `MakeRenderer` function is gone. Bubble Tea v2 handles color profile detection automatically.

```go
// Before
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    renderer := bubbletea.MakeRenderer(s)
    txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("10"))

    bg := "light"
    if renderer.HasDarkBackground() {
        bg = "dark"
    }

    m := model{
        txtStyle: txtStyle,
        bg:       bg,
    }
    return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// After
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    m := model{
        txtStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
    }
    return m, []tea.ProgramOption{}
}
```

### Use Declarative Views

In Bubble Tea v2, `View()` returns a `tea.View` struct instead of a `string`.

```go
// Before
func (m model) View() string {
    return "Hello, world!"
}

// After
func (m model) View() tea.View {
    v := tea.NewView("Hello, world!")
    v.AltScreen = true  // Move tea.WithAltScreen() here
    return v
}
```

### Get Background Color from Messages

Instead of querying at initialization, listen for `BackgroundColorMsg`:

```go
// Before
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    renderer := bubbletea.MakeRenderer(s)
    bg := "light"
    if renderer.HasDarkBackground() {
        bg = "dark"
    }
    m := model{bg: bg}
    return m, nil
}

// After
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    m := model{bg: "light"} // default
    return m, nil
}

func (m model) Init() tea.Cmd {
    return tea.RequestBackgroundColor
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.BackgroundColorMsg:
        if msg.IsDark() {
            m.bg = "dark"
        } else {
            m.bg = "light"
        }
    }
    return m, nil
}
```

### Get Color Profile from Messages

Similarly, color profile is now received as a message:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.ColorProfileMsg:
        m.profile = msg.String() // "TrueColor", "ANSI256", "ANSI", etc.
    }
    return m, nil
}
```

## Middleware API Changes

### Removed Functions

These functions have been removed from `bubbletea` middleware:

- **`MakeRenderer()`** — Use Lip Gloss styles directly; color profiles are automatic
- **`MiddlewareWithColorProfile()`** — No longer needed
- **`QueryTerminalFilter`** — Terminal queries now handled by Bubble Tea v2

### Updated Function Signatures

All middleware functions now return `charm.land/wish/v2.Middleware`:

```go
// Before
func Middleware(handler Handler) wish.Middleware

// After
func Middleware(handler Handler) wish.Middleware // same signature, new import
```

The `MiddlewareWithProgramHandler` signature was simplified:

```go
// Before
func MiddlewareWithProgramHandler(
    handler ProgramHandler,
    profile termenv.Profile,
) wish.Middleware

// After
func MiddlewareWithProgramHandler(handler ProgramHandler) wish.Middleware
```

## Program Options

Move most program options from `ProgramOption` to the `View` struct:

```go
// Before
return m, []tea.ProgramOption{
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(),
}

// After
func (m model) View() tea.View {
    v := tea.NewView(m.content)
    v.AltScreen = true
    v.MouseMode = tea.MouseModeCellMotion
    return v
}
```

You can still use options for things like input/output configuration:

```go
return m, bubbletea.MakeOptions(s) // Still needed for SSH I/O setup
```

## Bubble Tea v2 Features

Your SSH apps now get all the Bubble Tea v2 improvements:

### Key Messages

Key messages are split into `KeyPressMsg` and `KeyReleaseMsg`:

```go
// Before
case tea.KeyMsg:
    switch msg.String() {
    case " ":
        // space
    }

// After
case tea.KeyPressMsg:
    switch msg.String() {
    case "space":  // Note: "space" not " "
        // space
    case "shift+enter":
        // Now possible!
    }
```

### Mouse Messages

Mouse messages are now split by type:

```go
// Before
case tea.MouseMsg:
    switch msg.Type {
    case tea.MouseLeft:
        // click
    }

// After
case tea.MouseClickMsg:
    if msg.Button == tea.MouseLeft {
        // click
    }
case tea.MouseWheelMsg:
    // scroll
case tea.MouseMotionMsg:
    // movement
```

### Paste Events

Paste events are now their own message type:

```go
// Before
case tea.KeyMsg:
    if msg.Paste {
        // paste
    }

// After
case tea.PasteMsg:
    m.text += msg.Content
```

### Clipboard Support

You can now read and write the clipboard (OSC52 works over SSH!):

```go
case tea.KeyPressMsg:
    switch msg.String() {
    case "ctrl+c":
        return m, tea.SetClipboard("Copied text")
    case "ctrl+v":
        return m, tea.ReadClipboard()
    }
case tea.ClipboardMsg:
    m.clipboard = msg.String()
```

## Logging Middleware

The structured logging middleware signature changed:

```go
// Before
import "github.com/charmbracelet/log"

logging.StructuredMiddlewareWithLogger(logger, log.InfoLevel)

// After
import "charm.land/log/v2"

logging.StructuredMiddlewareWithLogger(logger, log.InfoLevel)
```

The `log.Logger` type is now from `charm.land/log/v2`.

## Complete Example

Here's a complete before/after for a typical Wish application:

### Before (v1)

```go
package main

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/ssh"
    "github.com/charmbracelet/wish"
    "github.com/charmbracelet/wish/bubbletea"
    "github.com/charmbracelet/wish/logging"
)

func main() {
    s, _ := wish.NewServer(
        wish.WithAddress(":2222"),
        wish.WithMiddleware(
            bubbletea.Middleware(teaHandler),
            logging.Middleware(),
        ),
    )
    s.ListenAndServe()
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    renderer := bubbletea.MakeRenderer(s)
    style := renderer.NewStyle().Foreground(lipgloss.Color("10"))

    m := model{style: style}
    return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type model struct {
    style lipgloss.Style
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "q" {
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m model) View() string {
    return m.style.Render("Hello, SSH!\n\nPress 'q' to quit")
}
```

### After (v2)

```go
package main

import (
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
    "charm.land/wish/v2"
    "charm.land/wish/v2/bubbletea"
    "charm.land/wish/v2/logging"
    "github.com/charmbracelet/ssh"
)

func main() {
    s, _ := wish.NewServer(
        wish.WithAddress(":2222"),
        wish.WithMiddleware(
            bubbletea.Middleware(teaHandler),
            logging.Middleware(),
        ),
    )
    s.ListenAndServe()
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    style := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
    m := model{style: style}
    return m, nil
}

type model struct {
    style lipgloss.Style
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        if msg.String() == "q" {
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m model) View() tea.View {
    v := tea.NewView(m.style.Render("Hello, SSH!\n\nPress 'q' to quit"))
    v.AltScreen = true
    return v
}
```

### Key Changes

1. Import paths use `charm.land/*/v2`
2. `MakeRenderer` removed—use Lip Gloss directly
3. `tea.WithAltScreen()` moved to `v.AltScreen = true`
4. `View()` returns `tea.View` instead of `string`
5. `tea.KeyMsg` changed to `tea.KeyPressMsg`

## Migration Checklist

- [ ] Update `go.mod` to require `charm.land/wish/v2`
- [ ] Update all import paths to `charm.land/*`
- [ ] Remove `bubbletea.MakeRenderer()` calls
- [ ] Remove `MiddlewareWithColorProfile()` usage
- [ ] Change `View() string` to `View() tea.View`
- [ ] Move program options to view fields (`v.AltScreen`, etc.)
- [ ] Update `tea.KeyMsg` to `tea.KeyPressMsg`
- [ ] Update `tea.MouseMsg` to specific mouse message types
- [ ] Handle background color via `tea.BackgroundColorMsg`
- [ ] Handle color profile via `tea.ColorProfileMsg`
- [ ] Test your SSH app with various terminals

## Accessing SSH Client Environment Variables

In Bubble Tea v2, you can access the SSH client's environment variables in two ways:

**Important:** Wish automatically passes the client's environment to Bubble Tea when you use `bubbletea.MakeOptions()`. This means `tea.EnvMsg` will contain the _client's_ environment, not the server's!

### Method 1: Use tea.EnvMsg (Recommended)

Bubble Tea v2 automatically sends an `EnvMsg` with the client's environment:

```go
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    return model{}, bubbletea.MakeOptions(s) // Passes client environment
}

type model struct {
    envMsg tea.EnvMsg
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.EnvMsg:
        m.envMsg = msg

        // Access specific CLIENT variables
        term := msg.Getenv("TERM")
        lang := msg.Getenv("LANG")
        user := msg.Getenv("USER")

        fmt.Printf("Client TERM: %s\n", term)
    }
    return m, nil
}
```

### Method 2: Pass from Handler

If you need environment variables before `Init()` runs, extract them in the handler:

```go
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
    // Get client environment variables from the SSH session
    env := make(map[string]string)
    for _, e := range s.Environ() {
        parts := strings.SplitN(e, "=", 2)
        if len(parts) == 2 {
            env[parts[0]] = parts[1]
        }
    }

    m := model{
        env: env,  // Pass to model
    }
    return m, bubbletea.MakeOptions(s)
}

type model struct {
    env map[string]string
}

func (m model) View() tea.View {
    // Access client's environment
    term := m.env["TERM"]    // Client's TERM
    lang := m.env["LANG"]    // Client's LANG
    user := m.env["USER"]    // Client's USER

    return tea.NewView(fmt.Sprintf("Your TERM: %s", term))
}
```

> [!WARNING]
> **Never use `os.Getenv()` in SSH apps**—it returns the _server's_ environment! Always use `tea.EnvMsg` (recommended) or `ssh.Session.Environ()`.

**Key Point:** These are the _client's_ environment variables, not the server's. This is especially important for SSH apps where `os.Getenv()` would give you incorrect server values.

## Need Help?

If you run into issues:

- Check the [Bubble Tea v2 Upgrade Guide][bbtea-upgrade] for Bubble Tea-specific changes
- See [examples/](examples/) for complete working examples
- Ask on [Discord](https://charm.land/chat) or [Matrix](https://charm.land/matrix)
- Open an issue on [GitHub](https://github.com/charmbracelet/wish/issues)

[bbtea-upgrade]: https://github.com/charmbracelet/bubbletea/blob/v2/UPGRADE_GUIDE_V2.md

---

Part of [Charm](https://charm.land).

<a href="https://charm.land/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source • نحنُ نحب المصادر المفتوحة
