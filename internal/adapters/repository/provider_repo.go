// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/internal/infrastructure/secret"
	"github.com/hzhan516/medmemo/pkg/models"
)

const providerMasterKeyName = "provider:api_key:master"

// ProviderRepoSQLite 基于 SQLite 的 Provider 配置仓库实现。
// API Key 使用 AES-256-GCM 加密存储，加密主密钥由 secret.Store 管理。
type ProviderRepoSQLite struct {
	db        *sql.DB
	masterKey []byte
}

// NewProviderRepoSQLite 创建 Provider 配置仓库。
// 从 secret.Store 获取或生成 32 字节 AES 主密钥，用于加解密 api_key 字段。
func NewProviderRepoSQLite(connector database.DBConnector, store secret.Store) (*ProviderRepoSQLite, error) {
	key, err := getOrCreateMasterKey(store)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize provider master key: %w", err)
	}

	return &ProviderRepoSQLite{
		db:        connector.DB(),
		masterKey: key,
	}, nil
}

// Create 创建新的 Provider 配置。
func (r *ProviderRepoSQLite) Create(ctx context.Context, provider *models.ProviderConfig) error {
	if err := validateProvider(provider); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	encryptedKey, err := encrypt(provider.APIKey, r.masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt api key: %w", err)
	}

	authParamsJSON, err := provider.MarshalAuthParams()
	if err != nil {
		return fmt.Errorf("failed to marshal auth params: %w", err)
	}

	now := time.Now().UnixMilli()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO providers (id, name, api_host, api_key, model_id, temperature, timeout_ms, max_retries, group_name, enabled, sort_order, created_at, updated_at, auth_method, auth_params, provider_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, provider.ID, provider.Name, provider.APIHost, encryptedKey, provider.ModelID, provider.Temperature,
		int64(provider.TimeoutMs), provider.MaxRetries, provider.GroupName, boolToInt(provider.Enabled),
		provider.SortOrder, now, now, string(provider.AuthMethod), authParamsJSON, string(resolveProviderType(provider)))

	if err != nil {
		// SQLite 约束冲突（重复主键）
		if isUniqueConstraintError(err) {
			return fmt.Errorf("provider %s already exists: %w", provider.ID, entity.ErrDuplicateEntry)
		}
		return fmt.Errorf("failed to create provider %s: %w", provider.ID, err)
	}
	return nil
}

// Update 更新已有 Provider 配置。
func (r *ProviderRepoSQLite) Update(ctx context.Context, provider *models.ProviderConfig) error {
	if err := validateProvider(provider); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	encryptedKey, err := encrypt(provider.APIKey, r.masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt api key: %w", err)
	}

	authParamsJSON, err := provider.MarshalAuthParams()
	if err != nil {
		return fmt.Errorf("failed to marshal auth params: %w", err)
	}

	now := time.Now().UnixMilli()
	res, err := r.db.ExecContext(ctx, `
		UPDATE providers SET
			name = ?, api_host = ?, api_key = ?, model_id = ?, temperature = ?,
			timeout_ms = ?, max_retries = ?, group_name = ?, enabled = ?, sort_order = ?, updated_at = ?, auth_method = ?, auth_params = ?, provider_type = ?
		WHERE id = ?
	`, provider.Name, provider.APIHost, encryptedKey, provider.ModelID, provider.Temperature,
		int64(provider.TimeoutMs), provider.MaxRetries, provider.GroupName, boolToInt(provider.Enabled),
		provider.SortOrder, now, string(provider.AuthMethod), authParamsJSON, string(resolveProviderType(provider)), provider.ID)

	if err != nil {
		return fmt.Errorf("failed to update provider %s: %w", provider.ID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("provider %s not found: %w", provider.ID, entity.ErrNotFound)
	}
	return nil
}

// Delete 删除指定 Provider 配置。
func (r *ProviderRepoSQLite) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider %s: %w", id, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("provider %s not found: %w", id, entity.ErrNotFound)
	}
	return nil
}

