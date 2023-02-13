// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package fs

import (
	"io"
	"path/filepath"

	"github.com/spf13/afero"
)

type LocalFileSystem struct {
	fs afero.Fs
}

func (fs *LocalFileSystem) Join(name ...string) string {
	return filepath.Join(name...)
}

func (fs *LocalFileSystem) Stat(name string) (*FileInfo, error) {
	fi, err := fs.fs.Stat(name)
	if err != nil {
		return nil, err
	}
	return NewFileInfo(fi.Name(), fi.ModTime(), fi.IsDir(), fi.Size()), nil
}

func (fs *LocalFileSystem) Open(name string) (io.ReadSeeker, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func NewLocalFileSystem(rootPath string) *LocalFileSystem {
	return &LocalFileSystem{
		fs: afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()), rootPath),
	}
}
