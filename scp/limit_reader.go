package scp

import (
	"io"
	"sync"
)

func newLimitReader(r io.Reader, limit int) io.Reader {
	return &limitReader{
		r:    r,
		left: limit,
	}
}

type limitReader struct {
	r io.Reader

	lock sync.Mutex
	left int
}

func (r *limitReader) Read(b []byte) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.left <= 0 {
		return 0, io.EOF
	}
	if len(b) > r.left {
		b = b[0:r.left]
	}
	n, err := r.r.Read(b)
	r.left -= n
	return n, err
}
