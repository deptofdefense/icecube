// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package fs

import (
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3FileSystem struct {
	bucket             string
	prefix             string
	s3                 *s3.S3
	bucketCreationDate time.Time
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

func (fs *S3FileSystem) Join(name ...string) string {
	return path.Join(name...)
}

func (fs *S3FileSystem) Stat(name string) (*FileInfo, error) {
	if name == "/" {
		_, err := fs.s3.HeadBucket(&s3.HeadBucketInput{
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
	headObjectOutput, err := fs.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(fs.key(name)),
	})
	if err != nil {
		return nil, err
	}
	fi := NewFileInfo(
		name,
		aws.TimeValue(headObjectOutput.LastModified),
		aws.Int64Value(headObjectOutput.ContentLength) == int64(0),
		aws.Int64Value(headObjectOutput.ContentLength),
	)
	return fi, nil
}

func (fs *S3FileSystem) Open(name string) (io.ReadSeeker, error) {
	fi, err := fs.Stat(name)
	if err != nil {
		return nil, err
	}
	rs := NewReadSeeker(
		0,
		fi.Size(),
		func(offset int64, p []byte) (int, error) {
			getObjectOutput, err := fs.s3.GetObject(&s3.GetObjectInput{
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

func NewS3FileSystem(bucket string, prefix string, s3 *s3.S3, bucketCreationDate time.Time) *S3FileSystem {
	return &S3FileSystem{
		bucket:             bucket,
		prefix:             prefix,
		s3:                 s3,
		bucketCreationDate: bucketCreationDate,
	}
}
