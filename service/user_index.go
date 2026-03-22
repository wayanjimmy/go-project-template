package service

import "context"

type UserSearchDocument struct {
	UserID   string
	Name     string
	Email    string
	Document string
}

type UserIndexer interface {
	Index(ctx context.Context, doc UserSearchDocument) error
	DeleteIndex(ctx context.Context, userID string) error
}
