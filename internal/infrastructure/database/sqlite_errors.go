package database

import (
	"errors"
	"strings"

	sqlcipher "github.com/mutecomm/go-sqlcipher"
	moderncsqlite "modernc.org/sqlite"
	modernclib "modernc.org/sqlite/lib"
)

// IsSQLitePrimaryOrUniqueConstraintOn 判断错误是否来自指定列的主键或唯一键冲突。
func IsSQLitePrimaryOrUniqueConstraintOn(err error, column string) bool {
	return isSQLiteConstraintOn(err, column, true, true)
}

// IsSQLiteUniqueConstraintOn 判断错误是否来自指定列的唯一键冲突。
func IsSQLiteUniqueConstraintOn(err error, column string) bool {
	return isSQLiteConstraintOn(err, column, false, true)
}

func isSQLiteConstraintOn(err error, column string, allowPrimary bool, allowUnique bool) bool {
	if err == nil || column == "" {
		return false
	}

	if code, ok := sqlCipherExtendedCode(err); ok {
		return sqlCipherConstraintMatches(code, allowPrimary, allowUnique) && errorMentionsColumn(err, column)
	}

	if code, ok := moderncErrorCode(err); ok {
		return moderncConstraintMatches(code, allowPrimary, allowUnique) && errorMentionsColumn(err, column)
	}

	return constraintMessageMatches(err, column, allowPrimary, allowUnique)
}

func sqlCipherExtendedCode(err error) (sqlcipher.ErrNoExtended, bool) {
	var valueErr sqlcipher.Error
	if errors.As(err, &valueErr) {
		return valueErr.ExtendedCode, true
	}

	var pointerErr *sqlcipher.Error
	if errors.As(err, &pointerErr) && pointerErr != nil {
		return pointerErr.ExtendedCode, true
	}

	return 0, false
}

func moderncErrorCode(err error) (int, bool) {
	var sqliteErr *moderncsqlite.Error
	if errors.As(err, &sqliteErr) && sqliteErr != nil {
		return sqliteErr.Code(), true
	}
	return 0, false
}

func sqlCipherConstraintMatches(code sqlcipher.ErrNoExtended, allowPrimary bool, allowUnique bool) bool {
	if allowPrimary && code == sqlcipher.ErrConstraintPrimaryKey {
		return true
	}
	if allowUnique && code == sqlcipher.ErrConstraintUnique {
		return true
	}
	return false
}

func moderncConstraintMatches(code int, allowPrimary bool, allowUnique bool) bool {
	if allowPrimary && code == modernclib.SQLITE_CONSTRAINT_PRIMARYKEY {
		return true
	}
	if allowUnique && code == modernclib.SQLITE_CONSTRAINT_UNIQUE {
		return true
	}
	return false
}

func errorMentionsColumn(err error, column string) bool {
	return strings.Contains(err.Error(), column)
}

func constraintMessageMatches(err error, column string, allowPrimary bool, allowUnique bool) bool {
	message := strings.ToLower(err.Error())
	if !strings.Contains(err.Error(), column) {
		return false
	}
	if allowPrimary && strings.Contains(message, "primary key") {
		return true
	}
	if allowUnique && strings.Contains(message, "unique constraint failed") {
		return true
	}
	return false
}
