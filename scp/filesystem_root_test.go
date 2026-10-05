package scp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/matryer/is"
)

// NewFileSystemHandler returns the Handler interface; assert to the concrete
// type to reach prefixed (this file lives in-package).
func newFSTestHandler(tb testing.TB, root string) *fileSystemHandler {
	tb.Helper()
	h := NewFileSystemHandler(root)
	fh, ok := h.(*fileSystemHandler)
	if !ok {
		tb.Fatalf("NewFileSystemHandler returned %T, want *fileSystemHandler", h)
	}
	return fh
}

// A handler rooted at "." must still resolve relative client paths.
// Regression: prefixed() compared the cleaned join with a raw string prefix,
// and filepath.Join(".", "/a.txt") cleans to "a.txt", so "./" never matched.
func TestFileSystemHandlerRelativeRoot(t *testing.T) {
	for _, root := range []string{".", "", "." + string(filepath.Separator), "root", "root" + string(filepath.Separator)} {
		t.Run(fmt.Sprintf("root=%q", root), func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			if filepath.Clean(root) == "root" {
				dir = filepath.Join(dir, "root")
				is.New(t).NoErr(os.Mkdir(dir, 0o755))
			}
			testFileSystemRootOperations(t, root, dir)
		})
	}
}

func TestFileSystemHandlerAbsoluteRoot(t *testing.T) {
	for _, trailingSeparator := range []bool{false, true} {
		t.Run(fmt.Sprintf("trailing_separator=%v", trailingSeparator), func(t *testing.T) {
			dir := t.TempDir()
			// Relative client paths must remain relative to root, even when cwd
			// is also inside root.
			cwd := filepath.Join(dir, "cwd")
			is.New(t).NoErr(os.Mkdir(cwd, 0o755))
			t.Chdir(cwd)
			root := dir
			if trailingSeparator {
				root += string(filepath.Separator)
			}
			testFileSystemRootOperations(t, root, dir)
		})
	}
}

// A handler rooted at a volume root ("/" or `C:\`) must work too. filepath.Clean
// can't strip the trailing separator there, so h.root+Separator becomes
// "//" / `C:\\` and nothing ever matches.
func TestFileSystemHandlerVolumeRoot(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	root := filepath.VolumeName(dir) + string(filepath.Separator)
	testFileSystemRootOperations(t, root, dir)
}

