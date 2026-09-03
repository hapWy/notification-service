// Command migrate applies (or reverts) the SQL migrations in migrations/
// against the database configured in configs/config.yaml.
//
// Usage (from the repo root):
//
//	go run ./cmd/migrate                 # apply all pending migrations
//	go run ./cmd/migrate -direction down # revert the last migration
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/wwoes/notification-service/internal/config"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	flag.Parse()

	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.SSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set goose dialect: %v", err)
	}

	ctx := context.Background()
	switch *direction {
	case "up":
		if err := goose.UpContext(ctx, db, "migrations"); err != nil {
			log.Fatalf("migrate up: %v", err)
		}
	case "down":
		if err := goose.DownContext(ctx, db, "migrations"); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
	default:
		log.Fatalf("unknown -direction %q (want \"up\" or \"down\")", *direction)
	}

	log.Println("migration complete")
}
