// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package lfs

import (
	"io/fs"
	"time"
)

type LocalDirectoryEntry struct {
	de fs.DirEntry
}

func (lde *LocalDirectoryEntry) IsDir() bool {
	return lde.de.IsDir()
}

func (lde *LocalDirectoryEntry) Name() string {
	return lde.de.Name()
}

func (lde *LocalDirectoryEntry) ModTime() time.Time {
	if i, err := lde.de.Info(); err == nil {
		return i.ModTime()
	}
	return time.Time{}
}

func (lde *LocalDirectoryEntry) Size() int64 {
	if i, err := lde.de.Info(); err == nil {
		return i.Size()
	}
	return -1
}