func TestFileSystemHandlerUnavailableWorkingDirectory(t *testing.T) {
	// Removing cwd makes filepath.Abs fail on Linux. Other platforms may
	// still return a path or disallow removing the working directory.
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux getwd behavior after removing the working directory")
	}
	is := is.New(t)
	root := t.TempDir()
	cwd := filepath.Join(root, "cwd")
	is.NoErr(os.Mkdir(cwd, 0o755))
	t.Chdir(cwd)
	// Resolving relative paths must fail safely when cwd no longer exists.
	is.NoErr(os.Remove(cwd))
	for _, tc := range []struct {
		name, root, wantPrefix string
	}{
		{"relative root", ".", `failed to resolve root "."`},
		{"relative path", root, `failed to resolve "a.txt"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := newFSTestHandler(t, tc.root).confined("a.txt")
			if !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("confined: got %v, want a wrapped not-exist error", err)
			}
			if !strings.HasPrefix(err.Error(), tc.wantPrefix) {
				t.Fatalf("confined: got %q, want prefix %q", err, tc.wantPrefix)
			}
		})
	}
}

// All fixtures stay in dir, including when root is an entire filesystem.
func testFileSystemRootOperations(t *testing.T, root, dir string) {
	t.Helper()
	is := is.New(t)
	h := newFSTestHandler(t, root)
	rootAbs, err := filepath.Abs(root)
	is.NoErr(err)
	prefix, err := filepath.Rel(rootAbs, dir)
	is.NoErr(err)
	clientPath := func(name string) string { return filepath.Join(prefix, name) }

	is.NoErr(os.Mkdir(filepath.Join(dir, "nested"), 0o755))
	for _, name := range []string{"a.txt", "b.txt", "nested/file.txt"} {
		is.NoErr(os.WriteFile(filepath.Join(dir, name), []byte("payload"), 0o644))
	}

	t.Run("paths", func(t *testing.T) {
		for _, name := range []string{"a.txt", "nested/file.txt", "missing.txt", "missing/child.txt"} {
			p, err := h.prefixed(clientPath(name))
			is.New(t).NoErr(err)
			abs, err := filepath.Abs(p)
			is.New(t).NoErr(err)
			is.New(t).Equal(filepath.Join(dir, name), abs)
		}
		for _, name := range []string{h.root, ".", "/"} {
			p, err := h.prefixed(name)
			is.New(t).NoErr(err)
			is.New(t).Equal(h.root, p)
		}
	})

	t.Run("read", func(t *testing.T) {
		is := is.New(t)
		entry, closeFn, err := h.NewFileEntry(nil, clientPath("a.txt"))
		is.NoErr(err)
		defer func() { is.NoErr(closeFn()) }()
		data, err := io.ReadAll(entry.Reader)
		is.NoErr(err)
		is.Equal("payload", string(data))
		_, _, err = h.NewFileEntry(nil, clientPath("missing.txt"))
		is.True(errors.Is(err, fs.ErrNotExist))
		entryDir, err := h.NewDirEntry(nil, clientPath("nested"))
		is.NoErr(err)
		is.True(entryDir.Mode.IsDir())
	})

	t.Run("write_and_mkdir", func(t *testing.T) {
		is := is.New(t)
		const mtime = int64(1323853868)
		is.NoErr(h.Mkdir(nil, &DirEntry{
			Filepath: clientPath("uploads"), Mode: 0o755, Mtime: mtime, Atime: mtime,
		}))
		info, err := os.Stat(filepath.Join(dir, "uploads"))
		is.NoErr(err)
		is.True(info.IsDir())
		is.Equal(mtime, info.ModTime().Unix())
		for _, name := range []string{"uploads/new.txt", "a.txt"} {
			written, err := h.Write(nil, &FileEntry{
				Filepath: clientPath(name), Mode: 0o644, Reader: strings.NewReader("updated"),
				Mtime: mtime, Atime: mtime,
			})
			is.NoErr(err)
			is.Equal(int64(len("updated")), written)
			data, err := os.ReadFile(filepath.Join(dir, name))
			is.NoErr(err)
			is.Equal("updated", string(data))
			info, err := os.Stat(filepath.Join(dir, name))
			is.NoErr(err)
			is.Equal(mtime, info.ModTime().Unix())
		}
	})

	t.Run("glob", func(t *testing.T) {
		is := is.New(t)
		matches, err := h.Glob(nil, clientPath("*.txt"))
		is.NoErr(err)
		is.Equal([]string{clientPath("a.txt"), clientPath("b.txt")}, matches)
		matches, err = h.Glob(nil, clientPath("nested/*.txt"))
		is.NoErr(err)
		is.Equal([]string{clientPath("nested/file.txt")}, matches)
		matches, err = h.Glob(nil, clientPath("absent*.txt"))
		is.NoErr(err)
		is.Equal(0, len(matches))
		_, err = h.Glob(nil, clientPath("["))
		is.True(errors.Is(err, filepath.ErrBadPattern))
	})

	t.Run("walk", func(t *testing.T) {
		is := is.New(t)
		var paths []string
		is.NoErr(h.WalkDir(nil, clientPath("nested"), func(path string, _ fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			paths = append(paths, abs)
			return nil
		}))
		is.Equal([]string{filepath.Join(dir, "nested"), filepath.Join(dir, "nested/file.txt")}, paths)
	})
}

func TestFileSystemHandlerRootConfinement(t *testing.T) {
	for _, root := range []string{".", "", "root", "absolute"} {
		t.Run(fmt.Sprintf("root=%q", root), func(t *testing.T) {
			is := is.New(t)
			base := t.TempDir()
			dir := filepath.Join(base, "root")
			outside := filepath.Join(base, "root-other")
			is.NoErr(os.Mkdir(dir, 0o755))
			is.NoErr(os.Mkdir(outside, 0o755))
			secret := filepath.Join(outside, "secret.txt")
			is.NoErr(os.WriteFile(secret, []byte("secret"), 0o644))
			t.Chdir(base)
			if root == "." || root == "" {
				t.Chdir(dir)
			} else if root == "absolute" {
				root = dir
			}
			h := newFSTestHandler(t, root)
			// Both an existing sibling and a missing descendant must be rejected.
			is.True(h.confined(secret) != nil)
			is.True(h.confined(filepath.Join(outside, "missing/child.txt")) != nil)
			for _, name := range []string{secret, "../root-other/secret.txt"} {
				_, closeFn, err := h.NewFileEntry(nil, name)
				if closeFn != nil {
					is.NoErr(closeFn())
				}
				is.True(err != nil)
				_, err = h.Write(nil, &FileEntry{Filepath: name, Mode: 0o644, Reader: strings.NewReader("changed")})
				is.True(err != nil)
			}
			matches, err := h.Glob(nil, "../root-other/*.txt")
			is.NoErr(err)
			is.Equal(0, len(matches))
			data, err := os.ReadFile(secret)
			is.NoErr(err)
			is.Equal("secret", string(data))
			is.NoErr(h.WalkDir(nil, ".", func(path string, _ fs.DirEntry, err error) error {
				if path == h.root {
					t.Error("WalkDir must omit the handler root")
				}
				return err
			}))
		})
	}
}

func TestFileSystemHandlerRelativeRootSymlinks(t *testing.T) {
	for _, root := range []string{".", "", "root"} {
		t.Run(fmt.Sprintf("root=%q", root), func(t *testing.T) {
			is := is.New(t)
			base := t.TempDir()
			dir := filepath.Join(base, "root")
			outside := filepath.Join(base, "root-other")
			is.NoErr(os.Mkdir(dir, 0o755))
			is.NoErr(os.Mkdir(outside, 0o755))
			is.NoErr(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("inside"), 0o644))
			secret := filepath.Join(outside, "secret.txt")
			is.NoErr(os.WriteFile(secret, []byte("secret"), 0o644))
			rootTestSymlink(t, filepath.Join(dir, "a.txt"), filepath.Join(dir, "inside-link.txt"))
			rootTestSymlink(t, secret, filepath.Join(dir, "outside-link.txt"))
			rootTestSymlink(t, outside, filepath.Join(dir, "dir-link"))
			t.Chdir(base)
			if root != "root" {
				t.Chdir(dir)
			}
			h := newFSTestHandler(t, root)
			entry, closeFn, err := h.NewFileEntry(nil, "inside-link.txt")
			is.NoErr(err)
			data, err := io.ReadAll(entry.Reader)
			is.NoErr(err)
			is.NoErr(closeFn())
			is.Equal("inside", string(data))
			for _, name := range []string{"outside-link.txt", "dir-link/secret.txt", "dir-link/missing.txt"} {
				_, closeFn, err := h.NewFileEntry(nil, name)
				if closeFn != nil {
					is.NoErr(closeFn())
				}
				is.True(err != nil)
				_, err = h.Write(nil, &FileEntry{Filepath: name, Mode: 0o644, Reader: strings.NewReader("changed")})
				is.True(err != nil)
			}
			is.True(h.Mkdir(nil, &DirEntry{Filepath: "dir-link/new-dir", Mode: 0o755}) != nil)
			_, err = h.Glob(nil, "dir-link/*.txt")
			is.True(err != nil)
			data, err = os.ReadFile(secret)
			is.NoErr(err)
			is.Equal("secret", string(data))
			_, err = os.Stat(filepath.Join(outside, "missing.txt"))
			is.True(errors.Is(err, fs.ErrNotExist))
			_, err = os.Stat(filepath.Join(outside, "new-dir"))
			is.True(errors.Is(err, fs.ErrNotExist))
		})
	}
}

func TestFileSystemHandlerSymlinkRoot(t *testing.T) {
	is := is.New(t)
	base := t.TempDir()
	dir := filepath.Join(base, "real")
	is.NoErr(os.Mkdir(dir, 0o755))
	is.NoErr(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("inside"), 0o644))
	rootTestSymlink(t, dir, filepath.Join(base, "root-link"))
	t.Chdir(base)
	h := newFSTestHandler(t, "root-link")
	entry, closeFn, err := h.NewFileEntry(nil, "a.txt")
	is.NoErr(err)
	defer func() { is.NoErr(closeFn()) }()
	data, err := io.ReadAll(entry.Reader)
	is.NoErr(err)
	is.Equal("inside", string(data))
	matches, err := h.Glob(nil, "*.txt")
	is.NoErr(err)
	is.Equal([]string{"a.txt"}, matches)
}

func rootTestSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(1314)) {
			t.Skip("creating symlinks requires Windows Developer Mode or symlink privileges")
		}
		t.Fatal(err)
	}
}

func TestFileSystemHandlerRootEndToEnd(t *testing.T) {
	for _, root := range []string{".", "", "volume"} {
		t.Run(fmt.Sprintf("root=%q", root), func(t *testing.T) {
			is := is.New(t)
			dir := filepath.Join(t.TempDir(), "root with spaces")
			is.NoErr(os.Mkdir(dir, 0o755))
			t.Chdir(dir)
			if root == "volume" {
				root = filepath.VolumeName(dir) + string(filepath.Separator)
			}
			h := newFSTestHandler(t, root)
			abs, err := filepath.Abs(root)
			is.NoErr(err)
			prefix, err := filepath.Rel(abs, dir)
			is.NoErr(err)
			clientPath := func(name string) string { return filepath.ToSlash(filepath.Join(prefix, name)) }
			is.NoErr(os.WriteFile("a file.txt", []byte("file-payload"), 0o644))
			is.NoErr(os.Mkdir("nested dir", 0o755))
			is.NoErr(os.WriteFile(filepath.Join("nested dir", "b.txt"), []byte("nested-payload"), 0o644))
			out, err := setup(t, h, nil).CombinedOutput(fmt.Sprintf("scp -f %q", clientPath("a file.txt")))
			is.NoErr(err)
			is.True(bytes.Contains(out, []byte("file-payload")))
			out, err = setup(t, h, nil).CombinedOutput(fmt.Sprintf("scp -r -f %q", clientPath("nested dir")))
			is.NoErr(err)
			is.True(bytes.Contains(out, []byte("nested-payload")))
			var in bytes.Buffer
			in.WriteString("C0644 7 upload.txt\n")
			in.WriteString("updated")
			in.Write(NULL)
			is.NoErr(os.Mkdir("upload dir", 0o755))
			session := setup(t, nil, h)
			session.Stdin = &in
			_, err = session.CombinedOutput(fmt.Sprintf("scp -t %q", clientPath("upload dir")))
			is.NoErr(err)
			data, err := os.ReadFile(filepath.Join("upload dir", "upload.txt"))
			is.NoErr(err)
			is.Equal("updated", string(data))
		})
	}
}
