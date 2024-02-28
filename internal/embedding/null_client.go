package embedding

import (
	"errors"
	"github.com/elnoro/foxyshot-indexer/internal/domain"
)

var ErrEmbeddingsSwitchedOff = errors.New("embeddings service is switched off")

type NullClient struct {
}

func (n NullClient) CreateEmbeddingForFile(_ string) (domain.Embedding, error) {
	return domain.Embedding{}, ErrEmbeddingsSwitchedOff
}

func (n NullClient) CreateEmbeddingFromBase64(_ string) (domain.Embedding, error) {
	return domain.Embedding{}, ErrEmbeddingsSwitchedOff
}
