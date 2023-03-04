package indexer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	dbadapter "github.com/elnoro/foxyshot-indexer/internal/db"
	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/elnoro/foxyshot-indexer/internal/monitoring"
)

//go:generate moq -out indexer_moq_test.go . ImageRepo FileStorage OCR
type ImageRepo interface {
	Get(ctx context.Context, fileID string) (domain.Image, error)
	GetLastModified(ctx context.Context) (time.Time, error)
	Upsert(ctx context.Context, image domain.Image) error
}

type FileStorage interface {
	ListFiles(start time.Time, ext string) ([]domain.File, error)
	Download(key string) (*os.File, error)
}

type OCR interface {
	Run(file string) (string, error)
}

type Indexer struct {
	imageRepo ImageRepo
	storage   FileStorage
	ocrEngine OCR

	tracker *monitoring.Tracker
}

func NewIndexer(
	imageRepo ImageRepo,
	storage FileStorage,
	ocrEngine OCR,
	tracker *monitoring.Tracker,
) *Indexer {
	return &Indexer{imageRepo: imageRepo, storage: storage, ocrEngine: ocrEngine, tracker: tracker}
}

func (i *Indexer) IndexNewList(ctx context.Context, pattern string) error {
	lastModified, err := i.imageRepo.GetLastModified(ctx)
	if err != nil {
		return fmt.Errorf("getting last modified, %w", err)
	}
	files, err := i.storage.ListFiles(lastModified, pattern)
	if err != nil {
		return fmt.Errorf("listing files, %w", err)
	}

	for _, file := range files {
		_, err := i.imageRepo.Get(ctx, file.Key)
		if err != nil && !errors.Is(err, dbadapter.ErrRecordNotFound) {
			log.Println("ERROR: failed to check:", err)
			continue
		}
		if nil == err {
			log.Printf("INFO: %s already processed, skipping\n", file.Key)
			continue
		}

		err = i.Index(file)
		if err != nil {
			log.Println("ERROR: failed to index:", err)
		} else {
			log.Println("INFO: added", file)
			i.tracker.OnIndex()
		}
	}

	return nil
}

func (i *Indexer) Index(file domain.File) error {
	f, err := i.storage.Download(file.Key)
	if f != nil {
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				log.Println("ERROR: removing temp file,", err)
			}
		}(f.Name())
	}
	if err != nil {
		return fmt.Errorf("cannot download file, %w", err)
	}
	desc, err := i.ocrEngine.Run(f.Name())
	if err != nil {
		return fmt.Errorf("running ocr, %w", err)
	}

	img := domain.Image{
		FileID:       file.Key,
		LastModified: file.LastModified,
		Description:  desc,
	}

	err = i.imageRepo.Upsert(context.TODO(), img)
	if err != nil {
		return fmt.Errorf("inserting image, %w", err)
	}

	return nil
}
