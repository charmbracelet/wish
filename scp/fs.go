package scp

import (
	"fmt"
	"io/fs"

	"github.com/charmbracelet/wish"
)

type fsHandler struct{ fsys fs.FS }

var _ CopyToClientHandler = &fsHandler{}

// NewFSReadHandler returns a read-only CopyToClientHandler that accepts any
// fs.FS as input.
func NewFSReadHandler(fsys fs.FS) CopyToClientHandler {
	return &fsHandler{fsys: fsys}
}

func (h *fsHandler) Glob(_ wish.Session, s string) ([]string, error) {
	return fs.Glob(h.fsys, s)
}

func (h *fsHandler) WalkDir(_ wish.Session, path string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(h.fsys, path, fn)
}

func (h *fsHandler) NewDirEntry(_ wish.Session, path string) (*DirEntry, error) {
	info, err := fs.Stat(h.fsys, path)
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

func (h *fsHandler) NewFileEntry(_ wish.Session, path string) (*FileEntry, func() error, error) {
	info, err := fs.Stat(h.fsys, path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat %q: %w", path, err)
	}
	f, err := h.fsys.Open(path)
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
