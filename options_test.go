package wish

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func TestWithIdleTimeout(t *testing.T) {
	s := ssh.Server{}
	requireNoError(t, WithIdleTimeout(time.Second)(&s))
	requireEqual(t, time.Second, s.IdleTimeout)
}

func TestWithMaxTimeout(t *testing.T) {
	s := ssh.Server{}
	requireNoError(t, WithMaxTimeout(time.Second)(&s))
	requireEqual(t, time.Second, s.MaxTimeout)
}

func TestParseAuthorizedKeys(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		keys, err := parseAuthorizedKeys("testdata/authorized_keys")
		requireNoError(t, err)
		requireEqual(t, 6, len(keys))
	})

	t.Run("invalid", func(t *testing.T) {
		keys, err := parseAuthorizedKeys("testdata/invalid_authorized_keys")
		requireEqual(t, `failed to parse "testdata/invalid_authorized_keys": ssh: no key found`, err.Error())
		requireEqual(t, 0, len(keys))
	})

	t.Run("file not found", func(t *testing.T) {
		keys, err := parseAuthorizedKeys("testdata/nope_authorized_keys")
		requireEqual(t, `failed to parse "testdata/nope_authorized_keys": open testdata/nope_authorized_keys: no such file or directory`, err.Error())
		requireEqual(t, 0, len(keys))
	})
}

func TestWithAuthorizedKeys(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		s := ssh.Server{}
		requireNoError(t, WithAuthorizedKeys("testdata/authorized_keys")(&s))

		for key, authorize := range map[string]bool{
			`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMJlb/qf2B2kMNdBxfpCQqI2ctPcsOkdZGVh5zTRhKtH k3@test`: true,
			`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOhsthN+zSFSJF7V2HFSO4+2OJYRghuAA43CIbVyvzF8 k7@test`: false,
		} {
			parts := strings.Fields(key)
			t.Run(parts[len(parts)-1], func(t *testing.T) {
				key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
				requireNoError(t, err)
				requireEqual(t, authorize, s.PublicKeyHandler(nil, key))
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		s := ssh.Server{}
		requireEqual(
			t,
			`failed to parse "testdata/invalid_authorized_keys": ssh: no key found`,
			WithAuthorizedKeys("testdata/invalid_authorized_keys")(&s).Error(),
		)
	})
}

func TestWithTrustedUserCAKeys(t *testing.T) {
	setup := func(tb testing.TB, certPath string) (*ssh.Server, *gossh.ClientConfig) {
		s := &ssh.Server{
			Handler: func(s ssh.Session) {
				fmt.Fprintln(s, "hello")
			},
		}
		requireNoError(t, WithTrustedUserCAKeys("testdata/ca.pub")(s))

		signer, err := gossh.ParsePrivateKey(getBytes(t, "testdata/foo"))
		requireNoError(t, err)

		cert, _, _, _, err := gossh.ParseAuthorizedKey(getBytes(t, certPath))
		requireNoError(t, err)

		certSigner, err := gossh.NewCertSigner(cert.(*gossh.Certificate), signer)
		requireNoError(t, err)
		return s, &gossh.ClientConfig{
			User: "foo",
			Auth: []gossh.AuthMethod{
				gossh.PublicKeys(certSigner),
			},
		}
	}

	t.Run("invalid ca key", func(t *testing.T) {
		s := &ssh.Server{}
		if err := WithTrustedUserCAKeys("testdata/invalid-path")(s); err == nil {
			t.Fatal("expedted an error, got nil")
		}
	})

	t.Run("valid", func(t *testing.T) {
		s, cc := setup(t, "testdata/valid-cert.pub")
		requireNoError(t, testsession.New(t, s, cc).Run(""))
	})

	t.Run("expired", func(t *testing.T) {
		s, cc := setup(t, "testdata/expired-cert.pub")
		_, err := testsession.NewClientSession(t, testsession.Listen(t, s), cc)
		requireAuthError(t, err)
	})

	t.Run("signed by another ca", func(t *testing.T) {
		s, cc := setup(t, "testdata/another-ca-cert.pub")
		_, err := testsession.NewClientSession(t, testsession.Listen(t, s), cc)
		requireAuthError(t, err)
	})

	t.Run("not a cert", func(t *testing.T) {
		s := &ssh.Server{
			Handler: func(s ssh.Session) {
				fmt.Fprintln(s, "hello")
			},
		}
		requireNoError(t, WithTrustedUserCAKeys("testdata/ca.pub")(s))

		signer, err := gossh.ParsePrivateKey(getBytes(t, "testdata/foo"))
		requireNoError(t, err)

		_, err = testsession.NewClientSession(t, testsession.Listen(t, s), &gossh.ClientConfig{
			User: "foo",
			Auth: []gossh.AuthMethod{
				gossh.PublicKeys(signer),
			},
		})
		requireAuthError(t, err)
	})
}

func getBytes(tb testing.TB, path string) []byte {
	tb.Helper()
	bts, err := os.ReadFile(path)
	requireNoError(tb, err)
	return bts
}

func requireEqual(tb testing.TB, a, b interface{}) {
	tb.Helper()
	if a != b {
		tb.Fatalf("expected %v, got %v", a, b)
	}
}

func requireNoError(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("expected no error, got %v", err)
	}
}

func requireAuthError(tb testing.TB, err error) {
	if err == nil {
		tb.Fatal("required an error, got nil")
	}
	requireEqual(tb, "ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain", err.Error())
}
