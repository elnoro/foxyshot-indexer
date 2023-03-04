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

var (
	ErrRecordNotFound = errors.New("record not found")
)

func NewImageRepo(db *sqlx.DB) *ImageRepo {
	return &ImageRepo{db: db}
}

func (i *ImageRepo) Upsert(ctx context.Context, image domain.Image) error {
	query := `INSERT INTO image_descriptions (file_id, description, last_modified) 
			VALUES (:file_id, :description, :last_modified)
			ON CONFLICT (file_id) DO UPDATE SET (description, last_modified) 
			    = (excluded.description, excluded.last_modified)`
	_, err := i.db.NamedExecContext(ctx, query, image)
	if err != nil {
		return fmt.Errorf("inserting image id=%s, %w", image.FileID, err)
	}

	return nil
}

func (i *ImageRepo) Search(ctx context.Context, searchString string, page, perPage int) ([]domain.Image, error) {
	pattern := "%" + searchString + "%"
	limit := perPage
	offset := (page - 1) * perPage

	imgs := []domain.Image{}
	query := `SELECT file_id, description, last_modified 
		FROM image_descriptions 
		WHERE (to_tsvector('simple', description) @@ plainto_tsquery('simple', $1) OR $1 = '')
		ORDER BY last_modified desc LIMIT $2 OFFSET $3`
	args := []any{pattern, limit, offset}
	err := i.db.SelectContext(ctx, &imgs, query, args...)
	if err != nil {
		return imgs, fmt.Errorf("searching for images with query %s, %w", query, err)
	}

	return imgs, nil
}

func (i *ImageRepo) Delete(ctx context.Context, fileID string) error {
	query := `DELETE FROM image_descriptions where file_id = $1`
	_, err := i.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("deleting image id=%s, %w", fileID, err)
	}

	return nil
}

func (i *ImageRepo) Get(ctx context.Context, fileID string) (domain.Image, error) {
	query := `SELECT file_id, description, last_modified FROM image_descriptions where file_id = $1`
	img := &domain.Image{}
	err := i.db.GetContext(ctx, img, query, fileID)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return domain.Image{}, fmt.Errorf("image with file id %s not found, %w", fileID, ErrRecordNotFound)
		default:
			return domain.Image{}, fmt.Errorf("looking for last modified, %w", err)
		}
	}

	return *img, nil
}

func (i *ImageRepo) GetLastModified(ctx context.Context) (time.Time, error) {
	query := `SELECT file_id, description, last_modified FROM image_descriptions ORDER BY last_modified DESC LIMIT 1`
	lastImg := &domain.Image{}
	err := i.db.GetContext(ctx, lastImg, query)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return time.Unix(0, 0).UTC(), nil
		default:
			return time.Time{}, fmt.Errorf("looking for last modified, %w", err)
		}
	}

	return lastImg.LastModified, nil
}
