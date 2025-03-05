package scp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/ssh"
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
		path  = info.Path
		r     = bufio.NewReader(s)
		mtime int64
		atime int64
	)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read line: %w", err)
		}
		line = strings.TrimSuffix(line, "\n")

		if matches := reTimestamp.FindAllStringSubmatch(line, 2); matches != nil {
			mtime, err = strconv.ParseInt(matches[0][1], 10, 64)
			if err != nil {
				return parseError{line}
			}
			atime, err = strconv.ParseInt(matches[0][2], 10, 64)
			if err != nil {
				return parseError{line}
			}

			// accepts the header
			_, _ = s.Write(NULL)
			continue
		}

		if matches := reNewFile.FindAllStringSubmatch(line, 3); matches != nil {
			if len(matches) != 1 || len(matches[0]) != 4 {
				return parseError{line}
			}

			mode, err := strconv.ParseUint(matches[0][1], 8, 32)
			if err != nil {
				return parseError{line}
			}

			size, err := strconv.ParseInt(matches[0][2], 10, 64)
			if err != nil {
				return parseError{line}
			}
			name := matches[0][3]

			// accepts the header
			_, _ = s.Write(NULL)

			written, err := handler.Write(s, &FileEntry{
				Name:     name,
				Filepath: filepath.Join(path, name),
				Mode:     fs.FileMode(mode), //nolint:gosec
				Size:     size,
				Mtime:    mtime,
				Atime:    atime,
				Reader:   newLimitReader(r, int(size)),
			})
			if err != nil {
				return fmt.Errorf("failed to write file: %q: %w", name, err)
			}
			if written != size {
				return fmt.Errorf("failed to write the file: %q: written %d out of %d bytes", name, written, size)
			}

			// read the trailing nil char
			_, _ = r.ReadByte()

			mtime = 0
			atime = 0
			// says 'hey im done'
			_, _ = s.Write(NULL)
			continue
		}

		if matches := reNewFolder.FindAllStringSubmatch(line, 2); matches != nil {
			if len(matches) != 1 || len(matches[0]) != 3 {
				return parseError{line}
			}

			mode, err := strconv.ParseUint(matches[0][1], 8, 32)
			if err != nil {
				return parseError{line}
			}
			name := matches[0][2]

			path = filepath.Join(path, name)
			if err := handler.Mkdir(s, &DirEntry{
				Name:     name,
				Filepath: path,
				Mode:     fs.FileMode(mode), //nolint:gosec
				Mtime:    mtime,
				Atime:    atime,
			}); err != nil {
				return fmt.Errorf("failed to create dir: %q: %w", name, err)
			}

			mtime = 0
			atime = 0
			// says 'hey im done'
			_, _ = s.Write(NULL)
			continue
		}

		if line == "E" {
			path = filepath.Dir(path)

			// says 'hey im done'
			_, _ = s.Write(NULL)
			continue
		}

		return fmt.Errorf("unhandled input: %q", line)
	}

	_, _ = s.Write(NULL)
	return nil
}
