package scp

import (
	"io/fs"
	"log"

	"github.com/gliderlabs/ssh"
)

func copyFilesToClient(s ssh.Session, handler CopyToClientHandler, paths []string) error {
	for _, path := range paths {
		entry, closer, err := handler.NewFileEntry(s, path)
		if err != nil {
			return err
		}
		defer func() {
			if closer != nil {
				_ = closer()
			}
		}()
		if err := entry.Write(s); err != nil {
			return err
		}
	}
	return nil
}

func copyToClient(s ssh.Session, info Info, handler CopyToClientHandler) error {
	paths, err := handler.Glob(s, info.Path)
	if err != nil {
		return err
	}

	if !info.Recursive {
		return copyFilesToClient(s, handler, paths)
	}

	rootEntry, err := getRootEntry(s, handler, info.Path)
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

	if err := handler.WalkDir(s, info.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == info.Path {
			return nil
		}

		if d.IsDir() {
			entry, err := handler.NewDirEntry(s, path)
			if err != nil {
				return err
			}
			rootEntry.Append(entry)
		} else {
			entry, closer, err := handler.NewFileEntry(s, path)
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

	return rootEntry.Write(s)
}

func getRootEntry(s ssh.Session, handler CopyToClientHandler, root string) (RootEntry, error) {
	if root == "/" || root == "." {
		return &NoDirRootEntry{}, nil
	}

	return handler.NewDirEntry(s, root)
}
