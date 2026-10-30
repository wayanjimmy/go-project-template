package testutil

import (
	"context"
	"os"
	"testing"

	"go-project-template/config"
	"go-project-template/database/sqldb"
	"go-project-template/logger"
)

// OpenMigratedDB uses TEST_DATABASE_URL when supplied, otherwise a disposable container.
// TEST_DATABASE_URL must point to a disposable test database: fixtures reset its tables.
func OpenMigratedDB(t *testing.T, log *logger.Logger) (context.Context, *sqldb.DB) {
	t.Helper()

	if log == nil {
		log = logger.Noop()
	}

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = StartPostgresContainer(t)
	}
	if err := sqldb.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	db, err := sqldb.Open(&config.Config{DatabaseURL: dbURL}, log)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	ctx := context.Background()
	t.Cleanup(func() {
		db.Close(ctx)
	})

	return ctx, db
}
