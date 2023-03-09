// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3FileSystem struct {
	bucket             string
	prefix             string
	s3                 *s3.Client
	bucketCreationDate time.Time
}

type S3DirectoryEntry struct {
	name string
	dir  bool
}

func (de *S3DirectoryEntry) IsDir() bool {
	return de.dir
}

func (de *S3DirectoryEntry) Name() string {
	return de.name
}

func (fs *S3FileSystem) key(name string) string {
	if len(fs.prefix) == 0 {
		if strings.HasPrefix(name, "/") {
			return name[1:]
		}
		return name
	}
	return fs.Join(fs.prefix, name)
}

func (fs *S3FileSystem) HeadObject(ctx context.Context, name string) (*FileInfo, error) {
	headObjectOutput, err := fs.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(fs.key(name)),
	})
	if err != nil {
		return nil, err
	}
	fi := NewFileInfo(
		name,
		aws.ToTime(headObjectOutput.LastModified),
		headObjectOutput.ContentLength == int64(0),
		headObjectOutput.ContentLength,
	)
	return fi, nil
}

func (fs *S3FileSystem) IsNotExist(err error) bool {
	var responseError *http.ResponseError
	if errors.As(err, &responseError) {
		if responseError.HTTPStatusCode() == 404 {
			return true
		}
	}
	return false
}

func (fs *S3FileSystem) Join(name ...string) string {
	return path.Join(name...)
}

func (fs *S3FileSystem) ReadDir(ctx context.Context, name string) ([]DirectoryEntry, error) {
	listObjectsV2Input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(fs.bucket),
		Delimiter: aws.String("/"),
		MaxKeys:   25,
	}
	if name != "/" {
		listObjectsV2Input.Prefix = aws.String(fs.key(name) + "/")
	} else {
		listObjectsV2Input.Prefix = aws.String("")
	}
	listObjectsV2Output, err := fs.s3.ListObjectsV2(ctx, listObjectsV2Input)
	if err != nil {
		return nil, err
	}
	directoryEntries := []DirectoryEntry{}
	for _, commonPrefix := range listObjectsV2Output.CommonPrefixes {
		directoryEntries = append(directoryEntries, &S3DirectoryEntry{name: aws.ToString(commonPrefix.Prefix), dir: true})
	}
	for _, object := range listObjectsV2Output.Contents {
		directoryEntries = append(directoryEntries, &S3DirectoryEntry{name: aws.ToString(object.Key), dir: (object.Size == 0)})
	}
	return directoryEntries, nil
}

func (fs *S3FileSystem) Stat(ctx context.Context, name string) (*FileInfo, error) {
	if name == "/" {
		_, err := fs.s3.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(fs.bucket),
		})
		if err != nil {
			return nil, err
		}
		fi := NewFileInfo(
			name,
			fs.bucketCreationDate,
			true,
			int64(0),
		)
		return fi, nil
	}

	directoryEntries, err := fs.ReadDir(ctx, name)
	if len(directoryEntries) > 0 {
		fi := NewFileInfo(
			name,
			fs.bucketCreationDate,
			true,
			int64(0),
		)
		return fi, nil
	}

	fi, err := fs.HeadObject(ctx, name)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func (fs *S3FileSystem) Open(ctx context.Context, name string) (io.ReadSeeker, error) {
	fi, err := fs.Stat(ctx, name)
	if err != nil {
		return nil, err
	}
	rs := NewReadSeeker(
		0,
		fi.Size(),
		func(offset int64, p []byte) (int, error) {
			getObjectOutput, err := fs.s3.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(fs.bucket),
				Key:    aws.String(fs.key(name)),
				Range:  aws.String(fmt.Sprintf("bytes=%d-%d", offset, int(offset)+len(p)-1)),
			})
			if err != nil {
				return 0, err
			}
			body, err := io.ReadAll(getObjectOutput.Body)
			if err != nil {
				return 0, err
			}
			copy(p, body)
			return len(p), nil
		},
	)
	return rs, nil
}

func NewS3FileSystem(bucket string, prefix string, s3 *s3.Client, bucketCreationDate time.Time) *S3FileSystem {
	return &S3FileSystem{
		bucket:             bucket,
		prefix:             prefix,
		s3:                 s3,
		bucketCreationDate: bucketCreationDate,
	}
}
