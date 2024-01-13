package s3wrapper

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/matryer/is"
)

func TestBucketClient_Download(t *testing.T) {
	d := &downloaderMock{
		DownloadFunc: func(_ io.WriterAt, _ *s3.GetObjectInput, _ ...func(*s3manager.Downloader)) (int64, error) {
			return 0, nil
		},
	}
	c := &clientMock{}
	l := slog.Default()

	t.Run("successful downloading", func(t *testing.T) {
		tt := is.New(t)

		bc := NewClient(c, d, l, "expected-bucket", "expected-prefix")

		f, err := bc.Download("expected-key")
		tt.True(f != nil) // temp file with downloaded content must be created
		defer os.Remove(f.Name())
		tt.NoErr(err)
		tt.Equal(f, d.DownloadCalls()[0].WriterAt) // temp file must have been passed to aws Download method
		tt.Equal(&s3.GetObjectInput{
			Bucket: aws.String("expected-bucket"),
			Key:    aws.String("expected-key"),
		}, d.DownloadCalls()[0].GetObjectInput)
	})

	t.Run("file error", func(t *testing.T) {
		tt := is.New(t)

		bc := NewClient(c, d, l, "expected-bucket", "expected/prefix")

		f, err := bc.Download("expected-key")
		tt.True(f == nil) // temp file with downloaded content must be created
		tt.True(err != nil)
	})

	t.Run("download error", func(t *testing.T) {
		tt := is.New(t)

		expectedErr := errors.New("expected-err")
		d := &downloaderMock{
			DownloadFunc: func(_ io.WriterAt, _ *s3.GetObjectInput, _ ...func(*s3manager.Downloader)) (int64, error) {
				return 0, expectedErr
			},
		}
		bc := NewClient(c, d, l, "expected-bucket", "expected-prefix")

		f, err := bc.Download("expected-key")
		tt.True(f != nil)                    // temp file with downloaded content must be created
		tt.True(errors.Is(err, expectedErr)) // unexpected error type
	})
}

func TestBucketClient_ListFiles(t *testing.T) {
	d := &downloaderMock{}
	c := &clientMock{
		ListObjectsV2Func: func(_ *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
			return &s3.ListObjectsV2Output{Contents: []*s3.Object{
				{LastModified: aws.Time(time.Unix(0, 0)), Key: aws.String("expected-key-skipped")},
				{LastModified: aws.Time(time.Unix(100, 0)), Key: aws.String("first-expected-key")},
				{LastModified: aws.Time(time.Unix(200, 0)), Key: aws.String("second-expected-key")},
				{LastModified: aws.Time(time.Unix(200, 0)), Key: aws.String("expected-invalid-ext")},
			}}, nil
		},
	}

	l := slog.Default()

	t.Run("filters out files with wrong extension and before start time", func(t *testing.T) {
		tt := is.New(t)

		bc := NewClient(c, d, l, "expected-bucket", "expected-prefix")

		files, err := bc.ListFiles(time.Unix(99, 0), "-key")
		tt.NoErr(err)

		tt.Equal(2, len(files)) // must be only 2 files after filtration
		tt.Equal(domain.File{Key: "first-expected-key", LastModified: time.Unix(100, 0)}, files[0])
		tt.Equal(domain.File{Key: "second-expected-key", LastModified: time.Unix(200, 0)}, files[1])
	})

	t.Run("s3 error", func(t *testing.T) {
		tt := is.New(t)

		expectedErr := errors.New("expected err")
		c := &clientMock{
			ListObjectsV2Func: func(_ *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				return nil, expectedErr
			},
		}
		bc := NewClient(c, d, l, "expected-bucket", "expected-prefix")

		files, err := bc.ListFiles(time.Unix(99, 0), "-key")

		tt.Equal(0, len(files))
		tt.True(errors.Is(err, expectedErr))
	})
}
