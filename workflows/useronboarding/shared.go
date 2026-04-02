package useronboarding

import "fmt"

const (
	WorkflowName = "UserOnboardingWorkflow"

	ActivityCreatePendingUser = "UserOnboardingCreatePendingUser"
	ActivitySendWelcomeEmail  = "UserOnboardingSendWelcomeEmail"
	ActivityActivateUser      = "UserOnboardingActivateUser"

	SignalEmailVerified        = "email_verified"
	UserStatusPending          = "pending"
	UserStatusActive           = "active"
	StatusActive               = "active"
	StatusVerificationTimedOut = "verification_timed_out"
)

type Input struct {
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type EmailVerifiedSignal struct {
	Verified bool `json:"verified"`
}

type Result struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

func InstanceID(userID string) string {
	return fmt.Sprintf("user-onboarding/%s", userID)
}
