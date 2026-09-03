package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wwoes/notification-service/internal/model"
)

// TemplateStore persists reusable notification message bodies.
type TemplateStore struct {
	pool *pgxpool.Pool
}

func NewTemplateStore(pool *pgxpool.Pool) *TemplateStore {
	return &TemplateStore{pool: pool}
}

func (s *TemplateStore) Create(ctx context.Context, name, body string) (*model.Template, error) {
	const q = `
		insert into templates (name, body)
		values ($1, $2)
		returning id::text, name, body, created_at, updated_at
	`

	var t model.Template
	err := s.pool.QueryRow(ctx, q, name, body).Scan(
		&t.ID, &t.Name, &t.Body, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	return &t, nil
}

func (s *TemplateStore) GetByName(ctx context.Context, name string) (*model.Template, error) {
	const q = `
		select id::text, name, body, created_at, updated_at
		from templates
		where name = $1
	`

	var t model.Template
	err := s.pool.QueryRow(ctx, q, name).Scan(
		&t.ID, &t.Name, &t.Body, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get template by name: %w", err)
	}
	return &t, nil
}

func (s *TemplateStore) GetByID(ctx context.Context, id string) (*model.Template, error) {
	const q = `
		select id::text, name, body, created_at, updated_at
		from templates
		where id = $1::uuid
	`

	var t model.Template
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.Name, &t.Body, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get template by id: %w", err)
	}
	return &t, nil
}

func (s *TemplateStore) List(ctx context.Context) ([]model.Template, error) {
	const q = `
		select id::text, name, body, created_at, updated_at
		from templates
		order by created_at desc
	`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	templates := make([]model.Template, 0)
	for rows.Next() {
		var t model.Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Body, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate templates: %w", err)
	}
	return templates, nil
}
