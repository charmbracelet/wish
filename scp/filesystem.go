package scp

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gliderlabs/ssh"
)

type fileSystemHandler struct{ root string }

var _ Handler = &fileSystemHandler{}

// NewFileSystemHandler return a Handler based on the given dir.
func NewFileSystemHandler(root string) Handler {
	return &fileSystemHandler{
		root: filepath.Clean(root),
	}
}

func (h *fileSystemHandler) prefixed(path string) string {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, h.root) {
		return path
	}
	return filepath.Join(h.root, path)
}

func (h *fileSystemHandler) WalkDir(_ ssh.Session, path string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(filepath.Join(h.root, path), fn)
}

func (h *fileSystemHandler) NewDirEntry(_ ssh.Session, name string) (*DirEntry, error) {
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
	}, nil
}

func (h *fileSystemHandler) NewFileEntry(_ ssh.Session, name string) (*FileEntry, func() error, error) {
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
		Reader:   f,
	}, f.Close, nil
}

func (h *fileSystemHandler) Mkdir(_ ssh.Session, entry *DirEntry) error {
	if err := os.Mkdir(h.prefixed(entry.Filepath), entry.Mode); err != nil {
		return fmt.Errorf("failed to create dir: %q: %w", entry.Filepath, err)
	}
	return nil
}

func (h *fileSystemHandler) Write(_ ssh.Session, entry *FileEntry) (int64, error) {
	f, err := os.OpenFile(h.prefixed(entry.Filepath), os.O_TRUNC|os.O_RDWR|os.O_CREATE, entry.Mode)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %q: %w", entry.Filepath, err)
	}
	written, err := io.Copy(f, entry.Reader)
	if err != nil {
		return 0, fmt.Errorf("failed to write file: %q: %w", entry.Filepath, err)
	}
	return written, nil
}
