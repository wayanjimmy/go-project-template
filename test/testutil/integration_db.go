package testutil

import (
	"context"
	"testing"

	"go-project-template/config"
	"go-project-template/database/sqldb"
	"go-project-template/logger"
)

// OpenMigratedDB starts a disposable postgres container, runs migrations, and opens sqldb.DB.
func OpenMigratedDB(t *testing.T, log *logger.Logger) (context.Context, *sqldb.DB) {
	t.Helper()

	if log == nil {
		log = logger.Noop()
	}

	dbURL := StartPostgresContainer(t)
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
