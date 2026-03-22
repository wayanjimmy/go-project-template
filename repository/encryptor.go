package repository

import "context"

// Encryptor defines the encryption behavior required by repositories.
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext string, aad []byte) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte, aad []byte) (string, error)
}
