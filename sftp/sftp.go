package sftp

import (
	"io"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/pkg/sftp"
)

// Subsystem returns a ssh.SubsystemHandler that creates a
// *sftp.Server and serves the contents of the root directory
// (unless specified differently via a sftp.ServerOption).
func Subsystem(opts ...sftp.ServerOption) ssh.SubsystemHandler {
	return func(s ssh.Session) {
		srv, err := sftp.NewServer(s, opts...)
		if err != nil {
			wish.Fatalln(s, "sftp:", err)
		}
		if err := srv.Serve(); err == io.EOF {
			if err := srv.Close(); err != nil {
				wish.Fatalln(s, "sftp:", err)
			}
		} else if err != nil {
			wish.Fatalln(s, "sftp:", err)
		}
	}
}
