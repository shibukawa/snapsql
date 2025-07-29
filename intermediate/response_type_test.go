package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	. "github.com/shibukawa/snapsql"
)

func TestDetermineResponseType(t *testing.T) {
	// Test with empty table info
	t.Run("EmptyTableInfo", func(t *testing.T) {
		// This test would require a proper SQL statement node
		// For now, we'll test the conversion function
		result := convertTableInfoToSchemas(nil)
		assert.Equal(t, []DatabaseSchema(nil), result)
	})

	// Test table info conversion
	t.Run("ConvertTableInfoToSchemas", func(t *testing.T) {
		tableInfo := map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"id":    {Name: "id", DataType: "int", Nullable: false, IsPrimaryKey: true},
					"name":  {Name: "name", DataType: "string", Nullable: true},
					"email": {Name: "email", DataType: "string", Nullable: true},
				},
			},
			"orders": {
				Name: "orders",
				Columns: map[string]*ColumnInfo{
					"id":      {Name: "id", DataType: "int", Nullable: false, IsPrimaryKey: true},
					"user_id": {Name: "user_id", DataType: "int", Nullable: false},
					"amount":  {Name: "amount", DataType: "decimal", Nullable: false},
				},
			},
		}

		schemas := convertTableInfoToSchemas(tableInfo)

		// Verify schema structure
		assert.Equal(t, 1, len(schemas))
		assert.Equal(t, "default", schemas[0].Name)
		assert.Equal(t, 2, len(schemas[0].Tables))

		// Find users table
		var usersTable *TableInfo
		var ordersTable *TableInfo
		for _, table := range schemas[0].Tables {
			if table.Name == "users" {
				usersTable = table
			} else if table.Name == "orders" {
				ordersTable = table
			}
		}

		// Verify users table
		assert.True(t, usersTable != nil)
		assert.Equal(t, 3, len(usersTable.Columns))

		// Verify orders table
		assert.True(t, ordersTable != nil)
		assert.Equal(t, 3, len(ordersTable.Columns))
	})
}
