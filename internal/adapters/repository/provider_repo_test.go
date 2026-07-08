package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSecretStore 是 secret.Store 的内存实现，用于测试。
type mockSecretStore struct {
	data map[string][]byte
}

func newMockSecretStore() *mockSecretStore {
	return &mockSecretStore{data: make(map[string][]byte)}
}

func (m *mockSecretStore) Set(key string, value []byte) error {
	m.data[key] = value
	return nil
}

func (m *mockSecretStore) Get(key string) ([]byte, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", key)
	}
	return v, nil
}

func (m *mockSecretStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func setupProviderRepo(t *testing.T) (*ProviderRepoSQLite, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)
	require.NotNil(t, repo)

	cleanup := func() {
		_ = conn.Close()
	}
	return repo, cleanup
}

func newTestProvider(id string) *models.ProviderConfig {
	return &models.ProviderConfig{
		ID:          id,
		Name:        "Test Provider " + id,
		APIHost:     "https://api.example.com",
		APIKey:      "sk-test-key-" + id,
		ModelID:     "gpt-4o",
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "default",
		Enabled:     true,
		SortOrder:   0,
	}
}

// TestProviderRepo_CreateAndGet 验证正常创建和读取，包含加密解密。
func TestProviderRepo_CreateAndGet(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("p001")
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, p.Name, got.Name)
	assert.Equal(t, p.APIHost, got.APIHost)
	assert.Equal(t, p.APIKey, got.APIKey) // 验证解密正确
	assert.Equal(t, p.ModelID, got.ModelID)
	assert.InDelta(t, p.Temperature, got.Temperature, 0.001)
	assert.Equal(t, p.TimeoutMs, got.TimeoutMs)
	assert.Equal(t, p.MaxRetries, got.MaxRetries)
	assert.Equal(t, p.GroupName, got.GroupName)
	assert.Equal(t, p.Enabled, got.Enabled)
	assert.Equal(t, p.SortOrder, got.SortOrder)
	assert.Greater(t, got.CreatedAt, int64(0))
	assert.Greater(t, got.UpdatedAt, int64(0))
}

// TestProviderRepo_Create_DuplicateID 验证重复 ID 返回错误。
func TestProviderRepo_Create_DuplicateID(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("dup001")
	require.NoError(t, repo.Create(ctx, p))

	err := repo.Create(ctx, p)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrDuplicateEntry))
}

// TestProviderRepo_Create_InvalidConfig 验证各种字段校验。
func TestProviderRepo_Create_InvalidConfig(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	tests := []struct {
		name string
		mod  func(p *models.ProviderConfig)
	}{
		{"empty_id", func(p *models.ProviderConfig) { p.ID = "" }},
		{"empty_name", func(p *models.ProviderConfig) { p.Name = "" }},
		{"empty_api_host", func(p *models.ProviderConfig) { p.APIHost = "" }},
		{"invalid_scheme", func(p *models.ProviderConfig) { p.APIHost = "ftp://example.com" }},
		{"empty_api_key", func(p *models.ProviderConfig) { p.APIKey = "" }},
		{"empty_model_id", func(p *models.ProviderConfig) { p.ModelID = "" }},
		{"negative_temperature", func(p *models.ProviderConfig) { p.Temperature = -0.1 }},
		{"excessive_temperature", func(p *models.ProviderConfig) { p.Temperature = 2.1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestProvider("invalid-" + tt.name)
			tt.mod(p)
			err := repo.Create(ctx, p)
			require.Error(t, err)
			assert.True(t, errors.Is(err, entity.ErrInvalidConfig), "expected ErrInvalidConfig, got: %v", err)
		})
	}
}

