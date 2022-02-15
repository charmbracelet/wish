package scp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

func TestFS(t *testing.T) {
	t.Run("copy to client", func(t *testing.T) {
		is := is.New(t)

		dir := t.TempDir()
		h := NewFSReadHandler(os.DirFS(dir))
		is.NoErr(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a text file"), 0644))

		session := setup(t, h, nil)
		bts, err := session.CombinedOutput("scp -f a.txt")
		is.NoErr(err)
		requireEqualGolden(t, bts)
	})

	t.Run("recursive copy to client", func(t *testing.T) {
		is := is.New(t)

		dir := t.TempDir()
		h := NewFSReadHandler(os.DirFS(dir))
		is.NoErr(os.MkdirAll(filepath.Join(dir, "a/b/c/d/e"), 0755))
		is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c.txt"), []byte("c text file"), 0644))
		is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c/d/e/e.txt"), []byte("e text file"), 0644))

		session := setup(t, h, nil)
		bts, err := session.CombinedOutput("scp -r -f a")
		is.NoErr(err)
		requireEqualGolden(t, bts)
	})
}
