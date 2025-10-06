package query

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/intermediate"
)

// TestIsDangerousQuery tests the dangerous query detection
func TestIsDangerousQuery(t *testing.T) {
	testCases := []struct {
		name     string
		sql      string
		expected bool
	}{
		{
			name:     "safe SELECT query",
			sql:      "SELECT * FROM users",
			expected: false,
		},
		{
			name:     "safe SELECT with WHERE",
			sql:      "SELECT * FROM users WHERE id = 1",
			expected: false,
		},
		{
			name:     "dangerous DELETE without WHERE",
			sql:      "DELETE FROM users",
			expected: true,
		},
		{
			name:     "safe DELETE with WHERE",
			sql:      "DELETE FROM users WHERE id = 1",
			expected: false,
		},
		{
			name:     "dangerous UPDATE without WHERE",
			sql:      "UPDATE users SET name = 'test'",
			expected: true,
		},
		{
			name:     "safe UPDATE with WHERE",
			sql:      "UPDATE users SET name = 'test' WHERE id = 1",
			expected: false,
		},
		{
			name:     "safe INSERT",
			sql:      "INSERT INTO users (name) VALUES ('test')",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsDangerousQuery(tc.sql)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetDialectFromDriver tests dialect conversion
func TestGetDialectFromDriver(t *testing.T) {
	testCases := []struct {
		driver   string
		expected string
	}{
		{"postgres", "postgresql"},
		{"pgx", "postgresql"},
		{"mysql", "mysql"},
		{"sqlite3", "sqlite"},
		{"unknown", "postgresql"}, // default
	}

	for _, tc := range testCases {
		t.Run(tc.driver, func(t *testing.T) {
			result := getDialectFromDriver(tc.driver)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestOpenDatabase tests database connection opening
func TestOpenDatabase(t *testing.T) {
	// Test with invalid driver
	_, err := OpenDatabase("invalid_driver", "invalid_connection", 30)
	assert.Error(t, err)
}

func TestBuildSQLAddsSpacingAfterBoundaryTokens(t *testing.T) {
	idx0, idx1 := 0, 1
	instructions := []intermediate.OptimizedInstruction{
		{Op: "EMIT_STATIC", Value: "SELECT id FROM t WHERE a ="},
		{Op: "EMIT_STATIC", Value: "?"},
		{Op: "ADD_PARAM", ExprIndex: &idx0},
		{Op: "EMIT_UNLESS_BOUNDARY", Value: "AND"},
		{Op: "EMIT_STATIC", Value: "b >"},
		{Op: "EMIT_STATIC", Value: "?"},
		{Op: "ADD_PARAM", ExprIndex: &idx1},
	}

	format := &intermediate.IntermediateFormat{
		CELExpressions: []intermediate.CELExpression{
			{Expression: "param1"},
			{Expression: "param2"},
		},
	}

	params := map[string]interface{}{
		"param1": 123,
		"param2": 456,
	}

	exec := &Executor{}
	sql, args, err := exec.buildSQLFromOptimized(instructions, format, params)
	assert.NoError(t, err)
	assert.Equal(t, "SELECT id FROM t WHERE a =? AND b >?", sql)
	assert.Equal(t, 2, len(args))
}
