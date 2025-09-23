package main

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/query"
)

func TestFormatSQLForDisplay_Postgres_BasicSpacing(t *testing.T) {
	in := "SELECT id FROM items WHERE id =?ORDER BY id"
	out := query.FormatSQLForDialect(in, "postgresql")
	assert.Equal(t, "SELECT id FROM items WHERE id =$1 ORDER BY id", out)
}

func TestFormatSQLForDisplay_Postgres_ParenNoExtraSpace(t *testing.T) {
	in := "INSERT INTO t(x) VALUES(?)"
	out := query.FormatSQLForDialect(in, "postgresql")
	assert.Equal(t, "INSERT INTO t(x) VALUES($1)", out)
}

func TestFormatSQLForDisplay_MySQL_NoConversion(t *testing.T) {
	in := "SELECT id,flag FROM t WHERE id=?AND flag=?"
	out := query.FormatSQLForDialect(in, "mysql")
	// mysqlは?のまま。直後スペース補正は方言に依らず行う
	assert.Equal(t, "SELECT id,flag FROM t WHERE id=? AND flag=?", out)
}

func TestFormatSQLForDisplay_SQLite_NoConversion(t *testing.T) {
	in := "UPDATE t SET name=?WHERE id=?"
	out := query.FormatSQLForDialect(in, "sqlite")
	assert.Equal(t, "UPDATE t SET name=? WHERE id=?", out)
}