// TestProviderRepo_Update 验证更新操作。
func TestProviderRepo_Update(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("upd001")
	require.NoError(t, repo.Create(ctx, p))

	// 修改字段后更新
	p.Name = "Updated Name"
	p.APIKey = "sk-updated-key"
	p.Temperature = 1.2
	p.Enabled = false
	p.SortOrder = 10
	time.Sleep(10 * time.Millisecond) // 确保 updated_at 有变化
	require.NoError(t, repo.Update(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", got.Name)
	assert.Equal(t, "sk-updated-key", got.APIKey)
	assert.InDelta(t, 1.2, got.Temperature, 0.001)
	assert.False(t, got.Enabled)
	assert.Equal(t, 10, got.SortOrder)
	assert.GreaterOrEqual(t, got.UpdatedAt, got.CreatedAt,
		"updated_at should not be before created_at")
}

// TestProviderRepo_Update_NotFound 验证更新不存在的记录返回错误。
func TestProviderRepo_Update_NotFound(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("not-exist")
	err := repo.Update(ctx, p)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

// TestProviderRepo_Delete 验证删除操作。
func TestProviderRepo_Delete(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("del001")
	require.NoError(t, repo.Create(ctx, p))

	require.NoError(t, repo.Delete(ctx, p.ID))

	_, err := repo.Get(ctx, p.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

// TestProviderRepo_Delete_NotFound 验证删除不存在的记录返回错误。
func TestProviderRepo_Delete_NotFound(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

// TestProviderRepo_List 验证列表查询和排序。
func TestProviderRepo_List(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	// 插入 3 条，设置不同 sort_order
	providers := []*models.ProviderConfig{
		{ID: "l2", Name: "P2", APIHost: "https://a.com", APIKey: "k", ModelID: "m", SortOrder: 2, Enabled: true},
		{ID: "l0", Name: "P0", APIHost: "https://a.com", APIKey: "k", ModelID: "m", SortOrder: 0, Enabled: true},
		{ID: "l1", Name: "P1", APIHost: "https://a.com", APIKey: "k", ModelID: "m", SortOrder: 1, Enabled: true},
	}
	for _, p := range providers {
		require.NoError(t, repo.Create(ctx, p))
	}

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// 验证按 sort_order ASC 排序
	assert.Equal(t, "l0", list[0].ID)
	assert.Equal(t, "l1", list[1].ID)
	assert.Equal(t, "l2", list[2].ID)
}

// TestProviderRepo_List_Empty 验证空表返回空切片。
func TestProviderRepo_List_Empty(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 0)
}

// TestProviderRepo_Get_NotFound 验证查询不存在的记录。
func TestProviderRepo_Get_NotFound(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

// TestProviderRepo_EncryptDecrypt 验证 AES-GCM 加解密正确性。
func TestProviderRepo_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := "my-secret-api-key-12345"
	ciphertext, err := encrypt(plaintext, key)
	require.NoError(t, err)
	require.NotEmpty(t, ciphertext)

	// 密文应不同于明文
	assert.NotEqual(t, plaintext, string(ciphertext))

	decrypted, err := decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// TestProviderRepo_EncryptDecrypt_WrongKey 验证错误密钥无法解密。
func TestProviderRepo_EncryptDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1

	ciphertext, err := encrypt("secret", key1)
	require.NoError(t, err)

	_, err = decrypt(ciphertext, key2)
	require.Error(t, err)
}

// TestProviderRepo_ConcurrentReadWrite 验证并发读写安全。
func TestProviderRepo_ConcurrentReadWrite(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	// 先写入基础数据
	for i := 0; i < 5; i++ {
		p := newTestProvider(fmt.Sprintf("concurrent-%d", i))
		require.NoError(t, repo.Create(ctx, p))
	}

	var wg sync.WaitGroup
	// 5 个 goroutine 并发读写
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p := newTestProvider(fmt.Sprintf("concurrent-new-%d", idx))
			_ = repo.Create(ctx, p)
			_, _ = repo.List(ctx)
			_, _ = repo.Get(ctx, fmt.Sprintf("concurrent-%d", idx))
		}(i)
	}
	wg.Wait()

	// 最终列表应包含至少 5 条原始记录
	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list), 5)
}

// TestProviderRepo_Persistence 验证数据在连接重建后仍然完整。
func TestProviderRepo_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)

	p := newTestProvider("persist001")
	require.NoError(t, repo.Create(ctx, p))
	require.NoError(t, conn.Close())

	// 重建连接和 repo（复用同一个 secret store，确保主密钥一致）
	conn2, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	require.NoError(t, conn2.Migrate(ctx))
	repo2, err := NewProviderRepoSQLite(conn2, store)
	require.NoError(t, err)

	got, err := repo2.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.Name, got.Name)
	assert.Equal(t, p.APIKey, got.APIKey)
	assert.Equal(t, p.APIHost, got.APIHost)
	_ = conn2.Close()
}

