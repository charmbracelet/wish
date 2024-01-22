package wish

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
)

func TestCommandNoPty(t *testing.T) {
	tmp := t.TempDir()
	sess := testsession.New(t, &ssh.Server{
		Handler: func(s ssh.Session) {
			if err := Command(s, "echo", "hello").Run(); err != nil {
				Fatal(s, "echo:", err)
			}

			cmd := Command(s, "env")
			cmd.SetEnv([]string{"HELLO=hello"})
			if err := cmd.Run(); err != nil {
				Fatal(s, "env:", err)
			}

			cmd = Command(s, "pwd")
			cmd.SetDir(tmp)
			if err := cmd.Run(); err != nil {
				Fatal(s, "pwd:", err)
			}
		},
	}, nil)
	var out bytes.Buffer
	sess.Stdout = &out
	if err := sess.Run(""); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expect := strings.Join([]string{
		"hello",
		"HELLO=hello",
		tmp,
	}, "\n") + "\n"
	if s := out.String(); s != expect {
		t.Errorf("expected output to be %q, got %q", expect, s)
	}
}
