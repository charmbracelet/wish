package wish

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/ssh"
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
	for _, op := range ops {
		if err := s.SetOption(op); err != nil {
			return nil, err
		}
	}
	if len(s.HostSigners) == 0 {
		k, err := keygen.New("id_ed25519", keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
		if err != nil {
			return nil, err
		}
		err = s.SetOption(WithHostKeyPEM(k.RawPrivateKey()))
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Fatal prints to the given session's STDERR and exits 1.
func Fatal(s ssh.Session, v ...interface{}) {
	Error(s, v...)
	_ = s.Exit(1)
	_ = s.Close()
}

// Fatalf formats according to the given format, prints to the session's STDERR
// followed by an exit 1.
//
// Notice that this might cause formatting issues if you don't add a \r\n in the end of your string.
func Fatalf(s ssh.Session, f string, v ...interface{}) {
	Errorf(s, f, v...)
	_ = s.Exit(1)
	_ = s.Close()
}

// Fatalln formats according to the default format, prints to the session's
// STDERR, followed by a new line and an exit 1.
func Fatalln(s ssh.Session, v ...interface{}) {
	Errorln(s, v...)
	Errorf(s, "\r")
	_ = s.Exit(1)
	_ = s.Close()
}

// Error prints the given error the the session's STDERR.
func Error(s ssh.Session, v ...interface{}) {
	_, _ = fmt.Fprint(s.Stderr(), v...)
}

// Errorf formats according to the given format and prints to the session's STDERR.
func Errorf(s ssh.Session, f string, v ...interface{}) {
	_, _ = fmt.Fprintf(s.Stderr(), f, v...)
}

// Errorf formats according to the default format and prints to the session's STDERR.
func Errorln(s ssh.Session, v ...interface{}) {
	_, _ = fmt.Fprintln(s.Stderr(), v...)
}

// Print writes to the session's STDOUT followed.
func Print(s ssh.Session, v ...interface{}) {
	_, _ = fmt.Fprint(s, v...)
}

// Printf formats according to the given format and writes to the session's STDOUT.
func Printf(s ssh.Session, f string, v ...interface{}) {
	_, _ = fmt.Fprintf(s, f, v...)
}

// Println formats according to the default format and writes to the session's STDOUT.
func Println(s ssh.Session, v ...interface{}) {
	_, _ = fmt.Fprintln(s, v...)
}

// WriteString writes the given string to the session's STDOUT.
func WriteString(s ssh.Session, v string) (int, error) {
	return io.WriteString(s, v)
}

// Command sets stdin, stdout, and stderr to the current session's PTY slave.
//
// If the current session does not have a PTY, it sets them to the session
// itself.
func Command(s ssh.Session, name string, args ...string) *Cmd {
	c := exec.Command(name, args...)
	pty, _, ok := s.Pty()
	if !ok {
		c.Stdin, c.Stdout, c.Stderr = s, s, s
		return &Cmd{cmd: c}
	}

	return &Cmd{c, &pty}
}

// Cmd wraps a *exec.Cmd and a ssh.Pty so a command can be properly run.
type Cmd struct {
	cmd *exec.Cmd
	pty *ssh.Pty
}

// SetDir set the underlying exec.Cmd env.
func (c *Cmd) SetEnv(env []string) {
	c.cmd.Env = env
}

// Environ returns the underlying exec.Cmd environment.
func (c *Cmd) Environ() []string {
	return c.cmd.Environ()
}

// SetDir set the underlying exec.Cmd dir.
func (c *Cmd) SetDir(dir string) {
	c.cmd.Dir = dir
}

// SetStderr conforms with tea.ExecCommand.
func (*Cmd) SetStderr(io.Writer) {}

// SetStdin conforms with tea.ExecCommand.
func (*Cmd) SetStdin(io.Reader) {}

// SetStdout conforms with tea.ExecCommand.
func (*Cmd) SetStdout(io.Writer) {}

// Run runs the program and waits for it to finish.
func (c *Cmd) Run() error {
	if c.pty == nil {
		return c.Run()
	}

	if err := c.pty.Start(c.cmd); err != nil {
		return err
	}
	start := time.Now()
	if runtime.GOOS == "windows" {
		for c.cmd.ProcessState == nil {
			if time.Since(start) > time.Second*10 {
				return fmt.Errorf("could not start process")
			}
			time.Sleep(100 * time.Millisecond)
		}

		if !c.cmd.ProcessState.Success() {
			return fmt.Errorf("process failed: exit %d", c.cmd.ProcessState.ExitCode())
		}
	} else {
		if err := c.cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}
