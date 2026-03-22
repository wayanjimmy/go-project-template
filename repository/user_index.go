package repository

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

type UserSearcher interface {
	Search(ctx context.Context, query string, limit int) ([]UserSearchDocument, error)
}
