package pull

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"
)

var ErrMockNotImplemented = errors.New("mock functionality not implemented")

func TestExtractorInterface(t *testing.T) {
	t.Run("PostgreSQLExtractorImplementsInterface", func(t *testing.T) {
		extractor := NewPostgreSQLExtractor()

		// Verify it implements the Extractor interface
		var _ Extractor = extractor

		assert.NotZero(t, extractor)
	})

	t.Run("MySQLExtractorImplementsInterface", func(t *testing.T) {
		extractor := NewMySQLExtractor()

		// Verify it implements the Extractor interface
		var _ Extractor = extractor

		assert.NotZero(t, extractor)
	})

	t.Run("SQLiteExtractorImplementsInterface", func(t *testing.T) {
		extractor := NewSQLiteExtractor()

		// Verify it implements the Extractor interface
		var _ Extractor = extractor

		assert.NotZero(t, extractor)
	})
}

func TestExtractorFactory(t *testing.T) {
	t.Run("CreatePostgreSQLExtractor", func(t *testing.T) {
		extractor, err := NewExtractor("postgresql")
		assert.NoError(t, err)
		assert.NotZero(t, extractor)

		// Verify it's the correct type
		_, ok := extractor.(*PostgreSQLExtractor)
		assert.True(t, ok)
	})

	t.Run("CreateMySQLExtractor", func(t *testing.T) {
		extractor, err := NewExtractor("mysql")
		assert.NoError(t, err)
		assert.NotZero(t, extractor)

		// Verify it's the correct type
		_, ok := extractor.(*MySQLExtractor)
		assert.True(t, ok)
	})

	t.Run("CreateSQLiteExtractor", func(t *testing.T) {
		extractor, err := NewExtractor("sqlite")
		assert.NoError(t, err)
		assert.NotZero(t, extractor)

		// Verify it's the correct type
		_, ok := extractor.(*SQLiteExtractor)
		assert.True(t, ok)
	})

	t.Run("CreateUnsupportedExtractor", func(t *testing.T) {
		extractor, err := NewExtractor("unsupported")
		assert.Error(t, err)
		assert.Zero(t, extractor)
		assert.Equal(t, ErrUnsupportedDatabase, err)
	})

	t.Run("CreateEmptyExtractor", func(t *testing.T) {
		extractor, err := NewExtractor("")
		assert.Error(t, err)
		assert.Zero(t, extractor)
		assert.Equal(t, ErrEmptyDatabaseType, err)
	})
}

func TestExtractConfigValidation(t *testing.T) {
	t.Run("ValidateBasicConfig", func(t *testing.T) {
		config := ExtractConfig{
			IncludeViews:   true,
			IncludeIndexes: true,
		}

		err := ValidateExtractConfig(config)
		assert.NoError(t, err)
	})

	t.Run("ValidateConfigWithFilters", func(t *testing.T) {
		config := ExtractConfig{
			IncludeSchemas: []string{"public", "auth"},
			ExcludeSchemas: []string{"information_schema"},
			IncludeTables:  []string{"users", "posts"},
			ExcludeTables:  []string{"migrations"},
			IncludeViews:   true,
			IncludeIndexes: false,
		}

		err := ValidateExtractConfig(config)
		assert.NoError(t, err)
	})

	t.Run("ValidateConfigWithConflictingSchemas", func(t *testing.T) {
		config := ExtractConfig{
			IncludeSchemas: []string{"public", "auth"},
			ExcludeSchemas: []string{"public"}, // Conflict: public is both included and excluded
		}

		err := ValidateExtractConfig(config)
		assert.Error(t, err)
		assert.Equal(t, ErrConflictingSchemaFilters, err)
	})

	t.Run("ValidateConfigWithConflictingTables", func(t *testing.T) {
		config := ExtractConfig{
			IncludeTables: []string{"users", "posts"},
			ExcludeTables: []string{"users"}, // Conflict: users is both included and excluded
		}

		err := ValidateExtractConfig(config)
		assert.Error(t, err)
		assert.Equal(t, ErrConflictingTableFilters, err)
	})
}

