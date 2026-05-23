package secret

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStore_SetGetDelete(t *testing.T) {
	fs, err := NewFileStore()
	require.NoError(t, err)

	key := "test-api-key"
	value := []byte("sk-test-secret-value-12345")

	err = fs.Set(key, value)
	require.NoError(t, err)

	got, err := fs.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	err = fs.Delete(key)
	require.NoError(t, err)

	_, err = fs.Get(key)
	require.Error(t, err)
}

func TestFileStore_Get_NotFound(t *testing.T) {
	fs, err := NewFileStore()
	require.NoError(t, err)

	_, err = fs.Get("non-existent-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFileStore_Delete_Idempotent(t *testing.T) {
	fs, err := NewFileStore()
	require.NoError(t, err)

	err = fs.Delete("never-existed")
	require.NoError(t, err)
}

func TestFileStore_MultipleKeys(t *testing.T) {
	fs, err := NewFileStore()
	require.NoError(t, err)

	data := map[string][]byte{
		"key-a": []byte("value-a"),
		"key-b": []byte("value-b"),
		"key-c": []byte("value-c"),
	}

	for k, v := range data {
		require.NoError(t, fs.Set(k, v))
	}

	for k, expected := range data {
		got, err := fs.Get(k)
		require.NoError(t, err)
		assert.Equal(t, expected, got)
	}
}

func TestKeyringStore_FileBackend(t *testing.T) {
	tmpDir := t.TempDir()
	ring, err := keyring.Open(keyring.Config{
		ServiceName:              "medmemo-test",
		AllowedBackends:          []keyring.BackendType{keyring.FileBackend},
		FileDir:                  tmpDir,
		FilePasswordFunc:         func(_ string) (string, error) { return "test-password", nil },
		KeychainTrustApplication: true,
	})
	require.NoError(t, err)

	store := &KeyringStore{ring: ring}

	key := "db_key"
	value := []byte("super-secret-32-byte-key!!!!")

	err = store.Set(key, value)
	require.NoError(t, err)

	got, err := store.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	err = store.Delete(key)
	require.NoError(t, err)

	_, err = store.Get(key)
	require.Error(t, err)
}

func TestKeyringStore_FallbackToFileStore(t *testing.T) {
	// 直接构造 fallback 实例，避免 CI 中 D-Bus 超时等待
	fs, err := NewFileStore()
	require.NoError(t, err)
	store := &KeyringStore{fallback: fs}

	key := "fallback-test-key"
	value := []byte("fallback-secret-value")

	err = store.Set(key, value)
	require.NoError(t, err)

	got, err := store.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	err = store.Delete(key)
	require.NoError(t, err)

	_, err = store.Get(key)
	require.Error(t, err)
}

func TestNewFileStore_HomeDirFailure(t *testing.T) {
	t.Setenv("HOME", "")
	_, err := NewFileStore()
	assert.Error(t, err)
}

func TestNewFileStore_MkdirAllFailure(t *testing.T) {
	// 使用一个文件路径作为 home，使 MkdirAll 失败
	tmpFile, err := os.CreateTemp("", "secret-test-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	t.Setenv("HOME", tmpFile.Name())
	_, err = NewFileStore()
	assert.Error(t, err)
}

func TestNewFileStore_SaltWriteFailure(t *testing.T) {
	tmpDir := t.TempDir()
	// 预创建 .salt 为目录，导致写入失败
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".medmemo", "secrets", ".salt"), 0755))
	t.Setenv("HOME", tmpDir)
	_, err := NewFileStore()
	assert.Error(t, err)
}

func TestFileStore_Set_WriteFails(t *testing.T) {
	tmpDir := t.TempDir()
	secretsDir := filepath.Join(tmpDir, "secrets")
	require.NoError(t, os.MkdirAll(secretsDir, 0755))

	fs := &FileStore{baseDir: secretsDir, key: make([]byte, 32)}
	// 将目录改为只读，使写入失败
	require.NoError(t, os.Chmod(secretsDir, 0555))
	defer os.Chmod(secretsDir, 0755)

	err := fs.Set("test-key", []byte("value"))
	assert.Error(t, err)
}

