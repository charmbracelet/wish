package scp

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/matryer/is"
)

func TestFilesystem(t *testing.T) {
	mtime := time.Unix(1323853868, 0)
	atime := time.Unix(1380425711, 0)

	t.Run("scp -f", func(t *testing.T) {
		t.Run("file", func(t *testing.T) {
			is := is.New(t)

			dir := t.TempDir()
			h := NewFileSystemHandler(dir)
			is.NoErr(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a text file"), 0o644))
			chtimesTree(t, dir, atime, mtime)

			session := setup(t, h, nil)
			bts, err := session.CombinedOutput("scp -f a.txt")
			is.NoErr(err)
			requireEqualGolden(t, bts)
		})

		t.Run("glob", func(t *testing.T) {
			is := is.New(t)

			dir := t.TempDir()
			h := NewFileSystemHandler(dir)
			is.NoErr(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a text file"), 0o644))
			is.NoErr(os.WriteFile(filepath.Join(dir, "b.txt"), []byte("another text file"), 0o644))
			chtimesTree(t, dir, atime, mtime)

			session := setup(t, h, nil)
			bts, err := session.CombinedOutput("scp -f *.txt")
			is.NoErr(err)
			requireEqualGolden(t, bts)
		})

		t.Run("invalid file", func(t *testing.T) {
			is := is.New(t)

			dir := t.TempDir()
			h := NewFileSystemHandler(dir)

			session := setup(t, h, nil)
			_, err := session.CombinedOutput("scp -f a.txt")
			is.True(err != nil)
		})

		t.Run("recursive", func(t *testing.T) {
			is := is.New(t)

			dir := t.TempDir()
			h := NewFileSystemHandler(dir)

			is.NoErr(os.MkdirAll(filepath.Join(dir, "a/b/c/d/e"), 0o755))
			is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c.txt"), []byte("c text file"), 0o644))
			is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c/d/e/e.txt"), []byte("e text file"), 0o644))
			chtimesTree(t, dir, atime, mtime)

			session := setup(t, h, nil)
			bts, err := session.CombinedOutput("scp -r -f a")
			is.NoErr(err)
			requireEqualGolden(t, bts)
		})

		t.Run("recursive glob", func(t *testing.T) {
			is := is.New(t)

			dir := t.TempDir()
			h := NewFileSystemHandler(dir)

			is.NoErr(os.MkdirAll(filepath.Join(dir, "a/b/c/d/e"), 0o755))
			is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c.txt"), []byte("c text file"), 0o644))
			is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c/d/e/e.txt"), []byte("e text file"), 0o644))
			chtimesTree(t, dir, atime, mtime)

			session := setup(t, h, nil)
			bts, err := session.CombinedOutput("scp -r -f a/*")
			is.NoErr(err)
			requireEqualGolden(t, bts)
		})

		t.Run("recursive invalid file", func(t *testing.T) {
			is := is.New(t)

			dir := t.TempDir()
			h := NewFileSystemHandler(dir)

			session := setup(t, h, nil)
			_, err := session.CombinedOutput("scp -r -f a")
			is.True(err != nil)
		})

		t.Run("recursive folder", func(t *testing.T) {
			is := is.New(t)

			dir := t.TempDir()
			h := NewFileSystemHandler(dir)

			is.NoErr(os.MkdirAll(filepath.Join(dir, "a/b/c/d/e"), 0o755))
			is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c.txt"), []byte("c text file"), 0o644))
			is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c/d/e/e.txt"), []byte("e text file"), 0o644))
			chtimesTree(t, dir, atime, mtime)

			session := setup(t, h, nil)
			bts, err := session.CombinedOutput("scp -r -f /")
			is.NoErr(err)
			requireEqualGolden(t, bts)
		})
	})

	t.Run("scp -t", func(t *testing.T) {
		t.Run("file", func(t *testing.T) {
			is := is.New(t)
			dir := t.TempDir()
			h := NewFileSystemHandler(dir)
			session := setup(t, nil, h)

			var in bytes.Buffer
			in.WriteString("T1183832947 0 1183833773 0\n")
			in.WriteString("C0644 6 a.txt\n")
			in.WriteString("hello\n")
			in.Write(NULL)
			session.Stdin = &in

			_, err := session.CombinedOutput("scp -t .")
			is.NoErr(err)

			bts, err := os.ReadFile(filepath.Join(dir, "a.txt"))
			is.NoErr(err)
			is.Equal("hello\n", string(bts))
		})

		t.Run("recursive", func(t *testing.T) {
			is := is.New(t)
			dir := t.TempDir()
			h := NewFileSystemHandler(dir)

			var in bytes.Buffer
			in.WriteString("T1183832947 0 1183833773 0\n")
			in.WriteString("D0755 0 folder1\n")
			in.WriteString("C0644 6 file1\n")
			in.WriteString("hello\n")
			in.Write(NULL)
			in.WriteString("D0755 0 folder2\n")
			in.WriteString("T1183832947 0 1183833773 0\n")
			in.WriteString("C0644 6 file2\n")
			in.WriteString("hello\n")
			in.Write(NULL)
			in.WriteString("E\n")
			in.WriteString("E\n")

			session := setup(t, nil, h)
			session.Stdin = &in
			_, err := session.CombinedOutput("scp -r -t .")
			is.NoErr(err)

			mtime := int64(1183832947)

			stat, err := os.Stat(filepath.Join(dir, "folder1"))
			is.NoErr(err)
			is.True(stat.IsDir())
			// TODO: check how scp behaves
			is.True(stat.ModTime().Unix() != mtime) // should be different because the folder was later modified again

			stat, err = os.Stat(filepath.Join(dir, "folder1/file1"))
			is.NoErr(err)
			is.True(stat.ModTime().Unix() != mtime)

			stat, err = os.Stat(filepath.Join(dir, "folder1/folder2/file2"))
			is.NoErr(err)
			is.Equal(stat.ModTime().Unix(), mtime)
		})
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("chtimes", func(t *testing.T) {
			h := &fileSystemHandler{t.TempDir()}
			is.New(t).True(h.chtimes("nope", 1212212, 323232) != nil) // should err
		})

		t.Run("glob", func(t *testing.T) {
			t.Run("invalid glob", func(t *testing.T) {
				is := is.New(t)
				h := &fileSystemHandler{t.TempDir()}
				matches, err := h.Glob(nil, "[asda")
				is.True(err != nil) // should err
				is.Equal(0, len(matches))
			})
		})

		t.Run("NewDirEntry", func(t *testing.T) {
			t.Run("do not exist", func(t *testing.T) {
				is := is.New(t)
				h := &fileSystemHandler{t.TempDir()}
				_, err := h.NewDirEntry(nil, "foo")
				is.True(err != nil) // should err
			})
		})

		t.Run("NewFileEntry", func(t *testing.T) {
			t.Run("do not exist", func(t *testing.T) {
				is := is.New(t)
				h := &fileSystemHandler{t.TempDir()}
				_, _, err := h.NewFileEntry(nil, "foo")
				is.True(err != nil) // should err
			})
		})

		t.Run("Mkdir", func(t *testing.T) {
			t.Run("parent do not exist", func(t *testing.T) {
				is := is.New(t)
				h := &fileSystemHandler{t.TempDir()}
				err := h.Mkdir(nil, &DirEntry{
					Name:     "foo",
					Filepath: "foo/bar/baz",
					Mode:     0o755,
				})
				is.True(err != nil) // should err
			})
		})

		t.Run("Write", func(t *testing.T) {
			t.Run("parent do not exist", func(t *testing.T) {
				is := is.New(t)
				h := &fileSystemHandler{t.TempDir()}
				_, err := h.Write(nil, &FileEntry{
					Name:     "foo.txt",
					Filepath: "baz/foo.txt",
					Mode:     0o644,
					Size:     10,
				})
				is.True(err != nil) // should err
			})

			t.Run("reader fails", func(t *testing.T) {
				is := is.New(t)
				h := &fileSystemHandler{t.TempDir()}
				_, err := h.Write(nil, &FileEntry{
					Name:     "foo.txt",
					Filepath: "foo.txt",
					Mode:     0o644,
					Size:     10,
					Reader:   iotest.ErrReader(fmt.Errorf("fake err")),
				})
				is.True(err != nil) // should err
			})
		})
	})
}

func TestPrefixedPathTraversal(t *testing.T) {
	root := t.TempDir()
	h := &fileSystemHandler{root: filepath.Clean(root)}

	cases := []struct {
		name       string
		path       string
		want       string
		wantInRoot bool
	}{
		{"relative traversal", "../../../etc/passwd", filepath.Join(root, "etc/passwd"), false},
		{"nested relative traversal", "foo/../../etc/passwd", filepath.Join(root, "etc/passwd"), false},
		{"deep relative traversal", "../../../tmp/pwned", filepath.Join(root, "tmp/pwned"), false},
		{"absolute path outside root", "/etc/passwd", filepath.Join(root, "etc/passwd"), false},
		{"root-prefixed traversal", root + "/../../../etc/shadow", "", true},
		{"valid relative path", "subdir/file.txt", filepath.Join(root, "subdir/file.txt"), false},
		{"valid path under root", filepath.Join(root, "file.txt"), filepath.Join(root, "file.txt"), false},
		{"root slash", "/", root, false},
		{"dot", ".", root, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			p, err := h.prefixed(tc.path)
			is.NoErr(err)
			if tc.wantInRoot {
				is.True(p == root || strings.HasPrefix(p, root+string(filepath.Separator)))
			} else {
				is.Equal(tc.want, p)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	cases := []struct {
		name    string
		invalid bool
	}{
		{"normal.txt", false},
		{"file-with-dashes", false},
		{".hidden", false},
		{"..", true},
		{".", true},
		{"", true},
		{"../etc/passwd", true},
		{"sub/dir", true},
		{"back\\slash", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateName(tc.name)
			if tc.invalid {
				is.New(t).True(err != nil)
			} else {
				is.New(t).NoErr(err)
			}
		})
	}
}

func TestPathTraversalEndToEnd(t *testing.T) {
	t.Run("write file", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		secret := t.TempDir()
		h := NewFileSystemHandler(root)

		var in bytes.Buffer
		in.WriteString("C0644 5 ../" + filepath.Base(secret) + "/pwned\n")
		in.WriteString("hello")
		in.Write(NULL)

		session := setup(t, nil, h)
		session.Stdin = &in
		_, err := session.CombinedOutput("scp -t .")
		is.True(err != nil)
		_, statErr := os.Stat(filepath.Join(secret, "pwned"))
		is.True(os.IsNotExist(statErr))
	})

	t.Run("write dir", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		h := NewFileSystemHandler(root)

		var in bytes.Buffer
		in.WriteString("D0755 0 ../../evil_dir\n")
		in.WriteString("C0644 7 payload\n")
		in.WriteString("payload")
		in.Write(NULL)
		in.WriteString("E\n")

		session := setup(t, nil, h)
		session.Stdin = &in
		_, err := session.CombinedOutput("scp -r -t .")
		is.True(err != nil)
		_, statErr := os.Stat(filepath.Join(root, "../../evil_dir"))
		is.True(os.IsNotExist(statErr))
	})

	t.Run("read file", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		secret := t.TempDir()
		is.NoErr(os.WriteFile(filepath.Join(secret, "secret.txt"), []byte("super-secret"), 0o644))
		h := NewFileSystemHandler(root)

		session := setup(t, h, nil)
		_, err := session.CombinedOutput("scp -f ../" + filepath.Base(secret) + "/secret.txt")
		is.True(err != nil)
	})

	t.Run("read glob", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		secret := t.TempDir()
		is.NoErr(os.WriteFile(filepath.Join(secret, "secret.txt"), []byte("super-secret"), 0o644))
		h := NewFileSystemHandler(root)

		session := setup(t, h, nil)
		_, err := session.CombinedOutput("scp -f ../" + filepath.Base(secret) + "/secret*")
		is.True(err != nil)
	})

	t.Run("filename with slash", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		h := NewFileSystemHandler(root)

		var in bytes.Buffer
		in.WriteString("C0644 5 sub/file\n")
		in.WriteString("hello")
		in.Write(NULL)

		session := setup(t, nil, h)
		session.Stdin = &in
		_, err := session.CombinedOutput("scp -t .")
		is.True(err != nil)
	})

	t.Run("filename with dotdot", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		h := NewFileSystemHandler(root)

		var in bytes.Buffer
		in.WriteString("C0644 5 ..\n")
		in.WriteString("hello")
		in.Write(NULL)

		session := setup(t, nil, h)
		session.Stdin = &in
		_, err := session.CombinedOutput("scp -t .")
		is.True(err != nil)
	})
}

