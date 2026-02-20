package store

import (
	"database/sql"
	"time"
)

type WeeklyReview struct {
	ID             int64
	WeekStart      string
	TasksCompleted int
	TasksCreated   int
	Notes          string
	AIInsights     string
	CompletedAt    sql.NullTime
	CreatedAt      time.Time
}

type ReviewStore struct {
	db *sql.DB
}

func NewReviewStore(db *sql.DB) *ReviewStore {
	return &ReviewStore{db: db}
}

func (s *ReviewStore) Create(r *WeeklyReview) error {
	res, err := s.db.Exec(`
		INSERT INTO weekly_reviews (week_start, tasks_completed, tasks_created, notes, ai_insights)
		VALUES (?, ?, ?, ?, ?)`,
		r.WeekStart, r.TasksCompleted, r.TasksCreated, r.Notes, r.AIInsights,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	r.ID = id
	return nil
}

func (s *ReviewStore) GetLatest() (*WeeklyReview, error) {
	r := &WeeklyReview{}
	err := s.db.QueryRow(`
		SELECT id, week_start, tasks_completed, tasks_created, notes, ai_insights, completed_at, created_at
		FROM weekly_reviews ORDER BY created_at DESC LIMIT 1`).Scan(
		&r.ID, &r.WeekStart, &r.TasksCompleted, &r.TasksCreated, &r.Notes, &r.AIInsights,
		&r.CompletedAt, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *ReviewStore) Complete(id int64, notes, insights string) error {
	_, err := s.db.Exec(`
		UPDATE weekly_reviews SET notes=?, ai_insights=?, completed_at=datetime('now')
		WHERE id=?`, notes, insights, id)
	return err
}

func (s *ReviewStore) List(limit int) ([]WeeklyReview, error) {
	rows, err := s.db.Query(`
		SELECT id, week_start, tasks_completed, tasks_created, notes, ai_insights, completed_at, created_at
		FROM weekly_reviews ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []WeeklyReview
	for rows.Next() {
		var r WeeklyReview
		if err := rows.Scan(&r.ID, &r.WeekStart, &r.TasksCompleted, &r.TasksCreated,
			&r.Notes, &r.AIInsights, &r.CompletedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	return reviews, nil
}
