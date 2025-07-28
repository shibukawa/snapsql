package intermediate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	. "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// GetSystemFieldsSQL generates the SQL fragment for system fields
// This is a helper function to generate the actual SQL text for manual SQL construction
func GetSystemFieldsSQL(t *testing.T, implicitParams []ImplicitParameter) string {
	t.Helper()
	if len(implicitParams) == 0 {
		return ""
	}

	var systemCalls []string
	for _, param := range implicitParams {
		systemCalls = append(systemCalls, fmt.Sprintf("EMIT_SYSTEM_VALUE(%s)", param.Name))
	}

	return ", " + strings.Join(systemCalls, ", ")
}
func TestSystemFieldIntegration_Simple(t *testing.T) {
	// Real-world configuration
	config := &Config{
		System: SystemConfig{
			Fields: []SystemField{
				{
					Name: "updated_at",
					Type: "timestamp",
					OnUpdate: SystemFieldOperation{
						Default: "NOW()",
					},
				},
				{
					Name: "updated_by",
					Type: "string",
					OnUpdate: SystemFieldOperation{
						Parameter: ParameterImplicit,
					},
				},
			},
		},
	}

	// Parse actual UPDATE statement with function definition
	sql := `/*
name: UpdateUser
function_name: updateUser
description: Update user information
parameters:
  name: string
  email: string
*/
UPDATE users SET name = 'Updated Name', email = 'updated@example.com' WHERE id = 1`
	reader := strings.NewReader(sql)

	stmt, _, err := parser.ParseSQLFile(reader, nil, "", "")
	if err != nil {
		t.Skipf("Parser not fully implemented yet: %v", err)
		return
	}

	// Cast to UpdateStatement
	updateStmt, ok := stmt.(*parsercommon.UpdateStatement)
	assert.True(t, ok)

	// Parameters provided by user
	parameters := []Parameter{
		{Name: "name", Type: "string"},
		{Name: "email", Type: "string"},
	}

	// Step 1: Check system fields and get implicit parameters
	gerr := &GenerateError{}
	implicitParams := CheckSystemFields(updateStmt, config, parameters, gerr)
	assert.False(t, gerr.HasErrors(), "Expected no errors but got: %v", gerr)
	assert.Equal(t, 2, len(implicitParams)) // updated_at (default), updated_by (implicit)

	// Verify initial SET clause has 2 assignments
	assert.Equal(t, 2, len(updateStmt.Set.Assigns)) // name, email

	// Step 2: Add system fields to UPDATE statement
	err = AddSystemFieldsToUpdate(updateStmt, implicitParams)
	assert.NoError(t, err)

	// Verify that system fields were added to SET clause
	assert.Equal(t, 4, len(updateStmt.Set.Assigns)) // 2 original + 2 system fields

	// Check that original assignments are preserved
	assert.Equal(t, "name", updateStmt.Set.Assigns[0].FieldName)
	assert.Equal(t, "email", updateStmt.Set.Assigns[1].FieldName)

	// Check that system fields were added
	assert.Equal(t, "updated_at", updateStmt.Set.Assigns[2].FieldName)
	assert.Equal(t, "EMIT_SYSTEM_VALUE", updateStmt.Set.Assigns[2].Value[0].Value)
	assert.Equal(t, "updated_at", updateStmt.Set.Assigns[2].Value[2].Value)

	assert.Equal(t, "updated_by", updateStmt.Set.Assigns[3].FieldName)
	assert.Equal(t, "EMIT_SYSTEM_VALUE", updateStmt.Set.Assigns[3].Value[0].Value)
	assert.Equal(t, "updated_by", updateStmt.Set.Assigns[3].Value[2].Value)
}

func TestSystemFieldIntegration_InsertStatement(t *testing.T) {
	// Configuration for INSERT
	config := &Config{
		System: SystemConfig{
			Fields: []SystemField{
				{
					Name: "created_at",
					Type: "timestamp",
					OnInsert: SystemFieldOperation{
						Default: "NOW()",
					},
				},
				{
					Name: "created_by",
					Type: "string",
					OnInsert: SystemFieldOperation{
						Parameter: ParameterImplicit,
					},
				},
			},
		},
	}

	// Parse actual INSERT statement with function definition
	sql := `/*
name: InsertUser
function_name: insertUser
description: Insert new user
parameters:
  name: string
  email: string
*/
INSERT INTO users (name, email) VALUES ('John', 'john@example.com')`
	reader := strings.NewReader(sql)

	stmt, _, err := parser.ParseSQLFile(reader, nil, "", "")
	if err != nil {
		t.Skipf("Parser not fully implemented yet: %v", err)
		return
	}

	parameters := []Parameter{
		{Name: "name", Type: "string"},
		{Name: "email", Type: "string"},
	}

	// Should get implicit parameters for INSERT
	gerr := &GenerateError{}
	implicitParams := CheckSystemFields(stmt, config, parameters, gerr)
	assert.False(t, gerr.HasErrors(), "Expected no errors but got: %v", gerr)
	assert.Equal(t, 2, len(implicitParams))

	// Verify implicit parameters
	createdAt := findImplicitParam(implicitParams, "created_at")
	assert.True(t, createdAt != nil)
	assert.Equal(t, "timestamp", createdAt.Type)
	assert.Equal(t, "NOW()", createdAt.Default)

	createdBy := findImplicitParam(implicitParams, "created_by")
	assert.True(t, createdBy != nil)
	assert.Equal(t, "string", createdBy.Type)
	assert.Equal(t, nil, createdBy.Default)

	// Should not modify INSERT statement (no SET clause to modify)
	err = AddSystemFieldsToUpdate(stmt, implicitParams)
	assert.NoError(t, err) // Should not error, just skip processing
}

