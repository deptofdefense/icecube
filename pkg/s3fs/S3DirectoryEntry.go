// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package s3fs

import (
	"time"
)

type S3DirectoryEntry struct {
	name    string
	dir     bool
	modTime time.Time
	size    int64
}

func (de *S3DirectoryEntry) IsDir() bool {
	return de.dir
}

func (de *S3DirectoryEntry) Name() string {
	return de.name
}

func (de *S3DirectoryEntry) ModTime() time.Time {
	return de.modTime
}

func (de *S3DirectoryEntry) Size() int64 {
	return de.size
}
