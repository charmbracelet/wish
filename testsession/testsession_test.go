package testsession

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gliderlabs/ssh"
)

func TestSession(t *testing.T) {
	const out = "hello world"
	session := New(t, &ssh.Server{
		Handler: func(s ssh.Session) {
			_, _ = fmt.Fprint(s, out)
		},
	}, nil)
	var w bytes.Buffer
	session.Stdout = &w
	if err := session.Run(""); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if w.String() != out {
		t.Errorf("expected %q, got %q", out, w.String())
	}
}
