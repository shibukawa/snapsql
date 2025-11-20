package pygen

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorClassesInGeneratedCode(t *testing.T) {
	// Create a simple intermediate format
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_user_by_id",
		Description:      "Get user by ID",
		StatementType:    "select",
		ResponseAffinity: "one",
		Parameters: []intermediate.Parameter{
			{
				Name:     "user_id",
				Type:     "int",
				Optional: false,
			},
		},
		Responses: []intermediate.Response{
			{
				Name:       "user_id",
				Type:       "int",
				IsNullable: false,
			},
			{
				Name:       "username",
				Type:       "string",
				IsNullable: false,
			},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT user_id, username FROM users WHERE id = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
	}

	// Generate code
	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf strings.Builder

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()
	runtimeCode := renderRuntimeCode(t)

	assert.Contains(t, code, "from .snapsql_runtime import (", "module should import runtime helpers")

	// Verify error class definitions exist in runtime
	assert.Contains(t, runtimeCode, "class SnapSQLError(Exception):", "should define SnapSQLError base class")
	assert.Contains(t, runtimeCode, "class NotFoundError(SnapSQLError):", "should define NotFoundError")
	assert.Contains(t, runtimeCode, "class ValidationError(SnapSQLError):", "should define ValidationError")
	assert.Contains(t, runtimeCode, "class DatabaseError(SnapSQLError):", "should define DatabaseError")
	assert.Contains(t, runtimeCode, "class UnsafeQueryError(SnapSQLError):", "should define UnsafeQueryError")

	// Verify error classes have enhanced constructors
	assert.Contains(t, runtimeCode, "def __init__(self, message: str, func_name: Optional[str] = None", "SnapSQLError should have enhanced constructor")
	assert.Contains(t, runtimeCode, "query: Optional[str] = None", "SnapSQLError should accept query parameter")
	assert.Contains(t, runtimeCode, "params: Optional[Dict[str, Any]] = None", "SnapSQLError should accept params parameter")

	// Verify error formatting method
	assert.Contains(t, runtimeCode, "def _format_message(self) -> str:", "should have _format_message method")
	assert.Contains(t, runtimeCode, "Function: {self.func_name}", "should format function name in error")
	assert.Contains(t, runtimeCode, "Query: {query_preview}", "should format query in error")
	assert.Contains(t, runtimeCode, "Parameters: {param_str}", "should format parameters in error")
}

func TestNotFoundErrorWithContext(t *testing.T) {
	// Create intermediate format with one affinity
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_user_by_id",
		Description:      "Get user by ID",
		StatementType:    "select",
		ResponseAffinity: "one",
		Parameters: []intermediate.Parameter{
			{
				Name:     "user_id",
				Type:     "int",
				Optional: false,
			},
		},
		Responses: []intermediate.Response{
			{
				Name:       "user_id",
				Type:       "int",
				IsNullable: false,
			},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT user_id FROM users WHERE id = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
	}

	// Generate code
	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf strings.Builder

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify NotFoundError is raised with context
	assert.Contains(t, code, "raise NotFoundError(", "should raise NotFoundError")
	assert.Contains(t, code, `func_name="get_user_by_id"`, "should include function name in NotFoundError")
	assert.Contains(t, code, "query=sql", "should include query in NotFoundError")

	// Verify the error includes message parameter
	assert.Contains(t, code, `message="Record not found"`, "should include message in NotFoundError")
}

func TestValidationErrorWithContext(t *testing.T) {
	// Create intermediate format with required parameter
	format := &intermediate.IntermediateFormat{
		FunctionName:     "create_user",
		Description:      "Create a new user",
		StatementType:    "insert",
		ResponseAffinity: "none",
		Parameters: []intermediate.Parameter{
			{
				Name:     "username",
				Type:     "string",
				Optional: false, // Required parameter
			},
			{
				Name:     "email",
				Type:     "string",
				Optional: true, // Optional parameter
			},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "INSERT INTO users (username, email) VALUES ("},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
			{Op: "EMIT_STATIC", Value: ", "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(1)},
			{Op: "EMIT_STATIC", Value: ")"},
		},
	}

	// Generate code
	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf strings.Builder

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify ValidationError is raised with context
	assert.Contains(t, code, "raise ValidationError(", "should raise ValidationError")
	assert.Contains(t, code, `param_name="username"`, "should include parameter name in ValidationError")
	assert.Contains(t, code, `func_name="create_user"`, "should include function name in ValidationError")

	// Verify validation message
	assert.Contains(t, code, "Required parameter 'username' cannot be None", "should have descriptive validation message")

	// Verify optional parameter is not validated
	assert.NotContains(t, code, `param_name="email"`, "should not validate optional parameters")
}

func TestDatabaseErrorInMockMode(t *testing.T) {
	// Create intermediate format
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_users",
		Description:      "Get all users",
		StatementType:    "select",
		ResponseAffinity: "many",
		Parameters:       []intermediate.Parameter{},
		Responses: []intermediate.Response{
			{
				Name:       "user_id",
				Type:       "int",
				IsNullable: false,
			},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT user_id FROM users"},
		},
	}

	// Generate code with mock path
	gen := New(format, WithDialect(snapsql.DialectPostgres), WithMockPath("mock_data.json"))

	var buf strings.Builder

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify DatabaseError is raised with context in mock mode
	assert.Contains(t, code, "raise DatabaseError(", "should raise DatabaseError")
	assert.Contains(t, code, `func_name="get_users"`, "should include function name in DatabaseError")
	assert.Contains(t, code, "query=sql", "should include query in DatabaseError")
}

