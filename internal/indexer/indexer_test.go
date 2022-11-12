package indexer

import (
	"os"
	"testing"

	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/matryer/is"
	"github.com/pkg/errors"
)

func TestIndexer_Index(t *testing.T) {
	tt := is.New(t)

	const testImg = "./testdata/expected-downloaded-image"
	const testKey = "expected-image-key"
	const testOCRResult = "expected-ocr-results"

	repo := &ImageRepoMock{InsertFunc: func(image domain.Image) error { return nil }}
	storage := &FileStorageMock{DownloadFunc: func(key string) (*os.File, error) { return os.Create(testImg) }}
	ocr := &OCRMock{RunFunc: func(file string) (string, error) { return testOCRResult, nil }}

	t.Run("successful run", func(t *testing.T) {
		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testKey)
		tt.NoErr(err)

		tt.Equal(storage.DownloadCalls()[0].Key, testKey)
		tt.Equal(ocr.RunCalls()[0].File, testImg)
		tt.Equal(repo.InsertCalls()[0].Image, domain.Image{
			FileID:      testKey,
			Description: testOCRResult,
		})

		_, err = os.Stat(testImg)
		if !os.IsNotExist(err) {
			t.Errorf("file %s must be removed after indexing", testImg)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		repo := &ImageRepoMock{InsertFunc: func(image domain.Image) error { return expectedErr }}

		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testKey)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("storage error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		storage := &FileStorageMock{DownloadFunc: func(key string) (*os.File, error) { return nil, expectedErr }}

		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testKey)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("ocr error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		ocr := &OCRMock{RunFunc: func(file string) (string, error) { return "", expectedErr }}

		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testKey)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("temp file was not created properly", func(t *testing.T) {
		storage := &FileStorageMock{DownloadFunc: func(key string) (*os.File, error) {
			f, _ := os.Create(testImg)
			_ = os.Remove(testImg)

			return f, nil
		}}

		indexer := NewIndexer(repo, storage, ocr)
		err := indexer.Index(testKey)

		tt.NoErr(err)
	})
}
