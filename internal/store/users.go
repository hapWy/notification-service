package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wwoes/notification-service/internal/model"
)

// UserStore persists the mapping between an internal user id and a
// Telegram chat id.
type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

// GetOrCreateByTelegramID upserts a user row for the given Telegram id.
// Notification requests only carry a telegram_id, so callers resolve (or
// silently register) the corresponding user this way rather than
// requiring a separate "register user" step.
func (s *UserStore) GetOrCreateByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	const q = `
		insert into users (telegram_id)
		values ($1)
		on conflict (telegram_id) do update set updated_at = now()
		returning id::text, telegram_id, is_active, created_at, updated_at
	`

	var u model.User
	err := s.pool.QueryRow(ctx, q, telegramID).Scan(
		&u.ID, &u.TelegramID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user by telegram id: %w", err)
	}
	return &u, nil
}

// GetByID loads a user by its internal (uuid) id.
func (s *UserStore) GetByID(ctx context.Context, id string) (*model.User, error) {
	const q = `
		select id::text, telegram_id, is_active, created_at, updated_at
		from users
		where id = $1::uuid
	`

	var u model.User
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.TelegramID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}
