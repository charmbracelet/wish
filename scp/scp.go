// Package scp provides a SCP middleware for wish.
package scp

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// CopyToClientHandler is a handler that can be implemented to handle files
// being copied from the server to the client.
type CopyToClientHandler interface {
	// Glob should be implemented if you want to provide server-side globbing
	// support.
	//
	// A minimal implementation to disable it is to return `[]string{s}, nil`.
	//
	// Note: if your other functions expect a relative path, make sure that
	// your Glob implementation returns relative paths as well.
	Glob(ssh.Session, string) ([]string, error)

	// WalkDir must be implemented if you want to allow recursive copies.
	WalkDir(ssh.Session, string, fs.WalkDirFunc) error

	// NewDirEntry should provide a *DirEntry for the given path.
	NewDirEntry(ssh.Session, string) (*DirEntry, error)

	// NewFileEntry should provide a *FileEntry for the given path.
	// Users may also provide a closing function.
	NewFileEntry(ssh.Session, string) (*FileEntry, func() error, error)
}

// CopyFromClientHandler is a handler that can be implemented to handle files
// being copied from the client to the server.
type CopyFromClientHandler interface {
	// Mkdir should created the given dir.
	// Note that this usually shouldn't use os.MkdirAll and the like.
	Mkdir(ssh.Session, *DirEntry) error

	// Write should write the given file.
	Write(ssh.Session, *FileEntry) (int64, error)
}

// Handler is a interface that can be implemented to handle both SCP
// directions.
type Handler interface {
	CopyFromClientHandler
	CopyToClientHandler
}

// Middleware provides a wish middleware using the given CopyToClientHandler
// and CopyFromClientHandler.
func Middleware(rh CopyToClientHandler, wh CopyFromClientHandler) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			info := GetInfo(s.Command())
			if !info.Ok {
				sh(s)
				return
			}

			var err error
			switch info.Op {
			case OpCopyToClient:
				if rh == nil {
					err = fmt.Errorf("no handler provided for scp -f")
					break
				}
				err = copyToClient(s, info, rh)
			case OpCopyFromClient:
				if wh == nil {
					err = fmt.Errorf("no handler provided for scp -t")
					break
				}
				err = copyFromClient(s, info, wh)
			}
			if err != nil {
				wish.Fatal(s, err)
				return
			}
		}
	}
}

// NULL is an array with a single NULL byte.
var NULL = []byte{'\x00'}

// Entry defines something that knows how to write itself and its path.
type Entry interface {
	// Write the current entry in SCP format.
	Write(io.Writer) error

	path() string
}

// AppendableEntry defines a special kind of Entry, which can contain
// children.
type AppendableEntry interface {
	// Write the current entry in SCP format.
	Write(io.Writer) error

	// Append another entry to the current entry.
	Append(entry Entry)
}

// FileEntry is an Entry that reads from a Reader, defining a file and
// its contents.
type FileEntry struct {
	Name     string
	Filepath string
	Mode     fs.FileMode
	Size     int64
	Reader   io.Reader
	Atime    int64
	Mtime    int64
}

func (e *FileEntry) path() string { return e.Filepath }

// Write a file to the given writer.
func (e *FileEntry) Write(w io.Writer) error {
	if e.Mtime > 0 && e.Atime > 0 {
		if _, err := fmt.Fprintf(w, "T%d 0 %d 0\n", e.Mtime, e.Atime); err != nil {
			return fmt.Errorf("failed to write file: %q: %w", e.Filepath, err)
		}
	}
	if _, err := fmt.Fprintf(w, "C%s %d %s\n", octalPerms(e.Mode), e.Size, e.Name); err != nil {
		return fmt.Errorf("failed to write file: %q: %w", e.Filepath, err)
	}

	if _, err := io.Copy(w, e.Reader); err != nil {
		return fmt.Errorf("failed to read file: %q: %w", e.Filepath, err)
	}

	if _, err := w.Write(NULL); err != nil {
		return fmt.Errorf("failed to write file: %q: %w", e.Filepath, err)
	}
	return nil
}

