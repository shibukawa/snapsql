package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

func TestSourceType_String(t *testing.T) {
	assert.Equal(t, "Table", SourceTypeTable.String())
	assert.Equal(t, "Expression", SourceTypeExpression.String())
	assert.Equal(t, "Subquery", SourceTypeSubquery.String())
	assert.Equal(t, "Aggregate", SourceTypeAggregate.String())
	assert.Equal(t, "Literal", SourceTypeLiteral.String())
}

func TestTableReference_GetField(t *testing.T) {
	field1 := &FieldSource{Name: "id", SourceType: SourceTypeTable}
	field2 := &FieldSource{Name: "name", Alias: "user_name", SourceType: SourceTypeTable}

	tableRef := &TableReference{
		Name:   "users",
		Fields: []*FieldSource{field1, field2},
	}

	// Test finding by name
	found := tableRef.GetField("id")
	assert.Equal(t, field1, found)

	// Test finding by alias
	found = tableRef.GetField("user_name")
	assert.Equal(t, field2, found)

	// Test not found
	found = tableRef.GetField("nonexistent")
	assert.Equal(t, (*FieldSource)(nil), found)
}

func TestDependencyType_String(t *testing.T) {
	assert.Equal(t, "CTE", cmn.SQDependencyCTE.String())
	assert.Equal(t, "Subquery", cmn.SQDependencySubquery.String())
	assert.Equal(t, "Main", cmn.SQDependencyMain.String())
}
