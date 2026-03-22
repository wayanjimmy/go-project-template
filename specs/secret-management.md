# Secret Management and Data Encryption

Study of the secret management and data encryption implementation in `go-project-example/ddd-lite`.

## Architecture Overview

```
internal/
  key/                    # Key management core
    ├── config.go          # Key manager configuration
    ├── keys.go            # KeyManager interface definitions
    ├── filesystem.go      # Filesystem-based implementation
    └── filesystem_test.go # Tests
  adapters/
    encryption/
      ├── encryptor.go     # High-level encryption adapter
      └── encryptor_test.go
  bootstrap/
      ├── keys.go          # Key manager initialization
      └── env.go           # Server environment setup
```

---

## 1. Key Management System (`internal/key/`)

### Core Interface

```go
type KeyManager interface {
    NewSigner(ctx, keyID) (crypto.Signer, error)      // ECDSA P-256
    Encrypt(ctx, keyID, plaintext, aad) ([]byte, error)
    Decrypt(ctx, keyID, ciphertext, aad) ([]byte, error)
}
```

### Extended Interfaces

| Interface | Purpose |
|-----------|---------|
| `SigningKeyManager` | Signing key lifecycle (create, version, destroy) |
| `EncryptionKeyManager` | Encryption key lifecycle (create, version, destroy) |
| `KeyVersionCreator` | Create new key versions under a parent key |
| `KeyVersionDestroyer` | Destroy key versions (idempotent) |

### Configuration (`config.go`)

```go
type Config struct {
    Type           string  // FILESYSTEM, GOOGLE_CLOUD_KMS, AWS_KMS, HASHICORP_VAULT
    FilesystemRoot string  // Required for FILESYSTEM type
}
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KEY_MANAGER` | `FILESYSTEM` | Key manager type |
| `KEY_FILESYSTEM_ROOT` | (required) | Filesystem key storage path |

### Registry Pattern

Key managers are registered via `init()` functions:

```go
func RegisterManager(name string, fn KeyManagerFunc)
func KeyManagerFor(ctx context.Context, cfg *Config) (KeyManager, error)
```

This enables pluggable implementations without changing application code.

---

## 2. Filesystem Implementation (`internal/key/filesystem.go`)

**Purpose**: Development and testing only (not for production)

### Key Storage Structure

```
{root}/
  tasks/
    notes-encryption/
      metadata      # {"t":"encryption"} or {"t":"signing"}
      1741694386... # key version file (timestamp in nanoseconds)
```

### Key Types

| Type | Algorithm | Format |
|------|-----------|--------|
| Signing | ECDSA P-256 | DER encoded (X.509) |
| Encryption | AES-256 | 32 random bytes |

### Key Versioning

- **Version ID**: Unix timestamp in nanoseconds (`time.Now().UnixNano()`)
- **Selection**: Latest version used for encryption (sorted descending)
- **Rotation**: New versions can be created; old ciphertext remains decryptable

### File Permissions

| Item | Permission |
|------|------------|
| Directories | `0700` |
| Key files | `0600` |
| Metadata | `0600` |

---

## 3. Encryption Adapter (`internal/adapters/encryption/`)

### Encryptor Interface

```go
type Encryptor interface {
    Encrypt(ctx context.Context, plaintext string, aad []byte) ([]byte, error)
    Decrypt(ctx context.Context, ciphertext []byte, aad []byte) (string, error)
}
```

### KeyManagerEncryptor

```go
type KeyManagerEncryptor struct {
    km    key.KeyManager
    keyID string
}
```

### Encryption Algorithm: AES-256-GCM

| Component | Details |
|-----------|---------|
| Cipher | AES-256-GCM |
| Key Derivation | SHA256 hash of stored key material |
| Nonce | 12 bytes (random per encryption) |
| AAD | Additional Authenticated Data (contextual binding) |

### Ciphertext Format

```
{keyVersion}:{nonce}{ciphertext}
```

Example: `1741694386123456789:<12-byte-nonce><encrypted-data>`

### Edge Case Handling

| Input | Behavior |
|-------|----------|
| Empty plaintext | Returns `nil` ciphertext |
| Empty ciphertext | Returns empty string |
| Wrong AAD | Decryption fails with error |

---

## 4. Bootstrap Integration (`internal/bootstrap/`)

### Key Manager Initialization

