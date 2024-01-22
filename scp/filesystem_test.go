package scp

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
				is.Equal([]string{}, matches)
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

func chtimesTree(tb testing.TB, dir string, atime, mtime time.Time) {
	is.New(tb).NoErr(filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chtimes(path, atime, mtime)
	}))
}
