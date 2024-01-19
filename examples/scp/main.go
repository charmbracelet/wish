package main

// An example SCP server. This will serve files from and to ./examples/scp/testdata.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/scp"
	"github.com/pkg/sftp"
)

const (
	host = "localhost"
	port = 23235
)

func main() {
	root, _ := filepath.Abs("./examples/scp/testdata")
	handler := scp.NewFileSystemHandler(root)
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithSubsystem("sftp", sftpSubsystem(root)),
		wish.WithMiddleware(
			scp.Middleware(handler, handler),
		),
	)
	if err != nil {
		log.Error("could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}

func sftpSubsystem(root string) ssh.SubsystemHandler {
	return func(s ssh.Session) {
		log.Info("sftp", "root", root)
		fs := &sftpHandler{root}
		srv := sftp.NewRequestServer(s, sftp.Handlers{
			FileList: fs,
			FileGet:  fs,
		})
		if err := srv.Serve(); err == io.EOF {
			if err := srv.Close(); err != nil {
				wish.Fatalln(s, "sftp:", err)
			}
		} else if err != nil {
			wish.Fatalln(s, "sftp:", err)
		}
	}
}

var (
	_ sftp.FileLister = &sftpHandler{}
	_ sftp.FileReader = &sftpHandler{}
)

// example readonly handler implementation for sftp.
type sftpHandler struct {
	root string
}

// listerAt satisfies [sftp.ListerAt].
type listerAt []fs.FileInfo

func (l listerAt) ListAt(ls []fs.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(ls, l[offset:])
	if n < len(ls) {
		return n, io.EOF
	}

	return n, nil
}

// Fileread implements sftp.FileReader.
func (s *sftpHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	var flags int
	pflags := r.Pflags()
	if pflags.Append {
		flags |= os.O_APPEND
	}
	if pflags.Creat {
		flags |= os.O_CREATE
	}
	if pflags.Excl {
		flags |= os.O_EXCL
	}
	if pflags.Trunc {
		flags |= os.O_TRUNC
	}

	if pflags.Read && pflags.Write {
		flags |= os.O_RDWR
	} else if pflags.Read {
		flags |= os.O_RDONLY
	} else if pflags.Write {
		flags |= os.O_WRONLY
	}

	f, err := os.OpenFile(filepath.Join(s.root, r.Filepath), flags, 0600)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// Filelist implements sftp.FileLister.
func (s *sftpHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	switch r.Method {
	case "List":
		entries, err := os.ReadDir(filepath.Join(s.root, r.Filepath))
		if err != nil {
			return nil, fmt.Errorf("sftp: %w", err)
		}
		infos := make([]fs.FileInfo, len(entries))
		for i, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			infos[i] = info
		}
		return listerAt(infos), nil
	case "Stat":
		fi, err := os.Stat(filepath.Join(s.root, r.Filepath))
		if err != nil {
			return nil, err
		}
		return listerAt{fi}, nil
	default:
		return nil, sftp.ErrSSHFxOpUnsupported
	}
}
