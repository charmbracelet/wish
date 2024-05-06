package wish

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
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

// WithBanner return an ssh.Option that sets the server banner.
func WithBanner(banner string) ssh.Option {
	return func(s *ssh.Server) error {
		s.Banner = banner
		return nil
	}
}

// WithBannerHandler return an ssh.Option that sets the server banner handler,
// overriding WithBanner.
func WithBannerHandler(h ssh.BannerHandler) ssh.Option {
	return func(s *ssh.Server) error {
		s.BannerHandler = h
		return nil
	}
}

// WithMiddleware composes the provided Middleware and returns an ssh.Option.
// This is useful if you manually create an ssh.Server and want to set the
// Server.Handler.
//
// Notice that middlewares are composed from first to last, which means the last one is executed first.
func WithMiddleware(mw ...Middleware) ssh.Option {
	return func(s *ssh.Server) error {
		h := func(ssh.Session) {}
		for _, m := range mw {
			h = m(h)
		}
		s.Handler = h
		return nil
	}
}

// WithHostKeyFile returns an ssh.Option that sets the path to the private key.
func WithHostKeyPath(path string) ssh.Option {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, err := keygen.New(path, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
		if err != nil {
			return func(*ssh.Server) error {
				return err
			}
		}
	}
	return ssh.HostKeyFile(path)
}

// WithHostKeyPEM returns an ssh.Option that sets the host key from a PEM block.
func WithHostKeyPEM(pem []byte) ssh.Option {
	return ssh.HostKeyPEM(pem)
}

// WithAuthorizedKeys allows the use of an SSH authorized_keys file to allowlist users.
func WithAuthorizedKeys(path string) ssh.Option {
	return func(s *ssh.Server) error {
		if _, err := os.Stat(path); err != nil {
			return err
		}
		return WithPublicKeyAuth(func(_ ssh.Context, key ssh.PublicKey) bool {
			return isAuthorized(path, func(k ssh.PublicKey) bool {
				return ssh.KeysEqual(key, k)
			})
		})(s)
	}
}

// WithTrustedUserCAKeys authorize certificates that are signed with the given
// Certificate Authority public key, and are valid.
// Analogous to the TrustedUserCAKeys OpenSSH option.
func WithTrustedUserCAKeys(path string) ssh.Option {
	return func(s *ssh.Server) error {
		if _, err := os.Stat(path); err != nil {
			return err
		}
		return WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			cert, ok := key.(*gossh.Certificate)
			if !ok {
				// not a certificate...
				return false
			}

			return isAuthorized(path, func(k ssh.PublicKey) bool {
				checker := &gossh.CertChecker{
					IsUserAuthority: func(auth gossh.PublicKey) bool {
						// its a cert signed by one of the CAs
						return bytes.Equal(auth.Marshal(), k.Marshal())
					},
				}

				if !checker.IsUserAuthority(cert.SignatureKey) {
					return false
				}

				if err := checker.CheckCert(ctx.User(), cert); err != nil {
					return false
				}

				return true
			})
		})(s)
	}
}

func isAuthorized(path string, checker func(k ssh.PublicKey) bool) bool {
	f, err := os.Open(path)
	if err != nil {
		log.Warn("failed to parse", "path", path, "error", err)
		return false
	}
	defer f.Close() // nolint: errcheck

	rd := bufio.NewReader(f)
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Warn("failed to parse", "path", path, "error", err)
			return false
		}
		if strings.TrimSpace(string(line)) == "" {
			continue
		}
		if bytes.HasPrefix(line, []byte{'#'}) {
			continue
		}
		upk, _, _, _, err := ssh.ParseAuthorizedKey(line)
		if err != nil {
			log.Warn("failed to parse", "path", path, "error", err)
			return false
		}
		if checker(upk) {
			return true
		}
	}
	return false
}

// WithPublicKeyAuth returns an ssh.Option that sets the public key auth handler.
func WithPublicKeyAuth(h ssh.PublicKeyHandler) ssh.Option {
	return ssh.PublicKeyAuth(h)
}

// WithPasswordAuth returns an ssh.Option that sets the password auth handler.
func WithPasswordAuth(p ssh.PasswordHandler) ssh.Option {
	return ssh.PasswordAuth(p)
}

// WithKeyboardInteractiveAuth returns an ssh.Option that sets the keyboard interactive auth handler.
func WithKeyboardInteractiveAuth(h ssh.KeyboardInteractiveHandler) ssh.Option {
	return ssh.KeyboardInteractiveAuth(h)
}

// WithIdleTimeout returns an ssh.Option that sets the connection's idle timeout.
func WithIdleTimeout(d time.Duration) ssh.Option {
	return func(s *ssh.Server) error {
		s.IdleTimeout = d
		return nil
	}
}

// WithMaxTimeout returns an ssh.Option that sets the connection's absolute timeout.
func WithMaxTimeout(d time.Duration) ssh.Option {
	return func(s *ssh.Server) error {
		s.MaxTimeout = d
		return nil
	}
}

// WithSubsystem returns an ssh.Option that sets the subsystem
// handler for a given protocol.
func WithSubsystem(key string, h ssh.SubsystemHandler) ssh.Option {
	return func(s *ssh.Server) error {
		if s.SubsystemHandlers == nil {
			s.SubsystemHandlers = map[string]ssh.SubsystemHandler{}
		}
		s.SubsystemHandlers[key] = h
		return nil
	}
}
