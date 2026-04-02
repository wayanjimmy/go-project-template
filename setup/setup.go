package setup

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	appconfig "go-project-template/config"
	"go-project-template/database/sqldb"
	"go-project-template/key"
	"go-project-template/logger"
	"go-project-template/repository"
	"go-project-template/serverenv"

	workflowbackend "github.com/cschleiden/go-workflows/backend"
	workflowpostgres "github.com/cschleiden/go-workflows/backend/postgres"
)

// Validatable is implemented by config structs that can self-validate.
type Validatable interface {
	Validate() error
}

// DatabaseConfigProvider exposes database config.
type DatabaseConfigProvider interface {
	DatabaseConfig() *appconfig.Config
}

// KeyManagerConfigProvider exposes key manager config.
type KeyManagerConfigProvider interface {
	KeyManagerConfig() *appconfig.Config
}

func Setup(ctx context.Context, log *logger.Logger, cfg any) (*serverenv.ServerEnv, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if validatable, ok := cfg.(Validatable); ok {
		if err := validatable.Validate(); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}
	}

	var serverEnvOpts []serverenv.Option

	if provider, ok := cfg.(DatabaseConfigProvider); ok {
		dbCfg := provider.DatabaseConfig()
		if dbCfg != nil && dbCfg.DatabaseURL != "" {
			log.Info(ctx, "setup", "status", "connecting to database")

			db, err := sqldb.Open(dbCfg, log)
			if err != nil {
				return nil, fmt.Errorf("unable to connect to database: %w", err)
			}

			log.Info(ctx, "setup", "status", "running database migrations")

			if err := sqldb.RunMigrations(dbCfg.DatabaseURL); err != nil {
				db.Close(ctx)
				return nil, fmt.Errorf("run db migrations: %w", err)
			}

			workflowDBCfg, err := parseWorkflowPostgresConfig(dbCfg.DatabaseURL)
			if err != nil {
				db.Close(ctx)
				return nil, fmt.Errorf("parse workflow database url: %w", err)
			}

			wb := workflowpostgres.NewPostgresBackend(
				workflowDBCfg.host,
				workflowDBCfg.port,
				workflowDBCfg.user,
				workflowDBCfg.password,
				workflowDBCfg.database,
				workflowpostgres.WithApplyMigrations(false),
				workflowpostgres.WithBackendOptions(
					workflowbackend.WithLogger(slog.Default()),
				),
			)

			serverEnvOpts = append(serverEnvOpts, serverenv.WithDatabase(db), serverenv.WithWorkflowBackend(wb))
		}
	}

	if provider, ok := cfg.(KeyManagerConfigProvider); ok {
		keyCfg := provider.KeyManagerConfig()
		if keyCfg != nil && keyCfg.SecretRoot != "" && keyCfg.SecretKeyID != "" {
			log.Info(ctx, "setup", "status", "init keys manager")

			km, err := key.KeyManagerFor(ctx, &key.Config{
				Type:           keyCfg.SecretType,
				FilesystemRoot: keyCfg.SecretRoot,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to setup key manager: %w", err)
			}

			if err := ensureEncryptionKey(ctx, km, keyCfg); err != nil {
				return nil, err
			}

			dataEncryptor := repository.NewDataEncryptor(km, keyCfg.SecretKeyID)
			serverEnvOpts = append(serverEnvOpts, serverenv.WithDataEncryptor(dataEncryptor))
		}
	}

	return serverenv.New(ctx, log, serverEnvOpts...), nil
}

type workflowPostgresConfig struct {
	host     string
	port     int
	user     string
	password string
	database string
}

func parseWorkflowPostgresConfig(databaseURL string) (*workflowPostgresConfig, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database url: %w", err)
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("unsupported database url scheme %q", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("database host is required")
	}

	port := 5432
	if p := u.Port(); p != "" {
		parsedPort, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid database port %q: %w", p, err)
		}
		port = parsedPort
	}

	user := ""
	password := ""
	if u.User != nil {
		user = u.User.Username()
		password, _ = u.User.Password()
	}
	if user == "" {
		return nil, fmt.Errorf("database user is required")
	}

	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		return nil, fmt.Errorf("database name is required")
	}

	return &workflowPostgresConfig{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		database: database,
	}, nil
}

func ensureEncryptionKey(ctx context.Context, km key.KeyManager, cfg *appconfig.Config) error {
	if strings.Count(cfg.SecretKeyID, "/") != 1 {
		return nil
	}

	ekm, ok := km.(key.EncryptionKeyManager)
	if !ok {
		return nil
	}

	if _, err := ekm.CreateEncryptionKey(ctx, cfg.SecretEncryptionParent, cfg.SecretEncryptionName); err != nil {
		return fmt.Errorf("failed to create encryption key: %w", err)
	}

	versions, err := km.ListKeyVersions(ctx, cfg.SecretKeyID)
	if err == nil && len(versions) > 0 {
		return nil
	}

	if _, err := ekm.CreateKeyVersion(ctx, cfg.SecretKeyID); err != nil {
		return fmt.Errorf("failed to create encryption key version: %w", err)
	}

	return nil
}
