package pygen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
)

// TestIntegration_MySQL_NoneAffinity tests complete code generation for MySQL with none affinity
func TestIntegration_MySQL_NoneAffinity(t *testing.T) {
	idx := 0
	format := &intermediate.IntermediateFormat{
		FunctionName:     "delete_user",
		Description:      "Delete a user by ID",
		ResponseAffinity: "none",
		Parameters: []intermediate.Parameter{
			{Name: "user_id", Type: "int", Optional: false},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "DELETE FROM users WHERE id = ?"},
			{Op: "ADD_PARAM", ExprIndex: &idx},
		},
		CELExpressions: []codegenerator.CELExpression{
			{Expression: "user_id", EnvironmentIndex: 0},
		},
		CELEnvironments: []codegenerator.CELEnvironment{
			{Index: 0, Container: "", AdditionalVariables: []codegenerator.CELVariableInfo{}},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectMySQL))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	code := buf.String()

	// Verify MySQL-specific imports
	if !strings.Contains(code, "import aiomysql") {
		t.Error("expected aiomysql import")
	}

	// Verify cursor execution
	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Error("expected cursor.execute call")
	}

	// Verify return rowcount
	if !strings.Contains(code, "return cursor.rowcount") {
		t.Error("expected return cursor.rowcount")
	}
}

// TestIntegration_SQLite_NoneAffinity tests complete code generation for SQLite with none affinity
func TestIntegration_SQLite_NoneAffinity(t *testing.T) {
	idx := 0
	format := &intermediate.IntermediateFormat{
		FunctionName:     "delete_user",
		Description:      "Delete a user by ID",
		ResponseAffinity: "none",
		Parameters: []intermediate.Parameter{
			{Name: "user_id", Type: "int", Optional: false},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "DELETE FROM users WHERE id = ?"},
			{Op: "ADD_PARAM", ExprIndex: &idx},
		},
		CELExpressions: []codegenerator.CELExpression{
			{Expression: "user_id", EnvironmentIndex: 0},
		},
		CELEnvironments: []codegenerator.CELEnvironment{
			{Index: 0, Container: "", AdditionalVariables: []codegenerator.CELVariableInfo{}},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectSQLite))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	code := buf.String()

	// Verify SQLite-specific imports
	if !strings.Contains(code, "import aiosqlite") {
		t.Error("expected aiosqlite import")
	}

	// Verify SQL with ? placeholder (SQLite uses ?)
	if !strings.Contains(code, "DELETE FROM users WHERE id = ?") {
		t.Error("expected SQL with ? placeholder for SQLite")
	}

	// Verify cursor execution
	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Error("expected cursor.execute call")
	}

	// Verify return rowcount
	if !strings.Contains(code, "return cursor.rowcount") {
		t.Error("expected return cursor.rowcount")
	}
}

// TestIntegration_SQLite_OneAffinity tests complete code generation for SQLite with one affinity
func TestIntegration_SQLite_OneAffinity(t *testing.T) {
	idx := 0
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_user",
		Description:      "Get a user by ID",
		ResponseAffinity: "one",
		Parameters: []intermediate.Parameter{
			{Name: "user_id", Type: "int", Optional: false},
		},
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "username", Type: "string", IsNullable: false},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT user_id, username FROM users WHERE id = ?"},
			{Op: "ADD_PARAM", ExprIndex: &idx},
		},
		CELExpressions: []codegenerator.CELExpression{
			{Expression: "user_id", EnvironmentIndex: 0},
		},
		CELEnvironments: []codegenerator.CELEnvironment{
			{Index: 0, Container: "", AdditionalVariables: []codegenerator.CELVariableInfo{}},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectSQLite))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	code := buf.String()

	// Verify row_factory comment
	if !strings.Contains(code, "# aiosqlite with row_factory returns dict-like object") {
		t.Error("expected row_factory comment for SQLite")
	}

	// Verify cursor execution
	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Error("expected cursor.execute call")
	}

	// Verify fetchone
	if !strings.Contains(code, "row = await cursor.fetchone()") {
		t.Error("expected cursor.fetchone call")
	}

	// Verify NotFoundError
	if !strings.Contains(code, "raise NotFoundError") {
		t.Error("expected NotFoundError for missing row")
	}
}

// TestIntegration_SQLite_ManyAffinity tests complete code generation for SQLite with many affinity
func TestIntegration_SQLite_ManyAffinity(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "list_users",
		Description:      "List all users",
		ResponseAffinity: "many",
		Parameters:       []intermediate.Parameter{},
		Responses: []intermediate.Response{
			{Name: "user_id", Type: "int", IsNullable: false},
			{Name: "username", Type: "string", IsNullable: false},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT user_id, username FROM users"},
		},
		CELExpressions:  []codegenerator.CELExpression{},
		CELEnvironments: []codegenerator.CELEnvironment{},
	}

	gen := New(format, WithDialect(snapsql.DialectSQLite))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	code := buf.String()

	// Verify async generator return type
	if !strings.Contains(code, "-> AsyncGenerator[ListUsersResult, None]:") {
		t.Error("expected AsyncGenerator return type for many affinity")
	}

	// Verify cursor execution
	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Error("expected cursor.execute call")
	}

	// Verify async for loop
	if !strings.Contains(code, "async for row in cursor:") {
		t.Error("expected async for loop over cursor")
	}

	// Verify yield
	if !strings.Contains(code, "yield ListUsersResult(**row)") {
		t.Error("expected yield with dataclass instantiation")
	}
}
