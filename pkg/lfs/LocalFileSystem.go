// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package lfs

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/deptofdefense/icecube/pkg/fs"
)

type LocalFileSystem struct {
	fs   afero.Fs
	iofs afero.IOFS
}

func (lfs *LocalFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (lfs *LocalFileSystem) Join(name ...string) string {
	return filepath.Join(name...)
}

func (lfs *LocalFileSystem) ReadDir(ctx context.Context, name string) ([]fs.DirectoryEntry, error) {
	directoryEntries := []fs.DirectoryEntry{}
	readDirOutput, err := lfs.iofs.ReadDir(name)
	if err != nil {
		return nil, err
	}
	for _, directoryEntry := range readDirOutput {
		directoryEntries = append(directoryEntries, &LocalDirectoryEntry{
			de: directoryEntry,
		})
	}
	return directoryEntries, nil
}

func (lfs *LocalFileSystem) Size(ctx context.Context, name string) (int64, error) {
	fi, err := lfs.fs.Stat(name)
	if err != nil {
		return int64(0), err
	}
	return fi.Size(), nil
}

func (lfs *LocalFileSystem) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	fi, err := lfs.fs.Stat(name)
	if err != nil {
		return nil, err
	}
	return NewLocalFileInfo(fi.Name(), fi.ModTime(), fi.IsDir(), fi.Size()), nil
}

func (lfs *LocalFileSystem) Open(ctx context.Context, name string) (io.ReadSeeker, error) {
	f, err := lfs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func NewLocalFileSystem(rootPath string) *LocalFileSystem {
	lfs := afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()), rootPath)
	return &LocalFileSystem{
		fs:   lfs,
		iofs: afero.NewIOFS(lfs),
	}
}