func TestErrorMessageFormatting(t *testing.T) {
	// Create intermediate format
	format := &intermediate.IntermediateFormat{
		FunctionName:     "test_function",
		Description:      "Test function",
		StatementType:    "select",
		ResponseAffinity: "one",
		Parameters: []intermediate.Parameter{
			{
				Name:     "param1",
				Type:     "string",
				Optional: false,
			},
		},
		Responses: []intermediate.Response{
			{
				Name:       "result",
				Type:       "string",
				IsNullable: false,
			},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT result FROM test WHERE param = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
	}

	// Generate code
	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf strings.Builder

	err := gen.Generate(&buf)
	require.NoError(t, err)

	runtimeCode := renderRuntimeCode(t)

	// Verify error formatting includes all context
	t.Run("SnapSQLError base class formatting", func(t *testing.T) {
		assert.Contains(t, runtimeCode, "def _format_message(self) -> str:", "should have formatting method")
		assert.Contains(t, runtimeCode, "parts = [self.message]", "should start with message")
		assert.Contains(t, runtimeCode, "if self.func_name:", "should check for function name")
		assert.Contains(t, runtimeCode, "if self.query:", "should check for query")
		assert.Contains(t, runtimeCode, "if self.params:", "should check for parameters")
		assert.Contains(t, runtimeCode, `parts.append(f"Function: {self.func_name}")`, "should append function name")
		assert.Contains(t, runtimeCode, `parts.append(f"Query: {query_preview}")`, "should append query")
		assert.Contains(t, runtimeCode, `parts.append(f"Parameters: {param_str}")`, "should append parameters")
	})

	t.Run("Query truncation for long queries", func(t *testing.T) {
		assert.Contains(t, runtimeCode, "query_preview = self.query[:200]", "should truncate long queries")
		assert.Contains(t, runtimeCode, `+ "..." if len(self.query) > 200`, "should add ellipsis for truncated queries")
	})

	t.Run("Parameter formatting", func(t *testing.T) {
		assert.Contains(t, runtimeCode, `param_str = ", ".join(f"{k}={repr(v)}" for k, v in self.params.items())`, "should format parameters")
		assert.Contains(t, runtimeCode, "if len(param_str) > 200:", "should truncate long parameter strings")
	})
}

func TestUnsafeQueryErrorContext(t *testing.T) {
	// Create intermediate format for UPDATE without WHERE
	format := &intermediate.IntermediateFormat{
		FunctionName:     "update_all_users",
		Description:      "Update all users",
		StatementType:    "update",
		ResponseAffinity: "none",
		Parameters: []intermediate.Parameter{
			{
				Name:     "status",
				Type:     "string",
				Optional: false,
			},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "UPDATE users SET status = "},
			{Op: "ADD_PARAM", ExprIndex: intPtr(0)},
		},
		WhereClauseMeta: &intermediate.WhereClauseMeta{
			Status: "fullscan",
		},
	}

	// Generate code
	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf strings.Builder

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify UnsafeQueryError guard is emitted for unsafe mutation
	assert.Contains(t, code, "UnsafeQueryError", "should reference the unsafe guard")
	assert.Contains(t, code, "if True:", "guard should be unconditional for fullscan")

	runtimeCode := renderRuntimeCode(t)
	assert.Contains(t, runtimeCode, "class UnsafeQueryError(SnapSQLError):", "should define UnsafeQueryError")
	assert.Contains(t, runtimeCode, "mutation_kind: Optional[str] = None", "UnsafeQueryError should accept mutation_kind")
}

func TestAllErrorTypesHaveProperInheritance(t *testing.T) {
	// Create a simple format
	format := &intermediate.IntermediateFormat{
		FunctionName:     "test_func",
		Description:      "Test",
		StatementType:    "select",
		ResponseAffinity: "one",
		Parameters:       []intermediate.Parameter{},
		Responses: []intermediate.Response{
			{Name: "id", Type: "int", IsNullable: false},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT id FROM test"},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf strings.Builder

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()
	assert.Contains(t, code, "from .snapsql_runtime import (", "module should import runtime helpers")
	runtimeCode := renderRuntimeCode(t)

	// Verify all error types inherit from SnapSQLError
	errorTypes := []string{
		"NotFoundError",
		"ValidationError",
		"DatabaseError",
		"UnsafeQueryError",
	}

	for _, errorType := range errorTypes {
		t.Run(errorType, func(t *testing.T) {
			// Check class definition
			classDefPattern := "class " + errorType + "(SnapSQLError):"
			assert.Contains(t, runtimeCode, classDefPattern, errorType+" should inherit from SnapSQLError")

			// Check __init__ method exists
			initPattern := "def __init__(self"
			// Find the error class and verify it has __init__
			classStart := strings.Index(runtimeCode, classDefPattern)
			require.Greater(t, classStart, -1, "should find "+errorType+" class definition")

			// Look for __init__ after class definition (within next 500 chars)
			classSection := runtimeCode[classStart : classStart+500]
			assert.Contains(t, classSection, initPattern, errorType+" should have __init__ method")
		})
	}
}
