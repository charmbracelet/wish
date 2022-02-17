package scp

import (
	"fmt"
	"io/fs"

	"github.com/gliderlabs/ssh"
)

func copyFilesToClient(s ssh.Session, handler CopyToClientHandler, paths []string) error {
	entries := &RootEntry{}
	var closers []func() error
	defer closeAll(closers)

	for _, path := range paths {
		entry, closer, err := handler.NewFileEntry(s, path)
		closers = append(closers, closer)
		if err != nil {
			return err
		}
		entries.Append(entry)
	}
	return entries.Write(s)
}

func copyToClient(s ssh.Session, info Info, handler CopyToClientHandler) error {
	paths, err := handler.Glob(s, info.Path)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no files matching %q", info.Path)
	}

	if !info.Recursive {
		return copyFilesToClient(s, handler, paths)
	}

	rootEntry := &RootEntry{}
	var closers []func() error
	defer closeAll(closers)

	for _, match := range paths {
		if err := handler.WalkDir(s, match, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
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
	}
	return rootEntry.Write(s)
}

func closeAll(closers []func() error) {
	for _, closer := range closers {
		if closer != nil {
			_ = closer()
		}
	}
}
