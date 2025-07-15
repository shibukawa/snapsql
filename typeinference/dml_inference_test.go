package typeinference2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep7"
)

// TestDMLInferenceEngine_NewDMLInferenceEngine tests creation of DML inference engine
func TestDMLInferenceEngine_NewDMLInferenceEngine(t *testing.T) {
	// Create test schema
	schemas := []snapsql.DatabaseSchema{
		{
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
				Name: "test",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":    {Name: "id", DataType: "INTEGER"},
						"name":  {Name: "name", DataType: "TEXT"},
						"email": {Name: "email", DataType: "TEXT"},
					},
				},
			},
		},
	}

	// Create base engine
	insertStmt := &parsercommon.InsertIntoStatement{}
	parseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
	}

	baseEngine := NewTypeInferenceEngine2(schemas, insertStmt, parseResult)

	// Create DML engine
	dmlEngine := NewDMLInferenceEngine(baseEngine)

	// Verify creation
	assert.NotEqual(t, nil, dmlEngine)
	assert.Equal(t, baseEngine, dmlEngine.baseEngine)
	assert.NotEqual(t, nil, dmlEngine.schemaResolver)
	assert.NotEqual(t, nil, dmlEngine.subqueryResolver)
}

// TestDMLInferenceEngine_InferInsertStatement tests INSERT statement inference
func TestDMLInferenceEngine_InferInsertStatement(t *testing.T) {
	// Create test schema
	schemas := []snapsql.DatabaseSchema{
		{
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
				Name: "test",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "INTEGER"},
						"name": {Name: "name", DataType: "TEXT"},
					},
				},
			},
		},
	}

	// Create INSERT statement without RETURNING clause
	insertStmt := &parsercommon.InsertIntoStatement{
		Into: &parsercommon.InsertIntoClause{},
	}

	parseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
	}

	baseEngine := NewTypeInferenceEngine2(schemas, insertStmt, parseResult)
	dmlEngine := NewDMLInferenceEngine(baseEngine)

	// Test inference
	fields, err := dmlEngine.inferInsertStatement(insertStmt)

	// Should return affected_rows field for INSERT without RETURNING
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "affected_rows", fields[0].Name)
	assert.Equal(t, "int", fields[0].Type.BaseType)
	assert.Equal(t, false, fields[0].Type.IsNullable)
	assert.Equal(t, true, fields[0].IsGenerated)
	assert.Equal(t, "function", fields[0].Source.Type)
}

// TestDMLInferenceEngine_InferUpdateStatement tests UPDATE statement inference
func TestDMLInferenceEngine_InferUpdateStatement(t *testing.T) {
	// Create test schema
	schemas := []snapsql.DatabaseSchema{
		{
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
				Name: "test",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "INTEGER"},
						"name": {Name: "name", DataType: "TEXT"},
					},
				},
			},
		},
	}

	// Create UPDATE statement without RETURNING clause
	updateStmt := &parsercommon.UpdateStatement{
		Update: &parsercommon.UpdateClause{},
	}

	parseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
	}

	baseEngine := NewTypeInferenceEngine2(schemas, updateStmt, parseResult)
	dmlEngine := NewDMLInferenceEngine(baseEngine)

	// Test inference
	fields, err := dmlEngine.inferUpdateStatement(updateStmt)

	// Should return affected_rows field for UPDATE without RETURNING
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "affected_rows", fields[0].Name)
	assert.Equal(t, "int", fields[0].Type.BaseType)
	assert.Equal(t, false, fields[0].Type.IsNullable)
	assert.Equal(t, true, fields[0].IsGenerated)
	assert.Equal(t, "function", fields[0].Source.Type)
}

// TestDMLInferenceEngine_InferDeleteStatement tests DELETE statement inference
func TestDMLInferenceEngine_InferDeleteStatement(t *testing.T) {
	// Create test schema
	schemas := []snapsql.DatabaseSchema{
		{
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
				Name: "test",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "INTEGER"},
						"name": {Name: "name", DataType: "TEXT"},
					},
				},
			},
		},
	}

	// Create DELETE statement without RETURNING clause
	deleteStmt := &parsercommon.DeleteFromStatement{
		From: &parsercommon.DeleteFromClause{},
	}

	parseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
	}

	baseEngine := NewTypeInferenceEngine2(schemas, deleteStmt, parseResult)
	dmlEngine := NewDMLInferenceEngine(baseEngine)

	// Test inference
	fields, err := dmlEngine.inferDeleteStatement(deleteStmt)

	// Should return affected_rows field for DELETE without RETURNING
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "affected_rows", fields[0].Name)
	assert.Equal(t, "int", fields[0].Type.BaseType)
	assert.Equal(t, false, fields[0].Type.IsNullable)
	assert.Equal(t, true, fields[0].IsGenerated)
	assert.Equal(t, "function", fields[0].Source.Type)
}

