package useronboarding

import (
	"context"
	"testing"
	"time"

	"github.com/cschleiden/go-workflows/tester"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserOnboardingWorkflow_HappyPath(t *testing.T) {
	wt := tester.NewWorkflowTester[Result](UserOnboardingWorkflow)

	in := Input{UserID: "u-1", Name: "Alice", Email: "alice@example.com", Address: "Street"}

	wt.OnActivityByName(ActivityCreatePendingUser, (&Activities{}).CreatePendingUser, mock.Anything, in).Return(nil)
	wt.OnActivityByName(ActivitySendWelcomeEmail, (&Activities{}).SendWelcomeEmail, mock.Anything, in).Return(nil)
	wt.OnActivityByName(ActivityActivateUser, (&Activities{}).ActivateUser, mock.Anything, in.UserID).Return(nil)

	wt.ScheduleCallback(100*time.Millisecond, func() {
		wt.SignalWorkflow(SignalEmailVerified, EmailVerifiedSignal{Verified: true})
	})

	wt.Execute(context.Background(), in)

	require.True(t, wt.WorkflowFinished())
	res, err := wt.WorkflowResult()
	require.NoError(t, err)
	require.Equal(t, in.UserID, res.UserID)
	require.Equal(t, StatusActive, res.Status)
}

func TestUserOnboardingWorkflow_TimeoutPath(t *testing.T) {
	wt := tester.NewWorkflowTester[Result](UserOnboardingWorkflow)

	in := Input{UserID: "u-2", Name: "Bob", Email: "bob@example.com", Address: "Road"}

	wt.OnActivityByName(ActivityCreatePendingUser, (&Activities{}).CreatePendingUser, mock.Anything, in).Return(nil)
	wt.OnActivityByName(ActivitySendWelcomeEmail, (&Activities{}).SendWelcomeEmail, mock.Anything, in).Return(nil)

	wt.Execute(context.Background(), in)

	require.True(t, wt.WorkflowFinished())
	res, err := wt.WorkflowResult()
	require.NoError(t, err)
	require.Equal(t, in.UserID, res.UserID)
	require.Equal(t, StatusVerificationTimedOut, res.Status)
}
