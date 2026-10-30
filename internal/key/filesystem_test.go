package key

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFileSystem(t *testing.T) *FileSystem {
	t.Helper()

	root := t.TempDir()
	kmFS, err := NewFileSystem(context.Background(), &Config{
		Type:           "FILESYSTEM",
		FilesystemRoot: root,
	})
	require.NoError(t, err)

	km, ok := kmFS.(*FileSystem)
	if !ok {
		t.Fatalf("expected *FileSystem, got %T", kmFS)
	}

	return km
}

func TestFileSystem_EncryptDecrypt_RoundTrip(t *testing.T) {
	ctx := context.Background()
	km := newTestFileSystem(t)

	keyID, err := km.CreateEncryptionKey(ctx, "tasks", "notes-encryption")
	assert.NoError(t, err)

	_, err = km.CreateKeyVersion(ctx, keyID)
	assert.NoError(t, err)

	plaintext := []byte("secret notes")
	aad := []byte("task-123")

	ciphertext, err := km.Encrypt(ctx, keyID, plaintext, aad)
	assert.NoError(t, err)

	got, err := km.Decrypt(ctx, keyID, ciphertext, aad)
	assert.NoError(t, err)

	assert.Equal(t, plaintext, got)
}

func TestFileSystem_Decrypt_WrongAAD_Fails(t *testing.T) {
	ctx := context.Background()
	km := newTestFileSystem(t)

	keyID, err := km.CreateEncryptionKey(ctx, "tasks", "notes-encryption")
	assert.NoError(t, err)

	_, err = km.CreateKeyVersion(ctx, keyID)
	assert.NoError(t, err)

	plaintext := []byte("secret notes")
	aad := []byte("task-123")

	ciphertext, err := km.Encrypt(ctx, keyID, plaintext, aad)
	assert.NoError(t, err)

	_, err = km.Decrypt(ctx, keyID, ciphertext, []byte("task-321"))
	assert.Error(t, err)
}

func TestFileSystem_KeyRotation_OldCiphertextStillDecrypts(t *testing.T) {
	ctx := context.Background()
	km := newTestFileSystem(t)

	keyID, err := km.CreateEncryptionKey(ctx, "tasks", "notes-encryption")
	assert.NoError(t, err)

	_, err = km.CreateKeyVersion(ctx, keyID)
	assert.NoError(t, err)

	plaintext := []byte("secret notes")
	aad := []byte("task-123")

	c1, err := km.Encrypt(ctx, keyID, plaintext, aad)
	assert.NoError(t, err)

	// create a new key version
	_, err = km.CreateKeyVersion(ctx, keyID)
	assert.NoError(t, err)

	plaintext2 := []byte("secret notes")
	aad2 := []byte("task-123")

	c2, err := km.Encrypt(ctx, keyID, plaintext2, aad2)
	assert.NoError(t, err)

	// Act:
	p1, err := km.Decrypt(ctx, keyID, c1, aad)
	assert.NoError(t, err)

	p2, err := km.Decrypt(ctx, keyID, c2, aad2)
	assert.NoError(t, err)

	assert.Equal(t, plaintext, p1)
	assert.Equal(t, plaintext2, p2)
	assert.NotEqual(t, c1, c2)
}
