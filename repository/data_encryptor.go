package repository

import (
	"context"
	"go-project-template/key"
)

var _ Encryptor = (*DataEncryptor)(nil)

// DataEncryptor adapts key.KeyManager to repository encryption needs.
type DataEncryptor struct {
	km    key.KeyManager
	keyID string
}

func NewDataEncryptor(km key.KeyManager, keyID string) *DataEncryptor {
	return &DataEncryptor{km: km, keyID: keyID}
}

func (s *DataEncryptor) Encrypt(ctx context.Context, plaintext string, aad []byte) ([]byte, error) {
	return s.km.Encrypt(ctx, s.keyID, []byte(plaintext), aad)
}

func (s *DataEncryptor) Decrypt(ctx context.Context, ciphertext []byte, aad []byte) (string, error) {
	result, err := s.km.Decrypt(ctx, s.keyID, ciphertext, aad)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