// TestDecrypt_ShortCiphertext 验证过短密文返回错误。
func TestDecrypt_ShortCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := decrypt([]byte{0x01}, key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

// TestEncrypt_NilKey 验证非法密钥返回错误。
func TestEncrypt_NilKey(t *testing.T) {
	_, err := encrypt("secret", nil)
	require.Error(t, err)
}

// TestGetOrCreateMasterKey_InvalidLength 验证密钥长度非法时返回错误。
func TestGetOrCreateMasterKey_InvalidLength(t *testing.T) {
	store := newMockSecretStore()
	store.data[providerMasterKeyName] = []byte("short")

	_, err := getOrCreateMasterKey(store)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid length")
}

// TestIsUniqueConstraintError 验证约束冲突检测函数。
func TestIsUniqueConstraintError(t *testing.T) {
	assert.False(t, isUniqueConstraintError(nil))
	assert.False(t, isUniqueConstraintError(errors.New("random error")))
	assert.True(t, isUniqueConstraintError(errors.New("UNIQUE constraint failed")))
	assert.True(t, isUniqueConstraintError(errors.New("unique constraint violated")))
}

// TestProviderRepo_Create_ExecError 验证数据库关闭后 Create 返回错误。
func TestProviderRepo_Create_ExecError(t *testing.T) {
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)
	require.NoError(t, conn.Close()) // 提前关闭连接

	p := newTestProvider("exec-err")
	err = repo.Create(ctx, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create provider")
}

// TestProviderRepo_Update_ExecError 验证数据库关闭后 Update 返回错误。
func TestProviderRepo_Update_ExecError(t *testing.T) {
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)

	p := newTestProvider("upd-exec-err")
	require.NoError(t, repo.Create(ctx, p))
	require.NoError(t, conn.Close())

	err = repo.Update(ctx, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update provider")
}

// TestProviderRepo_Delete_ExecError 验证数据库关闭后 Delete 返回错误。
func TestProviderRepo_Delete_ExecError(t *testing.T) {
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)

	p := newTestProvider("del-exec-err")
	require.NoError(t, repo.Create(ctx, p))
	require.NoError(t, conn.Close())

	err = repo.Delete(ctx, p.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete provider")
}

// failingSecretStore 是一个在 Set 时总是返回错误的 store，用于测试错误路径。
type failingSecretStore struct{}

func (f *failingSecretStore) Set(_ string, _ []byte) error { return errors.New("set failed") }
func (f *failingSecretStore) Get(_ string) ([]byte, error) { return nil, errors.New("get failed") }
func (f *failingSecretStore) Delete(_ string) error        { return nil }

// TestNewProviderRepoSQLite_KeyError 验证主密钥初始化失败时返回错误。
func TestNewProviderRepoSQLite_KeyError(t *testing.T) {
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	_, err = NewProviderRepoSQLite(conn, &failingSecretStore{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize provider master key")
}

// TestProviderRepo_List_QueryError 验证数据库关闭后 List 返回错误。
func TestProviderRepo_List_QueryError(t *testing.T) {
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	_, err = repo.List(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list providers")
}

// TestProviderRepo_MasterKey_Cached 验证主密钥在 repo 生命周期内复用。
func TestProviderRepo_MasterKey_Cached(t *testing.T) {
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo1, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)

	// 验证密钥已存入 store
	key1, err := store.Get(providerMasterKeyName)
	require.NoError(t, err)
	require.Len(t, key1, 32)

	// 创建第二个 repo，应复用同一密钥
	repo2, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)

	// 用 repo1 加密，repo2 解密，验证使用同一主密钥
	p := newTestProvider("keycache")
	require.NoError(t, repo1.Create(ctx, p))

	got, err := repo2.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.APIKey, got.APIKey)
	_ = conn.Close()
}

// TestProviderRepo_CreateAndGet_CLIToken 验证 cli_token 认证方式的序列化/反序列化。
func TestProviderRepo_CreateAndGet_CLIToken(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := &models.ProviderConfig{
		ID:          "cli-001",
		Name:        "Kimi CLI",
		APIHost:     "https://api.moonshot.cn",
		APIKey:      "", // cli_token 方式 api_key 可为空
		ModelID:     "moonshot-v1-8k",
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "cli",
		Enabled:     true,
		AuthMethod:  models.AuthMethodCLIToken,
		AuthParams:  models.AuthParams{CLICredentialPath: "~/.kimi/credentials/kimi-code.json"},
	}
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AuthMethodCLIToken, got.AuthMethod)
	assert.Equal(t, "~/.kimi/credentials/kimi-code.json", got.AuthParams.CLICredentialPath)
	assert.Empty(t, got.APIKey) // 空 api_key 解密后仍为空
}

