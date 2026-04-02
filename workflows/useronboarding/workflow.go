package useronboarding

import (
	"time"

	"github.com/cschleiden/go-workflows/workflow"
)

const VerificationTimeout = 30 * time.Minute

func UserOnboardingWorkflow(ctx workflow.Context, in Input) (Result, error) {
	if _, err := workflow.ExecuteActivity[any](ctx, workflow.DefaultActivityOptions, ActivityCreatePendingUser, in).Get(ctx); err != nil {
		return Result{}, err
	}

	if _, err := workflow.ExecuteActivity[any](ctx, workflow.DefaultActivityOptions, ActivitySendWelcomeEmail, in).Get(ctx); err != nil {
		return Result{}, err
	}

	signalCh := workflow.NewSignalChannel[EmailVerifiedSignal](ctx, SignalEmailVerified)
	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	timerF := workflow.ScheduleTimer(timerCtx, VerificationTimeout)

	verified := false
	workflow.Select(ctx,
		workflow.Receive(signalCh, func(ctx workflow.Context, v EmailVerifiedSignal, ok bool) {
			if ok && v.Verified {
				verified = true
				cancelTimer()
			}
		}),
		workflow.Await(timerF, func(ctx workflow.Context, f workflow.Future[any]) {
			// timeout path; no-op, checked below
		}),
	)

	if verified {
		if _, err := workflow.ExecuteActivity[any](ctx, workflow.DefaultActivityOptions, ActivityActivateUser, in.UserID).Get(ctx); err != nil {
			return Result{}, err
		}
		return Result{UserID: in.UserID, Status: StatusActive}, nil
	}

	return Result{UserID: in.UserID, Status: StatusVerificationTimedOut}, nil
}
