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

func MiddlewarePath(path string) wish.Middleware {
	// TODO: handle path being a file
	return MiddlewareFS(os.DirFS(path))
}

// Middleware handles SCPs from a file or dir to a filesystem.
func MiddlewareFS(fsys fs.FS) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			isScp, root, recursive := details(s.Command())
			if !isScp {
				sh(s)
				return
			}

			if !recursive {
				entry, closer, err := newFileEntry(fsys, root)
				if err != nil {
					errHandler(s, err)
					return
				}
				defer closer()
				if err := entry.Write(s); err != nil {
					errHandler(s, err)
					return
				}
				sh(s)
				return
			}

			rootEntry, err := getRootEntry(fsys, root)
			if err != nil {
				errHandler(s, err)
				return
			}

			var closers []func() error
			defer func() {
				for _, closer := range closers {
					if err := closer(); err != nil {
						log.Println("failed to close:", err)
					}
				}
			}()

			if err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if path == root {
					return nil
				}

				if d.IsDir() {
					entry, err := newDirEntry(fsys, path)
					if err != nil {
						return err
					}
					rootEntry.Append(entry)
				} else {
					entry, closer, err := newFileEntry(fsys, path)
					if err != nil {
						return err
					}
					closers = append(closers, closer)
					rootEntry.Append(entry)
				}

				return nil
			}); err != nil {
				errHandler(s, fmt.Errorf("walk failed: %w", err))
				return
			}
			if err := rootEntry.Write(s); err != nil {
				errHandler(s, err)
				return
			}

			sh(s)
		}
	}
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

func getRootEntry(fsys fs.FS, root string) (RootEntry, error) {
	if root == "/" || root == "." {
		return &NoDirRootEntry{}, nil
	}
	return newDirEntry(fsys, root)
}

func newFileEntry(fsys fs.FS, path string) (*FileEntry, func() error, error) {
	info, err := fs.Stat(fsys, path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat %q: %w", path, err)
	}
	f, err := fsys.Open(path)
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

func newDirEntry(fsys fs.FS, path string) (*DirEntry, error) {
	info, err := fs.Stat(fsys, path)
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

func details(cmd []string) (bool, string, bool) {
	if len(cmd) == 0 || cmd[0] != "scp" {
		return false, "", false
	}

	var name string
	var recursive bool
	for i, p := range cmd {
		if p == "-r" {
			recursive = true
		}
		if p == "-f" {
			name = strings.TrimPrefix(strings.TrimPrefix(cmd[i+1], "./"), "/")
			if name == "" {
				name = "."
			}
		}
	}
	return true, name, recursive
}

func errHandler(s ssh.Session, err error) {
	s.Stderr().Write([]byte(err.Error()))
	s.Exit(1)
}
