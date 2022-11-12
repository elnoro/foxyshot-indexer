package domain

import "time"

type File struct {
	Key          string
	LastModified time.Time
}
