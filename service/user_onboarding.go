package service

import (
	"context"
	"errors"
	"fmt"
	"go-project-template/workflows/useronboarding"

	workflowbackend "github.com/cschleiden/go-workflows/backend"
	workflowclient "github.com/cschleiden/go-workflows/client"
	"github.com/google/uuid"
)

type UserOnboardingService interface {
	Start(ctx context.Context, name, email, address string) (userID, workflowInstanceID string, err error)
	VerifyEmail(ctx context.Context, userID string) error
}

type userOnboardingService struct {
	client *workflowclient.Client
}

func NewUserOnboardingService(client *workflowclient.Client) UserOnboardingService {
	return &userOnboardingService{client: client}
}

func (s *userOnboardingService) Start(ctx context.Context, name, email, address string) (string, string, error) {
	userID := uuid.NewString()
	instanceID := useronboarding.InstanceID(userID)

	_, err := s.client.CreateWorkflowInstance(ctx, workflowclient.WorkflowInstanceOptions{
		InstanceID: instanceID,
	}, useronboarding.WorkflowName, useronboarding.Input{
		UserID:  userID,
		Name:    name,
		Email:   email,
		Address: address,
	})
	if err != nil {
		if errors.Is(err, workflowbackend.ErrInstanceAlreadyExists) {
			return "", "", fmt.Errorf("workflow instance already exists: %w", err)
		}
		return "", "", fmt.Errorf("start onboarding workflow: %w", err)
	}

	return userID, instanceID, nil
}

func (s *userOnboardingService) VerifyEmail(ctx context.Context, userID string) error {
	instanceID := useronboarding.InstanceID(userID)

	err := s.client.SignalWorkflow(ctx, instanceID, useronboarding.SignalEmailVerified, useronboarding.EmailVerifiedSignal{Verified: true})
	if err != nil {
		if errors.Is(err, workflowbackend.ErrInstanceNotFound) {
			return fmt.Errorf("workflow instance not found: %w", err)
		}
		return fmt.Errorf("signal onboarding workflow: %w", err)
	}

	return nil
}
