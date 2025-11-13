package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDeleteInstructions(t *testing.T) {
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
			name:        "basic_delete_all",
			sql:         "DELETE FROM users",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "DELETE FROM users", Pos: "1:1"},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:        "delete_with_where",
			sql:         "DELETE FROM users WHERE age < 18",
			dialect:     snapsql.DialectPostgres,
			expectError: false,
			expectedInstructions: []Instruction{
				{Op: OpEmitStatic, Value: "DELETE FROM users WHERE age < 18", Pos: "1:1"},
			},
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:             "delete_with_parameters",
			sql:              `/*# parameters: { min_age: int } */ DELETE FROM users WHERE age < /*= min_age */ 18`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 1,
			expectedEnvCount: 1,
		},
		{
			name:             "delete_with_complex_condition",
			sql:              `/*# parameters: { department: string, inactive_days: int } */ DELETE FROM users WHERE department = /*= department */ 'Sales' AND last_login < CURRENT_DATE - INTERVAL /*= inactive_days */ 90 DAY`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 2,
			expectedEnvCount: 1,
		},
		{
			name:             "delete_with_returning",
			sql:              "DELETE FROM users WHERE age < 18 RETURNING id, name, email",
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:             "delete_with_conditional_directive",
			sql:              `/*# parameters: { department: string, has_department: bool } */ DELETE FROM users WHERE 1=1 /*# if has_department */ AND department = /*= department */ 'Sales' /*# end */`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 2,
			expectedEnvCount: 1,
		},
		{
			name:             "delete_with_cte",
			sql:              `WITH inactive_users AS (SELECT id FROM users WHERE last_login < CURRENT_DATE - 180) DELETE FROM users WHERE id IN (SELECT id FROM inactive_users)`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:             "delete_with_subquery",
			sql:              `DELETE FROM users WHERE id IN (SELECT user_id FROM audit_log WHERE action = 'delete')`,
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 1,
		},
		{
			name:             "delete_with_returning_mariadb",
			sql:              "DELETE FROM users WHERE status = 'inactive' RETURNING id, email",
			dialect:          snapsql.DialectMariaDB,
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

			deleteStmt, ok := stmt.(*parser.DeleteFromStatement)
			require.True(t, ok, "Expected DeleteFromStatement")

			ctx := NewGenerationContext(tt.dialect)
			instructions, expressions, environments, err := GenerateDeleteInstructions(deleteStmt, ctx)

			if tt.expectError {
				require.Error(t, err, "Expected error")

				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				return
			}

			require.NoError(t, err, "GenerateDeleteInstructions should succeed")
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
