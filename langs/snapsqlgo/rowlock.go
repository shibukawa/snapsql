package snapsqlgo

import (
	"strings"

	"github.com/shibukawa/snapsql"
)

// EnsureRowLockAllowed panics if a row-lock directive is applied to an unsupported query type.
func EnsureRowLockAllowed(queryType QueryLogQueryType, mode RowLockMode) {
	if mode == RowLockNone {
		return
	}

	if queryType != QueryLogQueryTypeSelect {
		panic("snapsqlgo: WithRowLock is only supported for SELECT queries")
	}
}

// BuildRowLockClause returns the dialect-specific FOR clause for the requested lock mode.
// It returns an empty string when the dialect does not support pessimistic locking or when the
// mode is RowLockNone.
func BuildRowLockClause(dialect string, mode RowLockMode) (string, error) {
	if mode == RowLockNone {
		return "", nil
	}

	dl := strings.ToLower(dialect)
	switch dl {
	case "", string(snapsql.DialectPostgres), "postgresql", "pg":
		return postgresRowLockClause(mode)
	case string(snapsql.DialectMySQL), string(snapsql.DialectMariaDB):
		return mysqlRowLockClause(mode)
	case string(snapsql.DialectSQLite), "sqlite3":
		return "", nil
	default:
		return "", nil
	}
}

func postgresRowLockClause(mode RowLockMode) (string, error) {
	switch mode {
	case RowLockForUpdate:
		return " FOR UPDATE", nil
	case RowLockForShare:
		return " FOR SHARE", nil
	case RowLockForUpdateNoWait:
		return " FOR UPDATE NOWAIT", nil
	case RowLockForUpdateSkipLocked:
		return " FOR UPDATE SKIP LOCKED", nil
	}

	return "", nil
}

func mysqlRowLockClause(mode RowLockMode) (string, error) {
	switch mode {
	case RowLockForUpdate:
		return " FOR UPDATE", nil
	case RowLockForShare:
		return " FOR SHARE", nil
	case RowLockForUpdateNoWait:
		return " FOR UPDATE NOWAIT", nil
	case RowLockForUpdateSkipLocked:
		return " FOR UPDATE SKIP LOCKED", nil
	}

	return "", nil
}
