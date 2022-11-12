package db

import (
	"fmt"

	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/jmoiron/sqlx"
)

type ImageRepo struct {
	db *sqlx.DB
}

func NewImageRepo(db *sqlx.DB) *ImageRepo {
	return &ImageRepo{db: db}
}

func (i *ImageRepo) Insert(image domain.Image) error {
	query := `INSERT INTO image_descriptions (file_id, description) VALUES (:file_id, :description)`
	_, err := i.db.NamedExec(query, image)

	return fmt.Errorf("inserting image id=%s, %w", image.FileID, err)
}
