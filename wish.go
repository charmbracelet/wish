package wish

import (
	"fmt"
	"io"

	"github.com/charmbracelet/keygen"
	"github.com/gliderlabs/ssh"
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
		k, err := keygen.New("", nil, keygen.Ed25519)
		if err != nil {
			return nil, err
		}
		err = s.SetOption(WithHostKeyPEM(k.PrivateKeyPEM()))
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
	_ = s.Exit(1)
	_ = s.Close()
}

// Error prints the given error the the session's STDERR.
func Error(s ssh.Session, v ...interface{}) {
	_, _ = fmt.Fprint(newErrorCRLFWriter(s), v...)
}

// Errorf formats according to the given format and prints to the session's STDERR.
func Errorf(s ssh.Session, f string, v ...interface{}) {
	_, _ = fmt.Fprintf(newErrorCRLFWriter(s), f, v...)
}

// Errorf formats according to the default format and prints to the session's STDERR.
func Errorln(s ssh.Session, v ...interface{}) {
	_, _ = fmt.Fprintln(newErrorCRLFWriter(s), v...)
}

// Print writes to the session's STDOUT followed.
func Print(s ssh.Session, v ...interface{}) {
	_, _ = fmt.Fprint(newCRLFWriter(s), v...)
}

// Printf formats according to the given format and writes to the session's STDOUT.
func Printf(s ssh.Session, f string, v ...interface{}) {
	_, _ = fmt.Fprintf(newCRLFWriter(s), f, v...)
}

// Println formats according to the default format and writes to the session's STDOUT.
func Println(s ssh.Session, v ...interface{}) {
	_, _ = fmt.Fprintln(newCRLFWriter(s), v...)
}

// WriteString writes the given string to the session's STDOUT.
func WriteString(s ssh.Session, v string) (int, error) {
	return io.WriteString(s, v)
}

func newErrorCRLFWriter(s ssh.Session) io.Writer {
	return crlfWriter{s, s.Stderr()}
}

func newCRLFWriter(s ssh.Session) io.Writer {
	return crlfWriter{s, s}
}

type crlfWriter struct {
	s ptyier
	w io.Writer
}

func (w crlfWriter) Write(v []byte) (int, error) {
	if _, _, active := w.s.Pty(); active {
		var output []byte
		for i, b := range v {
			if b == '\n' && (i == 0 || v[i-1] != '\r') {
				output = append(output, '\r')
			}
			output = append(output, b)
		}
		return w.w.Write(output)
	}
	return w.w.Write(v)
}

type ptyier interface {
	Pty() (ssh.Pty, <-chan ssh.Window, bool)
}
