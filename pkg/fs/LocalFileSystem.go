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
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

type LocalFileSystem struct {
	fs   afero.Fs
	iofs afero.IOFS
}

type LocalDirectoryEntry struct {
	de fs.DirEntry
}

func (de *LocalDirectoryEntry) IsDir() bool {
	return de.de.IsDir()
}

func (de *LocalDirectoryEntry) Name() string {
	return de.de.Name()
}

func (de *LocalDirectoryEntry) ModTime() time.Time {
	if i, err := de.de.Info(); err == nil {
		return i.ModTime()
	}
	return time.Time{}
}

func (de *LocalDirectoryEntry) Size() int64 {
	if i, err := de.de.Info(); err == nil {
		return i.Size()
	}
	return -1
}

func (fs *LocalFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (fs *LocalFileSystem) Join(name ...string) string {
	return filepath.Join(name...)
}

func (fs *LocalFileSystem) ReadDir(ctx context.Context, name string) ([]DirectoryEntry, error) {
	directoryEntries := []DirectoryEntry{}
	readDirOutput, err := fs.iofs.ReadDir(name)
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

func (fs *LocalFileSystem) Stat(ctx context.Context, name string) (*FileInfo, error) {
	fi, err := fs.fs.Stat(name)
	if err != nil {
		return nil, err
	}
	return NewFileInfo(fi.Name(), fi.ModTime(), fi.IsDir(), fi.Size()), nil
}

func (fs *LocalFileSystem) Open(ctx context.Context, name string) (io.ReadSeeker, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func NewLocalFileSystem(rootPath string) *LocalFileSystem {
	fs := afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()), rootPath)
	return &LocalFileSystem{
		fs:   fs,
		iofs: afero.NewIOFS(fs),
	}
}
