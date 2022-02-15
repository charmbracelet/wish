package scp

import (
	"bytes"
	"io"
	"testing"
)

func TestLimitedReader(t *testing.T) {
	t.Run("partial", func(t *testing.T) {
		var b bytes.Buffer
		b.WriteString("writing some bytes")
		r := newLimitReader(&b, 7)

		bts, err := io.ReadAll(r)
		requireNoError(t, err)
		requireEqual(t, "writing", string(bts))
	})

	t.Run("full", func(t *testing.T) {
		var b bytes.Buffer
		b.WriteString("some text")
		r := newLimitReader(&b, int64(b.Len()))

		bts, err := io.ReadAll(r)
		requireNoError(t, err)
		requireEqual(t, "some text", string(bts))
	})

	t.Run("pass limit", func(t *testing.T) {
		var b bytes.Buffer
		b.WriteString("another text")
		r := newLimitReader(&b, int64(b.Len()+10))

		bts, err := io.ReadAll(r)
		requireNoError(t, err)
		requireEqual(t, "another text", string(bts))
	})
}
