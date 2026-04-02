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
	"go-project-template/setup"
	"go-project-template/worker"
	"go-project-template/workflows"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	workflowworker "github.com/cschleiden/go-workflows/worker"
	"golang.org/x/sync/errgroup"
)

func main() {
	correlationIDFn := requestid.FromContext

	ctx := context.Background()

	buildinfo.Service = "APP-WORKER"
	log := logger.New(os.Stdout, logger.LevelInfo, buildinfo.Service, correlationIDFn)

	logger.BuildInfo(ctx, log)

	err := runWorker(ctx, log)

	if err != nil {
		log.Error(ctx, "startup", "error", err.Error())
	}
}

func runWorker(ctx context.Context, log *logger.Logger) error {
	log.Info(ctx, "startup")

	cfg := &config.Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),

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

	wb := env.WorkflowBackend()
	if wb == nil {
		return errors.New("workflow backend not configured")
	}

	repos := repository.NewPostgresRepositories(db, encryptor, log)

	watermillLogger := logger.NewWatermillAdapter(log)
	pubSub := gochannel.NewGoChannel(gochannel.Config{}, watermillLogger)

	decoratedPub, err := publisher.CorrelationIDDecorator()(pubSub)
	if err != nil {
		return fmt.Errorf("failed to decorate publisher: %w", err)
	}
	eventPub := publisher.NewWatermillPublisher(decoratedPub)

	watermillRouter, err := worker.NewRouter(pubSub, repos.UserIndexer, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create worker router: %w", err)
	}

	w := workflowworker.New(wb, nil)
	if err := workflows.Register(w, workflows.Dependencies{UserRepo: repos.UserRepo, Publisher: eventPub, Log: log}); err != nil {
		return fmt.Errorf("register workflows: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, gctx := errgroup.WithContext(runCtx)

	g.Go(func() error {
		log.Info(gctx, "startup", "status", "workflow worker starting")
		if err := w.Start(gctx); err != nil {
			return fmt.Errorf("workflow worker start: %w", err)
		}
		<-gctx.Done()
		return w.WaitForCompletion()
	})

	g.Go(func() error {
		log.Info(gctx, "startup", "status", "watermill router starting")
		if err := watermillRouter.Run(gctx); err != nil {
			return fmt.Errorf("watermill router error: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sig)

		select {
		case <-sig:
			log.Info(gctx, "shutdown", "status", "stopping")
		case <-gctx.Done():
		}

		cancel()

		if err := watermillRouter.Close(); err != nil {
			return fmt.Errorf("watermill router close: %w", err)
		}

		return nil
	})

	return g.Wait()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
