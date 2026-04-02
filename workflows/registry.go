package workflows

import (
	"go-project-template/logger"
	"go-project-template/service"
	"go-project-template/workflows/useronboarding"

	"github.com/cschleiden/go-workflows/registry"
	workflowworker "github.com/cschleiden/go-workflows/worker"
)

type Dependencies struct {
	UserRepo  service.UserRepository
	Publisher service.EventPublisher
	Log       *logger.Logger
}

func Register(w *workflowworker.Worker, deps Dependencies) error {
	if err := w.RegisterWorkflow(useronboarding.UserOnboardingWorkflow, registry.WithName(useronboarding.WorkflowName)); err != nil {
		return err
	}

	acts := &useronboarding.Activities{
		UserRepo:  deps.UserRepo,
		Publisher: deps.Publisher,
		Log:       deps.Log,
	}

	if err := w.RegisterActivity(acts.CreatePendingUser, registry.WithName(useronboarding.ActivityCreatePendingUser)); err != nil {
		return err
	}

	if err := w.RegisterActivity(acts.SendWelcomeEmail, registry.WithName(useronboarding.ActivitySendWelcomeEmail)); err != nil {
		return err
	}

	if err := w.RegisterActivity(acts.ActivateUser, registry.WithName(useronboarding.ActivityActivateUser)); err != nil {
		return err
	}

	return nil
}
