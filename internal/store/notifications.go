package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wwoes/notification-service/internal/model"
)

// NotificationStore persists notification requests and their status
// history (notification_logs).
type NotificationStore struct {
	pool *pgxpool.Pool
}

func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore {
	return &NotificationStore{pool: pool}
}

// Create inserts a new notification in the "pending" status and its
// corresponding first log entry, in a single transaction.
func (s *NotificationStore) Create(ctx context.Context, userID, templateID string, params map[string]interface{}) (*model.Notification, error) {
	if params == nil {
		params = map[string]interface{}{}
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	const insertNotification = `
		insert into notifications (user_id, template_id, status, params)
		values ($1::uuid, $2::uuid, 'pending', $3::jsonb)
		returning id::text, user_id::text, template_id::text, status, params, error_message, created_at, updated_at
	`

	var (
		n           model.Notification
		rawParams   []byte
		errorMsg    *string
		notifStatus string
	)
	err = tx.QueryRow(ctx, insertNotification, userID, templateID, paramsJSON).Scan(
		&n.ID, &n.UserID, &n.TemplateID, &notifStatus, &rawParams, &errorMsg, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert notification: %w", err)
	}
	n.Status = model.NotificationStatus(notifStatus)
	n.ErrorMessage = errorMsg
	if err := json.Unmarshal(rawParams, &n.Params); err != nil {
		return nil, fmt.Errorf("unmarshal params: %w", err)
	}

	const insertLog = `
		insert into notification_logs (notification_id, old_status, new_status)
		values ($1::uuid, null, $2::notification_status)
	`
	if _, err := tx.Exec(ctx, insertLog, n.ID, string(n.Status)); err != nil {
		return nil, fmt.Errorf("insert notification log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &n, nil
}

func (s *NotificationStore) GetByID(ctx context.Context, id string) (*model.Notification, error) {
	const q = `
		select id::text, user_id::text, template_id::text, status, params, error_message, created_at, updated_at
		from notifications
		where id = $1::uuid
	`

	var (
		n         model.Notification
		rawParams []byte
		status    string
	)
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&n.ID, &n.UserID, &n.TemplateID, &status, &rawParams, &n.ErrorMessage, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get notification by id: %w", err)
	}
	n.Status = model.NotificationStatus(status)
	if err := json.Unmarshal(rawParams, &n.Params); err != nil {
		return nil, fmt.Errorf("unmarshal params: %w", err)
	}
	return &n, nil
}

// UpdateStatus moves a notification to newStatus, records the transition
// in notification_logs, and stores errMsg (nil clears it) — all in one
// transaction so the notifications row and its audit trail never drift
// apart.
func (s *NotificationStore) UpdateStatus(ctx context.Context, id string, newStatus model.NotificationStatus, errMsg *string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	var oldStatus string
	const selectStatus = `select status from notifications where id = $1::uuid for update`
	if err := tx.QueryRow(ctx, selectStatus, id).Scan(&oldStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("lock notification: %w", err)
	}

	const updateNotification = `
		update notifications
		set status = $2::notification_status, error_message = $3, updated_at = now()
		where id = $1::uuid
	`
	if _, err := tx.Exec(ctx, updateNotification, id, string(newStatus), errMsg); err != nil {
		return fmt.Errorf("update notification status: %w", err)
	}

	const insertLog = `
		insert into notification_logs (notification_id, old_status, new_status)
		values ($1::uuid, $2::notification_status, $3::notification_status)
	`
	if _, err := tx.Exec(ctx, insertLog, id, oldStatus, string(newStatus)); err != nil {
		return fmt.Errorf("insert notification log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
