package intermediate

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
)

// build simple table info for tests
func buildTableInfo(table string, pkCols []string, extraCols ...string) map[string]*snapsql.TableInfo {
	cols := map[string]*snapsql.ColumnInfo{}
	// primary keys
	for _, c := range pkCols {
		cols[c] = &snapsql.ColumnInfo{Name: c, DataType: "int", IsPrimaryKey: true}
	}
	// extras
	for _, c := range extraCols {
		if _, ok := cols[c]; !ok {
			cols[c] = &snapsql.ColumnInfo{Name: c, DataType: "int"}
		}
	}

	return map[string]*snapsql.TableInfo{
		table: {
			Name:    table,
			Columns: cols,
		},
	}
}

func TestAffinity_SingleTable_NonPKWhere_ShouldBeMany(t *testing.T) {
	sql := `/*#
name: ListsByBoard
function_name: listsByBoard
description: lists by board (non PK where)
*/
SELECT id, title FROM lists WHERE board_id = 1 AND is_archived = 0`

	stmt, _, err := parser.ParseSQLFile(strings.NewReader(sql), nil, ".", ".", parser.DefaultOptions)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ti := buildTableInfo("lists", []string{"id"}, "board_id", "is_archived", "title")
	aff := determineResponseAffinity(stmt, ti)
	assert.Equal(t, ResponseAffinityMany, aff)
}

func TestAffinity_SingleTable_PKEquality_ShouldBeOne(t *testing.T) {
	sql := `/*#
name: ListByID
function_name: listByID
description: list by id (PK equality)
*/
SELECT id, title FROM lists WHERE id = 1`

	stmt, _, err := parser.ParseSQLFile(strings.NewReader(sql), nil, ".", ".", parser.DefaultOptions)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	ti := buildTableInfo("lists", []string{"id"}, "title")
	aff := determineResponseAffinity(stmt, ti)
	assert.Equal(t, ResponseAffinityOne, aff)
}
