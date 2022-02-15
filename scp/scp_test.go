package scp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

var update = os.Getenv("UPDATE") != ""

func TestGetRootEntry(t *testing.T) {
	path := t.TempDir()
	handler := NewFileSystemHandler(path)

	t.Run("/", func(t *testing.T) {
		entry, err := getRootEntry(nil, handler, "/")
		requireNoError(t, err)
		_, ok := entry.(*NoDirRootEntry)
		requireEqual(t, true, ok)
	})

	t.Run(".", func(t *testing.T) {
		entry, err := getRootEntry(nil, handler, ".")
		requireNoError(t, err)
		_, ok := entry.(*NoDirRootEntry)
		requireEqual(t, true, ok)
	})

	t.Run("unknown folder", func(t *testing.T) {
		_, err := getRootEntry(nil, handler, "nope")
		requireError(t, err)
	})

	t.Run("folder", func(t *testing.T) {
		os.Mkdir(filepath.Join(path, "folder"), 0755)

		entry, err := getRootEntry(nil, handler, "folder")
		requireNoError(t, err)
		_, ok := entry.(*DirEntry)
		requireEqual(t, true, ok)
	})
}

func TestGetInfo(t *testing.T) {
	t.Run("no exec", func(t *testing.T) {
		info := GetInfo([]string{})
		if info.Ok {
			t.Fatal("not a scp")
		}
	})

	t.Run("exec is not scp", func(t *testing.T) {
		info := GetInfo([]string{"not-scp"})
		if info.Ok {
			t.Fatal("not a scp")
		}
	})

	t.Run("scp no recursive", func(t *testing.T) {
		info := GetInfo([]string{"scp", "-f", "file"})
		if !info.Ok {
			t.Fatal("is a scp")
		}
		if info.Recursive {
			t.Fatal("is not recursive")
		}
		if info.Path != "file" {
			t.Fatalf("path should have been 'file', was '%s'", info.Path)
		}
	})

	t.Run("scp recursive", func(t *testing.T) {
		info := GetInfo([]string{"scp", "-r", "--some-ignored-flag", "-f", "file", "ignored-arg"})
		if !info.Ok {
			t.Fatal("is a scp")
		}
		if !info.Recursive {
			t.Fatal("is recursive")
		}
		if info.Path != "file" {
			t.Fatalf("path should have been 'file', was '%s'", info.Path)
		}
	})
}

func TestNoDirRootEntry(t *testing.T) {
	root := NoDirRootEntry{}

	var f1m int64 = 1257894000
	var f1a int64 = 1257894400

	var f1 bytes.Buffer
	f1.WriteString("hello from file f1\n")

	var f2 bytes.Buffer
	f2.WriteString("hello from file f2\nwith multiple lines :)\n")

	dir := &DirEntry{
		Children: []Entry{},
		Name:     "dir1",
		Filepath: "dir1",
		Mode:     0755,
		Mtime:    f1m,
		Atime:    f1m,
	}

	dir.Append(&FileEntry{
		Name:     "f2",
		Filepath: "f2",
		Mode:     0600,
		Mtime:    f1a,
		Atime:    f1a,
		Size:     int64(f2.Len()),
		Reader:   &f2,
	})

	root.Append(&FileEntry{
		Name:     "f1",
		Filepath: "f1",
		Mode:     0644,
		Mtime:    f1m,
		Atime:    f1a,
		Size:     int64(f1.Len()),
		Reader:   &f1,
	})

	root.Append(dir)

	var out bytes.Buffer
	if err := root.Write(&out); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join("./testdata", t.Name()+".test")
	if update {
		if err := os.WriteFile(path, out.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}
	bts, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(bts, out.Bytes()) {
		t.Fatalf("output does not match for %q: \n%s \nvs\n%s\n", path, string(bts), out.String())
	}
}

func requireEqual(tb testing.TB, a, b interface{}) {
	tb.Helper()
	if a != b {
		tb.Errorf("expected %v, got %v", a, b)
	}
}

func requireNoError(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Errorf("expected no error, got %v", err)
	}
}

func requireError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
}
