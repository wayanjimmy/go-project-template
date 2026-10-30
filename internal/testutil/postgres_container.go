package testutil

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// StartPostgresContainer starts a disposable Postgres test container and returns its DSN.
// If no container runtime is available, the test is skipped.
func StartPostgresContainer(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("app_test"),
		postgres.WithUsername("app"),
		postgres.WithPassword("app"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("skipping integration test: cannot start postgres testcontainer: %v", err)
	}

	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("build postgres DSN from testcontainer: %v", err)
	}

	return dsn
}
