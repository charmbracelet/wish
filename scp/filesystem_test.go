package scp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

func TestFilesystem(t *testing.T) {
	t.Run("copy to client", func(t *testing.T) {
		is := is.New(t)

		dir := t.TempDir()
		h := NewFileSystemHandler(dir)
		is.NoErr(os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a text file"), 0644))

		session := setup(t, h, nil)
		bts, err := session.CombinedOutput("scp -f a.txt")
		is.NoErr(err)
		requireEqualGolden(t, bts)
	})

	t.Run("copy from client", func(t *testing.T) {
		t.SkipNow()
		is := is.New(t)
		dir := t.TempDir()
		h := NewFileSystemHandler(dir)
		session := setup(t, nil, h)

		var in bytes.Buffer
		session.Stdin = &in
		done := make(chan bool, 1)
		go func() {
			_, err := session.CombinedOutput("scp -t a.txt")
			is.NoErr(err)
			done <- true
		}()

		in.WriteString("C06444 6 a.txt\n")
		in.WriteString("hello\n")
		in.Write(NULL)

		<-done

		bts, err := os.ReadFile(filepath.Join(dir, "a.txt"))
		is.NoErr(err)
		is.Equal("hello\n", string(bts))
	})

	t.Run("recursive copy to client", func(t *testing.T) {
		is := is.New(t)

		dir := t.TempDir()
		h := NewFileSystemHandler(dir)
		is.NoErr(os.MkdirAll(filepath.Join(dir, "a/b/c/d/e"), 0755))
		is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c.txt"), []byte("c text file"), 0644))
		is.NoErr(os.WriteFile(filepath.Join(dir, "a/b/c/d/e/e.txt"), []byte("e text file"), 0644))

		session := setup(t, h, nil)
		bts, err := session.CombinedOutput("scp -r -f a")
		is.NoErr(err)
		requireEqualGolden(t, bts)
	})

	t.Run("recursive copy from client", func(t *testing.T) {
		t.SkipNow()
		is := is.New(t)
		dir := t.TempDir()
		h := NewFileSystemHandler(dir)

		session := setup(t, nil, h)
		var in bytes.Buffer
		session.Stdin = &in
		done := make(chan bool, 1)
		go func() {
			_, err := session.CombinedOutput("scp -r -t a")
			is.NoErr(err)
			done <- true
		}()

		in.WriteString("D0755 0 folder1\n")
		in.WriteString("C0644 6 file1\n")
		in.WriteString("hello\n")
		in.Write(NULL)
		in.WriteString("D0755 0 folder2\n")
		in.WriteString("C0644 6 file2\n")
		in.WriteString("hello\n")
		in.Write(NULL)
		in.WriteString("E\n")
		in.WriteString("E\n")
		in.Write(NULL)

		<-done

		stat, err := os.Stat(filepath.Join(dir, "folder1"))
		is.NoErr(err)
		is.True(stat.IsDir())
	})
}
