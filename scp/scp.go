// Package scp provides a SCP middleware for wish.
package scp

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// MiddlewarePath handles SCPs from the given folder.
func MiddlewarePath(path string) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			info := GetInfo(s.Command())
			if !info.Ok {
				sh(s)
				return
			}

			log.Printf("%+v", info)

			var err error
			switch info.Op {
			case OpCopyToClient:
				err = copyToClient(s, info, path)
			case OpCopyFromClient:
				err = copyFromClient(s, info, path)
			default:
				err = fmt.Errorf("invalid operation")
			}
			if err != nil {
				errHandler(s, err)
				return
			}

			sh(s)
		}
	}
}

func copyFromClient(s ssh.Session, info Info, path string) error {
	log.Println("copy from client")
	// TODO
	return nil
}

func copyToClient(s ssh.Session, info Info, path string) error {
	if !info.Recursive {
		entry, closer, err := newFileEntry(info.Path)
		if err != nil {
			return err
		}
		defer closer()
		if err := entry.Write(s); err != nil {
			return err
		}
		return nil
	}

	rootEntry, err := getRootEntry(path, info.Path)
	if err != nil {
		return err
	}

	var closers []func() error
	defer func() {
		for _, closer := range closers {
			if err := closer(); err != nil {
				log.Println("failed to close:", err)
			}
		}
	}()

	start := filepath.Join(path, info.Path)
	if err := filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == info.Path {
			return nil
		}

		if d.IsDir() {
			entry, err := newDirEntry(path)
			if err != nil {
				return err
			}
			rootEntry.Append(entry)
		} else {
			entry, closer, err := newFileEntry(path)
			if err != nil {
				return err
			}
			closers = append(closers, closer)
			rootEntry.Append(entry)
		}

		return nil
	}); err != nil {
		return err
	}
	if err := rootEntry.Write(s); err != nil {
		return err
	}
	return nil
}

// Entry defines something that knows how to write itself and its path.
type Entry interface {
	Write(io.Writer) error
	path() string
}

// RootEntry defines a special kind of Entry, which can contain
// children.
type RootEntry interface {
	Write(io.Writer) error
	Append(entry Entry)
}

// FileEntry is an Entry that reads from a Reader, defining a file and
// its contents.
type FileEntry struct {
	Name     string
	Filepath string
	Mode     string
	Mtime    int64
	Atime    int64
	Size     int64
	Reader   io.Reader
}

func (e *FileEntry) path() string { return e.Filepath }

// Write a file to the given writer.
func (e *FileEntry) Write(w io.Writer) error {
	content, err := io.ReadAll(e.Reader)
	if err != nil {
		return fmt.Errorf("failed to read file: %q: %w", e.Filepath, err)
	}
	for _, bts := range [][]byte{
		[]byte(fmt.Sprintf("T%d 0 %d 0\n", e.Mtime, e.Atime)),
		[]byte(fmt.Sprintf("C%s %d %s\n", e.Mode, e.Size, e.Name)),
		content,
		{'\x00'},
	} {
		if _, err := w.Write(bts); err != nil {
			return fmt.Errorf("failed to write file: %q: %w", e.Filepath, err)
		}
	}
	return nil
}

// NoDirRootEntry is a root entry that can only has children.
type NoDirRootEntry []Entry

// Appennd the given entry to a child directory, or the the itself if
// none matches.
func (e *NoDirRootEntry) Append(entry Entry) {
	parent := filepath.Dir(entry.path())

	for _, child := range *e {
		switch dir := child.(type) {
		case *DirEntry:
			if child.path() == parent {
				dir.Children = append(dir.Children, entry)
				return
			}
			if strings.HasPrefix(parent, dir.Filepath) {
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
func (e *NoDirRootEntry) Write(w io.Writer) error {
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
	Mode     string
	Mtime    int64
	Atime    int64
}

func (e *DirEntry) path() string { return e.Filepath }

// Write the current dir entry, all its contents (recursively), and the
// dir closing to the given writer.
func (e *DirEntry) Write(w io.Writer) error {
	for _, bts := range [][]byte{
		[]byte(fmt.Sprintf("T%d 0 %d 0\n", e.Mtime, e.Atime)),
		[]byte(fmt.Sprintf("D%s 0 %s\n", e.Mode, e.Name)),
	} {
		if _, err := w.Write(bts); err != nil {
			return fmt.Errorf("failed to write dir: %q: %w", e.Filepath, err)
		}
	}

	for _, child := range e.Children {
		if err := child.Write(w); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte{'E', '\n'}); err != nil {
		return fmt.Errorf("failed to write dir: %q: %w", e.Filepath, err)
	}
	return nil
}

// Appends an entry to the folder or their children.
func (e *DirEntry) Append(entry Entry) {
	parent := filepath.Dir(entry.path())

	for _, child := range e.Children {
		switch dir := child.(type) {
		case *DirEntry:
			if child.path() == parent {
				dir.Children = append(dir.Children, entry)
				return
			}
			if strings.HasPrefix(parent, dir.path()) {
				dir.Append(entry)
				return
			}
		default:
			continue
		}
	}

	e.Children = append(e.Children, entry)
}

func getRootEntry(start, root string) (RootEntry, error) {
	if root == "/" || root == "." {
		return &NoDirRootEntry{}, nil
	}
	return newDirEntry(filepath.Join(start, root))
}

func newFileEntry(path string) (*FileEntry, func() error, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat %q: %w", path, err)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %q: %w", path, err)
	}
	return &FileEntry{
		Name:     info.Name(),
		Filepath: path,
		Mode:     octalPerms(info),
		Mtime:    info.ModTime().Unix(),
		Atime:    info.ModTime().Unix(),
		Size:     info.Size(),
		Reader:   f,
	}, f.Close, nil
}

func newDirEntry(path string) (*DirEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open dir: %q: %w", path, err)
	}
	return &DirEntry{
		Children: []Entry{},
		Name:     info.Name(),
		Filepath: path,
		Mode:     octalPerms(info),
		Mtime:    info.ModTime().Unix(),
		Atime:    info.ModTime().Unix(),
	}, nil
}

func octalPerms(info fs.FileInfo) string {
	return "0" + strconv.FormatUint(uint64(info.Mode().Perm()), 8)
}

type Op byte

const (
	OpCopyToClient   Op = 'f'
	OpCopyFromClient Op = 't'
)

type Info struct {
	Ok        bool
	Recursive bool
	Path      string
	Op        Op
}

func GetInfo(cmd []string) Info {
	info := Info{}
	if len(cmd) == 0 || cmd[0] != "scp" {
		return info
	}

	info.Ok = true
	getPath := func(i int) string {
		// path := strings.TrimPrefix(strings.TrimPrefix(cmd[i+1], "./"), "/")
		// if path == "" {
		// 	path = "."
		// }
		// return path
		return cmd[i+1]
	}

	for i, p := range cmd {
		switch p {
		case "-r":
			info.Recursive = true
		case "-f":
			info.Path = getPath(i)
			info.Op = OpCopyToClient
		case "-t":
			info.Op = OpCopyFromClient
			info.Path = getPath(i)
		}
	}
	return info
}

func errHandler(s ssh.Session, err error) {
	s.Stderr().Write([]byte(err.Error()))
	s.Exit(1)
}
