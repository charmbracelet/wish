package scp

import (
	"io/fs"
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

		path := filepath.Join(dir, "a.txt")
		is.NoErr(os.WriteFile(path, []byte("a text file"), 0o644))
		is.NoErr(os.Chtimes(path, atime, mtime))

		session := setup(t, h, nil)
		bts, err := session.CombinedOutput("scp -f a.txt")
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

		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			is.NoErr(os.Chtimes(path, atime, mtime))
			return nil
		})

		session := setup(t, h, nil)
		bts, err := session.CombinedOutput("scp -r -f a")
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
}
