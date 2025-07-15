package typeinference2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
)

// TestSchemaResolver tests the basic functionality of SchemaResolver
func TestSchemaResolver(t *testing.T) {
	// Create test database schemas
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			Tables: []*snapsql.TableInfo{
				{
					Name:   "users",
					Schema: "testdb",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
						"name": {
							Name:     "name",
							DataType: "string",
							Nullable: false,
						},
						"email": {
							Name:     "email",
							DataType: "string",
							Nullable: true,
						},
					},
				},
				{
					Name:   "posts",
					Schema: "testdb",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
						"user_id": {
							Name:     "user_id",
							DataType: "int",
							Nullable: false,
						},
						"title": {
							Name:     "title",
							DataType: "string",
							Nullable: false,
						},
						"content": {
							Name:     "content",
							DataType: "string",
							Nullable: true,
						},
					},
				},
			},
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
			},
		},
	}

	resolver := NewSchemaResolver(schemas)

	t.Run("ResolveTableColumn - Valid", func(t *testing.T) {
		column, err := resolver.ResolveTableColumn("testdb", "users", "name")
		assert.NoError(t, err)
		assert.Equal(t, "name", column.Name)
		assert.Equal(t, "string", column.DataType)
		assert.Equal(t, false, column.Nullable)
	})

	t.Run("ResolveTableColumn - Invalid Table", func(t *testing.T) {
		_, err := resolver.ResolveTableColumn("testdb", "nonexistent", "name")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("ResolveTableColumn - Invalid Column", func(t *testing.T) {
		_, err := resolver.ResolveTableColumn("testdb", "users", "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("ValidateTableExists - Valid", func(t *testing.T) {
		err := resolver.ValidateTableExists("testdb", "users")
		assert.NoError(t, err)
	})

	t.Run("ValidateTableExists - Invalid Schema", func(t *testing.T) {
		err := resolver.ValidateTableExists("nonexistent", "users")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema 'nonexistent' does not exist")
	})

	t.Run("GetTableColumns", func(t *testing.T) {
		columns, err := resolver.GetTableColumns("testdb", "users")
		assert.NoError(t, err)
		assert.Equal(t, 3, len(columns))

		// Check that all expected columns are present
		columnNames := make(map[string]bool)
		for _, col := range columns {
			columnNames[col.Name] = true
		}
		assert.True(t, columnNames["id"])
		assert.True(t, columnNames["name"])
		assert.True(t, columnNames["email"])
	})

	t.Run("ConvertToFieldType", func(t *testing.T) {
		column := &snapsql.ColumnInfo{
			Name:     "test_col",
			DataType: "string",
			Nullable: true,
		}

		fieldType := resolver.ConvertToFieldType(column)
		assert.Equal(t, "string", fieldType.BaseType)
		assert.Equal(t, true, fieldType.IsNullable)
	})

	t.Run("FindColumnInTables", func(t *testing.T) {
		availableTables := []string{"testdb.users", "testdb.posts"}

		// Find column that exists in multiple tables
		matches := resolver.FindColumnInTables("id", availableTables)
		assert.Equal(t, 2, len(matches))
		found := false
		for _, match := range matches {
			if match == "testdb.users" {
				found = true
				break
			}
		}
		assert.True(t, found)

		found = false
		for _, match := range matches {
			if match == "testdb.posts" {
				found = true
				break
			}
		}
		assert.True(t, found)

		// Find column that exists in only one table
		matches = resolver.FindColumnInTables("name", availableTables)
		assert.Equal(t, 1, len(matches))
		found = false
		for _, match := range matches {
			if match == "testdb.users" {
				found = true
				break
			}
		}
		assert.True(t, found)

		// Find column that doesn't exist
		matches = resolver.FindColumnInTables("nonexistent", availableTables)
		assert.Equal(t, 0, len(matches))
	})
}

// TestSchemaValidator tests the schema validation functionality
func TestSchemaValidator(t *testing.T) {
	// Create test database schemas
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			Tables: []*snapsql.TableInfo{
				{
					Name:   "users",
					Schema: "testdb",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
						"name": {
							Name:     "name",
							DataType: "string",
							Nullable: false,
						},
					},
				},
			},
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
			},
		},
	}

	resolver := NewSchemaResolver(schemas)
	validator := NewSchemaValidator(resolver)
	validator.SetAvailableTables([]string{"testdb.users"})

	t.Run("ValidateTableColumn - Valid", func(t *testing.T) {
		err := validator.validateTableColumn(0, "users", "name")
		assert.Equal(t, (*ValidationError)(nil), err)
	})

	t.Run("ValidateTableColumn - Invalid Table", func(t *testing.T) {
		err := validator.validateTableColumn(0, "nonexistent", "name")
		assert.True(t, err != nil)
		assert.Equal(t, TableNotFound, err.ErrorType)
	})

	t.Run("ValidateTableColumn - Invalid Column", func(t *testing.T) {
		err := validator.validateTableColumn(0, "users", "nonexistent")
		assert.True(t, err != nil)
		assert.Equal(t, ColumnNotFound, err.ErrorType)
	})

	t.Run("ValidateSingleColumn - Valid", func(t *testing.T) {
		err := validator.validateSingleColumn(0, "name")
		assert.Equal(t, (*ValidationError)(nil), err)
	})

	t.Run("ValidateSingleColumn - Not Found", func(t *testing.T) {
		err := validator.validateSingleColumn(0, "nonexistent")
		assert.True(t, err != nil)
		assert.Equal(t, ColumnNotFound, err.ErrorType)
	})

	t.Run("TypeCompatibility", func(t *testing.T) {
		// Test compatible types
		assert.True(t, validator.areTypesCompatible("int", "int"))
		assert.True(t, validator.areTypesCompatible("int", "bigint"))
		assert.True(t, validator.areTypesCompatible("string", "text"))
		assert.True(t, validator.areTypesCompatible("string", "varchar"))

		// Test incompatible types
		assert.False(t, validator.areTypesCompatible("int", "string"))
		assert.False(t, validator.areTypesCompatible("bool", "timestamp"))
	})
}

