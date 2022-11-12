package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/jmoiron/sqlx"
)

type ImageRepo struct {
	db *sqlx.DB
}

func NewImageRepo(db *sqlx.DB) *ImageRepo {
	return &ImageRepo{db: db}
}

func (i *ImageRepo) Upsert(image domain.Image) error {
	query := `INSERT INTO image_descriptions (file_id, description, last_modified) 
			VALUES (:file_id, :description, :last_modified)
			ON CONFLICT (file_id) DO UPDATE SET (description, last_modified) 
			    = (excluded.description, excluded.last_modified)`
	_, err := i.db.NamedExec(query, image)
	if err != nil {
		return fmt.Errorf("inserting image id=%s, %w", image.FileID, err)
	}

	return nil
}

func (i *ImageRepo) GetLastModified(ctx context.Context) (time.Time, error) {
	query := `SELECT file_id, description, last_modified FROM image_descriptions ORDER BY last_modified DESC LIMIT 1`
	lastImg := &domain.Image{}
	err := i.db.GetContext(ctx, lastImg, query)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return time.Unix(0, 0), nil
		default:
			return time.Time{}, fmt.Errorf("looking for last modified, %w", err)
		}
	}

	return lastImg.LastModified, nil
}
