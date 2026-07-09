// Package secret 密钥安全存储。
// KeyringStore 优先使用平台原生密钥环，headless 环境降级到 FileStore（AES-GCM）。
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/99designs/keyring"
	"github.com/google/wire"
)

// Store 密钥安全存储接口。
type Store interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// KeyringStore 平台原生密钥环实现，不可用时自动降级到 FileStore。
type KeyringStore struct {
	mu       sync.RWMutex
	ring     keyring.Keyring
	fallback *FileStore
}

// NewKeyringStore 创建密钥环存储实例，失败时静默降级到 FileStore。
func NewKeyringStore() (*KeyringStore, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: "MedMemo",
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.WinCredBackend,
		},
	})
	if err == nil {
		return &KeyringStore{ring: ring}, nil
	}

	fs, err := NewFileStore()
	if err != nil {
		return nil, fmt.Errorf("platform keyring unavailable and fallback file store failed: %w", err)
	}
	return &KeyringStore{fallback: fs}, nil
}

// Set 存储密钥。
func (s *KeyringStore) Set(key string, value []byte) error {
	if fs := s.currentFallback(); fs != nil {
		return fs.Set(key, value)
	}
	if s.ring == nil {
		fs, err := s.ensureFallback()
		if err != nil {
			return fmt.Errorf("fallback file store unavailable for key %s: %w", key, err)
		}
		return fs.Set(key, value)
	}
	err := s.ring.Set(keyring.Item{
		Key:  key,
		Data: value,
	})
	if err == nil {
		return nil
	}

	fs, fallbackErr := s.ensureFallback()
	if fallbackErr != nil {
		return fmt.Errorf("keyring set failed for key %s and fallback unavailable: %w", key, errors.Join(err, fallbackErr))
	}
	if fallbackErr := fs.Set(key, value); fallbackErr != nil {
		return fmt.Errorf("keyring set failed for key %s and fallback set failed: %w", key, errors.Join(err, fallbackErr))
	}
	return nil
}

// Get 读取密钥。
func (s *KeyringStore) Get(key string) ([]byte, error) {
	if fs := s.currentFallback(); fs != nil {
		value, err := fs.Get(key)
		if err == nil || s.ring == nil {
			return value, err
		}
	}

	if s.ring == nil {
		fs, err := s.ensureFallback()
		if err != nil {
			return nil, fmt.Errorf("fallback file store unavailable for key %s: %w", key, err)
		}
		return fs.Get(key)
	}

	item, err := s.ring.Get(key)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			value, fs, fallbackErr := readFileFallback(key)
			if fallbackErr == nil {
				s.setFallback(fs)
				return value, nil
			}
			return nil, fmt.Errorf("secret not found for key %s: %w", key, err)
		}

		fs, fallbackErr := s.ensureFallback()
		if fallbackErr != nil {
			return nil, fmt.Errorf("keyring get failed for key %s and fallback unavailable: %w", key, errors.Join(err, fallbackErr))
		}
		value, fallbackErr := fs.Get(key)
		if fallbackErr != nil {
			return nil, fmt.Errorf("keyring get failed for key %s and fallback get failed: %w", key, errors.Join(err, fallbackErr))
		}
		return value, nil
	}
	return item.Data, nil
}

// Delete 删除密钥。
func (s *KeyringStore) Delete(key string) error {
	var keyringErr error
	if s.ring != nil {
		keyringErr = s.ring.Remove(key)
	}

	if fs := s.currentFallback(); fs != nil {
		if err := fs.Delete(key); err != nil {
			if keyringErr != nil && !errors.Is(keyringErr, keyring.ErrKeyNotFound) {
				return fmt.Errorf("keyring delete failed for key %s and fallback delete failed: %w", key, errors.Join(keyringErr, err))
			}
			return fmt.Errorf("fallback delete failed for key %s: %w", key, err)
		}
	}

	if keyringErr != nil && !errors.Is(keyringErr, keyring.ErrKeyNotFound) {
		return fmt.Errorf("keyring delete failed for key %s: %w", key, keyringErr)
	}
	return nil
}

func (s *KeyringStore) currentFallback() *FileStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fallback
}

