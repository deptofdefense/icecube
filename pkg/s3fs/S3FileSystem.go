// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package s3fs

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

	"github.com/deptofdefense/icecube/pkg/fs"
)

type S3FileSystem struct {
	defaultRegion        string
	bucket               string
	prefix               string
	clients              map[string]*s3.Client
	bucketRegions        map[string]string
	bucketCreationDates  map[string]time.Time
	earliestCreationDate time.Time
	maxEntries           int
}

// GetRegion returns the region for the bucket.
// If the bucket is not known, then returns the default region
func (s3fs *S3FileSystem) GetBucketRegion(bucket string) string {
	if bucketRegion, ok := s3fs.bucketRegions[bucket]; ok {
		return bucketRegion
	}
	return s3fs.defaultRegion
}

// parse returns the bucket and key for the given name
func (fs *S3FileSystem) parse(name string) (string, string) {
	// if not bucket is defined
	if len(fs.bucket) == 0 {
		if len(fs.prefix) != 0 {
			panic(fmt.Errorf("invalid configuration with bucket %q and prefix %q", fs.bucket, fs.prefix))
		}
		nameParts := strings.Split(strings.TrimPrefix(name, "/"), "/")
		return nameParts[0], fs.Join(nameParts[1:]...)
	}
	// If prefix is defined, then append the name
	if len(fs.prefix) > 0 {
		return fs.bucket, fs.Join(fs.prefix, name)
	}
	// If no prefix is defined, then return the name as a key
	return fs.bucket, strings.TrimPrefix(name, "/")
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

func (s3fs *S3FileSystem) HeadObject(ctx context.Context, name string) (*S3FileInfo, error) {
	bucket, key := s3fs.parse(name)
	headObjectOutput, err := s3fs.clients[s3fs.GetBucketRegion(bucket)].HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	fi := NewS3FileInfo(
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

func (s3fs *S3FileSystem) ReadDir(ctx context.Context, name string) ([]fs.DirectoryEntry, error) {
	bucket := ""
	delimiter := aws.String("/")
	prefix := ""
	if len(s3fs.bucket) == 0 {
		if len(name) > 0 {
			nameParts := strings.Split(strings.TrimPrefix(name, "/"), "/")
			bucket = nameParts[0]
			if len(nameParts) > 1 {
				if k := s3fs.Join(nameParts[1:]...); k != "/" {
					prefix = k + "/"
				}
			}
		}
	} else {
		bucket = s3fs.bucket
		if name != "/" || len(s3fs.prefix) > 0 {
			prefix = s3fs.key(name) + "/"
		}
	}
	//
	directoryEntries := []fs.DirectoryEntry{}
	// If listing s3 buckets in account
	if len(bucket) == 0 {
		listBucketsOutput, listBucketsError := s3fs.clients[s3fs.defaultRegion].ListBuckets(ctx, &s3.ListBucketsInput{})
		if listBucketsError != nil {
			return nil, fmt.Errorf("error listing buckets in account: %w", listBucketsError)
		}
		for _, b := range listBucketsOutput.Buckets {
			directoryEntries = append(directoryEntries, &S3DirectoryEntry{
				name:    aws.ToString(b.Name),
				dir:     true,
				modTime: aws.ToTime(b.CreationDate),
				size:    0,
			})
			if len(directoryEntries) == s3fs.maxEntries {
				break
			}
		}
		return directoryEntries, nil
	}
	// truncated, continuationToken, and startAfter are used to iterate through the bucket
	//truncated := true
	var marker *string
	// if truncated continue iterating through the bucket
	for i := 0; i < 20; i++ {
		listObjectsInput := &s3.ListObjectsInput{
			Bucket:    aws.String(bucket),
			Delimiter: delimiter,
		}
		if s3fs.maxEntries != -1 && s3fs.maxEntries < 1000 {
			listObjectsInput.MaxKeys = int32(s3fs.maxEntries)
		}
		listObjectsInput.Prefix = aws.String(prefix)
		if marker != nil {
			listObjectsInput.Marker = marker
		}
		listObjectsOutput, err := s3fs.clients[s3fs.GetBucketRegion(bucket)].ListObjects(ctx, listObjectsInput)
		if err != nil {
			return nil, err
		}
		if s3fs.maxEntries != -1 {
			// limit on number of directory entries
			for _, commonPrefix := range listObjectsOutput.CommonPrefixes {
				directoryName := ""
				if len(s3fs.bucket) == 0 {
					directoryName = "/" + s3fs.Join(bucket, aws.ToString(commonPrefix.Prefix))
				} else {
					directoryName = "/" + strings.TrimPrefix(aws.ToString(commonPrefix.Prefix), s3fs.prefix)
				}
				directoryEntries = append(directoryEntries, &S3DirectoryEntry{
					name:    directoryName,
					dir:     true,
					modTime: s3fs.bucketCreationDates[s3fs.bucket],
					size:    0,
				})
				if len(directoryEntries) == s3fs.maxEntries {
					break
				}
			}
			if len(directoryEntries) == s3fs.maxEntries {
				break
			}
			for _, object := range listObjectsOutput.Contents {
				fileName := ""
				if len(s3fs.bucket) == 0 {
					fileName = "/" + s3fs.Join(bucket, aws.ToString(object.Key))
				} else {
					fileName = "/" + strings.TrimPrefix(aws.ToString(object.Key), s3fs.prefix)
				}
				directoryEntries = append(directoryEntries, &S3DirectoryEntry{
					name:    fileName,
					dir:     (object.Size == 0),
					modTime: aws.ToTime(object.LastModified),
					size:    object.Size,
				})
				if len(directoryEntries) == s3fs.maxEntries {
					break
				}
			}
			if len(directoryEntries) == s3fs.maxEntries {
				break
			}
		} else {
			// no limit for number of directory entries
			for _, commonPrefix := range listObjectsOutput.CommonPrefixes {
				directoryName := ""
				if len(s3fs.bucket) == 0 {
					directoryName = "/" + s3fs.Join(bucket, aws.ToString(commonPrefix.Prefix))
				} else {
					directoryName = "/" + strings.TrimPrefix(aws.ToString(commonPrefix.Prefix), s3fs.prefix)
				}
				directoryEntries = append(directoryEntries, &S3DirectoryEntry{
					name:    directoryName,
					dir:     true,
					modTime: s3fs.bucketCreationDates[s3fs.bucket],
					size:    0,
				})
			}
			for _, object := range listObjectsOutput.Contents {
				fileName := ""
				if len(s3fs.bucket) == 0 {
					fileName = "/" + s3fs.Join(bucket, aws.ToString(object.Key))
				} else {
					fileName = "/" + strings.TrimPrefix(aws.ToString(object.Key), s3fs.prefix)
				}
				directoryEntries = append(directoryEntries, &S3DirectoryEntry{
					name:    fileName,
					dir:     (object.Size == 0),
					modTime: aws.ToTime(object.LastModified),
					size:    object.Size,
				})
			}
		}
		if !listObjectsOutput.IsTruncated {
			break
		}
		marker = listObjectsOutput.NextMarker
	}

	return directoryEntries, nil
}

func (s3fs *S3FileSystem) Size(ctx context.Context, name string) (int64, error) {
	bucket, key := s3fs.parse(name)
	headObjectOutput, err := s3fs.clients[s3fs.GetBucketRegion(bucket)].HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return int64(0), err
	}
	return headObjectOutput.ContentLength, nil
}

func (s3fs *S3FileSystem) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	if len(s3fs.bucket) == 0 {
		if len(s3fs.prefix) != 0 {
			return nil, fmt.Errorf("invalid configuration with bucket %q and prefix %q", s3fs.bucket, s3fs.prefix)
		}
		if name == "/" {
			fi := NewS3FileInfo(
				name,
				s3fs.earliestCreationDate,
				true,
				int64(0),
			)
			return fi, nil
		}
		nameParts := strings.Split(strings.TrimPrefix(name, "/"), "/")
		fmt.Println(fmt.Sprintf("Name parts: %#v", nameParts))
		if len(nameParts) == 1 {
			bucket := nameParts[0]
			_, err := s3fs.clients[s3fs.GetBucketRegion(bucket)].HeadBucket(ctx, &s3.HeadBucketInput{
				Bucket: aws.String(bucket),
			})
			if err != nil {
				return nil, err
			}
			fi := NewS3FileInfo(
				name,
				s3fs.bucketCreationDates[nameParts[0]],
				true,
				int64(0),
			)
			return fi, nil
		}
		directoryEntries, readDirError := s3fs.ReadDir(ctx, name)
		if readDirError != nil {
			return nil, fmt.Errorf("error reading directory %q: %w", name, readDirError)
		}
		if len(directoryEntries) > 0 {
			fi := NewS3FileInfo(
				name,
				s3fs.bucketCreationDates[nameParts[0]], // set creation date to the creation date of the bucket
				true,
				int64(0),
			)
			return fi, nil
		}
		// no directory entires, so this must be a file
		fi, err := s3fs.HeadObject(ctx, name)
		if err != nil {
			return nil, err
		}
		return fi, nil
	} else {
		// if bucket is defined
		if name == "/" && len(s3fs.prefix) == 0 {
			_, err := s3fs.clients[s3fs.GetBucketRegion(s3fs.bucket)].HeadBucket(ctx, &s3.HeadBucketInput{
				Bucket: aws.String(s3fs.bucket),
			})
			if err != nil {
				return nil, err
			}
			fi := NewS3FileInfo(
				name,
				s3fs.bucketCreationDates[s3fs.bucket],
				true,
				int64(0),
			)
			return fi, nil
		}
	}

	directoryEntries, readDirError := s3fs.ReadDir(ctx, name)
	if readDirError != nil {
		return nil, fmt.Errorf("error reading directory %q: %w", name, readDirError)
	}
	if len(directoryEntries) > 0 {
		fi := NewS3FileInfo(
			name,
			s3fs.bucketCreationDates[s3fs.bucket],
			true,
			int64(0),
		)
		return fi, nil
	}

	fi, err := s3fs.HeadObject(ctx, name)
	if err != nil {
		return nil, err
	}

	return fi, nil
}

func (s3fs *S3FileSystem) Open(ctx context.Context, name string) (io.ReadSeeker, error) {
	size, sizeError := s3fs.Size(ctx, name)
	if sizeError != nil {
		return nil, sizeError
	}
	bucket, key := s3fs.parse(name)
	rs := NewReadSeeker(
		0,
		size,
		func(offset int64, p []byte) (int, error) {
			getObjectOutput, err := s3fs.clients[s3fs.GetBucketRegion(bucket)].GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
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

func NewS3FileSystem(
	defaultRegion string,
	bucket string,
	prefix string,
	clients map[string]*s3.Client,
	bucketRegions map[string]string,
	bucketCreationDates map[string]time.Time,
	maxEntries int) *S3FileSystem {
	// calculate earliest creation date
	earliestCreationDate := time.Time{}
	for _, t := range bucketCreationDates {
		if t.Before(earliestCreationDate) {
			earliestCreationDate = t
		}
	}
	// return new file system
	return &S3FileSystem{
		bucket:               bucket,
		prefix:               prefix,
		clients:              clients,
		defaultRegion:        defaultRegion,
		bucketRegions:        bucketRegions,
		bucketCreationDates:  bucketCreationDates,
		earliestCreationDate: earliestCreationDate,
		maxEntries:           maxEntries,
	}
}
