package domain

type Image struct {
	FileID      string `db:"file_id"`
	Description string `db:"description"`
}
