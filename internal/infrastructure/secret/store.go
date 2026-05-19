// Package secret 封装密钥安全存储。
//
// KeyringStore 优先使用平台原生密钥环（macOS Keychain / Windows Credential Manager /
// Linux Secret Service），headless 环境无可用后端时自动降级到 FileStore。
// FileStore 基于 AES-GCM 加密文件系统存储，作为最终兜底方案。
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
	"github.com/google/wire"
)

// Store 密钥安全存储接口。
type Store interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// KeyringStore 基于平台原生密钥环的实现。
// 平台密钥环不可用时自动降级到 FileStore，保证任何环境都可正常启动。
type KeyringStore struct {
	ring     keyring.Keyring
	fallback *FileStore
}

// NewKeyringStore 创建密钥环存储实例。
// 优先尝试系统密钥环，失败时静默降级到 FileStore，避免 headless CI 阻塞。
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
	if s.fallback != nil {
		return s.fallback.Set(key, value)
	}
	return s.ring.Set(keyring.Item{
		Key:  key,
		Data: value,
	})
}

// Get 读取密钥。
func (s *KeyringStore) Get(key string) ([]byte, error) {
	if s.fallback != nil {
		return s.fallback.Get(key)
	}
	item, err := s.ring.Get(key)
	if err != nil {
		return nil, fmt.Errorf("keyring get failed for key %s: %w", key, err)
	}
	return item.Data, nil
}

// Delete 删除密钥。
func (s *KeyringStore) Delete(key string) error {
	if s.fallback != nil {
		return s.fallback.Delete(key)
	}
	return s.ring.Remove(key)
}

// FileStore 基于文件系统的加密存储（AES-GCM）。
// 密钥派生自 OS 用户目录路径哈希，提供基础安全性，不适合高安全场景。
type FileStore struct {
	baseDir string
	key     []byte
}

// NewFileStore 创建文件存储实例。
func NewFileStore() (*FileStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home dir: %w", err)
	}

	baseDir := filepath.Join(home, ".medmemo", "secrets")
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create secrets dir: %w", err)
	}

	// 使用用户主目录路径哈希派生加密密钥
	// 此方案仅提供基础保护，不适合高安全场景
	hash := sha256.Sum256([]byte(home + "/medmemo-secret"))
	key := hash[:]

	return &FileStore{
		baseDir: baseDir,
		key:     key,
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
func (s *FileStore) Get(key string) ([]byte, error) {
	path := s.filePath(key)
	encrypted, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("secret not found for key %s: %w", key, err)
		}
		return nil, fmt.Errorf("failed to read secret file: %w", err)
	}

	value, err := s.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return value, nil
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
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *FileStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, cipherData, nil)
}

// SecretSet 供 Wire 使用的 ProviderSet。
var SecretSet = wire.NewSet(
	NewKeyringStore,
)
