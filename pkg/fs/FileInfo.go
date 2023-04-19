// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package fs

import (
	"time"
)

type FileInfo interface {
	IsDir() bool
	Name() string
	ModTime() time.Time
	Size() int64
}
