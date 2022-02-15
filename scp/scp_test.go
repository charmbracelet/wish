package scp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/wish/testsession"
	"github.com/gliderlabs/ssh"
	"github.com/matryer/is"
	gossh "golang.org/x/crypto/ssh"
)

func TestGetRootEntry(t *testing.T) {
	path := t.TempDir()
	handler := NewFileSystemHandler(path)

	t.Run("/", func(t *testing.T) {
		is := is.New(t)
		entry, err := getRootEntry(nil, handler, "/")
		is.NoErr(err)
		_, ok := entry.(*NoDirRootEntry)
		is.True(ok)
	})

	t.Run(".", func(t *testing.T) {
		is := is.New(t)
		entry, err := getRootEntry(nil, handler, ".")
		is.NoErr(err)
		_, ok := entry.(*NoDirRootEntry)
		is.True(ok)
	})

	t.Run("unknown folder", func(t *testing.T) {
		is := is.New(t)
		_, err := getRootEntry(nil, handler, "nope")
		is.True(err != nil)
	})

	t.Run("folder", func(t *testing.T) {
		is := is.New(t)
		os.Mkdir(filepath.Join(path, "folder"), 0755)

		entry, err := getRootEntry(nil, handler, "folder")
		is.NoErr(err)
		_, ok := entry.(*DirEntry)
		is.True(ok)
	})
}

func TestGetInfo(t *testing.T) {
	t.Run("no exec", func(t *testing.T) {
		is := is.New(t)
		info := GetInfo([]string{})
		is.Equal(info.Ok, false)
	})

	t.Run("exec is not scp", func(t *testing.T) {
		is := is.New(t)
		info := GetInfo([]string{"not-scp"})
		is.Equal(info.Ok, false)
	})

	t.Run("scp no recursive", func(t *testing.T) {
		is := is.New(t)
		info := GetInfo([]string{"scp", "-f", "file"})
		is.True(info.Ok)
		is.Equal(info.Recursive, false)
		is.Equal("file", info.Path)
		is.Equal(info.Op, OpCopyToClient)
	})

	t.Run("scp recursive", func(t *testing.T) {
		is := is.New(t)
		info := GetInfo([]string{"scp", "-r", "--some-ignored-flag", "-f", "file", "ignored-arg"})
		is.True(info.Ok)
		is.True(info.Recursive)
		is.Equal("file", info.Path)
		is.Equal(info.Op, OpCopyToClient)
	})

	t.Run("scp op copy from client", func(t *testing.T) {
		is := is.New(t)
		info := GetInfo([]string{"scp", "-t", "file"})
		is.True(info.Ok)
		is.Equal(info.Op, OpCopyFromClient)
		is.Equal("file", info.Path)
	})
}

func TestNoDirRootEntry(t *testing.T) {
	is := is.New(t)
	root := NoDirRootEntry{}

	var f1 bytes.Buffer
	f1.WriteString("hello from file f1\n")

	var f2 bytes.Buffer
	f2.WriteString("hello from file f2\nwith multiple lines :)\n")

	dir := &DirEntry{
		Children: []Entry{},
		Name:     "dir1",
		Filepath: "dir1",
		Mode:     0755,
	}

	dir.Append(&FileEntry{
		Name:     "f2",
		Filepath: "f2",
		Mode:     0600,
		Size:     int64(f2.Len()),
		Reader:   &f2,
	})

	root.Append(&FileEntry{
		Name:     "f1",
		Filepath: "f1",
		Mode:     0644,
		Size:     int64(f1.Len()),
		Reader:   &f1,
	})

	root.Append(dir)

	var out bytes.Buffer
	is.NoErr(root.Write(&out))

	requireEqualGolden(t, out.Bytes())
}

func setup(tb testing.TB, rh CopyToClientHandler, wh CopyFromClientHandler) *gossh.Session {
	tb.Helper()
	return testsession.New(tb, &ssh.Server{
		Handler: Middleware(rh, wh)(func(s ssh.Session) {
			s.Exit(0)
			s.Close()
		}),
	}, nil)
}

func requireEqualGolden(tb testing.TB, out []byte) {
	tb.Helper()
	is := is.New(tb)

	golden := "testdata/" + tb.Name() + ".test"
	if os.Getenv("UPDATE") != "" {
		is.NoErr(os.MkdirAll(filepath.Dir(golden), 0o755))
		is.NoErr(os.WriteFile(golden, out, 0o655))
	}

	gbts, err := os.ReadFile(golden)
	is.NoErr(err)
	is.Equal(string(gbts), string(out))
}
