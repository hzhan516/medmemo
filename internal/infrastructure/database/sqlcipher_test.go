package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
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

// TestSQLCipherConnector_CreateAndOpen 验证首次启动创建加密库、二次启动正确打开。
func TestSQLCipherConnector_CreateAndOpen(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockSecretStore()

	// 首次启动：创建加密数据库
	conn1, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NotNil(t, conn1)

	// 验证密钥已生成并存储
	key, err := store.Get(dbKeyName)
	require.NoError(t, err)
	require.Len(t, key, 32)

	// 写入测试数据
	ctx := context.Background()
	_, err = conn1.DB().ExecContext(ctx, "CREATE TABLE test_kv (k TEXT PRIMARY KEY, v TEXT)")
	require.NoError(t, err)
	_, err = conn1.DB().ExecContext(ctx, "INSERT INTO test_kv VALUES ('hello', 'world')")
	require.NoError(t, err)
	require.NoError(t, conn1.Close())

	// 二次启动：使用相同密钥正确打开
	conn2, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NotNil(t, conn2)

	var v string
	err = conn2.DB().QueryRowContext(ctx, "SELECT v FROM test_kv WHERE k = 'hello'").Scan(&v)
	require.NoError(t, err)
	assert.Equal(t, "world", v)
	require.NoError(t, conn2.Close())
}

// TestSQLCipherConnector_KeyVerification 验证错误密钥无法打开已存在数据的加密数据库。
func TestSQLCipherConnector_KeyVerification(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockSecretStore()

	// 创建加密数据库并写入数据（确保文件非空，SQLCipher 才会验证密钥）
	conn, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	_, err = conn.DB().Exec("CREATE TABLE verify_test(x INTEGER)")
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	// 篡改密钥为另一个 32 字节值（合法长度但内容错误）
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(i)
	}
	store.data[dbKeyName] = wrongKey

	// 尝试打开应失败（密钥验证失败）
	_, err = NewSQLCipherConnector(tmpDir, store)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database key verification failed")
}

// TestSQLCipherConnector_MigrateFromPlaintext 验证明文数据库自动迁移为加密数据库。
func TestSQLCipherConnector_MigrateFromPlaintext(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockSecretStore()
	dbPath := tmpDir + "/medmemo.db"

	// 1. 使用 plain SQLite 创建明文数据库
	plainDB, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = plainDB.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	_, err = plainDB.Exec("INSERT INTO items VALUES (1, 'apple'), (2, 'banana')")
	require.NoError(t, err)
	require.NoError(t, plainDB.Close())

	// 2. 使用 SQLCipherConnector 打开，触发自动迁移
	conn, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// 3. 验证数据完整
	ctx := context.Background()
	var count int
	err = conn.DB().QueryRowContext(ctx, "SELECT count(*) FROM items").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	var name string
	err = conn.DB().QueryRowContext(ctx, "SELECT name FROM items WHERE id = 2").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "banana", name)

	require.NoError(t, conn.Close())

	// 4. 验证原文件已被加密（不是明文 SQLite 头）
	header := make([]byte, 16)
	f, err := os.Open(dbPath)
	require.NoError(t, err)
	_, err = f.Read(header)
	require.NoError(t, err)
	f.Close()
	assert.NotEqual(t, "SQLite format 3\x00", string(header))

	// 5. 验证 backup 文件存在且是明文
	backupPath := dbPath + ".backup"
	backupHeader := make([]byte, 16)
	f, err = os.Open(backupPath)
	require.NoError(t, err)
	_, err = f.Read(backupHeader)
	require.NoError(t, err)
	f.Close()
	assert.Equal(t, "SQLite format 3\x00", string(backupHeader))
}

// TestSQLCipherConnector_MigrateDataIntegrity 验证迁移后复杂 schema 和数据完整。
func TestSQLCipherConnector_MigrateDataIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockSecretStore()
	dbPath := tmpDir + "/medmemo.db"

	// 1. 创建包含外键、索引的明文数据库
	plainDB, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	schema := `
	CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT);
	CREATE TABLE orders (id TEXT PRIMARY KEY, user_id TEXT, amount REAL,
		FOREIGN KEY (user_id) REFERENCES users(id));
	CREATE INDEX idx_orders_user ON orders(user_id);
	`
	_, err = plainDB.Exec(schema)
	require.NoError(t, err)
	_, err = plainDB.Exec("INSERT INTO users VALUES ('u1', 'Alice'), ('u2', 'Bob')")
	require.NoError(t, err)
	_, err = plainDB.Exec("INSERT INTO orders VALUES ('o1', 'u1', 100.5), ('o2', 'u1', 200.0), ('o3', 'u2', 50.0)")
	require.NoError(t, err)
	require.NoError(t, plainDB.Close())

	// 2. 迁移
	conn, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	defer conn.Close()

	// 3. 验证 users
	ctx := context.Background()
	rows, err := conn.DB().QueryContext(ctx, "SELECT id, name FROM users ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()

	var users []struct{ id, name string }
	for rows.Next() {
		var u struct{ id, name string }
		require.NoError(t, rows.Scan(&u.id, &u.name))
		users = append(users, u)
	}
	require.NoError(t, rows.Err())
	require.Len(t, users, 2)
	assert.Equal(t, "u1", users[0].id)
	assert.Equal(t, "Alice", users[0].name)

	// 4. 验证 orders + JOIN
	var total float64
	err = conn.DB().QueryRowContext(ctx, "SELECT SUM(amount) FROM orders WHERE user_id = 'u1'").Scan(&total)
	require.NoError(t, err)
	assert.InDelta(t, 300.5, total, 0.001)

	// 5. 验证索引存在
	var idxCount int
	err = conn.DB().QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_orders_user'").Scan(&idxCount)
	require.NoError(t, err)
	assert.Equal(t, 1, idxCount)
}

