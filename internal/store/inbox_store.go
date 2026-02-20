package store

import (
	"database/sql"

	"github.com/deenaik/gtd-cli/internal/model"
)

type InboxStore struct {
	db *sql.DB
}

func NewInboxStore(db *sql.DB) *InboxStore {
	return &InboxStore{db: db}
}

func (s *InboxStore) Create(item *model.InboxItem) error {
	res, err := s.db.Exec(`
		INSERT INTO inbox_items (title, body, source, source_ref, source_meta)
		VALUES (?, ?, ?, ?, ?)`,
		item.Title, item.Body, item.Source, item.SourceRef, item.SourceMeta,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	item.ID = id
	return nil
}

func (s *InboxStore) ListUnprocessed() ([]model.InboxItem, error) {
	rows, err := s.db.Query(`
		SELECT id, title, body, source, source_ref, source_meta, processed, created_at
		FROM inbox_items WHERE processed = 0
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.InboxItem
	for rows.Next() {
		var item model.InboxItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Body, &item.Source, &item.SourceRef,
			&item.SourceMeta, &item.Processed, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *InboxStore) Count() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM inbox_items WHERE processed = 0`).Scan(&count)
	return count, err
}

func (s *InboxStore) MarkProcessed(id int64) error {
	_, err := s.db.Exec(`UPDATE inbox_items SET processed = 1 WHERE id = ?`, id)
	return err
}

func (s *InboxStore) GetByID(id int64) (*model.InboxItem, error) {
	item := &model.InboxItem{}
	err := s.db.QueryRow(`
		SELECT id, title, body, source, source_ref, source_meta, processed, created_at
		FROM inbox_items WHERE id = ?`, id).Scan(
		&item.ID, &item.Title, &item.Body, &item.Source, &item.SourceRef,
		&item.SourceMeta, &item.Processed, &item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *InboxStore) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM inbox_items WHERE id = ?`, id)
	return err
}
