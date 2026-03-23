package repository

import (
	"context"
	"fmt"
	"go-project-template/entity"
	"go-project-template/logger"
	"go-project-template/test/testutil"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEncryptor struct {
	decryptErr error
}

func (m *mockEncryptor) Encrypt(ctx context.Context, plaintext string, aad []byte) ([]byte, error) {
	return []byte(plaintext), nil
}

func (m *mockEncryptor) Decrypt(ctx context.Context, ciphertext []byte, aad []byte) (string, error) {
	if m.decryptErr != nil {
		return "", m.decryptErr
	}
	return string(ciphertext), nil
}

func TestPostgresUserRepository_List(t *testing.T) {
	ctx, db := testutil.OpenMigratedDB(t, logger.Noop())
	repo := NewPostgresUserRepository(db, &mockEncryptor{}, logger.Noop())

	t.Run("list from users_existing fixture", func(t *testing.T) {
		testutil.LoadFixtures(t, db.SQL().DB, "users_existing")

		users, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, "user-1", users[0].ID)
		assert.Equal(t, "Alice", users[0].Name)
		assert.Equal(t, "alice@example.com", users[0].Email)
	})

	t.Run("list from users_empty fixture", func(t *testing.T) {
		testutil.LoadFixtures(t, db.SQL().DB, "users_empty")

		users, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.Len(t, users, 0)
	})

	t.Run("list with decryption error", func(t *testing.T) {
		testutil.LoadFixtures(t, db.SQL().DB, "users_existing")
		failRepo := NewPostgresUserRepository(db, &mockEncryptor{decryptErr: fmt.Errorf("fail")}, logger.Noop())

		users, err := failRepo.List(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, "<error decrypting>", users[0].Address)
	})
}

func TestPostgresUserRepository_FindSaveDelete(t *testing.T) {
	ctx, db := testutil.OpenMigratedDB(t, logger.Noop())
	repo := NewPostgresUserRepository(db, &mockEncryptor{}, logger.Noop())

	t.Run("find by id from fixtures", func(t *testing.T) {
		testutil.LoadFixtures(t, db.SQL().DB, "users_existing")

		user, err := repo.FindByID(ctx, "user-1")
		require.NoError(t, err)
		assert.Equal(t, "user-1", user.ID)
		assert.Equal(t, "Alice", user.Name)
		assert.Equal(t, "alice@example.com", user.Email)
	})

	t.Run("save then find then delete", func(t *testing.T) {
		testutil.LoadFixtures(t, db.SQL().DB, "users_empty")

		id := uuid.NewString()
		err := repo.Save(ctx, &entity.User{
			ID:      id,
			Name:    "Bob",
			Email:   "bob@example.com",
			Address: "Jakarta",
		})
		require.NoError(t, err)

		saved, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "Bob", saved.Name)
		assert.Equal(t, "bob@example.com", saved.Email)
		assert.Equal(t, "Jakarta", saved.Address)

		err = repo.Delete(ctx, id)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, id)
		require.Error(t, err)
	})
}
