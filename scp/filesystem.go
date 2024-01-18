package scp

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/pkg/sftp"
)

// FileSystemHandler is Handler and SFTP implementation for a given root path.
type FileSystemHandler struct{ root string }

var (
	_ Handler = &FileSystemHandler{}
	_ Sftp    = &FileSystemHandler{}
)

// NewFileSystemHandler return a Handler based on the given dir.
func NewFileSystemHandler(root string) *FileSystemHandler {
	return &FileSystemHandler{
		root: filepath.Clean(root),
	}
}

func (h *FileSystemHandler) ServerOptions() []sftp.ServerOption {
	return []sftp.ServerOption{
		func(s *sftp.Server) error {
			abs, err := filepath.Abs(h.root)
			if err != nil {
				return err
			}
			return sftp.WithServerWorkingDirectory(abs)(s)
		},
	}
}

func (h *FileSystemHandler) chtimes(path string, mtime, atime int64) error {
	if mtime == 0 || atime == 0 {
		return nil
	}
	if err := os.Chtimes(
		h.prefixed(path),
		time.Unix(atime, 0),
		time.Unix(mtime, 0),
	); err != nil {
		return fmt.Errorf("failed to chtimes: %q: %w", path, err)
	}
	return nil
}

func (h *FileSystemHandler) prefixed(path string) string {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, h.root) {
		return path
	}
	return filepath.Join(h.root, path)
}

func (h *FileSystemHandler) Glob(_ ssh.Session, s string) ([]string, error) {
	matches, err := filepath.Glob(h.prefixed(s))
	if err != nil {
		return []string{}, err
	}

	for i, match := range matches {
		matches[i], err = filepath.Rel(h.root, match)
		if err != nil {
			return []string{}, err
		}
	}
	return matches, nil
}

func (h *FileSystemHandler) WalkDir(_ ssh.Session, path string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(h.prefixed(path), func(path string, d fs.DirEntry, err error) error {
		// if h.root is ./foo/bar, we don't want to server `bar` as the root,
		// but instead its contents.
		if path == h.root {
			return err
		}
		return fn(path, d, err)
	})
}

func (h *FileSystemHandler) NewDirEntry(_ ssh.Session, name string) (*DirEntry, error) {
	path := h.prefixed(name)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open dir: %q: %w", path, err)
	}
	return &DirEntry{
		Children: []Entry{},
		Name:     info.Name(),
		Filepath: path,
		Mode:     info.Mode(),
		Mtime:    info.ModTime().Unix(),
		Atime:    info.ModTime().Unix(),
	}, nil
}

func (h *FileSystemHandler) NewFileEntry(_ ssh.Session, name string) (*FileEntry, func() error, error) {
	path := h.prefixed(name)
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
		Mode:     info.Mode(),
		Size:     info.Size(),
		Mtime:    info.ModTime().Unix(),
		Atime:    info.ModTime().Unix(),
		Reader:   f,
	}, f.Close, nil
}

func (h *FileSystemHandler) Mkdir(_ ssh.Session, entry *DirEntry) error {
	if err := os.Mkdir(h.prefixed(entry.Filepath), entry.Mode); err != nil {
		return fmt.Errorf("failed to create dir: %q: %w", entry.Filepath, err)
	}
	return h.chtimes(entry.Filepath, entry.Mtime, entry.Atime)
}

func (h *FileSystemHandler) Write(_ ssh.Session, entry *FileEntry) (int64, error) {
	f, err := os.OpenFile(h.prefixed(entry.Filepath), os.O_TRUNC|os.O_RDWR|os.O_CREATE, entry.Mode)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %q: %w", entry.Filepath, err)
	}
	defer f.Close() //nolint:errcheck
	written, err := io.Copy(f, entry.Reader)
	if err != nil {
		return 0, fmt.Errorf("failed to write file: %q: %w", entry.Filepath, err)
	}
	if err := f.Close(); err != nil {
		return 0, fmt.Errorf("failed to close file: %q: %w", entry.Filepath, err)
	}
	return written, h.chtimes(entry.Filepath, entry.Mtime, entry.Atime)
}
