// Package scp provides a SCP middleware for wish.
package scp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

type CopyToClientHandler interface {
	WalkDir(context.Context, ssh.PublicKey, string, fs.WalkDirFunc) error
	NewDirEntry(context.Context, ssh.PublicKey, string) (*DirEntry, error)
	NewFileEntry(context.Context, ssh.PublicKey, string) (*FileEntry, func() error, error)
}

type CopyFromClientHandler interface {
	Mkdir(context.Context, ssh.PublicKey, *DirEntry) error
	Write(context.Context, ssh.PublicKey, *FileEntry) (int, error)
}

type Handler interface {
	CopyFromClientHandler
	CopyToClientHandler
}

func Middleware(handler Handler) wish.Middleware {
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
				err = copyToClient(s, info, handler)
			case OpCopyFromClient:
				err = copyFromClient(s, info, handler)
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

var (
	reTimestamp = regexp.MustCompile("^T(\\d{10}) 0 (\\d{10}) 0$")
	reNewFolder = regexp.MustCompile("^D(\\d{4}) 0 (.*)$")
	reNewFile   = regexp.MustCompile("^C(\\d{4}) (\\d+) (.*)$")
)

var NULL = []byte{'\x00'}

func copyFromClient(s ssh.Session, info Info, handler CopyFromClientHandler) error {
	// accepts the request
	s.Write(NULL)

	var (
		path        = "."
		mtime int64 = 0
		atime int64 = 0
		r           = bufio.NewReader(s)
	)

	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read line: %w", err)
		}

		matches := reTimestamp.FindAllStringSubmatch(string(line), 2)
		if matches != nil {
			if len(matches) != 1 || len(matches[0]) != 3 {
				return fmt.Errorf("cannot parse: %q", string(line))
			}
			mtime, err = strconv.ParseInt(matches[0][1], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to read line: %w", err)
			}
			atime, err = strconv.ParseInt(matches[0][2], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to read line: %w", err)
			}

			// accepts the header
			s.Write(NULL)
		}

		matches = reNewFile.FindAllStringSubmatch(string(line), 3)
		if matches != nil {
			if len(matches) != 1 || len(matches[0]) != 4 {
				return fmt.Errorf("cannot parse: %q", string(line))
			}

			name := matches[0][3]

			mode, err := strconv.ParseUint(matches[0][1], 10, 32)
			if err != nil {
				return fmt.Errorf("cannot parse: %q", string(line))
			}

			size, err := strconv.ParseInt(matches[0][2], 10, 64)
			if err != nil {
				return fmt.Errorf("cannot parse: %q", string(line))
			}

			// accepts the header
			s.Write(NULL)
			contents := make([]byte, size)
			if _, err := r.Read(contents); err != nil {
				return fmt.Errorf("cannot read %q: %w", name, err)
			}
			if int64(len(contents)) != size {
				return fmt.Errorf("sizes don't match: %q != %q", size, len(contents))
			}

			if _, err := handler.Write(s.Context(), s.PublicKey(), &FileEntry{
				Name:     name,
				Filepath: filepath.Join(path, name),
				Mode:     fs.FileMode(mode),
				Mtime:    mtime,
				Atime:    atime,
				Size:     size,
				Reader:   r,
			}); err != nil {
				return fmt.Errorf("failed to write file: %q: %w", name, err)
			}

			// read the trailing nil char
			_, _ = r.ReadByte() // TODO: check if it is indeed a NULL

			// says 'hey im done'
			s.Write(NULL)

			mtime = 0
			atime = 0

			continue
		}

		matches = reNewFolder.FindAllStringSubmatch(string(line), 2)
		if matches != nil {
			if len(matches) != 1 || len(matches[0]) != 3 {
				return fmt.Errorf("cannot parse: %q", string(line))
			}

			mode, err := strconv.ParseUint(matches[0][1], 10, 32)
			if err != nil {
				return fmt.Errorf("cannot parse: %q", string(line))
			}
			name := matches[0][2]

			path = filepath.Join(path, name)
			if err := handler.Mkdir(s.Context(), s.PublicKey(), &DirEntry{
				Name:     name,
				Filepath: path,
				Mode:     fs.FileMode(mode),
				Mtime:    mtime,
				Atime:    atime,
			}); err != nil {
				return fmt.Errorf("failed to create dir: %q: %w", name, err)
			}

			// says 'hey im done'
			s.Write(NULL)

			mtime = 0
			atime = 0

			continue
		}

		if string(line) == "E" {
			path = filepath.Dir(path)

			// says 'hey im done'
			s.Write(NULL)
			continue
		}

		// TODO: handle this better
		log.Println("unhandled", string(line))
	}
	return nil
}

func copyToClient(s ssh.Session, info Info, handler CopyToClientHandler) error {
	if !info.Recursive {
		entry, closer, err := handler.NewFileEntry(s.Context(), s.PublicKey(), info.Path) // newFileEntry(handler, info.Path)
		if err != nil {
			return err
		}
		defer closer()
		if err := entry.Write(s); err != nil {
			return err
		}
		return nil
	}

	rootEntry, err := getRootEntry(s.Context(), s.PublicKey(), handler, info.Path)
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

	if err := handler.WalkDir(s.Context(), s.PublicKey(), info.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == info.Path {
			return nil
		}

		if d.IsDir() {
			entry, err := handler.NewDirEntry(s.Context(), s.PublicKey(), path)
			if err != nil {
				return err
			}
			rootEntry.Append(entry)
		} else {
			entry, closer, err := handler.NewFileEntry(s.Context(), s.PublicKey(), path)
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
	Mode     fs.FileMode
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
		[]byte(fmt.Sprintf("C%s %d %s\n", octalPerms(e.Mode), e.Size, e.Name)),
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
	Mode     fs.FileMode
	Mtime    int64
	Atime    int64
}

func (e *DirEntry) path() string { return e.Filepath }

// Write the current dir entry, all its contents (recursively), and the
// dir closing to the given writer.
func (e *DirEntry) Write(w io.Writer) error {
	for _, bts := range [][]byte{
		[]byte(fmt.Sprintf("T%d 0 %d 0\n", e.Mtime, e.Atime)),
		[]byte(fmt.Sprintf("D%s 0 %s\n", octalPerms(e.Mode), e.Name)),
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

func getRootEntry(ctx context.Context, key ssh.PublicKey, handler CopyToClientHandler, root string) (RootEntry, error) {
	if root == "/" || root == "." {
		return &NoDirRootEntry{}, nil
	}

	return handler.NewDirEntry(ctx, key, root) // newDirEntry(filepath.Join(start, root))
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

func octalPerms(info fs.FileMode) string {
	return "0" + strconv.FormatUint(uint64(info.Perm()), 8)
}
