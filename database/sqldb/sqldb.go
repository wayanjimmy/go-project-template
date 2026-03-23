package sqldb

import (
	"context"
	"fmt"
	"go-project-template/config"
	"go-project-template/database/migrations"
	"go-project-template/logger"
	"net/url"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct {
	db   *sqlx.DB
	conf *config.Config
	log  *logger.Logger
}

func Open(config *config.Config, log *logger.Logger) (*DB, error) {
	db, err := sqlx.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection failed: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres failed: %w", err)
	}

	return &DB{
		db:   db,
		conf: config,
		log:  log,
	}, nil
}

func RunMigrations(dbURL string) error {
	u, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("parse database URL: %w", err)
	}

	m := dbmate.New(u)
	m.FS = migrations.Files
	m.MigrationsDir = []string{"."}
	m.AutoDumpSchema = false

	return m.Migrate()
}

func (d *DB) SQL() *sqlx.DB {
	if d == nil {
		return nil
	}
	return d.db
}

func (d *DB) Close(ctx context.Context) {
	if d == nil || d.db == nil {
		return
	}

	if err := d.db.Close(); err != nil {
		d.log.Error(ctx, "database", "error", err.Error())
	}
}
