package pull

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestPostgreSQLExtractor(t *testing.T) {
	t.Run("CreatePostgreSQLExtractor", func(t *testing.T) {
		extractor := NewPostgreSQLExtractor()
		assert.NotZero(t, extractor)

		// Verify it implements the Extractor interface
		var _ Extractor = extractor
	})

	t.Run("GetDatabaseType", func(t *testing.T) {
		extractor := NewPostgreSQLExtractor()
		assert.Equal(t, "postgresql", extractor.GetDatabaseType())
	})

	t.Run("GetSystemSchemas", func(t *testing.T) {
		extractor := NewPostgreSQLExtractor()
		systemSchemas := extractor.GetSystemSchemas()

		expectedSchemas := []string{
			"information_schema",
			"pg_catalog",
			"pg_toast",
			"pg_temp_1",
			"pg_toast_temp_1",
		}
		assert.Equal(t, expectedSchemas, systemSchemas)
	})
}

func TestPostgreSQLQueries(t *testing.T) {
	extractor := NewPostgreSQLExtractor()

	t.Run("BuildSchemasQuery", func(t *testing.T) {
		query := extractor.BuildSchemasQuery()
		assert.Contains(t, query, "information_schema.schemata")
		assert.Contains(t, query, "schema_name")
		assert.Contains(t, query, "NOT IN")
		assert.Contains(t, query, "information_schema")
		assert.Contains(t, query, "pg_catalog")
	})

	t.Run("BuildTablesQuery", func(t *testing.T) {
		query := extractor.BuildTablesQuery("public")
		assert.Contains(t, query, "information_schema.tables")
		assert.Contains(t, query, "table_name")
		assert.Contains(t, query, "obj_description")
		assert.Contains(t, query, "public")
	})

	t.Run("BuildColumnsQuery", func(t *testing.T) {
		query := extractor.BuildColumnsQuery("public", "users")
		assert.Contains(t, query, "information_schema.columns")
		assert.Contains(t, query, "column_name")
		assert.Contains(t, query, "data_type")
		assert.Contains(t, query, "is_nullable")
		assert.Contains(t, query, "column_default")
		assert.Contains(t, query, "table_schema")
		assert.Contains(t, query, "table_name")
		assert.Contains(t, query, "ordinal_position")
	})

	t.Run("BuildConstraintsQuery", func(t *testing.T) {
		query := extractor.BuildConstraintsQuery("public", "users")
		assert.Contains(t, query, "information_schema.table_constraints")
		assert.Contains(t, query, "information_schema.key_column_usage")
		assert.Contains(t, query, "constraint_name")
		assert.Contains(t, query, "constraint_type")
		assert.Contains(t, query, "column_name")
	})

	t.Run("BuildIndexesQuery", func(t *testing.T) {
		query := extractor.BuildIndexesQuery("public", "users")
		assert.Contains(t, query, "pg_class")
		assert.Contains(t, query, "pg_index")
		assert.Contains(t, query, "index_name")
		assert.Contains(t, query, "indisunique")
		assert.Contains(t, query, "public")
		assert.Contains(t, query, "users")
	})

	t.Run("BuildViewsQuery", func(t *testing.T) {
		query := extractor.BuildViewsQuery("public")
		assert.Contains(t, query, "information_schema.views")
		assert.Contains(t, query, "table_name")
		assert.Contains(t, query, "view_definition")
		assert.Contains(t, query, "table_schema")
	})

	t.Run("BuildDatabaseInfoQuery", func(t *testing.T) {
		query := extractor.BuildDatabaseInfoQuery()
		assert.Contains(t, query, "version()")
		assert.Contains(t, query, "current_database()")
	})
}

func TestPostgreSQLTypeMapping(t *testing.T) {
	extractor := NewPostgreSQLExtractor()

	t.Run("MapPostgreSQLTypes", func(t *testing.T) {
		testCases := []struct {
			pgType       string
			expectedType string
		}{
			{"integer", TypeInt},
			{"bigint", TypeInt},
			{"character varying(255)", TypeString},
			{"text", TypeString},
			{"boolean", TypeBool},
			{"timestamp with time zone", TypeDateTime},
			{"date", TypeDate},
			{"json", TypeJSON},
			{"jsonb", TypeJSON},
			{"bytea", TypeBinary},
			{"integer[]", TypeArray},
			{"text[]", TypeArray},
		}

		for _, tc := range testCases {
			result := extractor.MapColumnType(tc.pgType)
			assert.Equal(t, tc.expectedType, result, "Failed to map PostgreSQL type: %s", tc.pgType)
		}
	})
}

func TestPostgreSQLConstraintParsing(t *testing.T) {
	extractor := NewPostgreSQLExtractor()
	testConstraintParsing(t, extractor, "UNKNOWN")
}

