package logger

import (
	"context"
	"go-project-template/buildinfo"
)

func BuildInfo(ctx context.Context, log *Logger) {
	if log == nil {
		return
	}

	info := buildinfo.Get()
	log.Info(ctx, "build info",
		"commit", info.Commit,
		"build_date", info.BuildDate,
		"go_version", info.GoVersion,
		"platform", info.Platform,
	)
}