// Get 按 ID 查询 Provider 配置。
func (r *ProviderRepoSQLite) Get(ctx context.Context, id string) (*models.ProviderConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, api_host, api_key, model_id, temperature, timeout_ms, max_retries, group_name, enabled, sort_order, created_at, updated_at, auth_method, auth_params, provider_type
		FROM providers WHERE id = ?
	`, id)

	return r.scanProvider(row)
}

// List 查询全部 Provider 配置，按 sort_order ASC, updated_at DESC 排序。
func (r *ProviderRepoSQLite) List(ctx context.Context) ([]*models.ProviderConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, api_host, api_key, model_id, temperature, timeout_ms, max_retries, group_name, enabled, sort_order, created_at, updated_at, auth_method, auth_params, provider_type
		FROM providers
		ORDER BY sort_order ASC, updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []*models.ProviderConfig
	for rows.Next() {
		p, err := r.scanProvider(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate providers: %w", err)
	}
	return result, nil
}

// scanProvider 将单行扫描结果转换为 ProviderConfig。
func (r *ProviderRepoSQLite) scanProvider(scanner interface {
	Scan(dest ...any) error
}) (*models.ProviderConfig, error) {
	var p models.ProviderConfig
	var encryptedKey []byte
	var timeoutMs, createdAt, updatedAt int64
	var enabledInt int
	var authMethodStr, authParamsJSON, providerTypeStr string

	if err := scanner.Scan(&p.ID, &p.Name, &p.APIHost, &encryptedKey, &p.ModelID, &p.Temperature,
		&timeoutMs, &p.MaxRetries, &p.GroupName, &enabledInt, &p.SortOrder, &createdAt, &updatedAt, &authMethodStr, &authParamsJSON, &providerTypeStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("provider not found: %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to scan provider row: %w", err)
	}

	decryptedKey, err := decrypt(encryptedKey, r.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt api key: %w", err)
	}
	p.APIKey = decryptedKey

	p.TimeoutMs = int(timeoutMs)
	p.Enabled = enabledInt != 0
	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt
	p.AuthMethod = models.AuthMethod(authMethodStr)
	// 优先使用持久化的类型；旧行 provider_type 为空时回退按 api_host 推断，保持向后兼容。
	if providerTypeStr != "" {
		p.Type = models.ProviderType(providerTypeStr)
	} else {
		p.Type = models.InferProviderType(p.APIHost)
	}
	if err := p.UnmarshalAuthParams(authParamsJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth params for provider %s: %w", p.ID, err)
	}

	return &p, nil
}

// resolveProviderType 返回待持久化的 provider 类型。
// 若调用方已显式设置 Type 则直接使用；否则按 api_host 推断，
// 确保本地模型（ollama/local/回环地址）等创建路径也能写入正确类型。
func resolveProviderType(p *models.ProviderConfig) models.ProviderType {
	if p.Type != "" {
		return p.Type
	}
	return models.InferProviderType(p.APIHost)
}

// validateProvider 校验 Provider 配置字段合法性。
// 委托 ProviderConfig.Validate() 进行认证方式相关的校验。
func validateProvider(p *models.ProviderConfig) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("%w: %v", entity.ErrInvalidConfig, err)
	}
	return nil
}

// getOrCreateMasterKey 从 secret.Store 获取或生成 32 字节 AES 主密钥。
func getOrCreateMasterKey(store secret.Store) ([]byte, error) {
	key, err := store.Get(providerMasterKeyName)
	if err == nil {
		if len(key) == 32 {
			return key, nil
		}
		return nil, fmt.Errorf("provider master key has invalid length %d, expected 32", len(key))
	}

	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate provider master key: %w", err)
	}

	if err := store.Set(providerMasterKeyName, key); err != nil {
		return nil, fmt.Errorf("failed to store provider master key: %w", err)
	}
	return key, nil
}

// encrypt 使用 AES-256-GCM 加密明文。
// 返回格式: nonce || ciphertext。
func encrypt(plaintext string, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
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

	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// decrypt 使用 AES-256-GCM 解密密文。
// 输入格式: nonce || ciphertext。
func decrypt(ciphertext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	return string(plaintext), nil
}

// isUniqueConstraintError 判断是否为 SQLite 唯一约束冲突错误。
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// 兼容 modernc.org/sqlite 和 go-sqlcipher 的错误信息
	errStr := err.Error()
	return strings.Contains(errStr, "UNIQUE constraint failed") || strings.Contains(errStr, "unique constraint")
}

// boolToInt 将 bool 转为 SQLite 用的 0/1 整数，零分配。
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
