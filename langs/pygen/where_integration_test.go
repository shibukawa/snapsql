package pygen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestWhereClauseIntegration_UpdateWithoutWhere(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:  "update_user",
		Description:   "Update user without WHERE clause",
		StatementType: "update",
		Parameters: []intermediate.Parameter{
			{Name: "name", Type: "string", Optional: false},
		},
		Responses: []intermediate.Response{},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "UPDATE users SET name = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
		ResponseAffinity: "none",
		WhereClauseMeta: &intermediate.WhereClauseMeta{
			Status: "fullscan",
		},
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code := buf.String()

	// Verify WHERE clause enforcement is present
	if !strings.Contains(code, "enforce_non_empty_where_clause") {
		t.Error("Expected WHERE clause enforcement code")
	}

	// Verify mutation kind is set
	if !strings.Contains(code, "'update'") {
		t.Error("Expected mutation kind 'update'")
	}

	// Verify WHERE metadata is included
	if !strings.Contains(code, "'status': \"fullscan\"") {
		t.Error("Expected WHERE status 'fullscan'")
	}

	// Verify enforcement function is defined
	if !strings.Contains(code, "def enforce_non_empty_where_clause") {
		t.Error("Expected enforce_non_empty_where_clause function definition")
	}

	// Verify UnsafeQueryError is defined
	if !strings.Contains(code, "class UnsafeQueryError") {
		t.Error("Expected UnsafeQueryError class definition")
	}
}

func TestWhereClauseIntegration_DeleteWithWhere(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:  "delete_user",
		Description:   "Delete user with WHERE clause",
		StatementType: "delete",
		Parameters: []intermediate.Parameter{
			{Name: "user_id", Type: "int", Optional: false},
		},
		Responses: []intermediate.Response{},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "DELETE FROM users WHERE id = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
		ResponseAffinity: "none",
		WhereClauseMeta: &intermediate.WhereClauseMeta{
			Status:         "exists",
			ExpressionRefs: []int{0},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectMySQL))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code := buf.String()

	// Verify WHERE clause enforcement is present
	if !strings.Contains(code, "enforce_non_empty_where_clause") {
		t.Error("Expected WHERE clause enforcement code")
	}

	// Verify mutation kind is set
	if !strings.Contains(code, "'delete'") {
		t.Error("Expected mutation kind 'delete'")
	}

	// Verify WHERE metadata is included
	if !strings.Contains(code, "'status': \"exists\"") {
		t.Error("Expected WHERE status 'exists'")
	}

	// Verify expression refs are included
	if !strings.Contains(code, "'expression_refs': [0]") {
		t.Error("Expected expression_refs in WHERE metadata")
	}
}

func TestWhereClauseIntegration_ConditionalWhere(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:  "update_user_conditional",
		Description:   "Update user with conditional WHERE",
		StatementType: "update",
		Parameters: []intermediate.Parameter{
			{Name: "name", Type: "string", Optional: false},
			{Name: "user_id", Type: "int", Optional: true},
		},
		Responses: []intermediate.Response{},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "UPDATE users SET name = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
		ResponseAffinity: "none",
		WhereClauseMeta: &intermediate.WhereClauseMeta{
			Status: "conditional",
			DynamicConditions: []intermediate.WhereDynamicCondition{
				{
					ExprIndex:        1,
					NegatedWhenEmpty: true,
					HasElse:          false,
					Description:      "user_id filter",
				},
			},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectSQLite))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code := buf.String()

	// Verify WHERE clause enforcement is present
	if !strings.Contains(code, "enforce_non_empty_where_clause") {
		t.Error("Expected WHERE clause enforcement code")
	}

	// Verify WHERE metadata includes dynamic conditions
	if !strings.Contains(code, "'dynamic_conditions'") {
		t.Error("Expected dynamic_conditions in WHERE metadata")
	}

	if !strings.Contains(code, "'description': \"user_id filter\"") {
		t.Error("Expected condition description in WHERE metadata")
	}

	if !strings.Contains(code, "'negated_when_empty': True") {
		t.Error("Expected negated_when_empty flag in WHERE metadata")
	}
}

func TestWhereClauseIntegration_SelectNoEnforcement(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:  "get_users",
		Description:   "Get all users",
		StatementType: "select",
		Parameters:    []intermediate.Parameter{},
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "name", Type: "string", IsNullable: false},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT user_id, name FROM users"},
		},
		ResponseAffinity: "many",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code := buf.String()

	// Verify WHERE clause enforcement is NOT present for SELECT
	if strings.Contains(code, "enforce_non_empty_where_clause(ctx, \"get_users\"") {
		t.Error("WHERE clause enforcement should not be present for SELECT statements")
	}

	// But the function definition should still be present
	if !strings.Contains(code, "def enforce_non_empty_where_clause") {
		t.Error("Expected enforce_non_empty_where_clause function definition")
	}
}

func TestWhereClauseIntegration_RemovalCombos(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:  "update_with_combos",
		Description:   "Update with removal combos",
		StatementType: "update",
		Parameters: []intermediate.Parameter{
			{Name: "name", Type: "string", Optional: false},
		},
		Responses: []intermediate.Response{},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "UPDATE users SET name = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
		ResponseAffinity: "none",
		WhereClauseMeta: &intermediate.WhereClauseMeta{
			Status: "conditional",
			RemovalCombos: [][]intermediate.RemovalLiteral{
				{
					{ExprIndex: 0, When: false},
					{ExprIndex: 1, When: false},
				},
			},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code := buf.String()

	// Verify removal combos are included
	if !strings.Contains(code, "'removal_combos'") {
		t.Error("Expected removal_combos in WHERE metadata")
	}

	if !strings.Contains(code, "'expr_index': 0") {
		t.Error("Expected expr_index in removal combo")
	}

	if !strings.Contains(code, "'when': False") {
		t.Error("Expected when flag in removal combo")
	}
}

func TestWhereClauseHelperFunctions(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:  "update_test",
		Description:   "Test helper functions",
		StatementType: "update",
		Parameters: []intermediate.Parameter{
			{Name: "name", Type: "string", Optional: false},
		},
		Responses: []intermediate.Response{},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "UPDATE users SET name = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
		ResponseAffinity: "none",
		WhereClauseMeta: &intermediate.WhereClauseMeta{
			Status: "fullscan",
		},
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	code := buf.String()

	// Verify helper functions are defined
	helperFuncs := []string{
		"def _is_no_where_allowed",
		"def _describe_dynamic_conditions",
		"def _extract_top_level_where_clause",
		"def _is_identifier_char",
		"def _find_clause_end",
	}

	for _, funcName := range helperFuncs {
		if !strings.Contains(code, funcName) {
			t.Errorf("Expected helper function: %s", funcName)
		}
	}
}
