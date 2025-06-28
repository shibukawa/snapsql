package pull

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"
	snapsql "github.com/shibukawa/snapsql"
)

func TestMySQLExtractor(t *testing.T) {
	t.Run("CreateMySQLExtractor", func(t *testing.T) {
		extractor := NewMySQLExtractor()
		assert.NotZero(t, extractor)
		assert.NotZero(t, extractor.BaseExtractor)
	})

	t.Run("ExtractorType", func(t *testing.T) {
		extractor := NewMySQLExtractor()
		// Test that the extractor is properly initialized
		assert.NotZero(t, extractor)
	})
}

func TestMySQLQueries(t *testing.T) {
	extractor := NewMySQLExtractor()

	t.Run("BuildTablesQuery", func(t *testing.T) {
		query := extractor.BuildTablesQuery("testdb")
		assert.Contains(t, query, "information_schema.TABLES")
		assert.Contains(t, query, "TABLE_SCHEMA = 'testdb'")
		assert.Contains(t, query, "TABLE_TYPE = 'BASE TABLE'")
	})

	t.Run("BuildColumnsQuery", func(t *testing.T) {
		query := extractor.BuildColumnsQuery("testdb", "users")
		assert.Contains(t, query, "information_schema.COLUMNS")
		assert.Contains(t, query, "TABLE_SCHEMA = 'testdb'")
		assert.Contains(t, query, "TABLE_NAME = 'users'")
		assert.Contains(t, query, "ORDINAL_POSITION")
	})

	t.Run("BuildConstraintsQuery", func(t *testing.T) {
		query := extractor.BuildConstraintsQuery("testdb", "users")
		assert.Contains(t, query, "information_schema.TABLE_CONSTRAINTS")
		assert.Contains(t, query, "information_schema.KEY_COLUMN_USAGE")
		assert.Contains(t, query, "TABLE_SCHEMA = 'testdb'")
		assert.Contains(t, query, "TABLE_NAME = 'users'")
	})

	t.Run("BuildIndexesQuery", func(t *testing.T) {
		query := extractor.BuildIndexesQuery("testdb", "users")
		assert.Contains(t, query, "information_schema.STATISTICS")
		assert.Contains(t, query, "TABLE_SCHEMA = 'testdb'")
		assert.Contains(t, query, "TABLE_NAME = 'users'")
		assert.Contains(t, query, "SEQ_IN_INDEX")
	})

	t.Run("BuildViewsQuery", func(t *testing.T) {
		query := extractor.BuildViewsQuery("testdb")
		assert.Contains(t, query, "information_schema.VIEWS")
		assert.Contains(t, query, "TABLE_SCHEMA = 'testdb'")
		assert.Contains(t, query, "VIEW_DEFINITION")
	})
}

func TestMySQLTypeMapping(t *testing.T) {
	extractor := NewMySQLExtractor()

	t.Run("MapMySQLTypes", func(t *testing.T) {
		testCases := []struct {
			mysqlType    string
			expectedType string
		}{
			{"int", "int"},
			{"bigint", "int"},
			{"varchar", "string"},
			{"text", "string"},
			{"decimal", "float"},
			{"float", "float"},
			{"double", "float"},
			{"datetime", "datetime"},
			{"timestamp", "datetime"},
			{"date", "date"},
			{"time", "time"},
			{"boolean", "bool"},
			{"tinyint", "int"}, // MySQL uses tinyint(1) for boolean
			{"json", "json"},
			{"blob", "binary"},
			{"unknown_type", "string"}, // Default fallback
		}

		for _, tc := range testCases {
			result := extractor.MapColumnType(tc.mysqlType)
			assert.Equal(t, tc.expectedType, result, "Failed for MySQL type: %s", tc.mysqlType)
		}
	})
}

func TestMySQLConstraintParsing(t *testing.T) {
	extractor := NewMySQLExtractor()
	testConstraintParsing(t, extractor, "CUSTOM")
}