```go
func InitKeyManager(ctx context.Context, keysPath string) (key.KeyManager, error) {
    return key.KeyManagerFor(ctx, &key.Config{
        Type:           "FILESYSTEM",
        FilesystemRoot: keysPath,
    })
}
```

### Encryption Key Provisioning

```go
func EnsureEncryptionKey(ctx context.Context, km key.KeyManager, encryptionKeyID string) error {
    ekm, ok := km.(key.EncryptionKeyManager)
    if !ok {
        return nil
    }

    // Create key (idempotent)
    keyID, err := ekm.CreateEncryptionKey(ctx, "tasks", "notes-encryption")
    if err != nil {
        return err
    }

    // Ensure at least one version exists
    if _, err := ekm.CreateKeyVersion(ctx, keyID); err != nil {
        logger.Info(ctx, "Key version creation note", "error", err)
    }

    logger.Info(ctx, "Encryption key ready", "key_id", keyID)
    return nil
}
```

### Server Environment Setup

```go
func WithTaskRepository() Option {
    return func(ctx context.Context, env *ServerEnv) error {
        encryptor := encryption.New(env.KeyManager, env.Cfg.EncryptionKeyID)
        env.TaskRepo = postgres.NewTaskRepository(env.Pool, encryptor)
        return nil
    }
}
```

---

## 5. Application Usage (`internal/adapters/postgres/`)

### Task Repository Encryption

```go
type TaskRepository struct {
    pool      *pgxpool.Pool
    encryptor encryption.Encryptor
}

func (r *TaskRepository) encryptNotes(ctx context.Context, id task.TaskID, notes string) ([]byte, error) {
    return r.encryptor.Encrypt(ctx, notes, []byte(id.String()))  // AAD = task ID
}

func (r *TaskRepository) decryptNotes(ctx context.Context, id task.TaskID, encryptedNotes []byte) (string, error) {
    return r.encryptor.Decrypt(ctx, encryptedNotes, []byte(id.String()))
}
```

### Security Design

| Feature | Purpose |
|---------|---------|
| Field-level encryption | Only `notes` field is encrypted |
| AAD binding | Task ID prevents ciphertext substitution |
| Transactional updates | Encryption happens within DB transactions |

### Data Flow

```
Create Task:
  plaintext notes → encrypt(taskID) → ciphertext → PostgreSQL

Read Task:
  ciphertext → decrypt(taskID) → plaintext notes
```

---

## 6. Configuration (`internal/config/`)

```go
type Config struct {
    Addr            string
    DatabaseURL     string
    KeysPath        string        // KEYS_PATH
    EncryptionKeyID string        // ENCRYPTION_KEY_ID or NOTES_KEY_ID
    ShutdownTimeout time.Duration
    Env             string
    ServiceName     string
}
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KEYS_PATH` | `./keys` | Root directory for filesystem keys |
| `ENCRYPTION_KEY_ID` | `tasks/notes-encryption` | Key identifier for encryption |
| `NOTES_KEY_ID` | (fallback) | Legacy key ID variable |
| `KEY_MANAGER` | `FILESYSTEM` | Key manager type |
| `KEY_FILESYSTEM_ROOT` | (required) | Filesystem key storage path |

---

## 7. Test Infrastructure (`cmd/api/testinfra/`)

### TestKeyManager

```go
type TestKeyManager struct {
    key []byte  // Single 256-bit random key
}
```

### Characteristics

| Feature | Implementation |
|---------|----------------|
| Storage | In-memory (no persistence) |
| Key | Single random 256-bit key |
| Signing | Not implemented (returns error) |
| Encryption | AES-256-GCM |
| Versioning | Not supported |

### Usage

```go
func NewTestKeyManager() *TestKeyManager {
    key := make([]byte, 32)
    io.ReadFull(rand.Reader, key)
    return &TestKeyManager{key: key}
}
```

---

## 8. Security Features

### Key Security

| Feature | Description |
|---------|-------------|
| Key Versioning | Automatic version management enables key rotation |
| Secure Permissions | Directories: 0700, Files: 0600 |
| Key Isolation | Separate keys for signing vs encryption |
| Pluggable Backend | Interface abstraction for KMS integration |

### Encryption Security

| Feature | Description |
|---------|-------------|
| AAD Binding | Contextual data prevents ciphertext substitution |
| AES-256-GCM | Authenticated encryption with associated data |
| Random Nonce | Unique nonce per encryption operation |
| Key Derivation | SHA256 hash of key material |

