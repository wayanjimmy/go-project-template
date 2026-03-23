package repository

import (
	"go-project-template/logger"
	"go-project-template/service"
	"go-project-template/test/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresUserSearchRepository_SearchIndexDelete(t *testing.T) {
	ctx, db := testutil.OpenMigratedDB(t, logger.Noop())
	repo := NewPostgresUserSearchRepository(db, logger.Noop())

	t.Run("search from existing fixture", func(t *testing.T) {
		testutil.LoadFixtures(t, db.SQL(), "users_existing")

		items, err := repo.Search(ctx, "alice", 10)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "user-1", items[0].ID)
		assert.Equal(t, "Alice", items[0].Name)
		assert.Equal(t, "alice@example.com", items[0].Email)
	})

	t.Run("index then search", func(t *testing.T) {
		testutil.LoadFixtures(t, db.SQL(), "users_for_indexing")

		err := repo.Index(ctx, service.UserSearchDocument{
			UserID:   "user-2",
			Name:     "Charlie",
			Email:    "charlie@example.com",
			Document: "Charlie charlie@example.com",
		})
		require.NoError(t, err)

		items, err := repo.Search(ctx, "charlie", 10)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "user-2", items[0].ID)
	})

	t.Run("delete index", func(t *testing.T) {
		testutil.LoadFixtures(t, db.SQL(), "users_existing")

		err := repo.DeleteIndex(ctx, "user-1")
		require.NoError(t, err)

		items, err := repo.Search(ctx, "alice", 10)
		require.NoError(t, err)
		assert.Len(t, items, 0)
	})
}
