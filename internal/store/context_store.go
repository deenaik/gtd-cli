package store

import (
	"database/sql"

	"github.com/deenaik/gtd-cli/internal/model"
)

type ContextStore struct {
	db *sql.DB
}

func NewContextStore(db *sql.DB) *ContextStore {
	return &ContextStore{db: db}
}

func (s *ContextStore) Create(name string) (*model.Context, error) {
	res, err := s.db.Exec(`INSERT INTO contexts (name) VALUES (?)`, name)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &model.Context{ID: id, Name: name}, nil
}

func (s *ContextStore) List() ([]model.Context, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM contexts ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contexts []model.Context
	for rows.Next() {
		var c model.Context
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		contexts = append(contexts, c)
	}
	return contexts, nil
}

func (s *ContextStore) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM contexts WHERE id = ?`, id)
	return err
}

func (s *ContextStore) FindByName(name string) (*model.Context, error) {
	c := &model.Context{}
	err := s.db.QueryRow(`SELECT id, name, created_at FROM contexts WHERE name = ?`, name).Scan(
		&c.ID, &c.Name, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ContextStore) GetByID(id int64) (*model.Context, error) {
	c := &model.Context{}
	err := s.db.QueryRow(`SELECT id, name, created_at FROM contexts WHERE id = ?`, id).Scan(
		&c.ID, &c.Name, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}