### Operational Security

| Feature | Description |
|---------|-------------|
| Idempotent Operations | Key creation/versioning safe to retry |
| Test/Prod Separation | Different key managers for environments |
| Graceful Degradation | Empty plaintext/ciphertext handled safely |

---

## 9. Future Extensions

The `Config.Type` field indicates planned support for:

| Provider | Status |
|----------|--------|
| `FILESYSTEM` | ✅ Implemented |
| `GOOGLE_CLOUD_KMS` | 🔲 Planned |
| `AWS_KMS` | 🔲 Planned |
| `HASHICORP_VAULT` | 🔲 Planned |

### Implementation Pattern

New providers implement the `KeyManager` interface and register themselves:

```go
func init() {
    RegisterManager("GOOGLE_CLOUD_KMS", NewGoogleCloudKMS)
}
```

No application code changes required—only configuration updates.

---

## 10. Testing

### Unit Tests

| Test File | Coverage |
|-----------|----------|
| `filesystem_test.go` | Key creation, encryption, decryption, rotation, destruction |
| `encryptor_test.go` | Encryptor wrapper, edge cases, AAD validation |

### Test Cases

```go
// Encrypt/Decrypt roundtrip
func TestFilesystem_EncryptDecrypt(t *testing.T) {
    plaintext := []byte("secret notes content")
    aad := []byte("task-123")
    
    ciphertext, _ := km.Encrypt(ctx, keyID, plaintext, aad)
    decrypted, _ := km.Decrypt(ctx, keyID, ciphertext, aad)
    
    assert.Equal(t, plaintext, decrypted)
}

// AAD validation
func TestFilesystem_DecryptWithWrongAAD(t *testing.T) {
    ciphertext, _ := km.Encrypt(ctx, keyID, plaintext, []byte("task-123"))
    _, err := km.Decrypt(ctx, keyID, ciphertext, []byte("task-456"))
    
    assert.Error(t, err)  // Decryption fails with wrong AAD
}

// Key rotation
func TestFilesystem_KeyRotation(t *testing.T) {
    ciphertextV1, _ := km.Encrypt(ctx, keyID, plaintext, aad)
    km.CreateKeyVersion(ctx, keyID)  // Rotate key
    ciphertextV2, _ := km.Encrypt(ctx, keyID, plaintext, aad)
    
    // Old ciphertext still decrypts
    decrypted, _ := km.Decrypt(ctx, keyID, ciphertextV1, aad)
    assert.Equal(t, plaintext, decrypted)
    
    // New ciphertext differs
    assert.NotEqual(t, ciphertextV1, ciphertextV2)
}
```

---

## 11. Quick Reference

### Environment Setup

```bash
# Development
export KEYS_PATH=./keys
export KEY_MANAGER=FILESYSTEM
export KEY_FILESYSTEM_ROOT=./keys

# Production (example with AWS KMS)
export KEY_MANAGER=AWS_KMS
export AWS_KMS_KEY_ID=arn:aws:kms:us-east-1:123456789012:key/...
```

### Key Operations

```go
// Create encryption key
keyID, err := ekm.CreateEncryptionKey(ctx, "tasks", "notes-encryption")

// Create key version (rotation)
versionID, err := ekm.CreateKeyVersion(ctx, keyID)

// List key versions
versions, err := ekm.SigningKeyVersions(ctx, keyID)

// Destroy key version
err = ekm.DestroyKeyVersion(ctx, versionID)
```

### Encryption Operations

```go
// Encrypt
ciphertext, err := encryptor.Encrypt(ctx, "secret data", []byte("task-id-123"))

// Decrypt
plaintext, err := encryptor.Decrypt(ctx, ciphertext, []byte("task-id-123"))
```

---

## Summary

The `ddd-lite` project implements a clean, extensible secret management system with:

1. **Interface-driven design** - Pluggable key manager backends
2. **Field-level encryption** - Sensitive data encrypted at rest
3. **Key versioning** - Supports key rotation without data loss
4. **AAD binding** - Prevents ciphertext substitution attacks
5. **Test infrastructure** - In-memory key manager for testing
6. **Production-ready patterns** - Secure defaults, proper permissions, idempotent operations

The architecture allows seamless migration from filesystem-based keys to cloud KMS providers by changing only configuration.
