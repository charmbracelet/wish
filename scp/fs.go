package scp

import (
	"fmt"
	"io/fs"

	"github.com/gliderlabs/ssh"
)

type fsHandler struct{ fsys fs.FS }

var _ CopyToClientHandler = &fsHandler{}

func NewFSReadHandler(fsys fs.FS) CopyToClientHandler {
	return &fsHandler{fsys: fsys}
}

func (h *fsHandler) WalkDir(_ ssh.Session, path string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(h.fsys, path, fn)
}

func (h *fsHandler) NewDirEntry(_ ssh.Session, path string) (*DirEntry, error) {
	info, err := fs.Stat(h.fsys, path)
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

func (h *fsHandler) NewFileEntry(_ ssh.Session, path string) (*FileEntry, func() error, error) {
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
		Reader:   f,
	}, f.Close, nil
}
