package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Telegram TelegramConfig `mapstructure:"telegram"`
	SSE      SSEConfig      `mapstructure:"sse"`
	Worker   WorkerConfig   `mapstructure:"worker"`
}

type AppConfig struct {
	Name     string `mapstructure:"name"`
	Host     string `mapstructure:"host"`
	GRPCPort int    `mapstructure:"grpc_port"`
	HTTPPort int    `mapstructure:"http_port"`
	Env      string `mapstructure:"env"`
	// APIKey, when set, is required as the X-API-Key header on every
	// /api/v1 request. Empty disables auth (local dev only).
	APIKey string `mapstructure:"api_key"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Password      string `mapstructure:"password"`
	DB            int    `mapstructure:"db"`
	StreamName    string `mapstructure:"stream_name"`
	ConsumerGroup string `mapstructure:"consumer_group"`
}

type TelegramConfig struct {
	BotToken       string  `mapstructure:"bot_token"`
	ChatIDs        []int64 `mapstructure:"chat_ids"`
	TimeoutSeconds int     `mapstructure:"timeout_seconds"`
}

type SSEConfig struct {
	HeartbeatSeconds int `mapstructure:"heartbeat_seconds"`
}

type WorkerConfig struct {
	Concurrency       int `mapstructure:"concurrency"`
	RetryAttempts     int `mapstructure:"retry_attempts"`
	RetryDelaySeconds int `mapstructure:"retry_delay_seconds"`
}

func Load(configPath string) (*Config, error) {
	// Load .env into the process environment for local `go run`/`go run
	// ./cmd/migrate` usage. docker-compose already injects these vars
	// directly, and a missing .env (e.g. in the built image) is fine —
	// godotenv.Load's error is intentionally ignored.
	_ = godotenv.Load()

	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	overrideFromEnv(&cfg)

	return &cfg, nil
}

func overrideFromEnv(cfg *Config) {
	if value := os.Getenv("APP_ENV"); value != "" {
		cfg.App.Env = value
	}
	if value := os.Getenv("APP_API_KEY"); value != "" {
		cfg.App.APIKey = value
	}

	if value := os.Getenv("POSTGRES_USER"); value != "" {
		cfg.Database.User = value
	}
	if value := os.Getenv("POSTGRES_PASSWORD"); value != "" {
		cfg.Database.Password = value
	}
	if value := os.Getenv("POSTGRES_DB"); value != "" {
		cfg.Database.Name = value
	}
	if value := os.Getenv("DATABASE_HOST"); value != "" {
		cfg.Database.Host = value
	}

	if value := os.Getenv("REDIS_PASSWORD"); value != "" {
		cfg.Redis.Password = value
	}
	if value := os.Getenv("REDIS_HOST"); value != "" {
		cfg.Redis.Host = value
	}

	if value := os.Getenv("TELEGRAM_BOT_TOKEN"); value != "" {
		cfg.Telegram.BotToken = value
	}
}
