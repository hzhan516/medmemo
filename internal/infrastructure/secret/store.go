// Package secret 封装系统密钥环（macOS Keychain / Windows DPAPI / Linux Secret Service）。
package secret

import (
	"fmt"

	"github.com/google/wire"
)

// Store 密钥安全存储接口。
type Store interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// KeyringStore 基于平台密钥环的实现。
type KeyringStore struct{}

// NewKeyringStore 创建密钥环存储实例。
func NewKeyringStore() (*KeyringStore, error) {
	// TODO(作者): 接入 99designs/keyring 或平台原生 API [Issue#023]
	return &KeyringStore{}, nil
}

// Set 存储密钥。
func (s *KeyringStore) Set(key string, value []byte) error {
	return fmt.Errorf("KeyringStore.Set not implemented")
}

// Get 读取密钥。
func (s *KeyringStore) Get(key string) ([]byte, error) {
	return nil, fmt.Errorf("KeyringStore.Get not implemented")
}

// Delete 删除密钥。
func (s *KeyringStore) Delete(key string) error {
	return fmt.Errorf("KeyringStore.Delete not implemented")
}

// SecretSet 供 Wire 使用的 ProviderSet。
var SecretSet = wire.NewSet(
	NewKeyringStore,
)
