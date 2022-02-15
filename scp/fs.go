package scp

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/gliderlabs/ssh"
)

type fsHandler struct {
	fsys fs.FS
}

var _ Handler = &fsHandler{}

func NewFSHandler(fsys fs.FS) Handler {
	return &fsHandler{
		fsys: fsys,
	}
}

func (h *fsHandler) WalkDir(_ context.Context, _ ssh.PublicKey, path string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(h.fsys, path, fn)
}

func (h *fsHandler) NewDirEntry(_ context.Context, _ ssh.PublicKey, path string) (*DirEntry, error) {
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

func (h *fsHandler) NewFileEntry(_ context.Context, _ ssh.PublicKey, path string) (*FileEntry, func() error, error) {
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
		Mtime:    info.ModTime().Unix(),
		Atime:    info.ModTime().Unix(),
		Size:     info.Size(),
		Reader:   f,
	}, f.Close, nil
}

func (h *fsHandler) Mkdir(context.Context, ssh.PublicKey, *DirEntry) error         { return nil }
func (h *fsHandler) Write(context.Context, ssh.PublicKey, *FileEntry) (int, error) { return 0, nil }
