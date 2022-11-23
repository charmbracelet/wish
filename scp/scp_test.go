package scp

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/testsession"
	"github.com/google/go-cmp/cmp"
	"github.com/matryer/is"
	gossh "golang.org/x/crypto/ssh"
)

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
	root := RootEntry{}

	var f1 bytes.Buffer
	f1.WriteString("hello from file f1\n")

	var f2 bytes.Buffer
	f2.WriteString("hello from file f2\nwith multiple lines :)\n")

	dir := &DirEntry{
		Children: []Entry{},
		Name:     "dir1",
		Filepath: "dir1",
		Mode:     0o755,
	}

	dir.Append(&FileEntry{
		Name:     "f2",
		Filepath: "f2",
		Mode:     0o600,
		Size:     int64(f2.Len()),
		Reader:   &f2,
	})

	root.Append(&FileEntry{
		Name:     "f1",
		Filepath: "f1",
		Mode:     0o644,
		Size:     int64(f1.Len()),
		Reader:   &f1,
	})

	root.Append(dir)

	var out bytes.Buffer
	is.NoErr(root.Write(&out))

	requireEqualGolden(t, out.Bytes())
}

func TestInvalidOps(t *testing.T) {
	t.Run("not scp", func(t *testing.T) {
		_, err := setup(t, nil, nil).CombinedOutput("not-scp ign")
		is.New(t).NoErr(err)
	})

	t.Run("copy to client", func(t *testing.T) {
		_, err := setup(t, nil, nil).CombinedOutput("scp -t .")
		is.New(t).True(err != nil)
	})

	t.Run("copy from client", func(t *testing.T) {
		_, err := setup(t, nil, nil).CombinedOutput("scp -f .")
		is.New(t).True(err != nil)
	})
}

func setup(tb testing.TB, rh CopyToClientHandler, wh CopyFromClientHandler) *gossh.Session {
	tb.Helper()
	return testsession.New(tb, &ssh.Server{
		Handler: Middleware(rh, wh)(func(s ssh.Session) {
			s.Exit(0)
		}),
	}, nil)
}

func requireEqualGolden(tb testing.TB, out []byte) {
	tb.Helper()
	is := is.New(tb)

	fixOutput := func(bts []byte) []byte {
		bts = bytes.ReplaceAll(bts, []byte("\r"), []byte(""))
		if runtime.GOOS == "windows" {
			// perms always come different on Windows because, well, its Windows.
			bts = bytes.ReplaceAll(bts, []byte("0666"), []byte("0644"))
			bts = bytes.ReplaceAll(bts, []byte("0777"), []byte("0755"))
		}
		return bytes.ReplaceAll(bts, NULL, []byte("<NULL>"))
	}

	out = fixOutput(out)
	golden := "testdata/" + tb.Name() + ".test"
	if os.Getenv("UPDATE") != "" {
		is.NoErr(os.MkdirAll(filepath.Dir(golden), 0o755))
		is.NoErr(os.WriteFile(golden, out, 0o655))
	}

	gbts, err := os.ReadFile(golden)
	is.NoErr(err)
	gbts = fixOutput(gbts)

	if diff := cmp.Diff(string(gbts), string(out)); diff != "" {
		tb.Fatal("files do not match:", diff)
	}
}
