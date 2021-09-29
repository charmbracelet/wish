package wish

import (
	"github.com/charmbracelet/keygen"
	"github.com/gliderlabs/ssh"
)

// Middleware is a function that takes an ssh.Handler and returns an
// ssh.Handler. Implementations should call the provided handler argument.
type Middleware func(ssh.Handler) ssh.Handler

// NewServer is returns a default SSH server with the provided Middleware. A
// new SSH key pair of type ed25519 will be created if one does not exist. By
// default this server will accept all incoming connections, password and
// public key.
func NewServer(ops ...ssh.Option) (*ssh.Server, error) {
	s := &ssh.Server{}
	// Some sensible defaults
	s.Version = "OpenSSH_7.6p1"
	s.PasswordHandler = passHandler
	s.PublicKeyHandler = authHandler
	for _, op := range ops {
		if err := s.SetOption(op); err != nil {
			return nil, err
		}
	}
	if len(s.HostSigners) == 0 {
		k, err := keygen.New("", "", nil, keygen.Ed25519)
		if err != nil {
			return nil, err
		}
		err = s.SetOption(WithHostKeyPEM(k.PrivateKeyPEM))
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func passHandler(ctx ssh.Context, pass string) bool {
	return true
}
