package service

import (
	"context"
	"fmt"
	"go-project-template/entity"
	"go-project-template/event"
	"go-project-template/logger"
	"strings"

	"github.com/google/uuid"
)

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	List(ctx context.Context, limit, offset int) ([]entity.User, error)
	Save(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id string) error
}

type UserService interface {
	Create(ctx context.Context, name, email, address string) (*entity.User, error)
	Update(ctx context.Context, id, name, email, address string) (*entity.User, error)
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*entity.User, error)
	List(ctx context.Context, limit, offset int) ([]entity.User, error)
}

type userService struct {
	repo      UserRepository
	publisher EventPublisher
	log       *logger.Logger
}

func NewUserService(repo UserRepository, publisher EventPublisher, log *logger.Logger) UserService {
	if log == nil {
		log = logger.Noop()
	}

	return &userService{repo: repo, publisher: publisher, log: log}
}

func (s *userService) Create(ctx context.Context, name, email, address string) (*entity.User, error) {
	s.log.Info(ctx, "service.user.create")
	id := uuid.NewString()
	return s.saveAndPublish(ctx, event.UserCreated, id, name, email, address)
}

func (s *userService) Update(ctx context.Context, id, name, email, address string) (*entity.User, error) {
	s.log.Info(ctx, "service.user.update", "user_id", id)
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return nil, fmt.Errorf("find user failed: %w", err)
	}
	return s.saveAndPublish(ctx, event.UserUpdated, id, name, email, address)
}

func (s *userService) Delete(ctx context.Context, id string) error {
	s.log.Info(ctx, "service.user.delete", "user_id", id)
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user failed: %w", err)
	}

	if err := s.publisher.Publish(ctx, event.UserDeleted, event.UserDeletedEvent{UserID: id}); err != nil {
		return fmt.Errorf("publish user deleted event failed: %w", err)
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

func (s *userService) saveAndPublish(ctx context.Context, topic, id, name, email, address string) (*entity.User, error) {
	user := &entity.User{ID: id, Name: name, Email: email, Address: address}
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("save user failed: %w", err)
	}

	evt := event.UserUpsertedEvent{
		UserID:   id,
		Name:     name,
		Email:    email,
		Address:  address,
		Document: strings.Join([]string{name, email}, " "),
	}
	if err := s.publisher.Publish(ctx, topic, evt); err != nil {
		return nil, fmt.Errorf("publish user event failed: %w", err)
	}

	return user, nil
}
