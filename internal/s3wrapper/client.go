package s3wrapper

import (
	"fmt"
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

type BucketClient struct {
	client     *s3.S3
	downloader *s3manager.Downloader

	bucket string
}

func NewClient(client *s3.S3, downloader *s3manager.Downloader, bucket string) *BucketClient {
	return &BucketClient{client: client, downloader: downloader, bucket: bucket}
}

func NewFromSecrets(key, secret, endpoint, region, bucket string) (*BucketClient, error) {
	s3Config := &aws.Config{
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
	}

	sess, err := session.NewSession(s3Config)
	if err != nil {
		return nil, fmt.Errorf("creating s3 session, %w", err)
	}
	s3Client := s3.New(sess)
	s3Downloader := s3manager.NewDownloader(sess)

	return NewClient(s3Client, s3Downloader, bucket), nil
}

func (c *BucketClient) ListFiles(start time.Time, ext string) ([]domain.File, error) {
	listObjsResponse, err := c.client.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(c.bucket)})
	if err != nil {
		return nil, fmt.Errorf("listing objects from bucket %s, %w", c.bucket, err)
	}

	var files []domain.File
	for _, object := range listObjsResponse.Contents {
		if object.LastModified.Before(start) {
			continue
		}
		if !strings.HasSuffix(*object.Key, ext) {
			continue
		}

		files = append(files, domain.File{Key: *object.Key, LastModified: *object.LastModified})
	}

	return files, nil
}

func (c *BucketClient) Download(key string) (*os.File, error) {
	f, err := os.CreateTemp("", "foxyshot_indexer_")
	if err != nil {
		return nil, fmt.Errorf("creating local image file, %w", err)
	}

	_, err = c.downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("downloading image file from s3, %w", err)
	}

	return f, nil
}
