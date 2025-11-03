package snapsqlgo

import (
	"errors"
)

// ErrRowLockNotSupported is returned when a requested row-lock mode is not supported by the dialect.
var ErrRowLockNotSupported = errors.New("row lock is not supported for this dialect")

// EnsureRowLockAllowed panics if a row-lock directive is applied to an unsupported query type.
func EnsureRowLockAllowed(queryType QueryLogQueryType, mode RowLockMode) {
	if mode == RowLockNone {
		return
	}

	if queryType != QueryLogQueryTypeSelect {
		panic("snapsqlgo: WithRowLock is only supported for SELECT queries")
	}
}

// BuildRowLockClausePostgres is an exported helper for generated code to call
// when targeting Postgres. It is a thin wrapper around the internal
// postgresRowLockClause implementation.
func BuildRowLockClausePostgres(mode RowLockMode) (string, error) {
	return postgresRowLockClause(mode)
}

// BuildRowLockClauseMySQL is an exported helper for generated code to call
// when targeting MySQL. It is a thin wrapper around the internal
// mysqlRowLockClause implementation.
func BuildRowLockClauseMySQL(mode RowLockMode) (string, error) {
	return mysqlRowLockClause(mode)
}

// BuildRowLockClauseMariaDB is an exported helper for generated code to call
// when targeting MariaDB. MariaDB behaves like MySQL for row-lock clauses
// so this wraps the same internal implementation.
func BuildRowLockClauseMariaDB(mode RowLockMode) (string, error) {
	// MariaDB behaves like MySQL for row-lock clauses.
	return mysqlRowLockClause(mode)
}

// BuildRowLockClauseSQLite is an exported helper for generated code to call
// when targeting SQLite. SQLite does not support pessimistic row locking
// so this either returns an empty clause or ErrRowLockNotSupported.
func BuildRowLockClauseSQLite(mode RowLockMode) (string, error) {
	if mode == RowLockNone {
		return "", nil
	}

	return "", ErrRowLockNotSupported
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
