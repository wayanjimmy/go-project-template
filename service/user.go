package service

import (
	"context"
	"fmt"
	"go-project-template/entity"
	"go-project-template/event"
	"go-project-template/logger"
	"go-project-template/transaction"
	"strings"

	"github.com/google/uuid"
)

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	List(ctx context.Context, limit, offset int) ([]entity.User, error)
	Save(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id string) error
	RecordEvent(ctx context.Context, topic string, payload any) error
	ExecuteUnderTransaction(tx transaction.Transaction) (UserRepository, error)
}

type UserService interface {
	Create(ctx context.Context, name, email, address string) (*entity.User, error)
	Update(ctx context.Context, id, name, email, address string) (*entity.User, error)
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*entity.User, error)
	List(ctx context.Context, limit, offset int) ([]entity.User, error)
}

type userService struct {
	repo     UserRepository
	beginner transaction.Beginner
	log      *logger.Logger
}

func NewUserService(repo UserRepository, beginner transaction.Beginner, log *logger.Logger) UserService {
	if log == nil {
		log = logger.Noop()
	}

	return &userService{repo: repo, beginner: beginner, log: log}
}

func (s *userService) Create(ctx context.Context, name, email, address string) (*entity.User, error) {
	s.log.Info(ctx, "service.user.create")
	id := uuid.NewString()
	user := &entity.User{ID: id, Name: name, Email: email, Address: address}

	err := transaction.ExecuteUnderTransaction(ctx, s.beginner, s.log, func(tx transaction.Transaction) error {
		repo, err := s.repo.ExecuteUnderTransaction(tx)
		if err != nil {
			return err
		}
		if err := repo.Save(ctx, user); err != nil {
			return err
		}
		return recordUserEvent(ctx, repo, event.UserCreated, user)
	})
	if err != nil {
		return nil, fmt.Errorf("create user failed: %w", err)
	}

	return user, nil
}

func (s *userService) Update(ctx context.Context, id, name, email, address string) (*entity.User, error) {
	s.log.Info(ctx, "service.user.update", "user_id", id)
	user := &entity.User{ID: id, Name: name, Email: email, Address: address}

	err := transaction.ExecuteUnderTransaction(ctx, s.beginner, s.log, func(tx transaction.Transaction) error {
		txRepo, err := s.repo.ExecuteUnderTransaction(tx)
		if err != nil {
			return err
		}
		if _, err := txRepo.FindByID(ctx, id); err != nil {
			return fmt.Errorf("find user failed: %w", err)
		}
		if err := txRepo.Save(ctx, user); err != nil {
			return err
		}
		return recordUserEvent(ctx, txRepo, event.UserUpdated, user)
	})
	if err != nil {
		return nil, fmt.Errorf("update user failed: %w", err)
	}

	return user, nil
}

func (s *userService) Delete(ctx context.Context, id string) error {
	s.log.Info(ctx, "service.user.delete", "user_id", id)

	err := transaction.ExecuteUnderTransaction(ctx, s.beginner, s.log, func(tx transaction.Transaction) error {
		repo, err := s.repo.ExecuteUnderTransaction(tx)
		if err != nil {
			return err
		}
		if err := repo.Delete(ctx, id); err != nil {
			return err
		}
		return repo.RecordEvent(ctx, event.UserDeleted, event.UserDeletedEvent{UserID: id})
	})
	if err != nil {
		return fmt.Errorf("delete user failed: %w", err)
	}

	return nil
}

func (s *userService) FindByID(ctx context.Context, id string) (*entity.User, error) {
	s.log.Info(ctx, "service.user.find_by_id", "user_id", id)
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find user failed: %w", err)
	}

	return user, nil
}

func (s *userService) List(ctx context.Context, limit, offset int) ([]entity.User, error) {
	s.log.Info(ctx, "service.user.list", "limit", limit, "offset", offset)
	items, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users failed: %w", err)
	}
	return items, nil
}

func recordUserEvent(ctx context.Context, repo UserRepository, topic string, user *entity.User) error {
	evt := event.UserUpsertedEvent{
		UserID:   user.ID,
		Name:     user.Name,
		Email:    user.Email,
		Document: strings.Join([]string{user.Name, user.Email}, " "),
	}
	return repo.RecordEvent(ctx, topic, evt)
}
