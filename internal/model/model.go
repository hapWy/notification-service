// Package model holds the domain types shared across the notification
// service (API layer, storage layer, and the background worker).
package model

import "time"

// NotificationStatus mirrors the `notification_status` Postgres enum
// defined in migrations/003_create_notifications.sql.
type NotificationStatus string

const (
	StatusPending    NotificationStatus = "pending"
	StatusProcessing NotificationStatus = "processing"
	StatusSent       NotificationStatus = "sent"
	StatusDelivered  NotificationStatus = "delivered"
	StatusFailed     NotificationStatus = "failed"
)

// User maps an internal identity to the Telegram chat that notifications
// are delivered to.
type User struct {
	ID         string    `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Template is a reusable, named message body. Bodies are rendered with
// Go's text/template syntax (e.g. "Hello {{.name}}") against the params
// supplied on a notification request.
type Template struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Notification is a single request to deliver a rendered template to a
// user. It progresses through NotificationStatus values as the worker
// picks it up and sends it via Telegram.
type Notification struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	TemplateID   string                 `json:"template_id"`
	Status       NotificationStatus     `json:"status"`
	Params       map[string]interface{} `json:"params"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// NotificationLog records one status transition of a Notification, for
// audit/debugging purposes.
type NotificationLog struct {
	ID             string              `json:"id"`
	NotificationID string              `json:"notification_id"`
	OldStatus      *NotificationStatus `json:"old_status,omitempty"`
	NewStatus      NotificationStatus  `json:"new_status"`
	CreatedAt      time.Time           `json:"created_at"`
}
