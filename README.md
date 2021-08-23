# Wish

Serve Bubble Tea apps and Git over SSH.

## What is it?

Wish is a collection of middleware for [gliderlabs/ssh](https://github.com/gliderlabs/ssh).
The Glider Labs SSH library makes it easy to build custom SSH servers, and Wish
provides useful functionality as ready-to-go middleware.

## Middleware

### Bubble Tea

The Bubble Tea middleware makes it easy to serve any Bubble Tea application
over SSH. Each SSH session will get their own tea.Program with the SSH pty
input and output connected. Window dimension and resize messages are also
captured and sent to the tea.Program as tea.WindoSizeMsgs.

You can see a demo of the Wish middleware in action at: `ssh beta.charm.sh`

### Git

The Git middleware adds `git` server functionality to any ssh.Server. It
supports repo creation on initial push and public key based authorization. The
git server currently makes all repos publicly readable.

This middleware requires that `git` is installed on the server.

### Logging

The logging middleware provides basic connection logging. Connects are logged
with the remote address, invoked command, TERM setting, window dimensions and
if the auth was public key based. Disconnect will log the remote address and
connection duration.

## Default Server

Wish includes the ability to easily create an always authenticating default SSH
server with automatic server key generation.

## Examples

There are examples for a standalone [Bubble Tea application](examples/term-info)
and [Git server](examples/git) in the [examples](examples) folder. To see a
more real-world application that combines Bubble Tea and Git, you can take a
look at [Soft Serve](https://github.com/charmbracelet/soft-serve) which uses
Git as a CMS and provides a TUI over SSH for interacting with pushed repos.
