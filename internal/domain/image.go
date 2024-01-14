package domain

import (
	"database/sql/driver"
	"strconv"
	"strings"
	"time"
)

type Image struct {
	FileID       string    `db:"file_id"`
	Description  string    `db:"description"`
	LastModified time.Time `db:"last_modified"`
	Embedding    Embedding `db:"clip_embedding"`
}

type Embedding []float32

func (e Embedding) Value() (driver.Value, error) {
	var params []string
	for i, _ := range e {
		params = append(params, strconv.FormatFloat(float64(e[i]), 'f', -1, 32))
	}

	return "[" + strings.Join(params, ",") + "]", nil
}
