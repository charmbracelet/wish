package testsession

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/ssh"
)

func TestSession(t *testing.T) {
	const out = "hello world"
	session := New(t, &ssh.Server{
		Handler: func(s ssh.Session) {
			_, _ = fmt.Fprint(s, out)
		},
	}, nil)
	result, err := session.Output("")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if string(result) != out {
		t.Errorf("expected %q, got %q", out, string(result))
	}
}
