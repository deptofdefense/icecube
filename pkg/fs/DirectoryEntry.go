// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package fs

import "time"

type DirectoryEntry interface {
	Name() string
	IsDir() bool
	ModTime() time.Time
}
