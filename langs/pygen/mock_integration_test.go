package pygen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockIntegration_GeneratedCode(t *testing.T) {
	// Create intermediate format with mock path
	format := &intermediate.IntermediateFormat{
		FunctionName:     "GetUserByID",
		Description:      "Get user by ID with mock support",
		StatementType:    "SELECT",
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
			{
				Name:       "email",
				Type:       "string",
				IsNullable: true,
			},
		},
		Instructions: []intermediate.Instruction{
			{
				Op:    "EMIT_STATIC",
				Value: "SELECT user_id, username, email FROM users WHERE id = ",
			},
			{
				Op:        "ADD_PARAM",
				ExprIndex: intPtr(0),
			},
		},
	}

	// Create generator with mock path
	gen := New(
		format,
		WithDialect(snapsql.DialectPostgres),
		WithMockPath("./testdata/mocks.json"),
	)

	// Generate code
	var buf bytes.Buffer

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify mock loading code is present
	assert.Contains(t, code, "def load_mock_data", "should include mock loading function")
	assert.Contains(t, code, "MOCK_DATA_PATH = \"./testdata/mocks.json\"", "should include mock path constant")

	// Verify mock execution code is present
	assert.Contains(t, code, "ctx.mock_mode", "should check for mock mode")
	assert.Contains(t, code, "ctx.mock_data.get(\"GetUserByID\")", "should get mock data for function")

	// Verify mock error handling
	assert.Contains(t, code, "mock.get('error')", "should handle mock errors")
	assert.Contains(t, code, "error_type", "should check error type")
	assert.Contains(t, code, "raise NotFoundError", "should raise NotFoundError for not_found type")
	assert.Contains(t, code, "raise ValidationError", "should raise ValidationError for validation type")
	assert.Contains(t, code, "raise DatabaseError", "should raise DatabaseError for other types")

	// Verify mock returns correct type
	assert.Contains(t, code, "return GetUserByIDResult(**rows[0])", "should return correct result type")
}

func TestMockIntegration_ManyAffinity(t *testing.T) {
	// Create intermediate format with many affinity
	format := &intermediate.IntermediateFormat{
		FunctionName:     "ListUsers",
		Description:      "List all users with mock support",
		StatementType:    "SELECT",
		ResponseAffinity: "many",
		Parameters:       []intermediate.Parameter{},
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
			{
				Op:    "EMIT_STATIC",
				Value: "SELECT user_id, username FROM users",
			},
		},
	}

	// Create generator with mock path
	gen := New(
		format,
		WithDialect(snapsql.DialectPostgres),
		WithMockPath("./testdata/mocks.json"),
	)

	// Generate code
	var buf bytes.Buffer

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify async generator with mock support
	assert.Contains(t, code, "AsyncGenerator[ListUsersResult, None]", "should be async generator")
	assert.Contains(t, code, "for row in rows:", "should iterate over mock rows")
	assert.Contains(t, code, "yield ListUsersResult(**row)", "should yield results")
	assert.Contains(t, code, "return", "should return after yielding mock data")
}

func TestMockIntegration_NoneAffinity(t *testing.T) {
	// Create intermediate format with none affinity
	format := &intermediate.IntermediateFormat{
		FunctionName:     "DeleteUser",
		Description:      "Delete user with mock support",
		StatementType:    "DELETE",
		ResponseAffinity: "none",
		Parameters: []intermediate.Parameter{
			{
				Name:     "user_id",
				Type:     "int",
				Optional: false,
			},
		},
		Responses: []intermediate.Response{},
		Instructions: []intermediate.Instruction{
			{
				Op:    "EMIT_STATIC",
				Value: "DELETE FROM users WHERE id = ",
			},
			{
				Op:        "ADD_PARAM",
				ExprIndex: intPtr(0),
			},
		},
	}

	// Create generator with mock path
	gen := New(
		format,
		WithDialect(snapsql.DialectPostgres),
		WithMockPath("./testdata/mocks.json"),
	)

	// Generate code
	var buf bytes.Buffer

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify none affinity returns rows affected
	assert.Contains(t, code, "return mock.get('rows_affected', 0)", "should return rows affected from mock")
	assert.Contains(t, code, "-> int:", "should return int type")
}

func TestMockIntegration_WithoutMockPath(t *testing.T) {
	// Create intermediate format without mock path
	format := &intermediate.IntermediateFormat{
		FunctionName:     "GetUser",
		Description:      "Get user without mock support",
		StatementType:    "SELECT",
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
			{
				Op:    "EMIT_STATIC",
				Value: "SELECT user_id FROM users WHERE id = ",
			},
			{
				Op:        "ADD_PARAM",
				ExprIndex: intPtr(0),
			},
		},
	}

	// Create generator without mock path
	gen := New(
		format,
		WithDialect(snapsql.DialectPostgres),
	)

	// Generate code
	var buf bytes.Buffer

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify no mock code is generated
	assert.NotContains(t, code, "def load_mock_data", "should not include mock loading function")
	assert.NotContains(t, code, "MOCK_DATA_PATH", "should not include mock path constant")
	assert.NotContains(t, code, "ctx.mock_mode", "should not check for mock mode")
}

func TestMockIntegration_ErrorTypes(t *testing.T) {
	// Verify all error types are handled
	format := &intermediate.IntermediateFormat{
		FunctionName:     "TestFunction",
		Description:      "Test function",
		StatementType:    "SELECT",
		ResponseAffinity: "one",
		Parameters:       []intermediate.Parameter{},
		Responses: []intermediate.Response{
			{Name: "id", Type: "int", IsNullable: false},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT 1"},
		},
	}

	gen := New(
		format,
		WithDialect(snapsql.DialectPostgres),
		WithMockPath("./testdata/mocks.json"),
	)

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	require.NoError(t, err)

	code := buf.String()

	// Verify error handling logic
	errorHandling := []string{
		"if mock.get('error'):",
		"error_type = mock.get('error_type', 'database')",
		"error_msg = mock['error']",
		"if error_type == 'not_found':",
		"raise NotFoundError(",
		"message=error_msg",
		"elif error_type == 'validation':",
		"raise ValidationError(",
		"else:",
		"raise DatabaseError(",
	}

	for _, line := range errorHandling {
		assert.True(t, strings.Contains(code, line), "should contain error handling: %s", line)
	}
}