// TestFieldNameGenerator tests the field name generation functionality
func TestFieldNameGenerator(t *testing.T) {
	generator := NewFieldNameGenerator()

	t.Run("Basic Name Generation", func(t *testing.T) {
		name1 := generator.GenerateFieldName("id", "", "")
		assert.Equal(t, "id", name1)

		name2 := generator.GenerateFieldName("name", "", "")
		assert.Equal(t, "name", name2)
	})

	t.Run("Function Name Generation", func(t *testing.T) {
		name := generator.GenerateFieldName("", "COUNT", "")
		assert.Equal(t, "count", name)

		name = generator.GenerateFieldName("", "SUM", "")
		assert.Equal(t, "sum", name)
	})

	t.Run("Expression Type Name Generation", func(t *testing.T) {
		name := generator.GenerateFieldName("", "", "literal")
		assert.Equal(t, "literal", name)
	})

	t.Run("Duplicate Name Handling", func(t *testing.T) {
		gen := NewFieldNameGenerator()

		name1 := gen.GenerateFieldName("id", "", "")
		assert.Equal(t, "id", name1)

		name2 := gen.GenerateFieldName("id", "", "")
		assert.Equal(t, "id2", name2)

		name3 := gen.GenerateFieldName("id", "", "")
		assert.Equal(t, "id3", name3)
	})

	t.Run("Reserve Field Name", func(t *testing.T) {
		gen := NewFieldNameGenerator()
		gen.ReserveFieldName("reserved")

		name := gen.GenerateFieldName("reserved", "", "")
		assert.Equal(t, "reserved2", name)
	})
}

// TestTypeInferenceEngine2 tests the core type inference functionality
func TestTypeInferenceEngine2_Creation(t *testing.T) {
	// Create test database schemas
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			Tables: []*snapsql.TableInfo{
				{
					Name:   "users",
					Schema: "testdb",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
					},
				},
			},
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
			},
		},
	}

	t.Run("Engine Creation", func(t *testing.T) {
		engine := NewTypeInferenceEngine2(schemas, nil, nil)
		assert.True(t, engine != nil)
		assert.True(t, engine.schemaResolver != nil)
		assert.True(t, engine.fieldNameGen != nil)
		assert.True(t, engine.context != nil)
		assert.Equal(t, snapsql.DialectPostgres, engine.context.Dialect)
	})

	t.Run("Dialect Detection", func(t *testing.T) {
		// Test MySQL dialect
		mysqlSchemas := []snapsql.DatabaseSchema{
			{
				DatabaseInfo: snapsql.DatabaseInfo{Type: "mysql"},
			},
		}
		engine := NewTypeInferenceEngine2(mysqlSchemas, nil, nil)
		assert.Equal(t, snapsql.DialectMySQL, engine.context.Dialect)

		// Test SQLite dialect
		sqliteSchemas := []snapsql.DatabaseSchema{
			{
				DatabaseInfo: snapsql.DatabaseInfo{Type: "sqlite"},
			},
		}
		engine = NewTypeInferenceEngine2(sqliteSchemas, nil, nil)
		assert.Equal(t, snapsql.DialectSQLite, engine.context.Dialect)
	})
}

// Test helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("normalizeType", func(t *testing.T) {
		assert.Equal(t, "string", normalizeType("VARCHAR"))
		assert.Equal(t, "string", normalizeType("TEXT"))
		assert.Equal(t, "int", normalizeType("INTEGER"))
		assert.Equal(t, "int", normalizeType("BIGINT"))
		assert.Equal(t, "bool", normalizeType("BOOLEAN"))
		assert.Equal(t, "unknown", normalizeType("UNKNOWN"))
	})

	t.Run("levenshteinDistance", func(t *testing.T) {
		assert.Equal(t, 0, levenshteinDistance("test", "test"))
		assert.Equal(t, 1, levenshteinDistance("test", "tests"))
		assert.Equal(t, 1, levenshteinDistance("test", "tost"))
		assert.Equal(t, 4, levenshteinDistance("test", "abcd"))
	})
}
