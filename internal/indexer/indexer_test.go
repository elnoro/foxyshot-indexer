package indexer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/matryer/is"
	"github.com/pkg/errors"
)

func TestIndexer_Index(t *testing.T) {
	tt := is.New(t)

	testFile := domain.File{
		Key:          "expected-image-key",
		LastModified: time.Unix(10000, 0),
	}
	const testImg = "./testdata/expected-downloaded-image"
	const testOCRResult = "expected-ocr-results"

	repo := &ImageRepoMock{UpsertFunc: func(ctx context.Context, image domain.Image) error { return nil }}
	storage := &FileStorageMock{DownloadFunc: func(key string) (*os.File, error) { return os.Create(testImg) }}
	ocr := &OCRMock{RunFunc: func(file string) (string, error) { return testOCRResult, nil }}

	t.Run("successful run", func(t *testing.T) {
		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testFile)
		tt.NoErr(err)

		tt.Equal(storage.DownloadCalls()[0].Key, testFile.Key)
		tt.Equal(ocr.RunCalls()[0].File, testImg)
		tt.Equal(repo.UpsertCalls()[0].Image, domain.Image{
			FileID:       testFile.Key,
			Description:  testOCRResult,
			LastModified: testFile.LastModified,
		})

		_, err = os.Stat(testImg)
		if !os.IsNotExist(err) {
			t.Errorf("file %s must be removed after indexing", testImg)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		repo := &ImageRepoMock{UpsertFunc: func(ctx context.Context, image domain.Image) error { return expectedErr }}

		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testFile)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("storage error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		storage := &FileStorageMock{DownloadFunc: func(key string) (*os.File, error) { return nil, expectedErr }}

		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testFile)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("ocr error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		ocr := &OCRMock{RunFunc: func(file string) (string, error) { return "", expectedErr }}

		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testFile)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("temp file was not created properly", func(t *testing.T) {
		storage := &FileStorageMock{DownloadFunc: func(key string) (*os.File, error) {
			f, _ := os.Create(testImg)
			_ = os.Remove(testImg)

			return f, nil
		}}

		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testFile)

		tt.NoErr(err)
	})
}
