package gogen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestGenerator(t *testing.T) {
	// Helper function to create *int
	intPtr := func(i int) *int {
		return &i
	}

	// Test case based on find_user_by_id.go
	format := &intermediate.IntermediateFormat{
		FunctionName: "FindUserByID",
		Description:  "finds a user by ID and tenant ID",
		CELEnvironments: []intermediate.CELEnvironment{
			{
				Index: 0,
				AdditionalVariables: []intermediate.CELVariableInfo{
					{Name: "tenant_id", Type: "string"}, // idはParametersにあるので除外
				},
			},
		},
		CELExpressions: []intermediate.CELExpression{
			{
				ID:               "expr_001",
				Expression:       "id",
				EnvironmentIndex: 0,
			},
			{
				ID:               "expr_002",
				Expression:       "tenant_id",
				EnvironmentIndex: 0,
			},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT id, name, email, tenant_id, department_id, status, created_by, created_at, updated_at FROM users WHERE id = "},
			{Op: "EMIT_EVAL", ExprIndex: intPtr(0)},
			{Op: "EMIT_STATIC", Value: " AND tenant_id = "},
			{Op: "EMIT_EVAL", ExprIndex: intPtr(1)},
		},
		Parameters: []intermediate.Parameter{
			{Name: "id", Type: "int"},
		},
		ResponseAffinity: "one",
		Responses: []intermediate.Response{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "string"},
			{Name: "email", Type: "string"},
			{Name: "tenant_id", Type: "string"},
			{Name: "department_id", Type: "int"},
			{Name: "status", Type: "string"},
			{Name: "created_by", Type: "int"},
			{Name: "created_at", Type: "time.Time"},
			{Name: "updated_at", Type: "*time.Time"},
		},
	}

	gen := New(format,
		WithPackageName("generated"),
		WithMockPath("comprehensive/find_user_by_id"),
	)

	var buf strings.Builder
	err := gen.Generate(&buf)
	assert.NoError(t, err)

	// TODO: Add assertions for generated code
	t.Logf("Generated code:\n%s", buf.String())
}

func TestGeneratorWithoutDescription(t *testing.T) {
	gen := &Generator{
		PackageName: "generated",
		MockPath:    "test/mock",
		Format: &intermediate.IntermediateFormat{
			FunctionName: "TestFunction",
			// Description is empty
			Parameters: []intermediate.Parameter{
				{Name: "id", Type: "int"},
			},
			ResponseAffinity: "one",
			Responses: []intermediate.Response{
				{Name: "id", Type: "int"},
				{Name: "name", Type: "string"},
			},
		},
	}

	var buf bytes.Buffer
	err := gen.Generate(&buf)
	assert.NoError(t, err)

	generated := buf.String()

	// Should use fallback comment format when description is empty
	assert.Contains(t, generated, "// TestFunction - TestFunctionResult Affinity")
	assert.NotContains(t, generated, "finds a user")
}
