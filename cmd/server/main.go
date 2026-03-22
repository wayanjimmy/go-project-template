package main

import (
	"context"
	"errors"
	"fmt"
	"go-project-template/buildinfo"
	"go-project-template/config"
	"go-project-template/logger"
	"go-project-template/publisher"
	"go-project-template/repository"
	"go-project-template/requestid"
	"go-project-template/server"
	"go-project-template/service"
	"go-project-template/setup"
	"go-project-template/worker"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"golang.org/x/sync/errgroup"
)

func main() {
	correlationIDFn := requestid.FromContext

	ctx := context.Background()

	buildinfo.Service = "APP-API"
	log := logger.New(os.Stdout, logger.LevelInfo, buildinfo.Service, correlationIDFn)

	logger.BuildInfo(ctx, log)

	err := runServer(ctx, log)

	if err != nil {
		log.Error(ctx, "startup", "error", err.Error())
	}
}

func runServer(ctx context.Context, log *logger.Logger) error {
	log.Info(ctx, "startup")

	cfg := &config.Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),

		RedisAddr:  os.Getenv("REDIS_ADDR"),
		ServerPort: envOrDefault("SERVER_PORT", "3030"),

		SecretType:             envOrDefault("SECRET_DRIVER", "FILESYSTEM"),
		SecretRoot:             envOrDefault("SECRET_ROOT", ".secrets"),
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
	pubSub := gochannel.NewGoChannel(gochannel.Config{}, watermillLogger)

	decoratedPub, err := publisher.CorrelationIDDecorator()(pubSub)
	if err != nil {
		return fmt.Errorf("failed to decorate publisher: %w", err)
	}
	eventPub := publisher.NewWatermillPublisher(decoratedPub)

	userService := service.NewUserService(repos.UserRepo, eventPub, log)
	searchService := service.NewSearchService(repos.UserSearcher)

	watermillRouter, err := worker.NewRouter(pubSub, repos.UserIndexer, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create worker router: %w", err)
	}

	r := server.SetupRouter(userService, searchService, log)

	api := &http.Server{Addr: ":" + cfg.ServerPort, Handler: r}

	g, ctx := errgroup.WithContext(ctx)

	// Worker router
	g.Go(func() error {
		log.Info(ctx, "startup", "status", "worker router starting")
		if err := watermillRouter.Run(ctx); err != nil {
			return fmt.Errorf("worker router error: %w", err)
		}
		return nil
	})

	// HTTP server
	g.Go(func() error {
		log.Info(ctx, "startup", "status", "http server started", "addr", api.Addr)
		if err := api.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server error: %w", err)
		}
		return nil
	})

	// Graceful shutdown
	g.Go(func() error {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-shutdown:
			log.Info(ctx, "shutdown", "status", "stopping")
		case <-ctx.Done():
			// Context canceled by another group member's error
		}

		shutdownCtx, cancel := context.WithTimeoutCause(context.Background(), 20*time.Second, errors.New("graceful shutdown timeout"))
		defer cancel()

		var errs []error
		if err := watermillRouter.Close(); err != nil {
			errs = append(errs, fmt.Errorf("worker router close failed: %w", err))
		}

		if err := api.Shutdown(shutdownCtx); err != nil {
			_ = api.Close()
			errs = append(errs, fmt.Errorf("could not shutdown gracefully: %w", err))
		}

		return errors.Join(errs...)
	})

	return g.Wait()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
