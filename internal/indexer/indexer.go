package indexer

import (
	"context"
	"errors"
	"fmt"
	"github.com/elnoro/foxyshot-indexer/internal/embedding"
	"log/slog"
	"os"
	"time"

	dbadapter "github.com/elnoro/foxyshot-indexer/internal/db"
	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/elnoro/foxyshot-indexer/internal/monitoring"
)

//go:generate moq -out indexer_moq_test.go . ImageRepo FileStorage OCR CaptionSmith ImageEmbeddingClient
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

type CaptionSmith interface {
	Caption(filename string) (string, error)
}

type ImageEmbeddingClient interface {
	CreateEmbeddingForFile(filePath string) (domain.Embedding, error)
	CreateEmbeddingFromBase64(data string) (domain.Embedding, error)
}

type Indexer struct {
	imageRepo       ImageRepo
	storage         FileStorage
	ocrEngine       OCR
	captionSmith    CaptionSmith
	embeddingClient ImageEmbeddingClient

	log     *slog.Logger
	tracker *monitoring.Tracker
}

func NewIndexer(
	imageRepo ImageRepo,
	storage FileStorage,
	ocrEngine OCR,
	captionSmith CaptionSmith,
	embeddingClient ImageEmbeddingClient,
	log *slog.Logger,
	tracker *monitoring.Tracker,
) *Indexer {
	return &Indexer{
		imageRepo:       imageRepo,
		storage:         storage,
		ocrEngine:       ocrEngine,
		captionSmith:    captionSmith,
		embeddingClient: embeddingClient,
		log:             log.WithGroup("INDEXER"),
		tracker:         tracker,
	}
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
			i.log.Error("getting image from the database", slog.String("err", err.Error()))
			continue
		}
		if nil == err {
			i.log.Info("skipping, file already processed", slog.String("file", file.Key))
			continue
		}

		err = i.Index(ctx, file)
		if err != nil {
			i.log.Error("indexing file", slog.String("err", err.Error()))
		} else {
			i.log.Info("file processed", slog.String("file", file.Key))
			i.tracker.OnIndex()
		}
	}

	return nil
}

func (i *Indexer) Index(ctx context.Context, file domain.File) error {
	f, err := i.storage.Download(file.Key)
	if f != nil {
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				i.log.Error("removing temp file",
					slog.String("file", name),
					slog.String("err", err.Error()),
				)
			}
		}(f.Name())
	}
	if err != nil {
		return fmt.Errorf("cannot download file, %w", err)
	}

	ocrStream := runOnFile(f.Name(), i.ocrEngine.Run)
	captionStream := runOnFile(f.Name(), i.captionSmith.Caption)
	vectorStream := runOnFile(f.Name(), i.embeddingClient.CreateEmbeddingForFile)

	ocr := <-ocrStream
	if ocr.err != nil {
		return fmt.Errorf("running ocr, %w", ocr.err)
	}
	caption := <-captionStream
	if caption.err != nil {
		return fmt.Errorf("running captioning, %w", caption.err)
	}

	emb := <-vectorStream
	if emb.err != nil && !errors.Is(emb.err, embedding.ErrEmbeddingsSwitchedOff) {
		return fmt.Errorf("creating embedding, %w", emb.err)
	}

	desc := fmt.Sprintf("OCR:\n%s\nCaption:\n%s", ocr.val, caption.val)

	img := domain.Image{
		FileID:       file.Key,
		LastModified: file.LastModified,
		Description:  desc,
		Embedding:    emb.val,
	}

	err = i.imageRepo.Upsert(ctx, img)
	if err != nil {
		return fmt.Errorf("inserting image, %w", err)
	}

	return nil
}

type result[T any] struct {
	val T
	err error
}

type indexFunc[T any] func(filename string) (T, error)

func runOnFile[T any](filename string, f indexFunc[T]) <-chan result[T] {
	resStream := make(chan result[T])

	go func(resStream chan<- result[T]) {
		res, err := f(filename)

		resStream <- result[T]{res, err}
	}(resStream)

	return resStream
}
