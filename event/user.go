package event

const (
	UserCreated = "user.created"
	UserUpdated = "user.updated"
	UserDeleted = "user.deleted"
)

type UserUpsertedEvent struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Address  string `json:"address"`
	Document string `json:"document"`
}

type UserDeletedEvent struct {
	UserID string `json:"user_id"`
}
