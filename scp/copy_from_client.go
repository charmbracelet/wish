package scp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/gliderlabs/ssh"
)

var (
	reTimestamp = regexp.MustCompile(`^T(\d{10}) 0 (\d{10}) 0$`)
	reNewFolder = regexp.MustCompile(`^D(\d{4}) 0 (.*)$`)
	reNewFile   = regexp.MustCompile(`^C(\d{4}) (\d+) (.*)$`)
)

type parseError struct {
	subject string
}

func (e parseError) Error() string {
	return fmt.Sprintf("failed to parse: %q", e.subject)
}

func copyFromClient(s ssh.Session, info Info, handler CopyFromClientHandler) error {
	// accepts the request
	_, _ = s.Write(NULL)

	var (
		path = info.Path
		r    = bufio.NewReader(s)
	)

	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read line: %w", err)
		}

		if matches := reTimestamp.FindAllStringSubmatch(string(line), 2); matches != nil {
			// ignore for now
			// accepts the header
			_, _ = s.Write(NULL)
			continue
		}

		if matches := reNewFile.FindAllStringSubmatch(string(line), 3); matches != nil {
			if len(matches) != 1 || len(matches[0]) != 4 {
				return parseError{string(line)}
			}

			mode, err := strconv.ParseUint(matches[0][1], 8, 32)
			if err != nil {
				return parseError{string(line)}
			}

			size, err := strconv.ParseInt(matches[0][2], 10, 64)
			if err != nil {
				return parseError{string(line)}
			}
			name := matches[0][3]

			// accepts the header
			_, _ = s.Write(NULL)

			written, err := handler.Write(s, &FileEntry{
				Name:     name,
				Filepath: filepath.Join(path, name),
				Mode:     fs.FileMode(mode),
				Size:     size,
				Reader:   newLimitReader(r, int(size)),
			})
			if err != nil {
				return fmt.Errorf("failed to write file: %q: %w", name, err)
			}
			if written != size {
				return fmt.Errorf("failed to write the file: %q: written %d out of %d bytes", name, written, size)
			}

			// read the trailing nil char
			_, _ = r.ReadByte() // TODO: check if it is indeed a NULL?

			// says 'hey im done'
			_, _ = s.Write(NULL)
			continue
		}

		if matches := reNewFolder.FindAllStringSubmatch(string(line), 2); matches != nil {
			if len(matches) != 1 || len(matches[0]) != 3 {
				return parseError{string(line)}
			}

			mode, err := strconv.ParseUint(matches[0][1], 8, 32)
			if err != nil {
				return parseError{string(line)}
			}
			name := matches[0][2]

			path = filepath.Join(path, name)
			if err := handler.Mkdir(s, &DirEntry{
				Name:     name,
				Filepath: path,
				Mode:     fs.FileMode(mode),
			}); err != nil {
				return fmt.Errorf("failed to create dir: %q: %w", name, err)
			}

			// says 'hey im done'
			_, _ = s.Write(NULL)
			continue
		}

		if string(line) == "E" {
			path = filepath.Dir(path)

			// says 'hey im done'
			_, _ = s.Write(NULL)
			continue
		}

		if bytes.Equal(line, NULL) {
			log.Println("dangling NULL byte ignored")
			continue
		}

		return fmt.Errorf("unhandled input: %q", string(line))
	}

	_, _ = s.Write(NULL)
	return nil
}
