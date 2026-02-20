package model

import (
	"database/sql"
	"time"
)

type Project struct {
	ID          int64
	Name        string
	Description string
	Status      string // active, completed, on_hold, dropped
	Area        string
	DueDate     sql.NullString
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TaskCount   int // joined field
}