// TestProviderRepo_CreateAndGet_OAuthDevice 验证 oauth_device 认证方式的序列化/反序列化。
func TestProviderRepo_CreateAndGet_OAuthDevice(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := &models.ProviderConfig{
		ID:          "oauth-001",
		Name:        "OAuth Provider",
		APIHost:     "https://api.example.com",
		APIKey:      "",
		ModelID:     "gpt-4o",
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "oauth",
		Enabled:     true,
		AuthMethod:  models.AuthMethodOAuthDevice,
		AuthParams: models.AuthParams{
			OAuthClientID:     "client-123",
			OAuthAuthURL:      "https://auth.example.com/authorize",
			OAuthTokenURL:     "https://auth.example.com/token",
			OAuthRefreshToken: "refresh-abc",
			OAuthAccessToken:  "access-xyz",
			OAuthExpiresAt:    1234567890,
		},
	}
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AuthMethodOAuthDevice, got.AuthMethod)
	assert.Equal(t, "client-123", got.AuthParams.OAuthClientID)
	assert.Equal(t, "https://auth.example.com/authorize", got.AuthParams.OAuthAuthURL)
	assert.Equal(t, "https://auth.example.com/token", got.AuthParams.OAuthTokenURL)
	// OAuthRefreshToken / OAuthAccessToken 标记为 json:"-"，不持久化到数据库
	assert.Equal(t, "", got.AuthParams.OAuthRefreshToken)
	assert.Equal(t, "", got.AuthParams.OAuthAccessToken)
	assert.Equal(t, int64(1234567890), got.AuthParams.OAuthExpiresAt)
}

// TestProviderRepo_CreateAndGet_ServiceAccount 验证 service_account 认证方式的序列化/反序列化。
func TestProviderRepo_CreateAndGet_ServiceAccount(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := &models.ProviderConfig{
		ID:          "sa-001",
		Name:        "GCP Service Account",
		APIHost:     "https://us-central1-aiplatform.googleapis.com",
		APIKey:      "",
		ModelID:     "gemini-pro",
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "gcp",
		Enabled:     true,
		AuthMethod:  models.AuthMethodServiceAccount,
		AuthParams: models.AuthParams{
			GCPProjectID: "my-project-123",
			GCPRegion:    "us-central1",
			SAJSON:       `{"type":"service_account","project_id":"my-project-123"}`,
		},
	}
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AuthMethodServiceAccount, got.AuthMethod)
	assert.Equal(t, "my-project-123", got.AuthParams.GCPProjectID)
	assert.Equal(t, "us-central1", got.AuthParams.GCPRegion)
	assert.Equal(t, `{"type":"service_account","project_id":"my-project-123"}`, got.AuthParams.SAJSON)
}

// TestProviderRepo_BackwardCompatibility 验证旧数据（无 auth_method）向后兼容。
func TestProviderRepo_BackwardCompatibility(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	// 模拟旧数据：不设置 AuthMethod 和 AuthParams
	p := &models.ProviderConfig{
		ID:          "legacy-001",
		Name:        "Legacy Provider",
		APIHost:     "https://api.legacy.com",
		APIKey:      "sk-legacy-key",
		ModelID:     "legacy-model",
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "default",
		Enabled:     true,
	}
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "sk-legacy-key", got.APIKey)
	assert.Equal(t, models.AuthMethod(""), got.AuthMethod) // 旧数据反序列化后为空
	assert.Empty(t, got.AuthParams.CLICredentialPath)
}

// TestProviderRepo_ProviderType_StoredPreferred 验证持久化的类型优先于按 api_host 推断的类型。
func TestProviderRepo_ProviderType_StoredPreferred(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()

	ctx := context.Background()
	p := newTestProvider("stored-type")
	// api_host 会被推断为 Kimi，但显式设置 Type 为 local，读取应以存储值为准。
	p.APIHost = "https://api.moonshot.cn"
	p.Type = models.ProviderLocal
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, "stored-type")
	require.NoError(t, err)
	assert.Equal(t, models.ProviderLocal, got.Type, "应使用持久化类型，而非按 api_host 推断的 kimi")
}

