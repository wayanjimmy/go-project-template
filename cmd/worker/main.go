package main

import (
	"context"
	"go-project-template/buildinfo"
	"go-project-template/logger"
	"go-project-template/requestid"
	"os"
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

// runWorker is intentionally a no-op while using Watermill GoChannel.
// GoChannel is in-memory and requires publisher/subscriber in the same process,
// so event handling is started inside runServer.
func runWorker(ctx context.Context, log *logger.Logger) error {
	log.Info(ctx, "worker", "status", "no-op", "reason", "GoChannel requires in-process pub/sub; use rest mode")
	return nil
}
