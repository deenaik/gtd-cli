package store

import (
	"database/sql"

	"github.com/deenaik/gtd-cli/internal/model"
)

type ProjectStore struct {
	db *sql.DB
}

func NewProjectStore(db *sql.DB) *ProjectStore {
	return &ProjectStore{db: db}
}

func (s *ProjectStore) Create(p *model.Project) error {
	res, err := s.db.Exec(`
		INSERT INTO projects (name, description, status, area, due_date)
		VALUES (?, ?, ?, ?, ?)`,
		p.Name, p.Description, p.Status, p.Area, p.DueDate,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	p.ID = id
	return nil
}

func (s *ProjectStore) GetByID(id int64) (*model.Project, error) {
	p := &model.Project{}
	err := s.db.QueryRow(`
		SELECT p.id, p.name, p.description, p.status, p.area, p.due_date, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id AND t.status != 'done')
		FROM projects p WHERE p.id = ?`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Status, &p.Area, &p.DueDate,
		&p.CreatedAt, &p.UpdatedAt, &p.TaskCount,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProjectStore) List(status string) ([]model.Project, error) {
	query := `
		SELECT p.id, p.name, p.description, p.status, p.area, p.due_date, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id AND t.status != 'done')
		FROM projects p`
	args := []any{}
	if status != "" {
		query += " WHERE p.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY p.name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.Area, &p.DueDate,
			&p.CreatedAt, &p.UpdatedAt, &p.TaskCount); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *ProjectStore) Update(p *model.Project) error {
	_, err := s.db.Exec(`
		UPDATE projects SET name=?, description=?, status=?, area=?, due_date=?, updated_at=datetime('now')
		WHERE id=?`,
		p.Name, p.Description, p.Status, p.Area, p.DueDate, p.ID,
	)
	return err
}

func (s *ProjectStore) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

func (s *ProjectStore) FindByName(name string) (*model.Project, error) {
	p := &model.Project{}
	err := s.db.QueryRow(`
		SELECT id, name, description, status, area, due_date, created_at, updated_at
		FROM projects WHERE name = ?`, name).Scan(
		&p.ID, &p.Name, &p.Description, &p.Status, &p.Area, &p.DueDate,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}
