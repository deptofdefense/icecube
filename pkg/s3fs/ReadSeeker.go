// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package s3fs

import (
	"io"
)

type ReadSeeker struct {
	read   func(offset int64, p []byte) (n int, err error)
	offset int64
	size   int64
}

func (rs *ReadSeeker) Read(p []byte) (int, error) {
	if rs.offset >= rs.size {
		return 0, io.EOF
	}
	n, err := rs.read(rs.offset, p)
	if n > 0 {
		rs.offset += int64(n)
	}
	return n, err
}

func (rs *ReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		rs.offset = offset
	case io.SeekCurrent:
		rs.offset = rs.offset + offset
	case io.SeekEnd:
		rs.offset = rs.size + offset
	default:
		return 0, io.ErrUnexpectedEOF
	}
	return rs.offset, nil
}

func NewReadSeeker(offset int64, size int64, read func(offset int64, p []byte) (int, error)) *ReadSeeker {
	return &ReadSeeker{read: read, offset: offset, size: size}
}
