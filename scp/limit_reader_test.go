package scp

import (
	"bytes"
	"io"
	"testing"

	"github.com/matryer/is"
)

func TestLimitedReader(t *testing.T) {
	t.Run("partial", func(t *testing.T) {
		is := is.New(t)
		var b bytes.Buffer
		b.WriteString("writing some bytes")
		r := newLimitReader(&b, 7)

		bts, err := io.ReadAll(r)
		is.NoErr(err)
		is.Equal("writing", string(bts))
	})

	t.Run("full", func(t *testing.T) {
		is := is.New(t)
		var b bytes.Buffer
		b.WriteString("some text")
		r := newLimitReader(&b, b.Len())

		bts, err := io.ReadAll(r)
		is.NoErr(err)
		is.Equal("some text", string(bts))
	})

	t.Run("pass limit", func(t *testing.T) {
		is := is.New(t)
		var b bytes.Buffer
		b.WriteString("another text")
		r := newLimitReader(&b, b.Len()+10)

		bts, err := io.ReadAll(r)
		is.NoErr(err)
		is.Equal("another text", string(bts))
	})
}
