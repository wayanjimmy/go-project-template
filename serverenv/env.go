package serverenv

import (
	"context"
	"errors"
	"go-project-template/database/sqldb"
	"go-project-template/logger"
	"go-project-template/repository"

	"github.com/redis/go-redis/v9"
)

type Option func(*ServerEnv) *ServerEnv

type ServerEnv struct {
	database      *sqldb.DB
	rdb           *redis.Client
	dataEncryptor repository.Encryptor
}

func (s *ServerEnv) Database() *sqldb.DB {
	if s == nil {
		return nil
	}
	return s.database
}

func (s *ServerEnv) Redis() *redis.Client {
	if s == nil {
		return nil
	}
	return s.rdb
}

func (s *ServerEnv) DataEncryptor() repository.Encryptor {
	if s == nil {
		return nil
	}
	return s.dataEncryptor
}

func New(ctx context.Context, log *logger.Logger, opts ...Option) *ServerEnv {
	env := &ServerEnv{}

	for _, f := range opts {
		env = f(env)
	}

	return env
}

func WithDatabase(database *sqldb.DB) Option {
	return func(s *ServerEnv) *ServerEnv {
		s.database = database
		return s
	}
}

func WithRedis(rdb *redis.Client) Option {
	return func(s *ServerEnv) *ServerEnv {
		s.rdb = rdb
		return s
	}
}

func WithDataEncryptor(enc repository.Encryptor) Option {
	return func(s *ServerEnv) *ServerEnv {
		s.dataEncryptor = enc
		return s
	}
}

// func WithRedis

// func WithElastic and the list goes on ...

func (s *ServerEnv) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}

	var errs []error

	if s.database != nil {
		s.database.Close(ctx)
	}

	if s.rdb != nil {
		if err := s.rdb.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
