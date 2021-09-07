package wish

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/charm/keygen"
	"github.com/gliderlabs/ssh"
)

// WithAddress returns an ssh.Option that sets the address to listen on.
func WithAddress(addr string) ssh.Option {
	return func(s *ssh.Server) error {
		s.Addr = addr
		return nil
	}
}

// WithVersion returns an ssh.Option that sets the server version.
func WithVersion(version string) ssh.Option {
	return func(s *ssh.Server) error {
		s.Version = version
		return nil
	}
}

// WithMiddlewares composes the provided Middleware and return a
// ssh.Option. This useful if you manually create an ssh.Server and want to
// set the Server.Handler.
func WithMiddlewares(mw ...Middleware) ssh.Option {
	return func(s *ssh.Server) error {
		h := func(s ssh.Session) {}
		for _, m := range mw {
			h = m(h)
		}
		s.Handler = h
		return nil
	}
}

// WithHostKeyFile returns an ssh.Option that sets the path to the private.
func WithHostKeyPath(path string) ssh.Option {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		kps := strings.Split(path, string(filepath.Separator))
		kp := strings.Join(kps[:len(kps)-1], string(filepath.Separator))
		n := strings.TrimSuffix(kps[len(kps)-1], "_ed25519")
		_, err := keygen.NewSSHKeyPair(kp, n, nil, "ed25519")
		if err != nil {
			return func(*ssh.Server) error {
				return err
			}
		}
		path = filepath.Join(kp, n+"_ed25519")
	}
	return ssh.HostKeyFile(path)
}

// WithHostKeyPEM returns an ssh.Option that sets the host key from a PEM block.
func WithHostKeyPEM(pem []byte) ssh.Option {
	return ssh.HostKeyPEM(pem)
}

// WithPublicKeyAuth returns an ssh.Option that sets the public key auth handler.
func WithPublicKeyAuth(h ssh.PublicKeyHandler) ssh.Option {
	return ssh.PublicKeyAuth(h)
}

// WithPasswordAuth returns an ssh.Option that sets the password auth handler.
func WithPasswordAuth(p ssh.PasswordHandler) ssh.Option {
	return ssh.PasswordAuth(p)
}
