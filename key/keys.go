package key

import (
	"context"
	"crypto"
	"fmt"
	"sync"
)

// KeyManager provides signing and encryption operations.
type KeyManager interface {
	// NewSigner returns a crypto.Signer for the given key ID.
	NewSigner(ctx context.Context, keyID string) (crypto.Signer, error)

	// Encrypt encrypts plaintext with optional Additional Authenticated Data (AAD).
	// Returns ciphertext that can only be decrypted with the same key and AAD.
	Encrypt(ctx context.Context, keyID string, plaintext, aad []byte) ([]byte, error)

	// Decrypt decrypts ciphertext with optional Additional Authenticated Data (AAD).
	// The AAD must match what was provided during encryption.
	Decrypt(ctx context.Context, keyID string, ciphertext, aad []byte) ([]byte, error)

	// ListKeyVersions returns all versions for the given parent key.
	ListKeyVersions(ctx context.Context, parent string) ([]string, error)
}

// SigningKeyVersion represents a version of a signing key.
type SigningKeyVersion struct {
	ID        string
	CreatedAt int64
}

// SigningKeyManager extends KeyManager with signing key lifescyle operations.
type SigningKeyManager interface {
	KeyManager

	CreateSigningKey(ctx context.Context, parent, name string) (string, error)

	KeyVersionCreator
	KeyVersionDestroyer
}

// EncryptionKeyManager extends KeyManager with encryption key lifecycle operations.
type EncryptionKeyManager interface {
	KeyManager

	CreateEncryptionKey(ctx context.Context, parent, name string) (string, error)

	KeyVersionCreator
	KeyVersionDestroyer
}

// KeyVersionCreator creates new key version.
type KeyVersionCreator interface {
	// CreateKeyVersion creates a new key version under the given parent key.
	// Returns the full key version ID.
	CreateKeyVersion(ctx context.Context, parent string) (string, error)
}

// KeyVersionDestroyer destroys key version.
type KeyVersionDestroyer interface {
	// DestroyKeyVersion marks a key version for destruction.
	// This operation should be idempotent.
	DestroyKeyVersion(ctx context.Context, id string) error
}

// KeyManagerFunc is a factory function for creating key managers.
type KeyManagerFunc func(ctx context.Context, cfg *Config) (KeyManager, error)

var (
	managers     = make(map[string]KeyManagerFunc)
	managersLock sync.RWMutex
)

// RegisterManager register a key manager implementation.
// Called in init() functions of provider implementations.
// Panics if a manager with the same name is already registed.
func RegisterManager(name string, fn KeyManagerFunc) {
	managersLock.Lock()
	defer managersLock.Unlock()

	if _, exists := managers[name]; exists {
		panic(fmt.Sprintf("key manager %q already registered", name))
	}
	managers[name] = fn
}

// KeyManagerFor returns a KeyManager for the given configuration.
func KeyManagerFor(ctx context.Context, cfg *Config) (KeyManager, error) {
	managersLock.RLock()
	fn, exists := managers[cfg.Type]
	defer managersLock.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown key manager type: %s", cfg.Type)
	}

	return fn(ctx, cfg)
}
