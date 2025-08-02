package query

import (
	"testing"

	"github.com/alecthomas/assert/v2"
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
