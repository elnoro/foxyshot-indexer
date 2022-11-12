package domain

import "time"

type Image struct {
	FileID       string    `db:"file_id"`
	Description  string    `db:"description"`
	LastModified time.Time `db:"last_modified"`
}
