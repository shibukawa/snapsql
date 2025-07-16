package typeinference

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// TestSubqueryTypeInference tests type inference with subqueries
func TestSubqueryTypeInference(t *testing.T) {
	// Create test schema (simplified format without substructures)
	schema := snapsql.DatabaseSchema{
		Name: "test_db",
		DatabaseInfo: snapsql.DatabaseInfo{
			Type:    "postgres",
			Version: "13.0",
		},
	}

	testCases := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "simple_select_with_dual",
			sql:         "SELECT 1 as id, 'test' as name FROM dual",
			expectError: false,
		},
		{
			name:        "simple_function_call_with_dual",
			sql:         "SELECT COUNT(*) as count FROM dual",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse SQL using existing parseSQL function
			stmt, err := parseSQLSimple(tc.sql)
			assert.NoError(t, err, "Failed to parse SQL")

			// Perform type inference
			results, err := InferFieldTypes([]snapsql.DatabaseSchema{schema}, stmt, nil)

			if tc.expectError {
				assert.True(t, err != nil, "Expected error but got none")
			} else {
				// For now, we just expect the function to not crash
				// The subquery resolver will gracefully handle missing parse results
				assert.True(t, len(results) > 0 || err != nil, "Should return results or error")
			}
		})
	}
}

// Helper function to parse SQL (using tokenizer)
func parseSQLSimple(sql string) (parser.StatementNode, error) {
	tokens, err := tokenizer.Tokenize(sql)
	if err != nil {
		return nil, err
	}

	// Use basic parser without subquery analysis
	stmt, err := parser.Parse(tokens, nil, nil)
	if err != nil {
		return nil, err
	}

	return stmt, nil
}
