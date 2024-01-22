package wish

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
	gossh "golang.org/x/crypto/ssh"
)

func TestWithSubsystem(t *testing.T) {
	srv := &ssh.Server{
		Handler: func(s ssh.Session) {},
	}
	requireNoError(t, WithSubsystem("foo", func(s ssh.Session) {})(srv))
	if srv.SubsystemHandlers == nil {
		t.Fatalf("should not have been nil")
	}
	if _, ok := srv.SubsystemHandlers["foo"]; !ok {
		t.Fatalf("should have set the foo subsystem handler")
	}
}

func TestWithBanner(t *testing.T) {
	const banner = "a banner"
	var got string

	srv := &ssh.Server{
		Handler: func(s ssh.Session) {},
	}
	requireNoError(t, WithBanner(banner)(srv))

	requireNoError(t, testsession.New(t, srv, &gossh.ClientConfig{
		BannerCallback: func(message string) error {
			got = message
			return nil
		},
	}).Run(""))
	requireEqual(t, banner, got)
}

func TestWithBannerHandler(t *testing.T) {
	var got string

	srv := &ssh.Server{
		Handler: func(s ssh.Session) {},
	}
	requireNoError(t, WithBannerHandler(func(ctx ssh.Context) string {
		return fmt.Sprintf("banner for %s", ctx.User())
	})(srv))

	requireNoError(t, testsession.New(t, srv, &gossh.ClientConfig{
		User: "fulano",
		BannerCallback: func(message string) error {
			got = message
			return nil
		},
	}).Run(""))
	requireEqual(t, "banner for fulano", got)
}

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

func TestIsAuthorized(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		requireEqual(t, true, isAuthorized("testdata/authorized_keys", func(k ssh.PublicKey) bool { return true }))
	})

	t.Run("invalid", func(t *testing.T) {
		requireEqual(t, false, isAuthorized("testdata/invalid_authorized_keys", func(k ssh.PublicKey) bool { return true }))
	})

	t.Run("file not found", func(t *testing.T) {
		requireEqual(t, false, isAuthorized("testdata/nope_authorized_keys", func(k ssh.PublicKey) bool { return true }))
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
		requireNoError(
			t,
			WithAuthorizedKeys("testdata/invalid_authorized_keys")(&s),
		)
	})

	t.Run("file not found", func(t *testing.T) {
		s := ssh.Server{}
		if err := WithAuthorizedKeys("testdata/nope_authorized_keys")(&s); err == nil {
			t.Fatal("expected an error, got nil")
		}
	})
}

func TestWithTrustedUserCAKeys(t *testing.T) {
	setup := func(tb testing.TB, certPath string) (*ssh.Server, *gossh.ClientConfig) {
		tb.Helper()
		s := &ssh.Server{
			Handler: func(s ssh.Session) {
				cert, ok := s.PublicKey().(*gossh.Certificate)
				fmt.Fprintf(s, "cert? %v - principals: %v - type: %v", ok, cert.ValidPrincipals, cert.CertType)
			},
		}
		requireNoError(tb, WithTrustedUserCAKeys("testdata/ca.pub")(s))

		signer, err := gossh.ParsePrivateKey(getBytes(tb, "testdata/foo"))
		requireNoError(tb, err)

		cert, _, _, _, err := gossh.ParseAuthorizedKey(getBytes(tb, certPath))
		requireNoError(tb, err)

		certSigner, err := gossh.NewCertSigner(cert.(*gossh.Certificate), signer)
		requireNoError(tb, err)
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
			t.Fatal("expected an error, got nil")
		}
	})

	t.Run("valid", func(t *testing.T) {
		s, cc := setup(t, "testdata/valid-cert.pub")
		sess := testsession.New(t, s, cc)
		var b bytes.Buffer
		sess.Stdout = &b
		requireNoError(t, sess.Run(""))
		requireEqual(t, "cert? true - principals: [foo] - type: 1", b.String())
	})

	t.Run("valid wrong principal", func(t *testing.T) {
		s, cc := setup(t, "testdata/valid-cert.pub")
		cc.User = "not-foo"
		_, err := testsession.NewClientSession(t, testsession.Listen(t, s), cc)
		requireAuthError(t, err)
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
