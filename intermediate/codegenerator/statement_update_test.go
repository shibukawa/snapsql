package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateUpdateInstructions(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectError          bool
		errorContains        string
		expectedInstructions []Instruction
		expectedCELCount     int
		expectedEnvCount     int
	}{
		{
			name:        "basic_update_all",
			sql:         "UPDATE users SET status = 'active'",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "UPDATE users SET status = 'active'", Pos: "1:1"},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "update_with_where",
			sql:         "UPDATE users SET status = 'inactive' WHERE last_login < CURRENT_DATE - INTERVAL 180 DAY",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "UPDATE users SET status = 'inactive' WHERE last_login < CURRENT_DATE - INTERVAL 180 DAY", Pos: "1:1"},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "update_multiple_columns",
			sql:         "UPDATE users SET status = 'active', updated_at = NOW(), login_count = login_count + 1 WHERE id = 123",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "UPDATE users SET status = 'active', updated_at = NOW(), login_count = login_count + 1 WHERE id = 123", Pos: "1:1"},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:             "update_with_parameters",
			sql:              `/*# parameters: { user_id: int, new_status: string } */ UPDATE users SET status = /*= new_status */ 'active' WHERE id = /*= user_id */ 123`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 2,
			expectedEnvCount: 1,
		},
		{
			name:             "update_with_complex_condition",
			sql:              `/*# parameters: { department: string, min_salary: int, increment: int } */ UPDATE employees SET salary = salary + /*= increment */ 1000, updated_at = CURRENT_TIMESTAMP WHERE department = /*= department */ 'Engineering' AND salary < /*= min_salary */ 50000 AND status = 'active'`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 3,
			expectedEnvCount: 1,
		},
		{
			name:             "update_with_returning",
			sql:              "UPDATE users SET status = 'inactive', updated_at = CURRENT_TIMESTAMP WHERE last_login < CURRENT_DATE - 180 RETURNING id, email, status, updated_at",
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:             "update_with_conditional_directive",
			sql:              `/*# parameters: { status: string, reason: string } */ UPDATE users SET status = /*= status */ 'inactive', updated_at = CURRENT_TIMESTAMP /*# if reason */ , inactive_reason = /*= reason */ 'manual' /*# end */ WHERE id = 123`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 2,
			expectedEnvCount: 1,
		},
		{
			name:             "update_with_cte",
			sql:              `WITH inactive_users AS (SELECT id FROM users WHERE last_login < CURRENT_DATE - 180) UPDATE users SET status = 'inactive' WHERE id IN (SELECT id FROM inactive_users)`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:             "update_with_subquery",
			sql:              `UPDATE users SET department_name = (SELECT name FROM departments WHERE departments.id = users.department_id) WHERE department_id IS NOT NULL`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.sql)
			stmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt)

			updateStmt, ok := stmt.(*parser.UpdateStatement)
			require.True(t, ok, "Expected UpdateStatement")

			ctx := NewGenerationContext(tt.dialect)
			instructions, expressions, environments, err := GenerateUpdateInstructions(updateStmt, ctx)

			if tt.expectError {
				require.Error(t, err, "Expected error")

				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				return
			}

			require.NoError(t, err, "GenerateUpdateInstructions should succeed")
			require.NotNil(t, instructions)
			require.NotNil(t, expressions)
			require.NotNil(t, environments)

			if len(tt.expectedInstructions) > 0 {
				assert.Equal(t, tt.expectedInstructions, instructions,
					"Instructions should match exactly")
			}

			assert.Equal(t, tt.expectedCELCount, len(expressions),
				"CEL expression count should match")
			assert.Equal(t, tt.expectedEnvCount, len(environments),
				"CEL environment count should match")

			t.Logf("✓ Generated %d instructions", len(instructions))
			t.Logf("✓ Generated %d CEL expressions", len(expressions))
			t.Logf("✓ Generated %d environments", len(environments))
		})
	}
}