func TestSchemaFiltering(t *testing.T) {
	t.Run("ShouldIncludeSchema", func(t *testing.T) {
		testCases := []struct {
			name           string
			schemaName     string
			includeSchemas []string
			excludeSchemas []string
			expected       bool
		}{
			{
				name:           "NoFilters",
				schemaName:     "public",
				includeSchemas: []string{},
				excludeSchemas: []string{},
				expected:       true,
			},
			{
				name:           "IncludeOnly",
				schemaName:     "public",
				includeSchemas: []string{"public", "auth"},
				excludeSchemas: []string{},
				expected:       true,
			},
			{
				name:           "IncludeOnlyNotMatched",
				schemaName:     "test",
				includeSchemas: []string{"public", "auth"},
				excludeSchemas: []string{},
				expected:       false,
			},
			{
				name:           "ExcludeOnly",
				schemaName:     "public",
				includeSchemas: []string{},
				excludeSchemas: []string{"information_schema", "pg_catalog"},
				expected:       true,
			},
			{
				name:           "ExcludeOnlyMatched",
				schemaName:     "information_schema",
				includeSchemas: []string{},
				excludeSchemas: []string{"information_schema", "pg_catalog"},
				expected:       false,
			},
			{
				name:           "IncludeAndExclude",
				schemaName:     "public",
				includeSchemas: []string{"public", "auth"},
				excludeSchemas: []string{"test"},
				expected:       true,
			},
			{
				name:           "IncludeAndExcludeConflict",
				schemaName:     "public",
				includeSchemas: []string{"public", "auth"},
				excludeSchemas: []string{"public"},
				expected:       false, // Exclude takes precedence
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := ShouldIncludeSchema(tc.schemaName, tc.includeSchemas, tc.excludeSchemas)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("ShouldIncludeTable", func(t *testing.T) {
		testCases := []struct {
			name          string
			tableName     string
			includeTables []string
			excludeTables []string
			expected      bool
		}{
			{
				name:          "NoFilters",
				tableName:     "users",
				includeTables: []string{},
				excludeTables: []string{},
				expected:      true,
			},
			{
				name:          "IncludeOnly",
				tableName:     "users",
				includeTables: []string{"users", "posts"},
				excludeTables: []string{},
				expected:      true,
			},
			{
				name:          "IncludeOnlyNotMatched",
				tableName:     "comments",
				includeTables: []string{"users", "posts"},
				excludeTables: []string{},
				expected:      false,
			},
			{
				name:          "ExcludeOnly",
				tableName:     "users",
				includeTables: []string{},
				excludeTables: []string{"migrations", "temp_*"},
				expected:      true,
			},
			{
				name:          "ExcludeOnlyMatched",
				tableName:     "migrations",
				includeTables: []string{},
				excludeTables: []string{"migrations", "temp_*"},
				expected:      false,
			},
			{
				name:          "ExcludeWildcard",
				tableName:     "temp_data",
				includeTables: []string{},
				excludeTables: []string{"migrations", "temp_*"},
				expected:      false,
			},
			{
				name:          "IncludeAndExclude",
				tableName:     "users",
				includeTables: []string{"users", "posts"},
				excludeTables: []string{"migrations"},
				expected:      true,
			},
			{
				name:          "IncludeAndExcludeConflict",
				tableName:     "users",
				includeTables: []string{"users", "posts"},
				excludeTables: []string{"users"},
				expected:      false, // Exclude takes precedence
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := ShouldIncludeTable(tc.tableName, tc.includeTables, tc.excludeTables)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

func TestWildcardMatching(t *testing.T) {
	t.Run("MatchWildcard", func(t *testing.T) {
		testCases := []struct {
			pattern  string
			text     string
			expected bool
		}{
			{"*", "anything", true},
			{"temp_*", "temp_data", true},
			{"temp_*", "temp_", true},
			{"temp_*", "temporary", false},
			{"*_temp", "data_temp", true},
			{"*_temp", "_temp", true},
			{"*_temp", "temp", false},
			{"test*data", "testdata", true},
			{"test*data", "test_some_data", true},
			{"test*data", "testdata_extra", false},
			{"exact", "exact", true},
			{"exact", "not_exact", false},
		}

		for _, tc := range testCases {
			t.Run(tc.pattern+"_"+tc.text, func(t *testing.T) {
				result := MatchWildcard(tc.pattern, tc.text)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

// Mock database for testing
type MockDB struct {
	queries map[string][][]any
	errors  map[string]error
}

func NewMockDB() *MockDB {
	return &MockDB{
		queries: make(map[string][][]any),
		errors:  make(map[string]error),
	}
}

func (m *MockDB) SetQueryResult(query string, rows [][]any) {
	m.queries[query] = rows
}

func (m *MockDB) SetQueryError(query string, err error) {
	m.errors[query] = err
}

func (m *MockDB) Query(query string, args ...any) (*sql.Rows, error) {
	if err, exists := m.errors[query]; exists {
		return nil, err
	}

	// This is a simplified mock - in real tests we'd use a proper mock library
	// or testcontainers for integration tests
	return nil, ErrMockNotImplemented
}

func TestMockDatabase(t *testing.T) {
	t.Run("CreateMockDB", func(t *testing.T) {
		mockDB := NewMockDB()
		assert.NotZero(t, mockDB)

		// Set up mock data
		mockDB.SetQueryResult("SELECT id, email, name FROM users", [][]any{
			{"1", "john@example.com", "John Doe"},
			{"2", "jane@example.com", "Jane Smith"},
		})

		// Set up mock error
		mockDB.SetQueryError("SELECT id, email, name FROM nonexistent", ErrTableNotFound)

		// Verify mock setup
		assert.Equal(t, 2, len(mockDB.queries["SELECT id, email, name FROM users"]))
		assert.Equal(t, ErrTableNotFound, mockDB.errors["SELECT id, email, name FROM nonexistent"])
	})
}
