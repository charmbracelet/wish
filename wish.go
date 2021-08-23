package wish

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/charm/keygen"
	"github.com/gliderlabs/ssh"
)

// Middleware is a function that takes an ssh.Handler and returns an
// ssh.Handler. Implementations should call the provided handler argument.
type Middleware func(ssh.Handler) ssh.Handler

// NewServer is returns a default SSH server with the provided Middleware. A
// new SSH key pair of type ed25519 will be created if one does not exist. By
// default this server will accept all incoming connections, password and
// public key.
func NewServer(addr string, keyPath string, mw ...Middleware) (*ssh.Server,
	error) {
	s := &ssh.Server{}
	s.Version = "OpenSSH_7.6p1"
	s.Addr = addr
	s.PasswordHandler = passHandler
	s.PublicKeyHandler = authHandler
	kps := strings.Split(keyPath, string(filepath.Separator))
	kp := strings.Join(kps[:len(kps)-1], string(filepath.Separator))
	n := strings.TrimRight(kps[len(kps)-1], "_ed25519")
	_, err := keygen.NewSSHKeyPair(kp, n, nil, "ed25519")
	if err != nil {
		return nil, err
	}
	k := ssh.HostKeyFile(keyPath)
	err = s.SetOption(k)
	if err != nil {
		return nil, err
	}
	s.Handler = HandlerFromMiddleware(mw...)
	return s, nil
}

// HandlerFromMiddleware composes the provided Middleware and return a
// ssh.Handler. This useful if you manually create an ssh.Server and want to
// set the Server.Handler.
func HandlerFromMiddleware(mw ...Middleware) ssh.Handler {
	h := func(s ssh.Session) {}
	for _, m := range mw {
		h = m(h)
	}
	return h
}

func authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func passHandler(ctx ssh.Context, pass string) bool {
	return true
}
