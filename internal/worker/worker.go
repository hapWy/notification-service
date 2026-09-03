// Package worker consumes queued notifications and delivers them via
// Telegram, updating each notification's status as it goes.
package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/wwoes/notification-service/internal/config"
	"github.com/wwoes/notification-service/internal/model"
	"github.com/wwoes/notification-service/internal/queue"
	"github.com/wwoes/notification-service/internal/render"
	"github.com/wwoes/notification-service/internal/store"
	"github.com/wwoes/notification-service/internal/telegram"
)

const (
	readCount = 1
	readBlock = 5 * time.Second
)

// Worker is a pool of goroutines sharing one Redis consumer group; each
// goroutine claims and processes one notification at a time.
type Worker struct {
	queue         *queue.Queue
	users         *store.UserStore
	templates     *store.TemplateStore
	notifications *store.NotificationStore
	telegram      *telegram.Client
	cfg           config.WorkerConfig
}

func New(q *queue.Queue, users *store.UserStore, templates *store.TemplateStore, notifications *store.NotificationStore, tg *telegram.Client, cfg config.WorkerConfig) *Worker {
	return &Worker{
		queue:         q,
		users:         users,
		templates:     templates,
		notifications: notifications,
		telegram:      tg,
		cfg:           cfg,
	}
}

// Run ensures the consumer group exists and blocks running the worker
// pool until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.queue.EnsureGroup(ctx); err != nil {
		return fmt.Errorf("ensure consumer group: %w", err)
	}

	concurrency := w.cfg.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		consumerName := fmt.Sprintf("worker-%d", i)
		go func(name string) {
			defer wg.Done()
			w.loop(ctx, name)
		}(consumerName)
	}
	wg.Wait()
	return nil
}

func (w *Worker) loop(ctx context.Context, consumerName string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		streams, err := w.queue.ReadGroup(ctx, consumerName, readCount, readBlock)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("worker %s: read group: %v", consumerName, err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				w.handleMessage(ctx, msg)
			}
		}
	}
}

func (w *Worker) handleMessage(ctx context.Context, msg redis.XMessage) {
	notificationID, _ := msg.Values["notification_id"].(string)
	if notificationID == "" {
		log.Printf("worker: message %s has no notification_id, dropping", msg.ID)
		if err := w.queue.Ack(ctx, msg.ID); err != nil {
			log.Printf("worker: ack %s: %v", msg.ID, err)
		}
		return
	}

	if err := w.process(ctx, notificationID); err != nil {
		log.Printf("worker: notification %s failed: %v", notificationID, err)
	}

	if err := w.queue.Ack(ctx, msg.ID); err != nil {
		log.Printf("worker: ack %s: %v", msg.ID, err)
	}
}

func (w *Worker) process(ctx context.Context, notificationID string) error {
	n, err := w.notifications.GetByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("load notification: %w", err)
	}

	if err := w.notifications.UpdateStatus(ctx, n.ID, model.StatusProcessing, nil); err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	tmpl, err := w.templates.GetByID(ctx, n.TemplateID)
	if err != nil {
		return w.fail(ctx, n.ID, fmt.Errorf("load template: %w", err))
	}

	user, err := w.users.GetByID(ctx, n.UserID)
	if err != nil {
		return w.fail(ctx, n.ID, fmt.Errorf("load user: %w", err))
	}

	text, err := render.Render(tmpl.Body, n.Params)
	if err != nil {
		return w.fail(ctx, n.ID, fmt.Errorf("render template: %w", err))
	}

	if sendErr := w.sendWithRetry(ctx, user.TelegramID, text); sendErr != nil {
		return w.fail(ctx, n.ID, sendErr)
	}

	if err := w.notifications.UpdateStatus(ctx, n.ID, model.StatusSent, nil); err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	return nil
}

func (w *Worker) sendWithRetry(ctx context.Context, chatID int64, text string) error {
	attempts := w.cfg.RetryAttempts
	if attempts < 1 {
		attempts = 1
	}
	delay := time.Duration(w.cfg.RetryDelaySeconds) * time.Second

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		lastErr = w.telegram.Send(ctx, chatID, text)
		if lastErr == nil {
			return nil
		}
		if attempt < attempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return fmt.Errorf("send after %d attempt(s): %w", attempts, lastErr)
}

// fail records the notification as failed with cause's message and
// returns cause so the caller can log it.
func (w *Worker) fail(ctx context.Context, notificationID string, cause error) error {
	msg := cause.Error()
	if err := w.notifications.UpdateStatus(ctx, notificationID, model.StatusFailed, &msg); err != nil {
		log.Printf("worker: failed to persist failure state for %s: %v", notificationID, err)
	}
	return cause
}