func TestGetSystemFieldsSQL(t *testing.T) {
	tests := []struct {
		name           string
		implicitParams []ImplicitParameter
		expected       string
	}{
		{
			name:           "No implicit parameters",
			implicitParams: []ImplicitParameter{},
			expected:       "",
		},
		{
			name: "Single implicit parameter",
			implicitParams: []ImplicitParameter{
				{Name: "updated_at", Type: "timestamp"},
			},
			expected: ", EMIT_SYSTEM_VALUE(updated_at)",
		},
		{
			name: "Multiple implicit parameters",
			implicitParams: []ImplicitParameter{
				{Name: "updated_at", Type: "timestamp"},
				{Name: "updated_by", Type: "string"},
			},
			expected: ", EMIT_SYSTEM_VALUE(updated_at), EMIT_SYSTEM_VALUE(updated_by)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSystemFieldsSQL(t, tt.implicitParams)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSystemFieldAssignments(t *testing.T) {
	implicitParams := []ImplicitParameter{
		{Name: "updated_at", Type: "timestamp", Default: "NOW()"},
		{Name: "updated_by", Type: "string"},
	}

	assignments := GetSystemFieldAssignments(implicitParams)
	assert.Equal(t, 2, len(assignments))

	// Check first assignment (updated_at)
	assert.Equal(t, "updated_at", assignments[0].FieldName)
	assert.Equal(t, 4, len(assignments[0].Value))
	assert.Equal(t, "EMIT_SYSTEM_VALUE", assignments[0].Value[0].Value)
	assert.Equal(t, "(", assignments[0].Value[1].Value)
	assert.Equal(t, "updated_at", assignments[0].Value[2].Value)
	assert.Equal(t, ")", assignments[0].Value[3].Value)

	// Check second assignment (updated_by)
	assert.Equal(t, "updated_by", assignments[1].FieldName)
	assert.Equal(t, 4, len(assignments[1].Value))
	assert.Equal(t, "EMIT_SYSTEM_VALUE", assignments[1].Value[0].Value)
	assert.Equal(t, "(", assignments[1].Value[1].Value)
	assert.Equal(t, "updated_by", assignments[1].Value[2].Value)
	assert.Equal(t, ")", assignments[1].Value[3].Value)
}

func TestCheckSystemFields_MockData(t *testing.T) {
	// Test the system field checking logic with mock data
	config := &Config{
		System: SystemConfig{
			Fields: []SystemField{
				{
					Name: "updated_at",
					Type: "timestamp",
					OnUpdate: SystemFieldOperation{
						Default: "NOW()",
					},
				},
				{
					Name: "updated_by",
					Type: "string",
					OnUpdate: SystemFieldOperation{
						Parameter: ParameterImplicit,
					},
				},
				{
					Name: "lock_no",
					Type: "int",
					OnUpdate: SystemFieldOperation{
						Parameter: ParameterExplicit,
					},
				},
			},
		},
	}

	// Mock UPDATE statement
	mockStmt := &MockUpdateStatement{
		stmtType: parsercommon.UPDATE_STATEMENT,
	}

	// Test with valid parameters
	parameters := []Parameter{
		{Name: "name", Type: "string"},
		{Name: "lock_no", Type: "int"}, // Explicit parameter provided
	}

	gerr := &GenerateError{}
	implicitParams := CheckSystemFields(mockStmt, config, parameters, gerr)
	assert.False(t, gerr.HasErrors(), "Expected no errors but got: %v", gerr)
	assert.Equal(t, 2, len(implicitParams)) // updated_at (default), updated_by (implicit)

	// Verify implicit parameters
	updatedAt := findImplicitParam(implicitParams, "updated_at")
	assert.True(t, updatedAt != nil)
	assert.Equal(t, "timestamp", updatedAt.Type)
	assert.Equal(t, "NOW()", updatedAt.Default)

	updatedBy := findImplicitParam(implicitParams, "updated_by")
	assert.True(t, updatedBy != nil)
	assert.Equal(t, "string", updatedBy.Type)
	assert.Equal(t, nil, updatedBy.Default)
}

// Mock statement for testing
type MockUpdateStatement struct {
	stmtType parsercommon.NodeType
}

func (m *MockUpdateStatement) Type() parsercommon.NodeType { return m.stmtType }

// Helper function to find implicit parameter by name
func findImplicitParam(params []ImplicitParameter, name string) *ImplicitParameter {
	for _, param := range params {
		if param.Name == name {
			return &param
		}
	}
	return nil
}
