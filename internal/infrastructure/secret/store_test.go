package secret

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStore_SetGetDelete(t *testing.T) {
	fs, err := NewFileStore()
	require.NoError(t, err)

	key := "test-api-key"
	value := []byte("sk-test-secret-value-12345")

	// Set
	err = fs.Set(key, value)
	require.NoError(t, err)

	// Get
	got, err := fs.Get(key)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	// Delete
	err = fs.Delete(key)
	require.NoError(t, err)

	// Get after delete should fail
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

	// Delete non-existent key should not error
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

func TestKeyringStore_DelegatesToFileStore(t *testing.T) {
	store, err := NewKeyringStore()
	require.NoError(t, err)

	key := "ollama-api-key"
	value := []byte("secret-key")

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
