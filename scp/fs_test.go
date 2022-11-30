package scp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestFS(t *testing.T) {
	mtime := time.Unix(1323853868, 0)
	atime := time.Unix(1380425711, 0)

	t.Run("file", func(t *testing.T) {
		is := is.New(t)

		dir := t.TempDir()
		h := NewFSReadHandler(os.DirFS(dir))

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
		h := NewFSReadHandler(os.DirFS(dir))
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
		h := NewFSReadHandler(os.DirFS(dir))

		session := setup(t, h, nil)
		_, err := session.CombinedOutput("scp -f a.txt")
		is.True(err != nil)
	})

	t.Run("recursive", func(t *testing.T) {
		is := is.New(t)

		dir := t.TempDir()
		h := NewFSReadHandler(os.DirFS(dir))

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
		h := NewFSReadHandler(os.DirFS(dir))

		is.NoErr(os.MkdirAll(filepath.Join(dir, "a/b/c/d/e"), 0o755))
		is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c.txt"), []byte("c text file"), 0o644))
		is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c/d/e/e.txt"), []byte("e text file"), 0o644))
		chtimesTree(t, dir, atime, mtime)

		session := setup(t, h, nil)
		bts, err := session.CombinedOutput("scp -r -f a/*")
		is.NoErr(err)
		requireEqualGolden(t, bts)
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

	t.Run("recursive invalid file", func(t *testing.T) {
		is := is.New(t)

		dir := t.TempDir()
		h := NewFSReadHandler(os.DirFS(dir))

		session := setup(t, h, nil)
		_, err := session.CombinedOutput("scp -r -f a")
		is.True(err != nil)
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("glob", func(t *testing.T) {
			t.Run("invalid glob", func(t *testing.T) {
				is := is.New(t)
				h := &fsHandler{os.DirFS(t.TempDir())}
				matches, err := h.Glob(nil, "[asda")
				is.True(err != nil) // should err
				is.Equal(nil, matches)
			})
		})

		t.Run("NewDirEntry", func(t *testing.T) {
			t.Run("do not exist", func(t *testing.T) {
				is := is.New(t)
				h := &fsHandler{os.DirFS(t.TempDir())}
				_, err := h.NewDirEntry(nil, "foo")
				is.True(err != nil) // should err
			})
		})

		t.Run("NewFileEntry", func(t *testing.T) {
			t.Run("do not exist", func(t *testing.T) {
				is := is.New(t)
				h := &fsHandler{os.DirFS(t.TempDir())}
				_, _, err := h.NewFileEntry(nil, "foo")
				is.True(err != nil) // should err
			})
		})
	})
}
