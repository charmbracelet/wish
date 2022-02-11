package scp

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

// Middleware handles SCPs from a file or dir to a filesystem.
func MiddlewareFS(fsys fs.FS) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			isScp, root, recursive := details(s)
			if !isScp {
				sh(s)
				return
			}

			if recursive {
				rootEntry := &DirEntry{
					Children: []Entry{},
					Name:     root,
					Filepath: root,
					Mode:     "0755",
					Mtime:    time.Now().Unix(),
					Atime:    time.Now().Unix(),
				}
				if err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if path == root {
						return nil
					}
					info, err := d.Info()
					if err != nil {
						return fmt.Errorf("failed to walk: %w", err)
					}

					if d.IsDir() {
						rootEntry.Append(&DirEntry{
							Children: []Entry{},
							Name:     d.Name(),
							Filepath: path,
							Mode:     octalPerms(info),
							Mtime:    info.ModTime().Unix(),
							Atime:    info.ModTime().Unix(),
						})
					} else {
						f, err := fsys.Open(path)
						if err != nil {
							return err
						}
						rootEntry.Append(&FileEntry{
							Name:     d.Name(),
							Filepath: path,
							Mode:     octalPerms(info),
							Mtime:    info.ModTime().Unix(),
							Atime:    info.ModTime().Unix(),
							Size:     info.Size(),
							Reader:   f,
						})
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
			} else {
				f, err := fsys.Open(root)
				if err != nil {
					errHandler(s, err)
					return
				}
				info, err := f.Stat()
				if err != nil {
					errHandler(s, err)
					return
				}
				fe := &FileEntry{
					Name:     root,
					Filepath: root,
					Mode:     octalPerms(info),
					Mtime:    info.ModTime().Unix(),
					Atime:    info.ModTime().Unix(),
					Size:     info.Size(),
					Reader:   f,
				}
				if err := fe.Write(s); err != nil {
					errHandler(s, err)
					return
				}
			}

			sh(s)
		}
	}
}

func octalPerms(info fs.FileInfo) string {
	return "0" + strconv.FormatUint(uint64(info.Mode().Perm()), 8)
}

func MiddlewarePath(path string) wish.Middleware {
	// TODO: handle path being a file
	return MiddlewareFS(os.DirFS(path))
}

func details(s ssh.Session) (bool, string, bool) {
	cmd := s.Command()
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
			name = cmd[i+1]
		}
	}
	return true, name, recursive
}

func errHandler(s ssh.Session, err error) {
	s.Stderr().Write([]byte(err.Error()))
	s.Exit(1)
}

type Entry interface {
	Write(io.Writer) error
	Path() string
}

type FileEntry struct {
	Name     string
	Filepath string
	Mode     string
	Mtime    int64
	Atime    int64
	Size     int64
	Reader   io.ReadCloser
}

func (e *FileEntry) Path() string { return e.Filepath }

func (e *FileEntry) Write(w io.Writer) error {
	content, err := io.ReadAll(e.Reader)
	if err != nil {
		return fmt.Errorf("failed to read file: %q: %w", e.Filepath, err)
	}
	defer e.Reader.Close()
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

type DirEntry struct {
	Children []Entry
	Name     string
	Filepath string
	Mode     string
	Mtime    int64
	Atime    int64
}

func (e *DirEntry) Path() string { return e.Filepath }

func (e *DirEntry) Write(w io.Writer) error {
	for _, bts := range [][]byte{
		[]byte(fmt.Sprintf("T%d 0 %d 0\n", e.Mtime, e.Atime)),
		[]byte(fmt.Sprintf("D%s 0 %s\n", e.Mode, e.Name)),
	} {
		if _, err := w.Write(bts); err != nil {
			return fmt.Errorf("failed to write dir: %q: %w", e.Filepath, err)
		}
	}

	for _, ce := range e.Children {
		if err := ce.Write(w); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte{'E', '\n'}); err != nil {
		return fmt.Errorf("failed to write dir: %q: %w", e.Filepath, err)
	}
	return nil
}

func (e *DirEntry) Append(ce Entry) {
	parent := filepath.Dir(ce.Path())
	for _, ee := range e.Children {
		de, ok := ee.(*DirEntry)
		if !ok {
			continue
		}
		if ee.Path() == parent {
			de.Children = append(de.Children, ce)
			return
		}
		if strings.HasPrefix(parent, de.Path()) {
			de.Append(ce)
			return
		}
	}
	e.Children = append(e.Children, ce)
}
