// Package notification orchestrates turning an API request into a
// persisted, queued notification.
package notification

import (
	"context"
	"fmt"

	"github.com/wwoes/notification-service/internal/model"
	"github.com/wwoes/notification-service/internal/queue"
	"github.com/wwoes/notification-service/internal/store"
)

// Service is the entry point the API handlers call: it resolves (or
// registers) the target user, resolves the template, records the
// notification, and hands it off to the worker via the queue.
type Service struct {
	users         *store.UserStore
	templates     *store.TemplateStore
	notifications *store.NotificationStore
	queue         *queue.Queue
}

func NewService(users *store.UserStore, templates *store.TemplateStore, notifications *store.NotificationStore, q *queue.Queue) *Service {
	return &Service{
		users:         users,
		templates:     templates,
		notifications: notifications,
		queue:         q,
	}
}

// CreateInput is the caller-facing request to send a notification.
type CreateInput struct {
	TelegramID int64
	Template   string
	Params     map[string]interface{}
}

// CreateNotification upserts the user by Telegram id, resolves the named
// template, persists a pending notification, and publishes it to the
// worker queue. It returns store.ErrNotFound if Template does not exist.
func (s *Service) CreateNotification(ctx context.Context, in CreateInput) (*model.Notification, error) {
	user, err := s.users.GetOrCreateByTelegramID(ctx, in.TelegramID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	tmpl, err := s.templates.GetByName(ctx, in.Template)
	if err != nil {
		return nil, fmt.Errorf("resolve template: %w", err)
	}

	n, err := s.notifications.Create(ctx, user.ID, tmpl.ID, in.Params)
	if err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	if err := s.queue.Publish(ctx, n.ID); err != nil {
		return nil, fmt.Errorf("publish notification: %w", err)
	}

	return n, nil
}

// GetNotification looks up a notification's current status.
func (s *Service) GetNotification(ctx context.Context, id string) (*model.Notification, error) {
	return s.notifications.GetByID(ctx, id)
}
