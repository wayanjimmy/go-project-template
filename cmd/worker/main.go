package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-project-template/internal/alerting"
	"go-project-template/buildinfo"
	"go-project-template/client/webhook"
	"go-project-template/config"
	"go-project-template/logger"
	"go-project-template/internal/publisher"
	"go-project-template/repository"
	"go-project-template/setup"
	"go-project-template/worker"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx := context.Background()

	buildinfo.Service = "APP-WORKER"
	log, closeAlerts := alerting.NewLogger(ctx, os.Stdout)
	defer closeAlerts()

	logger.BuildInfo(ctx, log)

	if err := runWorker(ctx, log); err != nil {
		log.Error(ctx, "startup", "error", err.Error())
		// Flush pending alerts before exiting; os.Exit skips defers.
		closeAlerts()
		os.Exit(1)
	}
}

// runWorker consumes events via the worker router. When REDIS_ADDR is set it
// subscribes through Redis Streams so it can run as a standalone replica;
// otherwise it falls back to in-process pub/sub (which only sees events
// published by this same process).
func runWorker(ctx context.Context, log *logger.Logger) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Info(ctx, "startup")

	cfg := &config.Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		WebhookURL:    os.Getenv("WEBHOOK_URL"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),

		SecretType:             envOrDefault("SECRET_DRIVER", "FILESYSTEM"),
		SecretRoot:             os.Getenv("SECRET_ROOT"),
		SecretEncryptionParent: envOrDefault("SECRET_ENCRYPTION_PARENT", "local"),
		SecretEncryptionName:   envOrDefault("SECRET_ENCRYPTION_NAME", "user-data"),
	}

	env, err := setup.Setup(ctx, log, cfg)
	if err != nil {
		return fmt.Errorf("setup.Setup: %w", err)
	}
	defer env.Close(ctx)

	db := env.Database()
	if db == nil {
		return errors.New("database not configured")
	}

	encryptor := env.DataEncryptor()
	if encryptor == nil {
		return errors.New("data encryptor not configured")
	}

	repos := repository.NewPostgresRepositories(db, encryptor, log)
	watermillLogger := logger.NewWatermillAdapter(log)

	var inProcessSub message.Subscriber
	if env.Redis() == nil {
		log.Info(ctx, "startup", "status", "no REDIS_ADDR, using in-process pub/sub")
		inProcessSub = gochannel.NewGoChannel(gochannel.Config{}, watermillLogger)
	}

	router, err := worker.NewRouter(ctx, env.Redis(), inProcessSub, repos.UserIndexer, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create worker router: %w", err)
	}
	defer router.Close()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return router.Run(ctx) })
	if cfg.WebhookURL != "" {
		retryBase, err := time.ParseDuration(envOrDefault("WEBHOOK_RETRY_BASE", "5s"))
		if err != nil || retryBase < time.Second {
			return fmt.Errorf("WEBHOOK_RETRY_BASE must be a duration of at least 1s")
		}
		store := repository.NewWebhookRepository(db.SQL(), log)
		queue := publisher.NewAsynqQueue(cfg.RedisAddr)
		defer queue.Close()
		webhookRouter, err := worker.NewWebhookRouter(ctx, env.Redis(), worker.WebhookSubscriber(store, cfg.WebhookURL, log), watermillLogger)
		if err != nil {
			return err
		}
		defer webhookRouter.Close()
		g.Go(func() error { return webhookRouter.Run(ctx) })
		jobs, mux := worker.NewAsynqServer(cfg.RedisAddr, store, webhook.New(cfg.WebhookSecret), retryBase, log)
		if err := jobs.Start(mux); err != nil {
			return err
		}
		defer jobs.Shutdown()
		g.Go(func() error {
			return worker.Poll(ctx, log, "webhook.dispatch", func(ctx context.Context) error { return store.Dispatch(ctx, queue) })
		})
		log.Info(ctx, "startup", "status", "webhook subscriber and Asynq workers started", "queue", "webhooks", "concurrency", 4)
	}
	return g.Wait()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
