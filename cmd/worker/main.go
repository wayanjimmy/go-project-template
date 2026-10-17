package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go-project-template/alerting"
	"go-project-template/buildinfo"
	"go-project-template/config"
	"go-project-template/logger"
	"go-project-template/repository"
	"go-project-template/setup"
	"go-project-template/worker"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
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
	log.Info(ctx, "startup")

	cfg := &config.Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisAddr:   os.Getenv("REDIS_ADDR"),

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

	log.Info(ctx, "startup", "status", "worker router starting")
	return router.Run(ctx)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
