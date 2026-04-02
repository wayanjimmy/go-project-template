package serverenv

import (
	"context"
	"errors"
	"go-project-template/database/sqldb"
	"go-project-template/logger"
	"go-project-template/repository"

	workflowbackend "github.com/cschleiden/go-workflows/backend"
)

type Option func(*ServerEnv) *ServerEnv

type ServerEnv struct {
	database        *sqldb.DB
	workflowBackend workflowbackend.Backend
	dataEncryptor   repository.Encryptor
}

func (s *ServerEnv) Database() *sqldb.DB {
	if s == nil {
		return nil
	}
	return s.database
}

func (s *ServerEnv) DataEncryptor() repository.Encryptor {
	if s == nil {
		return nil
	}
	return s.dataEncryptor
}

func (s *ServerEnv) WorkflowBackend() workflowbackend.Backend {
	if s == nil {
		return nil
	}
	return s.workflowBackend
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

func WithDataEncryptor(enc repository.Encryptor) Option {
	return func(s *ServerEnv) *ServerEnv {
		s.dataEncryptor = enc
		return s
	}
}

func WithWorkflowBackend(workflowBackend workflowbackend.Backend) Option {
	return func(s *ServerEnv) *ServerEnv {
		s.workflowBackend = workflowBackend
		return s
	}
}

// func WithElastic and the list goes on ...

func (s *ServerEnv) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}

	var errs []error

	if s.database != nil {
		s.database.Close(ctx)
	}

	if s.workflowBackend != nil {
		if err := s.workflowBackend.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
