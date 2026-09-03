// Package queue implements the Redis Stream queue that decouples the API
// (which only records a notification) from the worker (which sends it).
package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/wwoes/notification-service/internal/config"
)

// NewClient builds a redis client from the app's redis config.
//
// ReadTimeout is set comfortably above the worker's XREADGROUP block
// duration: blocking stream reads wait server-side, and go-redis's
// default read timeout is shorter than that block, which would
// otherwise surface as spurious "i/o timeout" errors on every idle poll.
func NewClient(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:    cfg.Password,
		DB:          cfg.DB,
		ReadTimeout: 10 * time.Second,
	})
}

// Queue publishes notification ids onto a Redis Stream and lets workers
// consume them through a shared consumer group, so each notification is
// delivered by exactly one worker and survives a worker restart.
type Queue struct {
	client        *redis.Client
	stream        string
	consumerGroup string
}

func New(client *redis.Client, cfg config.RedisConfig) *Queue {
	return &Queue{
		client:        client,
		stream:        cfg.StreamName,
		consumerGroup: cfg.ConsumerGroup,
	}
}

// EnsureGroup creates the consumer group (and the stream, if missing).
// It is safe to call on every startup: an already-existing group is not
// an error.
func (q *Queue) EnsureGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.stream, q.consumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

// Publish enqueues a notification id for the worker pool to pick up.
func (q *Queue) Publish(ctx context.Context, notificationID string) error {
	err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		Values: map[string]interface{}{"notification_id": notificationID},
	}).Err()
	if err != nil {
		return fmt.Errorf("publish notification %s: %w", notificationID, err)
	}
	return nil
}

// ReadGroup blocks (up to block) waiting for new stream entries for
// consumerName within the shared consumer group. A nil, nil result means
// the block timed out with nothing to do — callers should just loop.
func (q *Queue) ReadGroup(ctx context.Context, consumerName string, count int64, block time.Duration) ([]redis.XStream, error) {
	res, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.consumerGroup,
		Consumer: consumerName,
		Streams:  []string{q.stream, ">"},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

// Ack marks a stream entry as processed so it won't be redelivered.
func (q *Queue) Ack(ctx context.Context, messageID string) error {
	return q.client.XAck(ctx, q.stream, q.consumerGroup, messageID).Err()
}
