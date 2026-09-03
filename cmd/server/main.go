package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wwoes/notification-service/internal/api"
	"github.com/wwoes/notification-service/internal/config"
	"github.com/wwoes/notification-service/internal/notification"
	"github.com/wwoes/notification-service/internal/queue"
	"github.com/wwoes/notification-service/internal/store"
	"github.com/wwoes/notification-service/internal/telegram"
	"github.com/wwoes/notification-service/internal/worker"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := store.NewPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	redisClient := queue.NewClient(cfg.Redis)
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}

	tgClient, err := telegram.New(cfg.Telegram.BotToken)
	if err != nil {
		log.Fatalf("connect telegram: %v", err)
	}

	users := store.NewUserStore(pool)
	templates := store.NewTemplateStore(pool)
	notifications := store.NewNotificationStore(pool)
	q := queue.New(redisClient, cfg.Redis)

	notifService := notification.NewService(users, templates, notifications, q)
	notifWorker := worker.New(q, users, templates, notifications, tgClient, cfg.Worker)

	workerCtx, cancelWorker := context.WithCancel(ctx)
	var workerWG sync.WaitGroup
	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		if err := notifWorker.Run(workerCtx); err != nil {
			log.Printf("worker stopped: %v", err)
		}
	}()

	router := buildRouter(cfg, notifService, templates)

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("%s on %s", cfg.App.Name, server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	waitForShutdown(ctx, server)

	cancelWorker()
	workerWG.Wait()
}

func buildRouter(cfg *config.Config, notifService *notification.Service, templates *store.TemplateStore) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": cfg.App.Name,
			"status":  "ok",
			"env":     cfg.App.Env,
		})
	})

	// Swagger UI (docs/index.html, CDN-loaded) + the OpenAPI spec it reads.
	// Both are static files, served relative to the working directory the
	// binary is run from (same convention as configs/config.yaml).
	router.StaticFile("/openapi.json", "docs/openapi.json")
	router.StaticFile("/docs", "docs/index.html")

	v1 := router.Group("/api/v1", api.APIKeyAuth(cfg.App.APIKey))
	api.NewHandler(notifService, templates).Register(v1)

	return router
}

func waitForShutdown(ctx context.Context, server *http.Server) {
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("server shutting down")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("force close failed: %v", closeErr)
		}
	}
}
