package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/deenaik/gtd-cli/internal/model"
)

type TaskStore struct {
	db *sql.DB
}

func NewTaskStore(db *sql.DB) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) Create(t *model.Task) error {
	res, err := s.db.Exec(`
		INSERT INTO tasks (title, description, category, status, priority, energy, time_estimate,
			project_id, context_id, delegated_to, due_date, defer_until, source, source_ref, calendar_event_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Title, t.Description, t.Category, t.Status, t.Priority, t.Energy, t.TimeEstimate,
		t.ProjectID, t.ContextID, t.DelegatedTo, t.DueDate, t.DeferUntil,
		t.Source, t.SourceRef, t.CalendarEventID,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	t.ID = id
	// Log activity
	_, _ = s.db.Exec(`INSERT INTO task_activity (task_id, action) VALUES (?, 'created')`, id)
	return nil
}

func (s *TaskStore) GetByID(id int64) (*model.Task, error) {
	t := &model.Task{}
	var projectName, contextName sql.NullString
	err := s.db.QueryRow(`
		SELECT t.id, t.title, t.description, t.category, t.status, t.priority, t.energy,
			t.time_estimate, t.project_id, t.context_id, t.delegated_to, t.due_date,
			t.defer_until, t.completed_at, t.source, t.source_ref, t.calendar_event_id,
			t.created_at, t.updated_at,
			p.name, c.name
		FROM tasks t
		LEFT JOIN projects p ON t.project_id = p.id
		LEFT JOIN contexts c ON t.context_id = c.id
		WHERE t.id = ?`, id).Scan(
		&t.ID, &t.Title, &t.Description, &t.Category, &t.Status, &t.Priority, &t.Energy,
		&t.TimeEstimate, &t.ProjectID, &t.ContextID, &t.DelegatedTo, &t.DueDate,
		&t.DeferUntil, &t.CompletedAt, &t.Source, &t.SourceRef, &t.CalendarEventID,
		&t.CreatedAt, &t.UpdatedAt,
		&projectName, &contextName,
	)
	if err != nil {
		return nil, err
	}
	if projectName.Valid {
		t.ProjectName = projectName.String
	}
	if contextName.Valid {
		t.ContextName = contextName.String
	}
	return t, nil
}

func (s *TaskStore) Update(t *model.Task) error {
	_, err := s.db.Exec(`
		UPDATE tasks SET title=?, description=?, category=?, status=?, priority=?, energy=?,
			time_estimate=?, project_id=?, context_id=?, delegated_to=?, due_date=?,
			defer_until=?, completed_at=?, calendar_event_id=?, updated_at=datetime('now')
		WHERE id=?`,
		t.Title, t.Description, t.Category, t.Status, t.Priority, t.Energy,
		t.TimeEstimate, t.ProjectID, t.ContextID, t.DelegatedTo, t.DueDate,
		t.DeferUntil, t.CompletedAt, t.CalendarEventID, t.ID,
	)
	return err
}

func (s *TaskStore) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

func (s *TaskStore) MarkDone(id int64) error {
	_, err := s.db.Exec(`
		UPDATE tasks SET status='done', completed_at=datetime('now'), updated_at=datetime('now')
		WHERE id=?`, id)
	if err == nil {
		_, _ = s.db.Exec(`INSERT INTO task_activity (task_id, action) VALUES (?, 'completed')`, id)
	}
	return err
}

type TaskFilter struct {
	Category  string
	Status    string
	ContextID int64
	ProjectID int64
	DueBefore string // date string
	Who       string // delegated_to filter
}

func (s *TaskStore) List(f TaskFilter) ([]model.Task, error) {
	where := []string{"1=1"}
	args := []any{}

	if f.Category != "" {
		where = append(where, "t.category = ?")
		args = append(args, f.Category)
	}
	if f.Status != "" {
		where = append(where, "t.status = ?")
		args = append(args, f.Status)
	} else {
		where = append(where, "t.status != 'done'")
	}
	if f.ContextID > 0 {
		where = append(where, "t.context_id = ?")
		args = append(args, f.ContextID)
	}
	if f.ProjectID > 0 {
		where = append(where, "t.project_id = ?")
		args = append(args, f.ProjectID)
	}
	if f.DueBefore != "" {
		where = append(where, "t.due_date IS NOT NULL AND t.due_date <= ?")
		args = append(args, f.DueBefore)
	}
	if f.Who != "" {
		where = append(where, "t.delegated_to LIKE ?")
		args = append(args, "%"+f.Who+"%")
	}

	query := fmt.Sprintf(`
		SELECT t.id, t.title, t.description, t.category, t.status, t.priority, t.energy,
			t.time_estimate, t.project_id, t.context_id, t.delegated_to, t.due_date,
			t.defer_until, t.completed_at, t.source, t.source_ref, t.calendar_event_id,
			t.created_at, t.updated_at,
			COALESCE(p.name, ''), COALESCE(c.name, '')
		FROM tasks t
		LEFT JOIN projects p ON t.project_id = p.id
		LEFT JOIN contexts c ON t.context_id = c.id
		WHERE %s
		ORDER BY t.priority DESC, t.due_date ASC, t.created_at ASC`,
		strings.Join(where, " AND "))

	return s.queryTasks(query, args...)
}

func (s *TaskStore) Search(query string) ([]model.Task, error) {
	return s.queryTasks(`
		SELECT t.id, t.title, t.description, t.category, t.status, t.priority, t.energy,
			t.time_estimate, t.project_id, t.context_id, t.delegated_to, t.due_date,
			t.defer_until, t.completed_at, t.source, t.source_ref, t.calendar_event_id,
			t.created_at, t.updated_at,
			COALESCE(p.name, ''), COALESCE(c.name, '')
		FROM tasks_fts fts
		JOIN tasks t ON fts.rowid = t.id
		LEFT JOIN projects p ON t.project_id = p.id
		LEFT JOIN contexts c ON t.context_id = c.id
		WHERE tasks_fts MATCH ?
		ORDER BY rank`, query)
}

func (s *TaskStore) Overdue() ([]model.Task, error) {
	return s.List(TaskFilter{DueBefore: "date('now')", Status: "pending"})
}

func (s *TaskStore) CountByStatus() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM tasks GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			m[status] = count
		}
	}
	return m, nil
}

func (s *TaskStore) CompletedSince(since string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status='done' AND completed_at >= ?`, since).Scan(&count)
	return count, err
}

func (s *TaskStore) CreatedSince(since string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE created_at >= ?`, since).Scan(&count)
	return count, err
}

func (s *TaskStore) queryTasks(query string, args ...any) ([]model.Task, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Category, &t.Status, &t.Priority, &t.Energy,
			&t.TimeEstimate, &t.ProjectID, &t.ContextID, &t.DelegatedTo, &t.DueDate,
			&t.DeferUntil, &t.CompletedAt, &t.Source, &t.SourceRef, &t.CalendarEventID,
			&t.CreatedAt, &t.UpdatedAt,
			&t.ProjectName, &t.ContextName,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
