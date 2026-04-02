package useronboarding

import (
	"context"
	"fmt"
	"go-project-template/entity"
	"go-project-template/event"
	"go-project-template/logger"
	"strings"
)

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	Save(ctx context.Context, user *entity.User) error
}

type Activities struct {
	UserRepo  UserRepository
	Publisher EventPublisher
	Log       *logger.Logger
}

func (a *Activities) CreatePendingUser(ctx context.Context, in Input) error {
	if a.Log != nil {
		a.Log.Info(ctx, "workflow.user_onboarding.create_pending_user", "user_id", in.UserID)
	}

	if err := a.UserRepo.Save(ctx, &entity.User{
		ID:      in.UserID,
		Name:    in.Name,
		Email:   in.Email,
		Address: in.Address,
		Status:  UserStatusPending,
	}); err != nil {
		return fmt.Errorf("save pending user: %w", err)
	}

	if err := a.Publisher.Publish(ctx, event.UserCreated, event.UserUpsertedEvent{
		UserID:   in.UserID,
		Name:     in.Name,
		Email:    in.Email,
		Address:  in.Address,
		Document: strings.Join([]string{in.Name, in.Email}, " "),
	}); err != nil && a.Log != nil {
		a.Log.Error(ctx, "workflow.user_onboarding.publish_user_created_failed", "user_id", in.UserID, "error", err.Error())
	}

	return nil
}

func (a *Activities) SendWelcomeEmail(ctx context.Context, in Input) error {
	if a.Log != nil {
		a.Log.Info(ctx, "workflow.user_onboarding.send_welcome_email", "user_id", in.UserID, "email", in.Email)
	}

	// Placeholder activity for demo purposes.
	return nil
}

func (a *Activities) ActivateUser(ctx context.Context, userID string) error {
	if a.Log != nil {
		a.Log.Info(ctx, "workflow.user_onboarding.activate_user", "user_id", userID)
	}

	u, err := a.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	u.Status = UserStatusActive
	if err := a.UserRepo.Save(ctx, u); err != nil {
		return fmt.Errorf("save active user: %w", err)
	}

	if err := a.Publisher.Publish(ctx, event.UserUpdated, event.UserUpsertedEvent{
		UserID:   u.ID,
		Name:     u.Name,
		Email:    u.Email,
		Address:  u.Address,
		Document: strings.Join([]string{u.Name, u.Email}, " "),
	}); err != nil && a.Log != nil {
		a.Log.Error(ctx, "workflow.user_onboarding.publish_user_updated_failed", "user_id", u.ID, "error", err.Error())
	}

	return nil
}