// TestProviderRepo_ProviderType_EmptyFallback 验证存储类型为空时回退按 api_host 推断（兼容迁移前的旧行）。
func TestProviderRepo_ProviderType_EmptyFallback(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()

	ctx := context.Background()
	p := newTestProvider("legacy-type")
	p.APIHost = "https://api.openai.com"
	require.NoError(t, repo.Create(ctx, p))

	// 模拟迁移前旧行：清空 provider_type，读取时应回退按 api_host 推断。
	_, err := repo.db.ExecContext(ctx, `UPDATE providers SET provider_type = '' WHERE id = ?`, "legacy-type")
	require.NoError(t, err)

	got, err := repo.Get(ctx, "legacy-type")
	require.NoError(t, err)
	assert.Equal(t, models.ProviderOpenAI, got.Type, "空类型应回退推断为 openai")
}

// TestProviderRepo_ProviderType_LocalPersisted 验证本地模型创建路径写入正确类型且读取一致。
func TestProviderRepo_ProviderType_LocalPersisted(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()

	ctx := context.Background()
	p := newTestProvider("local-ollama")
	// 未显式设置 Type，但回环 11434 端口应被推断并持久化为 ollama。
	p.APIHost = "http://localhost:11434"
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, "local-ollama")
	require.NoError(t, err)
	assert.Equal(t, models.ProviderOllama, got.Type)

	// List 路径同样返回持久化类型。
	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, models.ProviderOllama, list[0].Type)
}

// TestProviderRepo_Models_RoundTrip 验证模型列表（含 MaxContextLength）持久化后可完整读回。
func TestProviderRepo_Models_RoundTrip(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("models-rt")
	p.ModelID = "deepseek-v4-flash"
	p.Models = []models.ProviderModel{
		{ID: "deepseek-v4-flash", Name: "deepseek-v4-flash", Enabled: true, MaxContextLength: 1000000},
	}
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, got.Models, 1)
	assert.Equal(t, "deepseek-v4-flash", got.Models[0].ID)
	assert.Equal(t, "deepseek-v4-flash", got.Models[0].Name)
	assert.True(t, got.Models[0].Enabled)
	assert.Equal(t, 1000000, got.Models[0].MaxContextLength)
}

// TestProviderRepo_Models_UpdatePreservesMaxContextLength 验证更新 MaxContextLength 后读取到新值。
func TestProviderRepo_Models_UpdatePreservesMaxContextLength(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("models-upd")
	p.ModelID = "glm-5"
	p.Models = []models.ProviderModel{
		{ID: "glm-5", Name: "glm-5", Enabled: true, MaxContextLength: 128000},
	}
	require.NoError(t, repo.Create(ctx, p))

	// 更新 MaxContextLength
	p.Models[0].MaxContextLength = 256000
	require.NoError(t, repo.Update(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, got.Models, 1)
	assert.Equal(t, 256000, got.Models[0].MaxContextLength)
}

// TestProviderRepo_Models_ListCarriesModels 验证 List 返回的 provider 携带模型列表。
func TestProviderRepo_Models_ListCarriesModels(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("models-list")
	p.ModelID = "qwen-max"
	p.Models = []models.ProviderModel{
		{ID: "qwen-max", Name: "qwen-max", Enabled: true, MaxContextLength: 32768},
		{ID: "qwen-plus", Name: "qwen-plus", Enabled: false, MaxContextLength: 131072},
	}
	require.NoError(t, repo.Create(ctx, p))

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Len(t, list[0].Models, 2)
	assert.Equal(t, "qwen-max", list[0].Models[0].ID)
	assert.Equal(t, 32768, list[0].Models[0].MaxContextLength)
	assert.Equal(t, "qwen-plus", list[0].Models[1].ID)
	assert.Equal(t, 131072, list[0].Models[1].MaxContextLength)
}

// TestProviderRepo_Models_EmptyFallbackSynthesizesModel 验证旧行（空 models、非空 model_id）
// 读取时合成单个模型，保持向后兼容。
func TestProviderRepo_Models_EmptyFallbackSynthesizesModel(t *testing.T) {
	repo, cleanup := setupProviderRepo(t)
	defer cleanup()
	ctx := context.Background()

	p := newTestProvider("models-legacy")
	p.ModelID = "gpt-4o-mini"
	p.Models = nil // 模拟旧行：无模型列表
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, got.Models, 1)
	assert.Equal(t, "gpt-4o-mini", got.Models[0].ID)
	assert.Equal(t, "gpt-4o-mini", got.Models[0].Name)
	assert.True(t, got.Models[0].Enabled)
}