// TestSymlinkEscapeEndToEnd covers confinement against symlinks rather than
// against "..". The path checks were lexical, and os.Open and os.OpenFile both
// follow symlinks, so a link already sitting inside root led out of it in both
// directions. Something other than SCP has to place that link: an operator
// layout, an unpacked archive, or another part of the host application writing
// into the same directory.
func TestSymlinkEscapeEndToEnd(t *testing.T) {
	t.Run("read through symlink", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		outside := t.TempDir()

		secret := filepath.Join(outside, "secret.txt")
		is.NoErr(os.WriteFile(secret, []byte("super-secret"), 0o600))
		is.NoErr(os.Symlink(secret, filepath.Join(root, "link.txt")))

		session := setup(t, NewFileSystemHandler(root), nil)
		out, err := session.CombinedOutput("scp -f link.txt")
		is.True(err != nil)
		is.True(!bytes.Contains(out, []byte("super-secret")))
	})

	t.Run("overwrite through symlink", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		outside := t.TempDir()

		target := filepath.Join(outside, "target.txt")
		is.NoErr(os.WriteFile(target, []byte("original"), 0o600))
		is.NoErr(os.Symlink(target, filepath.Join(root, "link.txt")))

		var in bytes.Buffer
		in.WriteString("C0644 5 link.txt\n")
		in.WriteString("pwned")
		in.Write(NULL)

		session := setup(t, nil, NewFileSystemHandler(root))
		session.Stdin = &in
		_, err := session.CombinedOutput("scp -t .")
		is.True(err != nil)

		// The out-of-root file must be untouched, not merely unwritten: the
		// open uses O_TRUNC, so reaching it at all destroys the contents.
		got, err := os.ReadFile(target)
		is.NoErr(err)
		is.Equal(string(got), "original")
	})

	t.Run("create under symlinked dir", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()
		outside := t.TempDir()

		is.NoErr(os.Symlink(outside, filepath.Join(root, "dirlink")))

		var in bytes.Buffer
		in.WriteString("C0644 5 newfile\n")
		in.WriteString("pwned")
		in.Write(NULL)

		session := setup(t, nil, NewFileSystemHandler(root))
		session.Stdin = &in
		_, err := session.CombinedOutput("scp -t dirlink")
		is.True(err != nil)

		_, statErr := os.Stat(filepath.Join(outside, "newfile"))
		is.True(os.IsNotExist(statErr))
	})

	t.Run("symlink inside root still works", func(t *testing.T) {
		is := is.New(t)
		root := t.TempDir()

		target := filepath.Join(root, "real.txt")
		is.NoErr(os.WriteFile(target, []byte("in-root"), 0o600))
		is.NoErr(os.Symlink(target, filepath.Join(root, "link.txt")))

		session := setup(t, NewFileSystemHandler(root), nil)
		out, err := session.CombinedOutput("scp -f link.txt")
		is.NoErr(err)
		is.True(bytes.Contains(out, []byte("in-root")))
	})
}

func chtimesTree(tb testing.TB, dir string, atime, mtime time.Time) {
	is.New(tb).NoErr(filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chtimes(path, atime, mtime)
	}))
}