func TestPostgreSQLIndexParsing(t *testing.T) {
	extractor := NewPostgreSQLExtractor()

	t.Run("ParseUniqueIndex", func(t *testing.T) {
		indexDef := "CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email)"
		isUnique := extractor.ParseIndexUnique(indexDef)
		assert.True(t, isUnique)
	})

	t.Run("ParseNonUniqueIndex", func(t *testing.T) {
		indexDef := "CREATE INDEX idx_users_created_at ON public.users USING btree (created_at)"
		isUnique := extractor.ParseIndexUnique(indexDef)
		assert.False(t, isUnique)
	})

	t.Run("ParseIndexType", func(t *testing.T) {
		testCases := []struct {
			indexDef     string
			expectedType string
		}{
			{"CREATE INDEX idx_test ON table USING btree (col)", "BTREE"},
			{"CREATE INDEX idx_test ON table USING hash (col)", "HASH"},
			{"CREATE INDEX idx_test ON table USING gin (col)", "GIN"},
			{"CREATE INDEX idx_test ON table USING gist (col)", "GIST"},
			{"CREATE INDEX idx_test ON table (col)", "CREATE INDEX IDX_TEST ON TABLE (COL)"}, // Default - no USING clause
		}

		for _, tc := range testCases {
			result := extractor.ParseIndexType(tc.indexDef)
			assert.Equal(t, tc.expectedType, result, "Failed to parse index type from: %s", tc.indexDef)
		}
	})

	t.Run("ParseIndexColumns", func(t *testing.T) {
		testCases := []struct {
			indexDef        string
			expectedColumns []string
		}{
			{
				"CREATE INDEX idx_test ON table (col1)",
				[]string{"col1"},
			},
			{
				"CREATE INDEX idx_test ON table (col1, col2)",
				[]string{"col1", "col2"},
			},
			{
				"CREATE INDEX idx_test ON table USING btree (col1, col2 DESC)",
				[]string{"col1", "col2 DESC"},
			},
			{
				"CREATE UNIQUE INDEX idx_test ON table (email)",
				[]string{"email"},
			},
		}

		for _, tc := range testCases {
			result := extractor.ParseIndexColumns(tc.indexDef)
			assert.Equal(t, tc.expectedColumns, result, "Failed to parse index columns from: %s", tc.indexDef)
		}
	})
}

func TestPostgreSQLDefaultValues(t *testing.T) {
	extractor := NewPostgreSQLExtractor()

	t.Run("ParseDefaultValues", func(t *testing.T) {
		testCases := []struct {
			pgDefault       string
			expectedDefault string
		}{
			{"nextval('users_id_seq'::regclass)", "AUTO_INCREMENT"},
			{"now()", "now()"},
			{"'default_value'::character varying", "default_value"},
			{"true", "true"},
			{"false", "false"},
			{"0", "0"},
			{"NULL", ""},
			{"", ""},
		}

		for _, tc := range testCases {
			result := extractor.ParseDefaultValue(tc.pgDefault)
			assert.Equal(t, tc.expectedDefault, result, "Failed to parse default value: %s", tc.pgDefault)
		}
	})
}

func TestPostgreSQLSystemSchemaFiltering(t *testing.T) {
	extractor := NewPostgreSQLExtractor()

	t.Run("FilterSystemSchemas", func(t *testing.T) {
		allSchemas := []string{
			"public",
			"auth",
			"information_schema",
			"pg_catalog",
			"pg_toast",
			"pg_temp_1",
		}

		config := ExtractConfig{}
		filtered := extractor.FilterSystemSchemas(allSchemas, config)

		// Should exclude system schemas by default (including pg_temp_*)
		expected := []string{"public", "auth"}
		assert.Equal(t, expected, filtered)
	})

	t.Run("FilterSystemSchemasWithCustomExcludes", func(t *testing.T) {
		allSchemas := []string{
			"public",
			"auth",
			"test",
			"information_schema",
		}

		config := ExtractConfig{
			ExcludeSchemas: []string{"test"},
		}
		filtered := extractor.FilterSystemSchemas(allSchemas, config)

		// Should exclude both system schemas and custom excludes
		expected := []string{"public", "auth"}
		assert.Equal(t, expected, filtered)
	})
}

func TestPostgreSQLErrorHandling(t *testing.T) {
	extractor := NewPostgreSQLExtractor()

	t.Run("HandleDatabaseConnectionError", func(t *testing.T) {
		// Test error handling for database connection issues
		mockErr := errors.New("connection refused")
		err := extractor.HandleDatabaseError(mockErr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PostgreSQL connection refused")
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("HandleQueryExecutionError", func(t *testing.T) {
		// Test error handling for query execution issues
		mockErr := errors.New("syntax error")
		err := extractor.HandleDatabaseError(mockErr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "syntax error")
	})

	t.Run("HandleSchemaNotFoundError", func(t *testing.T) {
		// Test error handling for schema not found
		mockErr := errors.New("database does not exist")
		err := extractor.HandleDatabaseError(mockErr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PostgreSQL database does not exist")
		assert.Contains(t, err.Error(), "does not exist")
	})
}
