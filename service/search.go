package service

import (
	"context"
	"fmt"
	"go-project-template/entity"
)

type UserSearcher interface {
	Search(ctx context.Context, query string, limit int) ([]entity.User, error)
}

type SearchService interface {
	Users(ctx context.Context, query string, limit int) ([]entity.User, error)
}

type searchService struct {
	searcher UserSearcher
}

func NewSearchService(searcher UserSearcher) SearchService {
	return &searchService{searcher: searcher}
}

func (s *searchService) Users(ctx context.Context, query string, limit int) ([]entity.User, error) {
	items, err := s.searcher.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search users failed: %w", err)
	}
	return items, nil
}
