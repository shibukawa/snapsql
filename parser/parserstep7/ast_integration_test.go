package parserstep7

import (
	"testing"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Test extractCTEDependencies extracts CTE information correctly
func TestExtractCTEDependencies_SingleCTE(t *testing.T) {
	// Create a simple CTE: WITH cte AS (SELECT id, name FROM users) SELECT * FROM cte
	cteSelect := &cmn.SelectStatement{
		Select: &cmn.SelectClause{
			Fields: []cmn.SelectField{
				{
					FieldKind:     cmn.SingleField,
					OriginalField: "id",
					FieldName:     "id",
				},
				{
					FieldKind:     cmn.SingleField,
					OriginalField: "name",
					FieldName:     "name",
				},
			},
		},
		From: &cmn.FromClause{
			Tables: []cmn.TableReferenceForFrom{
				{
					TableReference: cmn.TableReference{
						TableName: "users",
						Name:      "users",
					},
				},
			},
		},
	}

	withClause := &cmn.WithClause{
		CTEs: []cmn.CTEDefinition{
			{
				Name:   "cte",
				Select: cteSelect,
			},
		},
	}

	mainStmt := cmn.NewSelectStatement([]tokenizer.Token{}, withClause, []cmn.ClauseNode{})

	// Create parser and integrator
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Extract CTE dependencies
	err := integrator.extractCTEDependencies(mainStmt.CTE(), mainStmt)
	if err != nil {
		t.Fatalf("extractCTEDependencies failed: %v", err)
	}

	// Verify DerivedTables
	derivedTables := parser.GetDerivedTables()
	if len(derivedTables) != 1 {
		t.Fatalf("expected 1 derived table, got %d", len(derivedTables))
	}

	dt := derivedTables[0]
	if dt.Name != "cte" {
		t.Errorf("expected CTE name 'cte', got '%s'", dt.Name)
	}

	if dt.SourceType != "cte" {
		t.Errorf("expected SourceType 'cte', got '%s'", dt.SourceType)
	}

	if len(dt.SelectFields) != 2 {
		t.Errorf("expected 2 SelectFields, got %d", len(dt.SelectFields))
	}

	if len(dt.ReferencedTables) != 1 {
		t.Errorf("expected 1 referenced table, got %d", len(dt.ReferencedTables))
	}

	if dt.ReferencedTables[0] != "users" {
		t.Errorf("expected referenced table 'users', got '%s'", dt.ReferencedTables[0])
	}
}

// Test multiple CTEs
func TestExtractCTEDependencies_MultipleCTEs(t *testing.T) {
	// WITH cte1 AS (SELECT id FROM users), cte2 AS (SELECT user_id FROM orders) SELECT ...
	cte1Select := &cmn.SelectStatement{
		Select: &cmn.SelectClause{
			Fields: []cmn.SelectField{
				{
					FieldKind:     cmn.SingleField,
					OriginalField: "id",
					FieldName:     "id",
				},
			},
		},
		From: &cmn.FromClause{
			Tables: []cmn.TableReferenceForFrom{
				{
					TableReference: cmn.TableReference{
						TableName: "users",
						Name:      "users",
					},
				},
			},
		},
	}

	cte2Select := &cmn.SelectStatement{
		Select: &cmn.SelectClause{
			Fields: []cmn.SelectField{
				{
					FieldKind:     cmn.SingleField,
					OriginalField: "user_id",
					FieldName:     "user_id",
				},
			},
		},
		From: &cmn.FromClause{
			Tables: []cmn.TableReferenceForFrom{
				{
					TableReference: cmn.TableReference{
						TableName: "orders",
						Name:      "orders",
					},
				},
			},
		},
	}

	withClause := &cmn.WithClause{
		CTEs: []cmn.CTEDefinition{
			{
				Name:   "cte1",
				Select: cte1Select,
			},
			{
				Name:   "cte2",
				Select: cte2Select,
			},
		},
	}

	mainStmt := cmn.NewSelectStatement([]tokenizer.Token{}, withClause, []cmn.ClauseNode{})

	// Create parser and integrator
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Extract CTE dependencies
	err := integrator.extractCTEDependencies(mainStmt.CTE(), mainStmt)
	if err != nil {
		t.Fatalf("extractCTEDependencies failed: %v", err)
	}

	// Verify DerivedTables
	derivedTables := parser.GetDerivedTables()
	if len(derivedTables) != 2 {
		t.Fatalf("expected 2 derived tables, got %d", len(derivedTables))
	}

	// Check first CTE
	dt1 := derivedTables[0]
	if dt1.Name != "cte1" {
		t.Errorf("expected first CTE name 'cte1', got '%s'", dt1.Name)
	}

	if dt1.SourceType != "cte" {
		t.Errorf("expected first SourceType 'cte', got '%s'", dt1.SourceType)
	}

	if len(dt1.ReferencedTables) != 1 || dt1.ReferencedTables[0] != "users" {
		t.Errorf("expected first CTE to reference 'users'")
	}

	// Check second CTE
	dt2 := derivedTables[1]
	if dt2.Name != "cte2" {
		t.Errorf("expected second CTE name 'cte2', got '%s'", dt2.Name)
	}

	if dt2.SourceType != "cte" {
		t.Errorf("expected second SourceType 'cte', got '%s'", dt2.SourceType)
	}

	if len(dt2.ReferencedTables) != 1 || dt2.ReferencedTables[0] != "orders" {
		t.Errorf("expected second CTE to reference 'orders'")
	}
}

// Test CTE with JOIN
func TestExtractCTEDependencies_CTEWithJoin(t *testing.T) {
	// WITH cte AS (SELECT u.id, o.total FROM users u JOIN orders o ON u.id = o.user_id) SELECT ...
	cteSelect := &cmn.SelectStatement{
		Select: &cmn.SelectClause{
			Fields: []cmn.SelectField{
				{
					FieldKind:     cmn.SingleField,
					OriginalField: "u.id",
					TableName:     "u",
					FieldName:     "id",
				},
				{
					FieldKind:     cmn.SingleField,
					OriginalField: "o.total",
					TableName:     "o",
					FieldName:     "total",
				},
			},
		},
		From: &cmn.FromClause{
			Tables: []cmn.TableReferenceForFrom{
				{
					TableReference: cmn.TableReference{
						TableName: "users",
						Name:      "u",
					},
					JoinType: cmn.JoinNone,
				},
				{
					TableReference: cmn.TableReference{
						TableName: "orders",
						Name:      "o",
					},
					JoinType: cmn.JoinInner,
				},
			},
		},
	}

	withClause := &cmn.WithClause{
		CTEs: []cmn.CTEDefinition{
			{
				Name:   "cte",
				Select: cteSelect,
			},
		},
	}

	mainStmt := cmn.NewSelectStatement([]tokenizer.Token{}, withClause, []cmn.ClauseNode{})

	// Create parser and integrator
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Extract CTE dependencies
	err := integrator.extractCTEDependencies(mainStmt.CTE(), mainStmt)
	if err != nil {
		t.Fatalf("extractCTEDependencies failed: %v", err)
	}

	// Verify DerivedTables
	derivedTables := parser.GetDerivedTables()
	if len(derivedTables) != 1 {
		t.Fatalf("expected 1 derived table, got %d", len(derivedTables))
	}

	dt := derivedTables[0]
	if dt.Name != "cte" {
		t.Errorf("expected CTE name 'cte', got '%s'", dt.Name)
	}

	if len(dt.SelectFields) != 2 {
		t.Errorf("expected 2 SelectFields, got %d", len(dt.SelectFields))
	}

	if len(dt.ReferencedTables) != 2 {
		t.Errorf("expected 2 referenced tables, got %d", len(dt.ReferencedTables))
	}
	// Referenced tables should include both 'u' and 'o' (aliases)
	foundU := false
	foundO := false

	for _, tableName := range dt.ReferencedTables {
		if tableName == "u" {
			foundU = true
		}

		if tableName == "o" {
			foundO = true
		}
	}

	if !foundU || !foundO {
		t.Errorf("expected referenced tables to include 'u' and 'o', got %v", dt.ReferencedTables)
	}
}

// Test FROM clause subquery extraction
func TestExtractFromClauseSubqueries_SimpleSubquery(t *testing.T) {
	// SELECT * FROM (SELECT id FROM users) AS user_summary
	mainStmt := &cmn.SelectStatement{
		From: &cmn.FromClause{
			Tables: []cmn.TableReferenceForFrom{
				{
					TableReference: cmn.TableReference{
						Name: "user_summary",
					},
					// Note: RawTokens would contain the subquery content in real parsing
					RawTokens: []tokenizer.Token{},
				},
			},
		},
	}

	// Create parser and integrator
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Extract FROM clause subqueries
	err := integrator.extractFromClauseSubqueries(mainStmt)
	if err != nil {
		t.Fatalf("extractFromClauseSubqueries failed: %v", err)
	}

	// Verify DerivedTables
	derivedTables := parser.GetDerivedTables()
	if len(derivedTables) != 1 {
		t.Fatalf("expected 1 derived table, got %d", len(derivedTables))
	}

	dt := derivedTables[0]
	if dt.Name != "user_summary" {
		t.Errorf("expected subquery alias 'user_summary', got '%s'", dt.Name)
	}

	if dt.SourceType != "subquery" {
		t.Errorf("expected SourceType 'subquery', got '%s'", dt.SourceType)
	}
}

// Test mixed CTEs and subqueries
func TestExtractSubqueries_MixedCTEAndSubquery(t *testing.T) {
	// WITH cte AS (SELECT id FROM users) SELECT * FROM cte JOIN (SELECT id FROM orders) AS o ON ...
	cteSelect := &cmn.SelectStatement{
		Select: &cmn.SelectClause{
			Fields: []cmn.SelectField{
				{
					FieldKind:     cmn.SingleField,
					OriginalField: "id",
					FieldName:     "id",
				},
			},
		},
		From: &cmn.FromClause{
			Tables: []cmn.TableReferenceForFrom{
				{
					TableReference: cmn.TableReference{
						TableName: "users",
						Name:      "users",
					},
				},
			},
		},
	}

	withClause := &cmn.WithClause{
		CTEs: []cmn.CTEDefinition{
			{
				Name:   "cte",
				Select: cteSelect,
			},
		},
	}

	mainStmt := cmn.NewSelectStatement([]tokenizer.Token{}, withClause, []cmn.ClauseNode{})
	mainStmt.From = &cmn.FromClause{
		Tables: []cmn.TableReferenceForFrom{
			{
				TableReference: cmn.TableReference{
					Name:      "cte",
					TableName: "cte",
				},
			},
			{
				TableReference: cmn.TableReference{
					Name: "o",
				},
				// Note: RawTokens would contain the subquery content in real parsing
				RawTokens: []tokenizer.Token{},
				JoinType:  cmn.JoinInner,
			},
		},
	}

	// Create parser and integrator
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Extract both CTEs and subqueries
	err := integrator.ExtractSubqueries(mainStmt)
	if err != nil {
		t.Fatalf("ExtractSubqueries failed: %v", err)
	}

	// Verify DerivedTables
	derivedTables := parser.GetDerivedTables()
	if len(derivedTables) != 2 {
		t.Fatalf("expected 2 derived tables (1 CTE + 1 subquery), got %d", len(derivedTables))
	}

	// First should be CTE
	dt1 := derivedTables[0]
	if dt1.SourceType != "cte" {
		t.Errorf("expected first derived table to be CTE, got '%s'", dt1.SourceType)
	}

	// Second should be subquery
	dt2 := derivedTables[1]
	if dt2.SourceType != "subquery" {
		t.Errorf("expected second derived table to be subquery, got '%s'", dt2.SourceType)
	}
}

// Test that SelectFields contain detailed field information for type inference
func TestDerivedTableInfo_SelectFieldsDetail(t *testing.T) {
	// Create a CTE with various field types: WITH cte AS (SELECT id, name, COUNT(*) as cnt FROM users) SELECT * FROM cte
	cteSelect := &cmn.SelectStatement{
		Select: &cmn.SelectClause{
			Fields: []cmn.SelectField{
				{
					FieldKind:     cmn.TableField,
					OriginalField: "users.id",
					TableName:     "users",
					FieldName:     "id",
				},
				{
					FieldKind:     cmn.SingleField,
					OriginalField: "name",
					FieldName:     "name",
				},
				{
					FieldKind:     cmn.FunctionField,
					OriginalField: "COUNT(*)",
					FieldName:     "cnt",
				},
			},
		},
		From: &cmn.FromClause{
			Tables: []cmn.TableReferenceForFrom{
				{
					TableReference: cmn.TableReference{
						TableName: "users",
						Name:      "users",
					},
				},
			},
		},
	}

	withClause := &cmn.WithClause{
		CTEs: []cmn.CTEDefinition{
			{
				Name:   "cte",
				Select: cteSelect,
			},
		},
	}

	mainStmt := cmn.NewSelectStatement([]tokenizer.Token{}, withClause, []cmn.ClauseNode{})

	// Create parser and integrator
	parser := NewSubqueryParser()
	integrator := NewASTIntegrator(parser)

	// Extract CTE dependencies
	err := integrator.extractCTEDependencies(mainStmt.CTE(), mainStmt)
	if err != nil {
		t.Fatalf("extractCTEDependencies failed: %v", err)
	}

	// Verify DerivedTables
	derivedTables := parser.GetDerivedTables()
	if len(derivedTables) != 1 {
		t.Fatalf("expected 1 derived table, got %d", len(derivedTables))
	}

	dt := derivedTables[0]

	// Verify SelectFields detail
	if len(dt.SelectFields) != 3 {
		t.Fatalf("expected 3 SelectFields, got %d", len(dt.SelectFields))
	}

	// Verify first field (users.id - TableField)
	field1 := dt.SelectFields[0]
	if field1.FieldKind != cmn.TableField {
		t.Errorf("expected field 1 to be TableField, got %v", field1.FieldKind)
	}
	if field1.FieldName != "id" {
		t.Errorf("expected field 1 name to be 'id', got '%s'", field1.FieldName)
	}
	if field1.TableName != "users" {
		t.Errorf("expected field 1 table name to be 'users', got '%s'", field1.TableName)
	}

	// Verify second field (name - SingleField)
	field2 := dt.SelectFields[1]
	if field2.FieldKind != cmn.SingleField {
		t.Errorf("expected field 2 to be SingleField, got %v", field2.FieldKind)
	}
	if field2.FieldName != "name" {
		t.Errorf("expected field 2 name to be 'name', got '%s'", field2.FieldName)
	}

	// Verify third field (COUNT(*) - FunctionField)
	field3 := dt.SelectFields[2]
	if field3.FieldKind != cmn.FunctionField {
		t.Errorf("expected field 3 to be FunctionField, got %v", field3.FieldKind)
	}
	if field3.FieldName != "cnt" {
		t.Errorf("expected field 3 name to be 'cnt', got '%s'", field3.FieldName)
	}
	if field3.OriginalField != "COUNT(*)" {
		t.Errorf("expected field 3 original field to be 'COUNT(*)', got '%s'", field3.OriginalField)
	}
}
