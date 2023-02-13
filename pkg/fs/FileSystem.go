// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package fs

import (
	"io"
)

type FileSystem interface {
	Join(name ...string) string
	Stat(name string) (*FileInfo, error)
	Open(name string) (io.ReadSeeker, error)
}