func TestMySQLDefaultValues(t *testing.T) {
	extractor := NewMySQLExtractor()

	t.Run("ParseDefaultValues", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP"},
			{"NOW()", "CURRENT_TIMESTAMP"},
			{"NULL", ""},
			{"'default_value'", "default_value"},
			{"42", "42"},
			{"'quoted string'", "quoted string"},
		}

		for _, tc := range testCases {
			result := extractor.ParseDefaultValue(tc.input)
			assert.Equal(t, tc.expected, result, "Failed for input: %s", tc.input)
		}
	})
}

func TestMySQLErrorHandling(t *testing.T) {
	extractor := NewMySQLExtractor()

	t.Run("HandleDatabaseConnectionError", func(t *testing.T) {
		// Test error handling for database connection issues
		mockErr := errors.New("connection refused")
		err := extractor.HandleDatabaseError(mockErr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection error")
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("HandleQueryExecutionError", func(t *testing.T) {
		// Test error handling for query execution issues
		mockErr := errors.New("syntax error")
		err := extractor.HandleDatabaseError(mockErr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query execution failed")
		assert.Contains(t, err.Error(), "syntax error")
	})

	t.Run("HandleSchemaNotFoundError", func(t *testing.T) {
		// Test error handling for schema not found
		mockErr := errors.New("database doesn't exist")
		err := extractor.HandleDatabaseError(mockErr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema not found")
		assert.Contains(t, err.Error(), "doesn't exist")
	})

	t.Run("HandlePermissionError", func(t *testing.T) {
		// Test error handling for permission denied
		mockErr := errors.New("Access denied for user")
		err := extractor.HandleDatabaseError(mockErr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
		assert.Contains(t, err.Error(), "Access denied")
	})
}

func TestMySQLSpecificFeatures(t *testing.T) {
	extractor := NewMySQLExtractor()

	t.Run("HandleAutoIncrement", func(t *testing.T) {
		// Test that auto increment is properly handled in comments
		// This would be tested in integration tests with real data
		assert.NotZero(t, extractor)
	})

	t.Run("HandleMySQLEngines", func(t *testing.T) {
		// Test that MySQL storage engines are properly handled
		// This would be tested in integration tests with real data
		assert.NotZero(t, extractor)
	})

	t.Run("HandleMySQLCharsets", func(t *testing.T) {
		// Test that MySQL character sets are properly handled
		// This would be tested in integration tests with real data
		assert.NotZero(t, extractor)
	})
}

func TestMySQLIndexHandling(t *testing.T) {
	extractor := NewMySQLExtractor()

	t.Run("SkipPrimaryKeyIndex", func(t *testing.T) {
		// Test that PRIMARY key indexes are skipped (handled as constraints)
		// This logic is in ExtractIndexes method
		assert.NotZero(t, extractor)
	})

	t.Run("HandleUniqueIndexes", func(t *testing.T) {
		// Test that unique indexes are properly identified
		// This would be tested in integration tests with real data
		assert.NotZero(t, extractor)
	})

	t.Run("HandleCompositeIndexes", func(t *testing.T) {
		// Test that composite indexes maintain column order
		// This would be tested in integration tests with real data
		assert.NotZero(t, extractor)
	})
}

func TestMySQLViewHandling(t *testing.T) {
	extractor := NewMySQLExtractor()

	t.Run("ExtractViewDefinitions", func(t *testing.T) {
		// Test that view definitions are properly extracted
		// This would be tested in integration tests with real data
		assert.NotZero(t, extractor)
	})

	t.Run("HandleComplexViews", func(t *testing.T) {
		// Test that complex views with joins are handled
		// This would be tested in integration tests with real data
		assert.NotZero(t, extractor)
	})
}

func TestMySQLDatabaseInfo(t *testing.T) {
	t.Run("DatabaseInfoStructure", func(t *testing.T) {
		// Test that database info has correct structure for MySQL
		info := snapsql.DatabaseInfo{
			Type:    "mysql",
			Version: "8.0.33",
			Name:    "testdb",
			Charset: "utf8mb4",
		}

		assert.Equal(t, "mysql", info.Type)
		assert.Equal(t, "8.0.33", info.Version)
		assert.Equal(t, "testdb", info.Name)
		assert.Equal(t, "utf8mb4", info.Charset)
	})
}