// TestDMLInferenceEngine_InferDMLStatementType tests unified DML inference
func TestDMLInferenceEngine_InferDMLStatementType(t *testing.T) {
	// Create test schema
	schemas := []snapsql.DatabaseSchema{
		{
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
				Name: "test",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "INTEGER"},
						"name": {Name: "name", DataType: "TEXT"},
					},
				},
			},
		},
	}

	parseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
	}

	// Test INSERT statement
	insertStmt := &parsercommon.InsertIntoStatement{
		Into: &parsercommon.InsertIntoClause{},
	}

	baseEngine := NewTypeInferenceEngine2(schemas, insertStmt, parseResult)
	dmlEngine := NewDMLInferenceEngine(baseEngine)

	fields, err := dmlEngine.InferDMLStatementType(insertStmt)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "affected_rows", fields[0].Name)

	// Test unsupported statement type
	selectStmt := &parsercommon.SelectStatement{}
	fields, err = dmlEngine.InferDMLStatementType(selectStmt)
	assert.Error(t, err)
	assert.Equal(t, 0, len(fields))
	assert.Contains(t, err.Error(), "unsupported DML statement type")
}

// TestDMLInferenceEngine_InferReturningClause tests RETURNING clause inference
func TestDMLInferenceEngine_InferReturningClause(t *testing.T) {
	// Create test schema
	schemas := []snapsql.DatabaseSchema{
		{
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
				Name: "test",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "INTEGER"},
						"name": {Name: "name", DataType: "TEXT"},
					},
				},
			},
		},
	}

	// Create RETURNING clause with simple fields
	returningClause := &parsercommon.ReturningClause{
		Fields: []parsercommon.SelectField{
			{OriginalField: "id"},
			{OriginalField: "name"},
		},
	}

	parseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
	}

	insertStmt := &parsercommon.InsertIntoStatement{
		Into: &parsercommon.InsertIntoClause{},
	}

	baseEngine := NewTypeInferenceEngine2(schemas, insertStmt, parseResult)
	dmlEngine := NewDMLInferenceEngine(baseEngine)

	// Test RETURNING clause inference
	fields, err := dmlEngine.inferReturningClause(returningClause, "users")

	// Should return fields for each RETURNING field
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fields))
	// Note: Since we're reusing the SELECT field inference, the exact field names and types
	// will depend on the implementation of inferFieldType
}

// TestTypeInferenceEngine2_InferTypes tests unified inference method
func TestTypeInferenceEngine2_InferTypes(t *testing.T) {
	// Create test schema
	schemas := []snapsql.DatabaseSchema{
		{
			DatabaseInfo: snapsql.DatabaseInfo{
				Type: "postgres",
				Name: "test",
			},
			Tables: []*snapsql.TableInfo{
				{
					Name: "users",
					Columns: map[string]*snapsql.ColumnInfo{
						"id":   {Name: "id", DataType: "INTEGER"},
						"name": {Name: "name", DataType: "TEXT"},
					},
				},
			},
		},
	}

	parseResult := &parserstep7.ParseResult{
		DependencyGraph: parserstep7.NewDependencyGraph(),
	}

	// Test INSERT statement via unified method
	insertStmt := &parsercommon.InsertIntoStatement{
		Into: &parsercommon.InsertIntoClause{},
	}

	engine := NewTypeInferenceEngine2(schemas, insertStmt, parseResult)

	fields, err := engine.InferTypes()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "affected_rows", fields[0].Name)

	// Test UPDATE statement via unified method
	updateStmt := &parsercommon.UpdateStatement{
		Update: &parsercommon.UpdateClause{},
	}

	engine = NewTypeInferenceEngine2(schemas, updateStmt, parseResult)

	fields, err = engine.InferTypes()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "affected_rows", fields[0].Name)

	// Test DELETE statement via unified method
	deleteStmt := &parsercommon.DeleteFromStatement{
		From: &parsercommon.DeleteFromClause{},
	}

	engine = NewTypeInferenceEngine2(schemas, deleteStmt, parseResult)

	fields, err = engine.InferTypes()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "affected_rows", fields[0].Name)
}
