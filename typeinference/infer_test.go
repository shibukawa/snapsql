package typeinference2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

func TestInferFieldTypes_SimpleSelect(t *testing.T) {
	// Setup test schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			DatabaseInfo: snapsql.DatabaseInfo{
				Name: "testdb",
				Type: "postgres",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name:   "users",
					Schema: "public",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
						"name": {
							Name:     "name",
							DataType: "varchar",
							Nullable: true,
						},
						"age": {
							Name:     "age",
							DataType: "int",
							Nullable: true,
						},
					},
				},
			},
		},
	}

	// Setup test SELECT statement
	selectStmt := &parsercommon.SelectStatement{
		Select: &parsercommon.SelectClause{
			Fields: []parsercommon.SelectField{
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "id",
					ExplicitName:  false,
				},
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "name",
					ExplicitName:  false,
				},
			},
		},
		From: &parsercommon.FromClause{
			Tables: []parsercommon.TableReferenceForFrom{
				{
					TableReference: parsercommon.TableReference{
						TableName: "users",
						Name:      "users",
					},
				},
			},
		},
	}

	// Test InferFieldTypes
	result, err := InferFieldTypes(schemas, selectStmt, nil)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))

	// Check first field (id)
	assert.Equal(t, "id", result[0].Name)
	assert.Equal(t, "id", result[0].OriginalName)
	assert.Equal(t, "int", result[0].Type.BaseType)
	assert.Equal(t, false, result[0].Type.IsNullable)
	assert.Equal(t, "column", result[0].Source.Type)
	assert.Equal(t, "users", result[0].Source.Table)
	assert.Equal(t, "id", result[0].Source.Column)

	// Check second field (name)
	assert.Equal(t, "name", result[1].Name)
	assert.Equal(t, "name", result[1].OriginalName)
	assert.Equal(t, "varchar", result[1].Type.BaseType) // Raw type from schema
	assert.Equal(t, true, result[1].Type.IsNullable)
	assert.Equal(t, "column", result[1].Source.Type)
	assert.Equal(t, "users", result[1].Source.Table)
	assert.Equal(t, "name", result[1].Source.Column)
}

func TestInferSelectFieldTypes_SpecializedFunction(t *testing.T) {
	// Setup test schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			DatabaseInfo: snapsql.DatabaseInfo{
				Name: "testdb",
				Type: "postgres",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name:   "users",
					Schema: "public",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
						"name": {
							Name:     "name",
							DataType: "varchar",
							Nullable: true,
						},
					},
				},
			},
		},
	}

	// Setup test SELECT statement
	selectStmt := &parsercommon.SelectStatement{
		Select: &parsercommon.SelectClause{
			Fields: []parsercommon.SelectField{
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "name",
					ExplicitName:  false,
				},
			},
		},
		From: &parsercommon.FromClause{
			Tables: []parsercommon.TableReferenceForFrom{
				{
					TableReference: parsercommon.TableReference{
						TableName: "users",
						Name:      "users",
					},
				},
			},
		},
	}

	// Test InferSelectFieldTypes
	result, err := InferSelectFieldTypes(schemas, selectStmt, nil)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "name", result[0].Name)
	assert.Equal(t, "varchar", result[0].Type.BaseType) // Raw type from schema
}

func TestInferFieldTypesSimple_NoSubqueryInfo(t *testing.T) {
	// Setup test schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			DatabaseInfo: snapsql.DatabaseInfo{
				Name: "testdb",
				Type: "postgres",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name:   "users",
					Schema: "public",
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
		},
	}

	// Setup test SELECT statement
	selectStmt := &parsercommon.SelectStatement{
		Select: &parsercommon.SelectClause{
			Fields: []parsercommon.SelectField{
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "id",
					ExplicitName:  false,
				},
			},
		},
		From: &parsercommon.FromClause{
			Tables: []parsercommon.TableReferenceForFrom{
				{
					TableReference: parsercommon.TableReference{
						TableName: "users",
						Name:      "users",
					},
				},
			},
		},
	}

	// Test InferFieldTypesSimple
	result, err := InferFieldTypesSimple(schemas, selectStmt)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "id", result[0].Name)
	assert.Equal(t, "int", result[0].Type.BaseType)
}

func TestInferFieldTypesWithOptions_CustomOptions(t *testing.T) {
	// Setup test schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			DatabaseInfo: snapsql.DatabaseInfo{
				Name: "testdb",
				Type: "mysql",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name:   "products",
					Schema: "mydb",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
						"price": {
							Name:     "price",
							DataType: "decimal",
							Nullable: true,
						},
					},
				},
			},
		},
	}

	// Setup test SELECT statement with alias
	selectStmt := &parsercommon.SelectStatement{
		Select: &parsercommon.SelectClause{
			Fields: []parsercommon.SelectField{
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "p",
					OriginalField: "price",
					ExplicitName:  false,
				},
			},
		},
		From: &parsercommon.FromClause{
			Tables: []parsercommon.TableReferenceForFrom{
				{
					TableReference: parsercommon.TableReference{
						TableName:    "products",
						Name:         "p",
						ExplicitName: true,
					},
				},
			},
		},
	}

	// Test with custom options
	options := &InferenceOptions{
		Dialect: snapsql.DialectMySQL,
		TableAliases: map[string]string{
			"p": "products",
		},
		CurrentTables: []string{"p"},
	}

	result, err := InferFieldTypesWithOptions(schemas, selectStmt, nil, options)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "price", result[0].Name)
	assert.Equal(t, "decimal", result[0].Type.BaseType)
}

func TestValidateStatementSchema_ValidSelect(t *testing.T) {
	// Setup test schema
	schemas := []snapsql.DatabaseSchema{
		{
			Name: "testdb",
			DatabaseInfo: snapsql.DatabaseInfo{
				Name: "testdb",
				Type: "postgres",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name:   "users",
					Schema: "public",
					Columns: map[string]*snapsql.ColumnInfo{
						"id": {
							Name:         "id",
							DataType:     "int",
							Nullable:     false,
							IsPrimaryKey: true,
						},
						"name": {
							Name:     "name",
							DataType: "varchar",
							Nullable: true,
						},
					},
				},
			},
		},
	}

	// Setup valid SELECT statement
	selectStmt := &parsercommon.SelectStatement{
		Select: &parsercommon.SelectClause{
			Fields: []parsercommon.SelectField{
				{
					FieldKind:     parsercommon.TableField,
					TableName:     "users",
					OriginalField: "id",
					ExplicitName:  false,
				},
			},
		},
		From: &parsercommon.FromClause{
			Tables: []parsercommon.TableReferenceForFrom{
				{
					TableReference: parsercommon.TableReference{
						TableName: "users",
						Name:      "users",
					},
				},
			},
		},
	}

	// Test ValidateStatementSchema
	validationErrors, err := ValidateStatementSchema(schemas, selectStmt, nil)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(validationErrors)) // No validation errors expected
}

func TestInferFieldTypes_UnsupportedStatement(t *testing.T) {
	// Setup test schema
	schemas := []snapsql.DatabaseSchema{}

	// Setup unsupported statement (using nil which will be treated as unsupported)
	var unsupportedStmt parsercommon.StatementNode = nil

	// Test InferFieldTypes with unsupported statement
	result, err := InferFieldTypes(schemas, unsupportedStmt, nil)

	assert.Error(t, err)
	assert.Zero(t, len(result))
	assert.Contains(t, err.Error(), "unsupported statement type")
}
