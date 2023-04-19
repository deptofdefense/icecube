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

type S3FileInfo struct {
	name string
	size int64
	//mode fs.FileMode
	modTime time.Time
	dir     bool
	//sys any
}

func (fi *S3FileInfo) IsDir() bool {
	return fi.dir
}

func (fi *S3FileInfo) Name() string {
	return fi.name
}

func (fi *S3FileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *S3FileInfo) Size() int64 {
	return fi.size
}

func NewS3FileInfo(name string, modTime time.Time, dir bool, size int64) *S3FileInfo {
	return &S3FileInfo{
		name:    name,
		modTime: modTime,
		dir:     dir,
		size:    size,
	}
}
