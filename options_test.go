package wish

import (
	"strings"
	"testing"
	"time"

	"github.com/gliderlabs/ssh"
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
	t.Skip("TODO: find a way to test this")
}

func requireEqual(tb testing.TB, a, b interface{}) {
	tb.Helper()
	if a != b {
		tb.Errorf("expected %v, got %v", a, b)
	}
}

func requireNoError(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Errorf("expected no error, got %v", err)
	}
}
