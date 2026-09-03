// Package telegram wraps the Telegram Bot API client used to deliver
// rendered notifications to end users.
package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Client sends plain-text messages through a Telegram bot.
type Client struct {
	bot *tgbotapi.BotAPI
}

// New authenticates against the Telegram Bot API using token (see
// telegram.bot_token in configs/config.yaml / TELEGRAM_BOT_TOKEN env).
func New(token string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("telegram bot token is empty")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("init telegram bot: %w", err)
	}

	return &Client{bot: bot}, nil
}

// Send delivers text to the given Telegram chat id. Plain text is used
// (no ParseMode) so arbitrary rendered template output never fails on
// unescaped Markdown/HTML characters.
func (c *Client) Send(ctx context.Context, chatID int64, text string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := c.bot.Send(msg); err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	return nil
}
