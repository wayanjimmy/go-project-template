package setup

import (
	"context"
	"fmt"
	appconfig "go-project-template/config"
	"go-project-template/database/sqldb"
	"go-project-template/key"
	"go-project-template/logger"
	"go-project-template/repository"
	"go-project-template/serverenv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Validatable is implemented by config structs that can self-validate.
type Validatable interface {
	Validate() error
}

// DatabaseConfigProvider exposes database config.
type DatabaseConfigProvider interface {
	DatabaseConfig() *appconfig.Config
}

// RedisConfigProvider exposes redis config.
type RedisConfigProvider interface {
	RedisConfig() *appconfig.Config
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

			serverEnvOpts = append(serverEnvOpts, serverenv.WithDatabase(db))
		}
	}

	if provider, ok := cfg.(RedisConfigProvider); ok {
		redisCfg := provider.RedisConfig()
		if redisCfg != nil && redisCfg.RedisAddr != "" {
			log.Info(ctx, "setup", "status", "connecting to redis")

			rdb := redis.NewClient(&redis.Options{Addr: redisCfg.RedisAddr})
			serverEnvOpts = append(serverEnvOpts, serverenv.WithRedis(rdb))
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
