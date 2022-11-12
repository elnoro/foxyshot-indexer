package indexer

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/elnoro/foxyshot-indexer/internal/domain"
)

//go:generate moq -out indexer_moq_test.go . ImageRepo FileStorage OCR
type ImageRepo interface {
	Upsert(ctx context.Context, image domain.Image) error
}

type FileStorage interface {
	Download(key string) (*os.File, error)
}

type OCR interface {
	Run(file string) (string, error)
}

type Indexer struct {
	imageRepo ImageRepo
	storage   FileStorage
	ocrEngine OCR
}

func NewIndexer(imageRepo ImageRepo, storage FileStorage, ocrEngine OCR) *Indexer {
	return &Indexer{imageRepo: imageRepo, storage: storage, ocrEngine: ocrEngine}
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
