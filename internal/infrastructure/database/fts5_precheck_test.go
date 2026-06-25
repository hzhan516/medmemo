package database

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestFTS5Availability_Stage0 是 Stage 0 预检：验证两个 driver 的 FTS5 可用性。
// 结论决定后续路径：
//   - 若 SQLCipher 与 modernc SQLite 均支持 FTS5，则迁移到 FTS5 全文检索；
//   - 否则按 SQL LIKE 数据库层过滤方案执行，不强行迁移 FTS5。
func TestFTS5Availability_Stage0(t *testing.T) {
	t.Run("modernc_sqlite_supports_fts5", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		assert.NoError(t, probeFTS5(db), "modernc.org/sqlite 应支持 FTS5")
	})

	t.Run("sqlcipher_lacks_fts5", func(t *testing.T) {
		// SQLCipher（go-sqlcipher）与 go-sqlite3 共享 sqlite3 驱动名
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		assert.Error(t, probeFTS5(db), "go-sqlcipher 不应支持 FTS5，必须回退到 SQL LIKE 方案")
	})
}

func probeFTS5(db *sql.DB) error {
	_, err := db.Exec(`CREATE VIRTUAL TABLE fts5_test USING fts5(content)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO fts5_test(content) VALUES ('hello world')`)
	if err != nil {
		return err
	}
	var cnt int
	err = db.QueryRow(`SELECT count(*) FROM fts5_test WHERE fts5_test MATCH 'hello'`).Scan(&cnt)
	if err != nil {
		return err
	}
	if cnt != 1 {
		return assert.AnError
	}
	return nil
}
