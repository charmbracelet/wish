package scp

import (
	"io"
	"sync"
)

func newLimitReader(r io.Reader, limit int64) io.Reader {
	return &limitReader{
		r:    r,
		left: limit,
	}
}

type limitReader struct {
	r io.Reader

	lock sync.Mutex
	left int64
}

func (r *limitReader) Read(b []byte) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.left <= 0 {
		return 0, io.EOF
	}
	if int64(len(b)) > r.left {
		b = b[0:r.left]
	}
	n, err := r.r.Read(b)
	r.left -= int64(n)
	return n, err
}
