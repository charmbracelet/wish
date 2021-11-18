# Wish

Make SSH apps, just like that! 💫

SSH is an excellent platform to build remotely accessible applications on. It
offers secure communication without the hassle of HTTPS certificates, it has
user identification with SSH keys and it's accessible from anywhere with a
terminal. Powerful protocols like Git work over SSH and you can even render
TUIs directly over an SSH connection.

Wish is an SSH server with sensible defaults and a collection of middleware that
makes building SSH apps easy. Wish is built on [gliderlabs/ssh][gliderlabs/ssh]
and should be easy to integrate into any existing projects.

## Middleware

### Bubble Tea

The [`bubbletea`](bubbletea) middleware makes it easy to serve any
[Bubble Tea][bubbletea] application over SSH. Each SSH session will get their own
`tea.Program` with the SSH pty input and output connected. Client window
dimension and resize messages are also natively handled by the `tea.Program`.

You can see a demo of the Wish middleware in action at: `ssh git.charm.sh`

### Git

The [`git`](git) middleware adds `git` server functionality to any ssh server.
It supports repo creation on initial push and custom public key based auth.

This middleware requires that `git` is installed on the server.

### Logging

The [`logging`](logging)  middleware provides basic connection logging. Connects
are logged with the remote address, invoked command, TERM setting, window
dimensions and if the auth was public key based. Disconnect will log the remote
address and connection duration.

### Access Control

Not all applications will support general SSH connections. To restrict access
to supported methods, you can use the [`activeterm`](activeterm) middleware to
only allow connections with active terminals connected and the
[`accesscontrol`](accesscontrol) middleware that lets you specify allowed
commands.

## Default Server

Wish includes the ability to easily create an always authenticating default SSH
server with automatic server key generation.

## Examples

There are examples for a standalone [Bubble Tea application](examples/bubbletea)
and [Git server](examples/git) in the [examples](examples) folder. To see a
more real-world application that combines Bubble Tea and Git, you can take a
look at [Soft Serve](https://github.com/charmbracelet/soft-serve) which uses
Git as a CMS and provides a TUI over SSH for interacting with pushed repos.

[bubbletea]: https://github.com/charmbracelet/bubbletea
[gliderlabs/ssh]: https://github.com/gliderlabs/ssh
