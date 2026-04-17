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
)

// fileSystemHandler is a Handler implementation for a given root path.
type fileSystemHandler struct{ root string }

var _ Handler = &fileSystemHandler{}

// NewFileSystemHandler return a Handler based on the given dir.
func NewFileSystemHandler(root string) Handler {
	return &fileSystemHandler{
		root: filepath.Clean(root),
	}
}

func (h *fileSystemHandler) chtimes(path string, mtime, atime int64) error {
	if mtime == 0 || atime == 0 {
		return nil
	}
	p, err := h.prefixed(path)
	if err != nil {
		return err
	}
	if err := os.Chtimes(
		p,
		time.Unix(atime, 0),
		time.Unix(mtime, 0),
	); err != nil {
		return fmt.Errorf("failed to chtimes: %q: %w", path, err)
	}
	return nil
}

func (h *fileSystemHandler) prefixed(path string) (string, error) {
	clean := filepath.Clean(path)
	if clean == h.root || strings.HasPrefix(clean, h.root+string(filepath.Separator)) {
		return clean, nil
	}
	safe := filepath.Clean("/" + path)
	joined := filepath.Join(h.root, safe)
	if joined != h.root && !strings.HasPrefix(joined, h.root+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal detected: %q resolves outside root", path)
	}
	return joined, nil
}

func (h *fileSystemHandler) Glob(_ ssh.Session, s string) ([]string, error) {
	p, err := h.prefixed(s)
	if err != nil {
		return []string{}, err
	}
	matches, err := filepath.Glob(p)
	if err != nil {
		return []string{}, err //nolint:wrapcheck
	}

	var safe []string
	for _, match := range matches {
		if match != h.root && !strings.HasPrefix(match, h.root+string(filepath.Separator)) {
			continue
		}
		rel, err := filepath.Rel(h.root, match)
		if err != nil {
			return []string{}, err //nolint:wrapcheck
		}
		safe = append(safe, rel)
	}
	return safe, nil
}

func (h *fileSystemHandler) WalkDir(_ ssh.Session, path string, fn fs.WalkDirFunc) error {
	p, err := h.prefixed(path)
	if err != nil {
		return err
	}
	return filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error { //nolint:wrapcheck
		// if h.root is ./foo/bar, we don't want to server `bar` as the root,
		// but instead its contents.
		if path == h.root {
			return err
		}
		return fn(path, d, err)
	})
}

func (h *fileSystemHandler) NewDirEntry(_ ssh.Session, name string) (*DirEntry, error) {
	path, err := h.prefixed(name)
	if err != nil {
		return nil, err
	}
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

func (h *fileSystemHandler) NewFileEntry(_ ssh.Session, name string) (*FileEntry, func() error, error) {
	path, err := h.prefixed(name)
	if err != nil {
		return nil, nil, err
	}
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

func (h *fileSystemHandler) Mkdir(_ ssh.Session, entry *DirEntry) error {
	p, err := h.prefixed(entry.Filepath)
	if err != nil {
		return err
	}
	if err := os.Mkdir(p, entry.Mode); err != nil {
		return fmt.Errorf("failed to create dir: %q: %w", entry.Filepath, err)
	}
	return h.chtimes(entry.Filepath, entry.Mtime, entry.Atime)
}

func (h *fileSystemHandler) Write(_ ssh.Session, entry *FileEntry) (int64, error) {
	p, err := h.prefixed(entry.Filepath)
	if err != nil {
		return 0, err
	}
	f, err := os.OpenFile(p, os.O_TRUNC|os.O_RDWR|os.O_CREATE, entry.Mode)
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
