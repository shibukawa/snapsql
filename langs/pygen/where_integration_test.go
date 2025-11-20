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

	runtimeCode := renderRuntimeCode(t)
	// Verify WHERE clause guard is present for full scan mutations
	if !strings.Contains(code, "raise UnsafeQueryError") {
		t.Error("Expected UnsafeQueryError guard for unsafe mutation")
	}

	if !strings.Contains(code, "if True:") {
		t.Error("Expected unconditional guard for fullscan status")
	}

	if !strings.Contains(runtimeCode, "class UnsafeQueryError") {
		t.Error("Expected UnsafeQueryError class definition in runtime")
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
	if strings.Contains(code, "raise UnsafeQueryError") {
		t.Error("UnsafeQueryError guard should not fire when WHERE clause exists")
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

	if !strings.Contains(code, "raise UnsafeQueryError") {
		t.Error("Expected UnsafeQueryError guard for conditional WHERE removal")
	}

	if !strings.Contains(code, "not (user_id filter)") {
		t.Error("Expected condition-based guard in generated code")
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

	// Verify WHERE clause guard is not emitted for SELECT
	if strings.Contains(code, "raise UnsafeQueryError") {
		t.Error("WHERE clause guard should not be present for SELECT statements")
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

	if !strings.Contains(code, "raise UnsafeQueryError") {
		t.Error("Expected guard for removal combos")
	}

	if !strings.Contains(code, "and") {
		t.Error("Expected combined condition in guard")
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
	if !strings.Contains(code, "from .snapsql_runtime import") {
		t.Error("Expected runtime import in generated module")
	}

	runtimeCode := renderRuntimeCode(t)

	if !strings.Contains(runtimeCode, "class UnsafeQueryError") {
		t.Error("Expected UnsafeQueryError definition in runtime")
	}
}
