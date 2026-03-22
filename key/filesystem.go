package key

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

func init() {
	RegisterManager("FILESYSTEM", NewFileSystem)
}

var _ EncryptionKeyManager = (*FileSystem)(nil)

type FileSystem struct {
	root string
}

func NewFileSystem(ctx context.Context, cfg *Config) (KeyManager, error) {
	if cfg.FilesystemRoot == "" {
		return nil, errors.New("cfg.FilesystemRoot is required")
	}

	if err := os.MkdirAll(cfg.FilesystemRoot, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key root: %w", err)
	}

	return &FileSystem{root: cfg.FilesystemRoot}, nil
}

type keyMetadata struct {
	Type string `json:"t"` // "signing" or "encryption"
}

// NewSigner implements [EncryptionKeyManager].
func (f *FileSystem) NewSigner(ctx context.Context, keyID string) (crypto.Signer, error) {
	return nil, errors.New("signing not implemented")
}

// CreateEncryptionKey implements [EncryptionKeyManager].
func (f *FileSystem) CreateEncryptionKey(ctx context.Context, parent string, name string) (string, error) {
	keyPath := filepath.Join(f.root, parent, name)

	if err := os.MkdirAll(keyPath, 0700); err != nil {
		return "", fmt.Errorf("failed to create key directory: %w", err)
	}

	metaPath := filepath.Join(keyPath, "metadata")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		meta := keyMetadata{Type: "encryption"}
		data, _ := json.Marshal(meta)
		if err := os.WriteFile(metaPath, data, 0600); err != nil {
			return "", fmt.Errorf("failed to write metadata: %w", err)
		}
	}

	return filepath.Join(parent, name), nil
}

// CreateKeyVersion implements [EncryptionKeyManager].
func (f *FileSystem) CreateKeyVersion(ctx context.Context, parent string) (string, error) {
	keyPath := filepath.Join(f.root, parent)

	metapath := filepath.Join(keyPath, "metadata")
	metaData, err := os.ReadFile(metapath)
	if err != nil {
		return "", fmt.Errorf("failed to read key metadata: %w", err)
	}

	var meta keyMetadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return "", fmt.Errorf("failed to parse metadata: %w", err)
	}

	version := strconv.FormatInt(time.Now().UnixNano(), 10)
	versionPath := filepath.Join(keyPath, version)

	var keyData []byte
	switch meta.Type {
	case "signing":
		return "", errors.New("signing not implemented")
	case "encryption":
		keyData = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, keyData); err != nil {
			return "", fmt.Errorf("failed to generate encryption key: %w", err)
		}
	default:
		return "", fmt.Errorf("unknown key type: %s", meta.Type)
	}

	if err := os.WriteFile(versionPath, keyData, 0600); err != nil {
		return "", fmt.Errorf("failed to write key version: %w", err)
	}

	return filepath.Join(parent, version), nil
}

// ListKeyVersions implements [EncryptionKeyManager].
func (f *FileSystem) ListKeyVersions(ctx context.Context, parent string) ([]string, error) {
	keyPath := filepath.Join(f.root, parent)
	return f.getKeyVersions(keyPath)
}

// Decrypt implements ciphertext using AES-256-GCM with optional AAD.
func (f *FileSystem) Decrypt(ctx context.Context, keyID string, ciphertext []byte, aad []byte) ([]byte, error) {
	keyPath := filepath.Join(f.root, keyID)

	// Parse version from ciphertext
	parts := bytes.SplitN(ciphertext, []byte(":"), 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid ciphertext format")
	}

	version := string(parts[0])
	data := parts[1]

	versionPath := filepath.Join(keyPath, version)
	keyData, err := os.ReadFile(versionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key version: %w", err)
	}

	// Derive AES key from stored key material
	aesKey := sha256.Sum256(keyData)

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertextData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextData, aad)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// DestroyKeyVersion implements [EncryptionKeyManager].
func (f *FileSystem) DestroyKeyVersion(ctx context.Context, id string) error {
	versionPath := filepath.Join(f.root, id)

	if err := os.Remove(versionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to destroy key version: %w", err)
	}

	return nil
}

// Encrypt implements [EncryptionKeyManager].
func (f *FileSystem) Encrypt(ctx context.Context, keyID string, plaintext []byte, aad []byte) ([]byte, error) {
	keyPath := filepath.Join(f.root, keyID)

	// Get latest version
	versions, err := f.getKeyVersions(keyPath)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, errors.New("no key versions available")
	}

	latestVersion := versions[0]
	versionPath := filepath.Join(keyPath, latestVersion)

	keyData, err := os.ReadFile(versionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}

	// Derive AES key from stored key material
	aesKey := sha256.Sum256(keyData)

	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, aad)

	// Prepend version identifier
	result := []byte(latestVersion + ":")
	result = append(result, ciphertext...)

	return result, nil
}

// getKeyVersions returns versions identifiers sorted by timestamp (newes first).
func (f *FileSystem) getKeyVersions(keyPath string) ([]string, error) {
	entries, err := os.ReadDir(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.Name() == "metadata" {
			continue
		}

		versions = append(versions, entry.Name())
	}

	sort.Slice(versions, func(i, j int) bool {
		ti, _ := strconv.ParseInt(versions[i], 10, 64)
		tj, _ := strconv.ParseInt(versions[j], 10, 64)
		return ti > tj
	})

	return versions, nil
}
