package scp

import (
	"fmt"
	"io/fs"

	"github.com/charmbracelet/ssh"
)

func copyToClient(s ssh.Session, info Info, handler CopyToClientHandler) error {
	matches, err := handler.Glob(s, info.Path)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("no files matching %q", info.Path)
	}

	rootEntry := &RootEntry{}
	var closers []func() error
	defer func() {
		closeAll(closers)
	}()

	for _, match := range matches {
		if !info.Recursive {
			entry, closer, err := handler.NewFileEntry(s, match)
			closers = append(closers, closer)
			if err != nil {
				return err
			}
			rootEntry.Append(entry)
			continue
		}

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
