package repository

import (
	"context"
	"fmt"
	"go-project-template/database/sqldb"
	"go-project-template/entity"
	"go-project-template/logger"
	"go-project-template/service"
)

type PostgresUserSearchRepository struct {
	db  *sqldb.DB
	log *logger.Logger
}

var _ service.UserIndexer = (*PostgresUserSearchRepository)(nil)
var _ service.UserSearcher = (*PostgresUserSearchRepository)(nil)

func NewPostgresUserSearchRepository(db *sqldb.DB, log *logger.Logger) *PostgresUserSearchRepository {
	if log == nil {
		log = logger.Noop()
	}

	return &PostgresUserSearchRepository{db: db, log: log}
}

func (r *PostgresUserSearchRepository) Index(ctx context.Context, doc service.UserSearchDocument) error {
	r.log.Info(ctx, "repository.user_search.index", "user_id", doc.UserID)
	query := `
		INSERT INTO user_search_index (user_id, name, email, document)
		VALUES ($1, $2, $3, to_tsvector('english', $4))
		ON CONFLICT (user_id)
		DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			document = EXCLUDED.document,
			updated_at = NOW()
	`
	if _, err := r.db.SQL().ExecContext(ctx, query, doc.UserID, doc.Name, doc.Email, doc.Document); err != nil {
		return fmt.Errorf("index user document failed: %w", err)
	}
	return nil
}

func (r *PostgresUserSearchRepository) DeleteIndex(ctx context.Context, userID string) error {
	r.log.Info(ctx, "repository.user_search.delete", "user_id", userID)
	if _, err := r.db.SQL().ExecContext(ctx, `DELETE FROM user_search_index WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete user index failed: %w", err)
	}
	return nil
}

func (r *PostgresUserSearchRepository) Search(ctx context.Context, query string, limit int) ([]entity.User, error) {
	r.log.Info(ctx, "repository.user_search.search", "query", query, "limit", limit)
	if limit <= 0 {
		limit = 20
	}

	sqlQuery := `
		SELECT user_id, name, email
		FROM user_search_index
		WHERE document @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(document, plainto_tsquery('english', $1)) DESC, updated_at DESC
		LIMIT $2
	`

	rows, err := r.db.SQL().QueryContext(ctx, sqlQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search users failed: %w", err)
	}
	defer rows.Close()

	results := make([]entity.User, 0)
	for rows.Next() {
		var item entity.User
		if err := rows.Scan(&item.ID, &item.Name, &item.Email); err != nil {
			return nil, fmt.Errorf("scan search result failed: %w", err)
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search result failed: %w", err)
	}

	return results, nil
}
