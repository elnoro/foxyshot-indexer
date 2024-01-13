package indexer

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/elnoro/foxyshot-indexer/internal/db"
	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/elnoro/foxyshot-indexer/internal/monitoring"
	"github.com/matryer/is"
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
	logger := slog.Default()
	tracker := monitoring.NewTracker()

	t.Run("successful run", func(t *testing.T) {
		indexer := NewIndexer(repo, storage, ocr, logger, tracker)
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

		indexer := NewIndexer(repo, storage, ocr, logger, tracker)
		err := indexer.Index(testFile)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("storage error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		storage := &FileStorageMock{DownloadFunc: func(key string) (*os.File, error) { return nil, expectedErr }}

		indexer := NewIndexer(repo, storage, ocr, logger, tracker)
		err := indexer.Index(testFile)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("ocr error", func(t *testing.T) {
		expectedErr := errors.New("expected err")
		ocr := &OCRMock{RunFunc: func(file string) (string, error) { return "", expectedErr }}

		indexer := NewIndexer(repo, storage, ocr, logger, tracker)
		err := indexer.Index(testFile)

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("temp file was not created properly", func(t *testing.T) {
		storage := &FileStorageMock{DownloadFunc: func(key string) (*os.File, error) {
			f, _ := os.Create(testImg)
			_ = os.Remove(testImg)

			return f, nil
		}}

		indexer := NewIndexer(repo, storage, ocr, logger, tracker)
		err := indexer.Index(testFile)

		tt.NoErr(err)
	})
}

func TestIndexer_IndexNewList(t *testing.T) {
	tt := is.New(t)

	const (
		expectedKey   = "expected-image-key"
		testImg       = "./testdata/expected-downloaded-image"
		testOCRResult = "expected-ocr-results"
	)
	expectedErr := errors.New("expected error")

	ctx := context.Background()
	tracker := monitoring.NewTracker()
	logger := slog.Default()

	repo := &ImageRepoMock{
		UpsertFunc: func(ctx context.Context, image domain.Image) error { return nil },
		GetLastModifiedFunc: func(_ context.Context) (time.Time, error) {
			return time.Unix(99, 0), nil
		},
		GetFunc: func(_ context.Context, fileID string) (domain.Image, error) {
			if fileID == expectedKey {
				return domain.Image{FileID: expectedKey}, db.ErrRecordNotFound
			}

			return domain.Image{}, errors.New("expected error")
		},
	}
	storage := &FileStorageMock{
		DownloadFunc: func(key string) (*os.File, error) { return os.Create(testImg) },
		ListFilesFunc: func(_ time.Time, _ string) ([]domain.File, error) {
			return []domain.File{{Key: expectedKey}, {Key: "invalid-key"}}, nil
		},
	}
	ocr := &OCRMock{RunFunc: func(file string) (string, error) { return testOCRResult, nil }}

	t.Run("successful run", func(t *testing.T) {
		indexer := NewIndexer(repo, storage, ocr, logger, tracker)
		err := indexer.IndexNewList(ctx, "expected-pattern")

		tt.NoErr(err)
		tt.Equal(storage.ListFilesCalls()[0].Start, time.Unix(99, 0)) // must match GetLastModified result
		tt.Equal(storage.ListFilesCalls()[0].Ext, "expected-pattern") // must match GetLastModified result

		tt.Equal(len(repo.UpsertCalls()), 1)                      // only one file passes the filter
		tt.Equal(repo.UpsertCalls()[0].Image.FileID, expectedKey) // only one file passes the filter
	})
	t.Run("getting last modified error", func(t *testing.T) {
		repo := &ImageRepoMock{GetLastModifiedFunc: func(_ context.Context) (time.Time, error) {
			return time.Time{}, expectedErr
		}}
		indexer := NewIndexer(repo, storage, ocr, logger, tracker)

		err := indexer.IndexNewList(ctx, "expected-pattern")

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("listing files error", func(t *testing.T) {
		storage := &FileStorageMock{
			ListFilesFunc: func(start time.Time, ext string) ([]domain.File, error) {
				return nil, expectedErr
			},
		}
		indexer := NewIndexer(repo, storage, ocr, logger, tracker)

		err := indexer.IndexNewList(ctx, "expected-pattern")

		tt.True(errors.Is(err, expectedErr))
	})

	t.Run("file index err", func(t *testing.T) {
		storage := &FileStorageMock{
			DownloadFunc: func(key string) (*os.File, error) { return nil, expectedErr },
			ListFilesFunc: func(_ time.Time, _ string) ([]domain.File, error) {
				return []domain.File{{Key: expectedKey}, {Key: "invalid-key"}}, nil
			},
		}
		indexer := NewIndexer(repo, storage, ocr, logger, tracker)

		err := indexer.IndexNewList(ctx, "expected-pattern")

		tt.NoErr(err)                             // individual file indexing errors do not break stop the whole method
		tt.Equal(len(storage.DownloadCalls()), 1) // must get to the index stage
	})

	t.Run("skips processing if image is already in the repo", func(t *testing.T) {
		storage := &FileStorageMock{
			DownloadFunc: func(key string) (*os.File, error) { return os.Create(testImg) },
			ListFilesFunc: func(_ time.Time, _ string) ([]domain.File, error) {
				return []domain.File{{Key: expectedKey}, {Key: "invalid-key"}}, nil
			},
		}
		repo := &ImageRepoMock{
			UpsertFunc: func(ctx context.Context, image domain.Image) error { return nil },
			GetLastModifiedFunc: func(_ context.Context) (time.Time, error) {
				return time.Unix(99, 0), nil
			},
			GetFunc: func(_ context.Context, fileID string) (domain.Image, error) { return domain.Image{}, nil },
		}
		indexer := NewIndexer(repo, storage, ocr, logger, tracker)

		err := indexer.IndexNewList(ctx, "expected-pattern")

		tt.NoErr(err)                             // individual file indexing errors do not break stop the whole method
		tt.Equal(len(storage.DownloadCalls()), 0) // must not get to the index stage
	})
}
