package secret

import (
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
