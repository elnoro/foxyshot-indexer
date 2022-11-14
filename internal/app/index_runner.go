package app

import (
	"context"
	"fmt"
	"log"
	"time"
)

//go:generate moq -out index_runner_moq_test.go . listIndexer
type listIndexer interface {
	IndexNewList(context.Context, string) error
}

type IndexRunner struct {
	indexer listIndexer

	ext      string
	interval time.Duration
}

func NewIndexRunner(indexer listIndexer, ext string, interval time.Duration) *IndexRunner {
	return &IndexRunner{indexer: indexer, ext: ext, interval: interval}
}

func (i *IndexRunner) Start(ctx context.Context) error {
	log.Println("INFO: running indexer")
	timer := time.NewTimer(0) // starting immediately
	for {
		select {
		case <-timer.C:
			err := i.indexer.IndexNewList(ctx, i.ext)
			if err != nil {
				return fmt.Errorf("indexing new, %w", err)
			}
			timer = time.NewTimer(i.interval)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
