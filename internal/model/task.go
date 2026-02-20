package model

import (
	"database/sql"
	"time"
)

type Task struct {
	ID              int64
	Title           string
	Description     string
	Category        string // next_action, waiting_for, someday_maybe
	Status          string // pending, active, done, delegated, deferred
	Priority        int    // 0=none, 1=low, 2=medium, 3=high, 4=urgent
	Energy          string // low, medium, high
	TimeEstimate    int    // minutes
	ProjectID       sql.NullInt64
	ContextID       sql.NullInt64
	DelegatedTo     string
	DueDate         sql.NullString
	DeferUntil      sql.NullString
	CompletedAt     sql.NullTime
	Source          string
	SourceRef       string
	CalendarEventID string
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// Joined fields (not stored directly)
	ProjectName string
	ContextName string
}

func (t Task) PriorityLabel() string {
	switch t.Priority {
	case 1:
		return "low"
	case 2:
		return "medium"
	case 3:
		return "high"
	case 4:
		return "urgent"
	default:
		return ""
	}
}

type InboxItem struct {
	ID         int64
	Title      string
	Body       string
	Source     string
	SourceRef  string
	SourceMeta string
	Processed  bool
	CreatedAt  time.Time
}
