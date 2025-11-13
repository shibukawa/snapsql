package pygen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
)

// TestResponseAffinity_None_ReturnsInt tests that none affinity returns int (affected rows)
func TestResponseAffinity_None_ReturnsInt(t *testing.T) {
	idx := 0
	format := &intermediate.IntermediateFormat{
		FunctionName:     "update_user",
		Description:      "Update user email",
		ResponseAffinity: "none",
		Parameters: []intermediate.Parameter{
			{Name: "user_id", Type: "int", Optional: false},
			{Name: "email", Type: "string", Optional: false},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "UPDATE users SET email = ? WHERE id = ?"},
			{Op: "ADD_PARAM", ExprIndex: &idx},
		},
		CELExpressions: []codegenerator.CELExpression{
			{Expression: "email", EnvironmentIndex: 0},
			{Expression: "user_id", EnvironmentIndex: 0},
		},
		CELEnvironments: []codegenerator.CELEnvironment{
			{Index: 0, Container: "", AdditionalVariables: []codegenerator.CELVariableInfo{}},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	code := buf.String()

	// Verify return type is int
	if !strings.Contains(code, ") -> int:") {
		t.Error("expected return type int for none affinity")
	}

	// Verify returns affected rows
	if !strings.Contains(code, "affected_rows") || !strings.Contains(code, "return affected_rows") {
		t.Error("expected to return affected_rows for none affinity")
	}
}

// TestResponseAffinity_One_RaisesNotFoundError tests that one affinity raises NotFoundError
func TestResponseAffinity_One_RaisesNotFoundError(t *testing.T) {
	idx := 0
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_user_by_id",
		Description:      "Get user by ID",
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

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	code := buf.String()

	// Verify return type is the dataclass
	if !strings.Contains(code, ") -> GetUserByIdResult:") {
		t.Error("expected return type GetUserByIdResult for one affinity")
	}

	// Verify raises NotFoundError when row is None
	if !strings.Contains(code, "if row is None:") {
		t.Error("expected None check for one affinity")
	}

	if !strings.Contains(code, "raise NotFoundError") {
		t.Error("expected NotFoundError for one affinity when row not found")
	}

	// Verify returns single dataclass instance
	if !strings.Contains(code, "return GetUserByIdResult(**") {
		t.Error("expected to return dataclass instance for one affinity")
	}
}

// TestResponseAffinity_Many_AsyncGenerator tests that many affinity returns AsyncGenerator
func TestResponseAffinity_Many_AsyncGenerator(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "list_all_users",
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

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	code := buf.String()

	// Verify return type is AsyncGenerator
	if !strings.Contains(code, "-> AsyncGenerator[ListAllUsersResult, None]:") {
		t.Error("expected AsyncGenerator return type for many affinity")
	}

	// Verify uses yield
	if !strings.Contains(code, "yield ListAllUsersResult(**") {
		t.Error("expected yield for many affinity")
	}

	// Verify the main function uses yield, not return
	// We check that the generated function body contains yield
	funcStart := strings.Index(code, "async def list_all_users(")
	if funcStart == -1 {
		t.Fatal("could not find function definition")
	}

	// Find the next function or end of file
	nextFunc := strings.Index(code[funcStart+1:], "\nasync def ")

	var funcBody string
	if nextFunc == -1 {
		funcBody = code[funcStart:]
	} else {
		funcBody = code[funcStart : funcStart+nextFunc+1]
	}

	// The function body should contain yield
	if !strings.Contains(funcBody, "yield ") {
		t.Error("expected yield in many affinity function body")
	}
}

// TestResponseAffinity_PostgreSQL_Differences tests PostgreSQL-specific differences
func TestResponseAffinity_PostgreSQL_Differences(t *testing.T) {
	tests := []struct {
		name     string
		affinity string
		expected []string
	}{
		{
			name:     "none affinity uses conn.execute",
			affinity: "none",
			expected: []string{"await conn.execute(sql, *args)"},
		},
		{
			name:     "one affinity uses conn.fetchrow",
			affinity: "one",
			expected: []string{"await conn.fetchrow(sql, *args)", "dict(row)"},
		},
		{
			name:     "many affinity uses conn.fetch",
			affinity: "many",
			expected: []string{"await conn.fetch(sql, *args)", "for row in rows:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := &intermediate.IntermediateFormat{
				FunctionName:     "test_func",
				Description:      "Test function",
				ResponseAffinity: tt.affinity,
				Parameters:       []intermediate.Parameter{},
				Instructions: []intermediate.Instruction{
					{Op: "EMIT_STATIC", Value: "SELECT 1"},
				},
				CELExpressions:  []codegenerator.CELExpression{},
				CELEnvironments: []codegenerator.CELEnvironment{},
			}

			if tt.affinity != "none" {
				format.Responses = []intermediate.Response{
					{Name: "result", Type: "int", IsNullable: false},
				}
			}

			gen := New(format, WithDialect(snapsql.DialectPostgres))

			var buf bytes.Buffer

			err := gen.Generate(&buf)
			if err != nil {
				t.Fatalf("failed to generate code: %v", err)
			}

			code := buf.String()

			for _, exp := range tt.expected {
				if !strings.Contains(code, exp) {
					t.Errorf("expected %q in generated code", exp)
				}
			}
		})
	}
}

// TestResponseAffinity_MySQL_Differences tests MySQL-specific differences
func TestResponseAffinity_MySQL_Differences(t *testing.T) {
	tests := []struct {
		name     string
		affinity string
		expected []string
	}{
		{
			name:     "none affinity uses cursor.execute and rowcount",
			affinity: "none",
			expected: []string{"await cursor.execute(sql, args)", "return cursor.rowcount"},
		},
		{
			name:     "one affinity uses cursor.execute and fetchone",
			affinity: "one",
			expected: []string{"await cursor.execute(sql, args)", "row = await cursor.fetchone()"},
		},
		{
			name:     "many affinity uses async for cursor",
			affinity: "many",
			expected: []string{"await cursor.execute(sql, args)", "async for row in cursor:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := &intermediate.IntermediateFormat{
				FunctionName:     "test_func",
				Description:      "Test function",
				ResponseAffinity: tt.affinity,
				Parameters:       []intermediate.Parameter{},
				Instructions: []intermediate.Instruction{
					{Op: "EMIT_STATIC", Value: "SELECT 1"},
				},
				CELExpressions:  []codegenerator.CELExpression{},
				CELEnvironments: []codegenerator.CELEnvironment{},
			}

			if tt.affinity != "none" {
				format.Responses = []intermediate.Response{
					{Name: "result", Type: "int", IsNullable: false},
				}
			}

			gen := New(format, WithDialect(snapsql.DialectMySQL))

			var buf bytes.Buffer

			err := gen.Generate(&buf)
			if err != nil {
				t.Fatalf("failed to generate code: %v", err)
			}

			code := buf.String()

			for _, exp := range tt.expected {
				if !strings.Contains(code, exp) {
					t.Errorf("expected %q in generated code", exp)
				}
			}
		})
	}
}