// TestSQLCipherConnector_MigrateSchema 验证 SQLCipherConnector 的 Migrate 方法正确执行 schema 升级。
func TestSQLCipherConnector_MigrateSchema(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockSecretStore()

	conn, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, conn.Migrate(ctx))

	// 验证 conversations 表存在
	var tableCount int
	err = conn.DB().QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'conversations'").Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 1, tableCount)

	// 验证 messages 表存在
	err = conn.DB().QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'messages'").Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 1, tableCount)

	// 验证 memories 表存在
	err = conn.DB().QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'memories'").Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 1, tableCount)

	// 验证 disclaimer_acceptance 表存在
	err = conn.DB().QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'disclaimer_acceptance'").Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 1, tableCount)

	// 验证 user_version = 9（含 v1.1 三层记忆表结构 + is_sensitive 列 + audit_logs 表）
	var version int
	err = conn.DB().QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 9, version)
}

// TestSQLCipherConnector_PRAGMAForeignKeys 验证外键约束在加密数据库中正常工作。
func TestSQLCipherConnector_PRAGMAForeignKeys(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockSecretStore()

	conn, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()
	_, err = conn.DB().ExecContext(ctx, `
		CREATE TABLE parent (id TEXT PRIMARY KEY);
		CREATE TABLE child (id TEXT PRIMARY KEY, parent_id TEXT,
			FOREIGN KEY (parent_id) REFERENCES parent(id));
	`)
	require.NoError(t, err)

	// 外键约束应阻止插入不存在的 parent_id
	_, err = conn.DB().ExecContext(ctx, "INSERT INTO child VALUES ('c1', 'nonexistent')")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FOREIGN KEY")
}

// TestSQLCipherConnector_isPlaintextDB 验证明文检测逻辑。
func TestSQLCipherConnector_isPlaintextDB(t *testing.T) {
	t.Run("file_not_exists", func(t *testing.T) {
		got, err := isPlaintextDB("/nonexistent/path/db.sqlite")
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("plaintext_db", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/plain.db"
		db, err := sql.Open("sqlite", dbPath)
		require.NoError(t, err)
		_, err = db.Exec("CREATE TABLE t(x INTEGER)")
		require.NoError(t, err)
		require.NoError(t, db.Close())

		got, err := isPlaintextDB(dbPath)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("encrypted_db", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := newMockSecretStore()
		conn, err := NewSQLCipherConnector(tmpDir, store)
		require.NoError(t, err)
		require.NoError(t, conn.Close())

		got, err := isPlaintextDB(tmpDir + "/medmemo.db")
		require.NoError(t, err)
		assert.False(t, got)
	})
}

// TestSQLCipherConnector_getOrCreateKey 验证密钥生成与复用。
func TestSQLCipherConnector_getOrCreateKey(t *testing.T) {
	store := newMockSecretStore()

	// 首次调用生成密钥
	key1, err := getOrCreateKey(store)
	require.NoError(t, err)
	require.Len(t, key1, 32)

	// 二次调用复用同一密钥
	key2, err := getOrCreateKey(store)
	require.NoError(t, err)
	assert.Equal(t, key1, key2)
}

// TestSQLCipherConnector_getOrCreateKey_InvalidLength 验证密钥长度非法时返回错误。
func TestSQLCipherConnector_getOrCreateKey_InvalidLength(t *testing.T) {
	store := newMockSecretStore()
	store.data[dbKeyName] = []byte("short-key")

	_, err := getOrCreateKey(store)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid length")
}

// TestSQLCipherConnector_MigrateFromPlaintext_AlreadyEncrypted 验证已加密数据库不会重复迁移。
func TestSQLCipherConnector_MigrateFromPlaintext_AlreadyEncrypted(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockSecretStore()

	// 第一次：创建加密数据库
	conn1, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	_, err = conn1.DB().Exec("CREATE TABLE encrypted_only(id INTEGER)")
	require.NoError(t, err)
	require.NoError(t, conn1.Close())

	// 第二次：用相同的 store（相同密钥）打开，不应触发迁移
	conn2, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)

	// 验证表仍然存在
	var count int
	err = conn2.DB().QueryRow("SELECT count(*) FROM encrypted_only").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	require.NoError(t, conn2.Close())
}

// TestSQLCipherConnector_MigrateFromPlaintext_EmptyPlaintext 验证空明文文件不会触发迁移。
func TestSQLCipherConnector_MigrateFromPlaintext_EmptyPlaintext(t *testing.T) {
	tmpDir := t.TempDir()
	store := newMockSecretStore()
	dbPath := tmpDir + "/medmemo.db"

	// 创建一个空的明文文件
	require.NoError(t, os.WriteFile(dbPath, []byte{}, 0644))

	// 空文件不应触发迁移（isPlaintextDB 中 info.Size() == 0 且无 WAL 返回 false）
	conn, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.NoError(t, conn.Close())
}
