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

type FileInfo struct {
	name string
	size int64
	//mode fs.FileMode
	modTime time.Time
	dir     bool
	//sys any
}

func (fi *FileInfo) IsDir() bool {
	return fi.dir
}

func (fi *FileInfo) Name() string {
	return fi.name
}

func (fi *FileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *FileInfo) Size() int64 {
	return fi.size
}

func NewFileInfo(name string, modTime time.Time, dir bool, size int64) *FileInfo {
	return &FileInfo{
		name:    name,
		modTime: modTime,
		dir:     dir,
		size:    size,
	}
}
