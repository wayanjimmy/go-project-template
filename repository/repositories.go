package repository

import (
	"go-project-template/database/sqldb"
	"go-project-template/logger"
	"go-project-template/service"
)

type Repositories struct {
	UserRepo     service.UserRepository
	UserIndexer  service.UserIndexer
	UserSearcher service.UserSearcher
}

func NewPostgresRepositories(db *sqldb.DB, encryptor Encryptor, log *logger.Logger) *Repositories {
	userRepo := NewPostgresUserRepository(db, encryptor, log)
	userSearchRepo := NewPostgresUserSearchRepository(db, log)

	return &Repositories{
		UserRepo:     userRepo,
		UserIndexer:  userSearchRepo,
		UserSearcher: userSearchRepo,
	}
}
