// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package fs

import (
	"context"
	"io"
)

type FileSystem interface {
	IsNotExist(err error) bool
	Join(name ...string) string
	ReadDir(ctx context.Context, name string) ([]DirectoryEntry, error)
	Size(ctx context.Context, name string) (int64, error)
	Stat(ctx context.Context, name string) (FileInfo, error)
	Open(ctx context.Context, name string) (io.ReadSeeker, error)
}
