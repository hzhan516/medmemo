package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestIsSQLiteConstraint_NilError(t *testing.T) {
	t.Parallel()

	assert.False(t, IsSQLiteUniqueConstraintOn(nil, "email"))
	assert.False(t, IsSQLitePrimaryOrUniqueConstraintOn(nil, "email"))
}

func TestIsSQLiteConstraint_EmptyColumn(t *testing.T) {
	t.Parallel()

	err := errors.New("UNIQUE constraint failed: users.email")
	// 未指定列名时应直接返回 false
	assert.False(t, IsSQLiteUniqueConstraintOn(err, ""))
	assert.False(t, IsSQLitePrimaryOrUniqueConstraintOn(err, ""))
}

func TestIsSQLiteUniqueConstraintOn_MessagePath(t *testing.T) {
	t.Parallel()

	err := errors.New("UNIQUE constraint failed: users.email")

	// 命中列名且为唯一约束
	assert.True(t, IsSQLiteUniqueConstraintOn(err, "email"))
	assert.True(t, IsSQLitePrimaryOrUniqueConstraintOn(err, "email"))

	// 列名不匹配时应返回 false
	assert.False(t, IsSQLiteUniqueConstraintOn(err, "username"))
	assert.False(t, IsSQLitePrimaryOrUniqueConstraintOn(err, "username"))
}

func TestIsSQLitePrimaryConstraintOn_MessagePath(t *testing.T) {
	t.Parallel()

	err := errors.New("PRIMARY KEY constraint failed: users.id")

	// 主键冲突：仅 PrimaryOrUnique 判定为真
	assert.True(t, IsSQLitePrimaryOrUniqueConstraintOn(err, "id"))
	// 纯唯一约束判定不应命中主键消息
	assert.False(t, IsSQLiteUniqueConstraintOn(err, "id"))
}

func TestIsSQLiteConstraint_MessagePath_UnrelatedError(t *testing.T) {
	t.Parallel()

	// 既非唯一也非主键约束，即便列名出现也应返回 false
	err := errors.New("no such column: email")
	assert.False(t, IsSQLiteUniqueConstraintOn(err, "email"))
	assert.False(t, IsSQLitePrimaryOrUniqueConstraintOn(err, "email"))
}

func TestIsSQLiteConstraint_MessagePath_ColumnMissing(t *testing.T) {
	t.Parallel()

	// 约束消息命中但列名不在错误文本中
	err := errors.New("UNIQUE constraint failed: users.email")
	assert.False(t, IsSQLiteUniqueConstraintOn(err, "phone"))
}

// TestIsSQLiteUniqueConstraintOn_ModerncError 使用真实 modernc.org/sqlite
// 触发唯一约束错误，覆盖基于扩展错误码的判定分支。
func TestIsSQLiteUniqueConstraintOn_ModerncError(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT UNIQUE)`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO t (name) VALUES ('alice')`)
	require.NoError(t, err)

	// 再次插入同名记录触发 UNIQUE 冲突
	_, dupErr := db.Exec(`INSERT INTO t (name) VALUES ('alice')`)
	require.Error(t, dupErr)

	assert.True(t, IsSQLiteUniqueConstraintOn(dupErr, "name"))
	assert.True(t, IsSQLitePrimaryOrUniqueConstraintOn(dupErr, "name"))
	// 列名不匹配时应返回 false
	assert.False(t, IsSQLiteUniqueConstraintOn(dupErr, "id"))
}

// TestIsSQLitePrimaryConstraintOn_ModerncError 触发主键冲突，
// 验证主键约束码不会被误判为纯唯一约束。
func TestIsSQLitePrimaryConstraintOn_ModerncError(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE p (id INTEGER PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO p (id, name) VALUES (1, 'x')`)
	require.NoError(t, err)

	_, dupErr := db.Exec(`INSERT INTO p (id, name) VALUES (1, 'y')`)
	require.Error(t, dupErr)

	assert.True(t, IsSQLitePrimaryOrUniqueConstraintOn(dupErr, "id"))
}
