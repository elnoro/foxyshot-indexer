package s3wrapper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/elnoro/foxyshot-indexer/internal/domain"
)

const tempFilePrefix = "foxyshot_indexer_"

//go:generate moq -out client_moq_test.go . downloader client
type downloader interface {
	Download(io.WriterAt, *s3.GetObjectInput, ...func(*s3manager.Downloader)) (n int64, err error)
}

type client interface {
	HeadBucket(*s3.HeadBucketInput) (*s3.HeadBucketOutput, error)
	ListObjectsV2(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
	DeleteObject(object *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
}

type BucketClient struct {
	client     client
	downloader downloader

	bucket         string
	tempFilePrefix string
}

var ErrNoAttempts = errors.New("invalid number of attempts, must be > 0")

func NewClient(c client, d downloader, bucket, tempFilePrefix string) *BucketClient {
	return &BucketClient{client: c, downloader: d, bucket: bucket, tempFilePrefix: tempFilePrefix}
}

func NewFromSecrets(key, secret, endpoint, region, bucket string, insecure bool) (*BucketClient, error) {
	s3Config := &aws.Config{
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		DisableSSL:       aws.Bool(insecure),
	}

	sess, err := session.NewSession(s3Config)
	if err != nil {
		return nil, fmt.Errorf("creating s3 session, %w", err)
	}
	s3Client := s3.New(sess)
	s3Downloader := s3manager.NewDownloader(sess)

	return NewClient(s3Client, s3Downloader, bucket, tempFilePrefix), nil
}

func (c *BucketClient) CheckConnectivity(attempts int, dur time.Duration) error {
	if attempts < 1 {
		return fmt.Errorf("checking connectivity, passed %d, %w", attempts, ErrNoAttempts)
	}
	var err error
	for i := 0; i < attempts; i++ {
		_, err = c.client.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(c.bucket)})
		if err != nil {
			time.Sleep(dur)
			continue
		}
		return nil
	}

	return fmt.Errorf("failed to initialize bucket client, %w", err)
}

func (c *BucketClient) ListFiles(start time.Time, ext string) ([]domain.File, error) {
	log.Println("[s3] listing objects with ext", ext, "starting from", start)
	listObjsResponse, err := c.client.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(c.bucket)})
	if err != nil {
		return nil, fmt.Errorf("listing objects from bucket %s, %w", c.bucket, err)
	}
	log.Println("[s3] received", len(listObjsResponse.Contents), "objects")

	var files []domain.File //nolint:prealloc
	for _, object := range listObjsResponse.Contents {
		if object.LastModified.Before(start) {
			continue
		}
		if !strings.HasSuffix(*object.Key, ext) {
			continue
		}

		files = append(files, domain.File{Key: *object.Key, LastModified: *object.LastModified})
	}

	log.Println("[s3]", len(files), "files to process")

	return files, nil
}

func (c *BucketClient) Download(key string) (*os.File, error) {
	f, err := os.CreateTemp("", c.tempFilePrefix)
	if err != nil {
		return nil, fmt.Errorf("creating local image file, %w", err)
	}

	_, err = c.downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return f, fmt.Errorf("downloading image file from s3, %w", err)
	}

	return f, nil
}

func (c *BucketClient) DeleteFile(_ context.Context, key string) error {
	_, err := c.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting file %s from s3, %w", key, err)
	}

	return nil
}