func TestFileStore_Get_OldKeyMigration(t *testing.T) {
	tmpDir := t.TempDir()
	// 使用旧 key derivation 加密数据
	oldHash := sha256.Sum256([]byte(tmpDir + "/medmemo-secret"))
	oldKey := oldHash[:]

	fsOld := &FileStore{baseDir: tmpDir, key: oldKey}
	err := fsOld.Set("migration-key", []byte("secret-value"))
	require.NoError(t, err)

	// 使用新 FileStore（含 oldKey）读取，应自动迁移
	newHash := sha256.Sum256([]byte(fmt.Sprintf("%s/medmemo-secret/%d/%x", tmpDir, os.Getuid(), []byte("salt"))))
	fsNew := &FileStore{baseDir: tmpDir, key: newHash[:], oldKey: oldKey}

	got, err := fsNew.Get("migration-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("secret-value"), got)

	// 验证已迁移：再次用新 key 读取应成功（无需 oldKey）
	fsNew2 := &FileStore{baseDir: tmpDir, key: newHash[:]}
	got2, err := fsNew2.Get("migration-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("secret-value"), got2)
}

func TestFileStore_Get_OldKeyMigrationFail(t *testing.T) {
	tmpDir := t.TempDir()
	// 用旧 key 写入
	oldHash := sha256.Sum256([]byte(tmpDir + "/medmemo-secret"))
	oldKey := oldHash[:]
	fsOld := &FileStore{baseDir: tmpDir, key: oldKey}
	err := fsOld.Set("migration-fail-key", []byte("value"))
	require.NoError(t, err)

	// 新 FileStore：oldKey 正确，但将 baseDir 设为只读使迁移写入失败
	newHash := sha256.Sum256([]byte(fmt.Sprintf("%s/medmemo-secret/%d/%x", tmpDir, os.Getuid(), []byte("salt"))))
	fsNew := &FileStore{baseDir: tmpDir, key: newHash[:], oldKey: oldKey}
	// 使迁移写入失败：创建 .salt 目录阻止 MkdirAll... 不，这里 FileStore 已创建好
	// 更简单：修改文件权限使 os.WriteFile 失败
	path := fsNew.filePath("migration-fail-key")
	require.NoError(t, os.Chmod(path, 0444))
	defer os.Chmod(path, 0644)

	got, err := fsNew.Get("migration-fail-key")
	require.NoError(t, err) // 迁移失败不影响读取
	assert.Equal(t, []byte("value"), got)
}

func TestFileStore_Get_BothKeysFail(t *testing.T) {
	fs, err := NewFileStore()
	require.NoError(t, err)
	err = fs.Set("corrupt-key", []byte("value"))
	require.NoError(t, err)

	// 篡改密文使解密失败
	path := fs.filePath("corrupt-key")
	require.NoError(t, os.WriteFile(path, []byte("corrupted-data"), 0600))

	_, err = fs.Get("corrupt-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decryption failed")
}

func TestDecryptWithKey_CiphertextTooShort(t *testing.T) {
	fs := &FileStore{key: make([]byte, 32)}
	_, err := fs.decryptWithKey([]byte("short"), fs.key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestDecryptWithKey_CorruptedData(t *testing.T) {
	fs := &FileStore{key: make([]byte, 32)}
	// 正确加密后再篡改
	encrypted, err := fs.encrypt([]byte("plaintext"))
	require.NoError(t, err)
	// 篡改 nonce 后的数据
	encrypted[len(encrypted)-1] ^= 0xFF

	_, err = fs.decryptWithKey(encrypted, fs.key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decryption failed")
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	// AES-128 需要 16 字节 key，但 GCM 同样需要 16/24/32 字节
	// 用 31 字节 key 触发 aes.NewCipher 错误
	fs := &FileStore{key: make([]byte, 31)}
	_, err := fs.encrypt([]byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aes new cipher failed")
}