func (s *KeyringStore) setFallback(fs *FileStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fallback == nil {
		s.fallback = fs
	}
}

func (s *KeyringStore) ensureFallback() (*FileStore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fallback != nil {
		return s.fallback, nil
	}
	fs, err := NewFileStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback file store: %w", err)
	}
	s.fallback = fs
	return fs, nil
}

func readFileFallback(key string) ([]byte, *FileStore, error) {
	fs, err := NewFileStore()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create fallback file store: %w", err)
	}
	value, err := fs.Get(key)
	if err != nil {
		return nil, nil, err
	}
	return value, fs, nil
}

// FileStore 文件系统加密存储（AES-GCM）。
// 安全性低于密钥环，仅作为降级方案。
type FileStore struct {
	baseDir string
	key     []byte
	oldKey  []byte // 旧 key derivation 兼容，用于迁移旧数据
}

// NewFileStore 创建文件存储实例。
// 密钥派生自 home + uid + 随机 salt，salt 持久化在 ~/.medmemo/secrets/.salt。
// 保留旧 key derivation（无 salt/uid）作为兼容路径，自动迁移旧数据。
func NewFileStore() (*FileStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home dir: %w", err)
	}

	baseDir := filepath.Join(home, ".medmemo", "secrets")
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create secrets dir: %w", err)
	}

	// 读取或生成随机 salt，增加密钥派生熵
	saltPath := filepath.Join(baseDir, ".salt")
	salt, err := os.ReadFile(saltPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read salt: %w", err)
		}
		salt = make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
		if err := os.WriteFile(saltPath, salt, 0600); err != nil {
			return nil, fmt.Errorf("failed to write salt: %w", err)
		}
	}

	// 新密钥派生：home + uid + salt
	uid := os.Getuid()
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s/medmemo-secret/%d/%x", home, uid, salt)))
	key := hash[:]

	// 旧密钥派生（兼容旧数据）：home + "/medmemo-secret"
	oldHash := sha256.Sum256([]byte(home + "/medmemo-secret"))
	oldKey := oldHash[:]

	return &FileStore{
		baseDir: baseDir,
		key:     key,
		oldKey:  oldKey,
	}, nil
}

// Set 加密并存储密钥。
func (s *FileStore) Set(key string, value []byte) error {
	encrypted, err := s.encrypt(value)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	path := s.filePath(key)
	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write secret file: %w", err)
	}
	return nil
}

// Get 读取并解密密钥。
// 若新 key 解密失败，尝试旧 key 解密并自动迁移。
func (s *FileStore) Get(key string) ([]byte, error) {
	path := s.filePath(key)
	encrypted, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("secret not found for key %s: %w", key, err)
		}
		return nil, fmt.Errorf("failed to read secret file: %w", err)
	}

	// 先用新 key 解密
	value, err := s.decryptWithKey(encrypted, s.key)
	if err == nil {
		return value, nil
	}

	// 新 key 失败，尝试旧 key 兼容解密
	if s.oldKey != nil {
		value, err = s.decryptWithKey(encrypted, s.oldKey)
		if err == nil {
			// 旧数据解密成功，自动迁移：用新 key 重新加密保存
			if migrateErr := s.Set(key, value); migrateErr == nil {
				return value, nil
			}
			// 迁移失败不影响读取，仍返回解密结果
			return value, nil
		}
	}

	return nil, fmt.Errorf("decryption failed (key mismatch or corrupted data): %w", err)
}

// Delete 删除密钥文件。
func (s *FileStore) Delete(key string) error {
	path := s.filePath(key)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove secret file: %w", err)
	}
	return nil
}

func (s *FileStore) filePath(key string) string {
	// SHA-256 哈希文件名，避免特殊字符问题
	hash := sha256.Sum256([]byte(key))
	name := fmt.Sprintf("%x.secret", hash)
	return filepath.Join(s.baseDir, name)
}

func (s *FileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init failed: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce generation failed: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *FileStore) decryptWithKey(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init failed: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (corrupted or tampered data): %w", err)
	}
	return plaintext, nil
}

// Set 供 Wire 使用的 ProviderSet。
var Set = wire.NewSet(
	NewKeyringStore,
)