// RootEntry is a root entry that can only have children.
type RootEntry []Entry

// Appennd the given entry to a child directory, or the the itself if
// none matches.
func (e *RootEntry) Append(entry Entry) {
	parent := normalizePath(filepath.Dir(entry.path()))

	for _, child := range *e {
		switch dir := child.(type) {
		case *DirEntry:
			if child.path() == parent {
				dir.Children = append(dir.Children, entry)
				return
			}
			if strings.HasPrefix(parent, normalizePath(dir.Filepath)) {
				dir.Append(entry)
				return
			}
		default:
			continue
		}
	}

	*e = append(*e, entry)
}

// Write recursively writes all the children to the given writer.
func (e *RootEntry) Write(w io.Writer) error {
	for _, child := range *e {
		if err := child.Write(w); err != nil {
			return err
		}
	}
	return nil
}

// DirEntry is an Entry with mode, possibly children, and possibly a
// parent.
type DirEntry struct {
	Children []Entry
	Name     string
	Filepath string
	Mode     fs.FileMode
	Atime    int64
	Mtime    int64
}

func (e *DirEntry) path() string { return e.Filepath }

// Write the current dir entry, all its contents (recursively), and the
// dir closing to the given writer.
func (e *DirEntry) Write(w io.Writer) error {
	if e.Mtime > 0 && e.Atime > 0 {
		if _, err := fmt.Fprintf(w, "T%d 0 %d 0\n", e.Mtime, e.Atime); err != nil {
			return fmt.Errorf("failed to write dir: %q: %w", e.Filepath, err)
		}
	}
	if _, err := fmt.Fprintf(w, "D%s 0 %s\n", octalPerms(e.Mode), e.Name); err != nil {
		return fmt.Errorf("failed to write dir: %q: %w", e.Filepath, err)
	}

	for _, child := range e.Children {
		if err := child.Write(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(w, "E\n"); err != nil {
		return fmt.Errorf("failed to write dir: %q: %w", e.Filepath, err)
	}
	return nil
}

// Appends an entry to the folder or their children.
func (e *DirEntry) Append(entry Entry) {
	parent := normalizePath(filepath.Dir(entry.path()))

	for _, child := range e.Children {
		switch dir := child.(type) {
		case *DirEntry:
			if child.path() == parent {
				dir.Children = append(dir.Children, entry)
				return
			}
			if strings.HasPrefix(parent, normalizePath(dir.path())) {
				dir.Append(entry)
				return
			}
		default:
			continue
		}
	}

	e.Children = append(e.Children, entry)
}

// Op defines which kind of SCP Operation is going on.
type Op byte

const (
	// OpCopyToClient is when a file is being copied from the server to the client.
	OpCopyToClient Op = 'f'

	// OpCopyFromClient is when a file is being copied from the client into the server.
	OpCopyFromClient Op = 't'
)

// Info provides some information about the current SCP Operation.
type Info struct {
	// Ok is true if the current session is a SCP.
	Ok bool

	// Recursive is true if its a recursive SCP.
	Recursive bool

	// Path is the server path of the scp operation.
	Path string

	// Op is the SCP operation kind.
	Op Op
}

// GetInfo return information about the given command.
func GetInfo(cmd []string) Info {
	info := Info{}
	if len(cmd) == 0 || cmd[0] != "scp" {
		return info
	}

	for i, p := range cmd {
		switch p {
		case "-r":
			info.Recursive = true
		case "-f":
			info.Op = OpCopyToClient
			info.Path = cmd[i+1]
		case "-t":
			info.Op = OpCopyFromClient
			info.Path = cmd[i+1]
		}
	}

	info.Ok = true
	return info
}

func octalPerms(info fs.FileMode) string {
	return "0" + strconv.FormatUint(uint64(info.Perm()), 8)
}

func normalizePath(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(p, "\\", "/")
	}
	return p
}
