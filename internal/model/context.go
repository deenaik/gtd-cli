package model

import "time"

type Context struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}
