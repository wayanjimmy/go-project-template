package repository

import (
	"context"
	"fmt"
	"go-project-template/config"
	"go-project-template/database/sqldb"
	"go-project-template/entity"
	"go-project-template/logger"
	"os"
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
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	log := logger.Noop()

	require.NoError(t, sqldb.RunMigrations(dbURL))

	cfg := &config.Config{DatabaseURL: dbURL}
	db, err := sqldb.Open(cfg, log)
	require.NoError(t, err)
	defer db.Close(ctx)

	encryptor := &mockEncryptor{}
	repo := NewPostgresUserRepository(db, encryptor, log)

	// Clean up
	_, err = db.SQL().Exec("DELETE FROM users")
	require.NoError(t, err)

	// Seed users
	for i := 0; i < 5; i++ {
		user := &entity.User{
			ID:      uuid.New().String(),
			Name:    fmt.Sprintf("User %d", i),
			Email:   fmt.Sprintf("user%d@example.com", i),
			Address: fmt.Sprintf("Address %d", i),
		}
		err := repo.Save(ctx, user)
		require.NoError(t, err)
	}

	t.Run("list with pagination", func(t *testing.T) {
		users, err := repo.List(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("list with offset", func(t *testing.T) {
		users, err := repo.List(ctx, 10, 4)
		require.NoError(t, err)
		assert.Len(t, users, 1)
	})

	t.Run("list with decryption error", func(t *testing.T) {
		failRepo := NewPostgresUserRepository(db, &mockEncryptor{decryptErr: fmt.Errorf("fail")}, log)
		users, err := failRepo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.Len(t, users, 5)
		assert.Equal(t, "<error decrypting>", users[0].Address)
	})
}
