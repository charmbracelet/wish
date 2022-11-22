package wish

import (
	"context"
	"sync"

	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// Option is a functional option handler for Server.
type Option = ssh.Option

// Handler is a callback for handling established SSH sessions.
type Handler = ssh.Handler

// PublicKey is an abstraction of different types of public keys.
type PublicKey = ssh.PublicKey

// The Permissions type holds fine-grained permissions that are specific to a
// user or a specific authentication method for a user. Permissions, except for
// "source-address", must be enforced in the server application layer, after
// successful authentication.
type Permissions = ssh.Permissions

// A Signer can create signatures that verify against a public key.
type Signer = ssh.Signer

// PublicKeyHandler is a callback for performing public key authentication.
type PublicKeyHandler func(ctx Context, key PublicKey) bool

// PasswordHandler is a callback for performing password authentication.
type PasswordHandler func(ctx Context, password string) bool

// KeyboardInteractiveHandler is a callback for performing keyboard-interactive authentication.
type KeyboardInteractiveHandler func(ctx Context, challenger gossh.KeyboardInteractiveChallenge) bool

// Context is a package specific context interface. It exposes connection
// metadata and allows new values to be easily written to it. It's used in
// authentication handlers and callbacks, and its underlying context.Context is
// exposed on Session in the session Handler. A connection-scoped lock is also
// embedded in the context to make it easier to limit operations per-connection.
type Context interface {
	context.Context
	sync.Locker
	ssh.Context
}

// Server defines parameters for running an SSH server. The zero value for
// Server is a valid configuration. When both PasswordHandler and
// PublicKeyHandler are nil, no client authentication is performed.
type Server = ssh.Server

// Session provides access to information about an SSH session and methods
// to read and write to the SSH channel with an embedded Channel interface from
// crypto/ssh.
//
// When Command() returns an empty slice, the user requested a shell. Otherwise
// the user is performing an exec with those command arguments.
type Session = ssh.Session

var (
	// HostKeyFile returns a functional option that adds HostSigners to the server
	// from a PEM file at filepath.
	HostKeyFile = ssh.HostKeyFile

	// HostKeyPEM returns a functional option that adds HostSigners to the server
	// from a PEM file as bytes.
	HostKeyPEM = ssh.HostKeyPEM

	// KeysEqual is constant time compare of the keys to avoid timing attacks.
	KeysEqual = ssh.KeysEqual
)
