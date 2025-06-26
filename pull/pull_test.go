package pull

import (
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func TestPullConfig(t *testing.T) {
	t.Run("CreateBasicPullConfig", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:  "postgres://user:pass@localhost/testdb",
			DatabaseType: "postgresql",
			OutputPath:   ".snapsql/schema",
			OutputFormat: OutputPerTable,
			SchemaAware:  true,
		}

		assert.Equal(t, "postgres://user:pass@localhost/testdb", config.DatabaseURL)
		assert.Equal(t, "postgresql", config.DatabaseType)
		assert.Equal(t, ".snapsql/schema", config.OutputPath)
		assert.Equal(t, OutputPerTable, config.OutputFormat)
		assert.Equal(t, true, config.SchemaAware)
	})

	t.Run("CreatePullConfigWithFilters", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:    "mysql://user:pass@localhost/testdb",
			DatabaseType:   "mysql",
			OutputPath:     ".snapsql/schema",
			OutputFormat:   OutputSingleFile,
			SchemaAware:    true,
			IncludeSchemas: []string{"public", "auth"},
			ExcludeSchemas: []string{"information_schema", "mysql"},
			IncludeTables:  []string{"users", "posts", "comments"},
			ExcludeTables:  []string{"migrations", "temp_*"},
			IncludeViews:   true,
			IncludeIndexes: true,
		}

		assert.Equal(t, "mysql", config.DatabaseType)
		assert.Equal(t, OutputSingleFile, config.OutputFormat)
		assert.Equal(t, []string{"public", "auth"}, config.IncludeSchemas)
		assert.Equal(t, []string{"information_schema", "mysql"}, config.ExcludeSchemas)
		assert.Equal(t, []string{"users", "posts", "comments"}, config.IncludeTables)
		assert.Equal(t, []string{"migrations", "temp_*"}, config.ExcludeTables)
		assert.Equal(t, true, config.IncludeViews)
		assert.Equal(t, true, config.IncludeIndexes)
	})

	t.Run("CreateSQLitePullConfig", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:  "sqlite:///path/to/database.db",
			DatabaseType: "sqlite",
			OutputPath:   ".snapsql/schema",
			OutputFormat: OutputPerSchema,
			SchemaAware:  false, // SQLite uses global schema
		}

		assert.Equal(t, "sqlite:///path/to/database.db", config.DatabaseURL)
		assert.Equal(t, "sqlite", config.DatabaseType)
		assert.Equal(t, OutputPerSchema, config.OutputFormat)
		assert.Equal(t, false, config.SchemaAware)
	})
}

func TestPullResult(t *testing.T) {
	t.Run("CreateBasicPullResult", func(t *testing.T) {
		extractedAt := time.Now()
		result := PullResult{
			Schemas: []DatabaseSchema{
				{
					Name: "public",
					Tables: []TableSchema{
						{
							Name:   "users",
							Schema: "public",
							Columns: []ColumnSchema{
								{
									Name:         "id",
									Type:         "integer",
									SnapSQLType:  "int",
									Nullable:     false,
									IsPrimaryKey: true,
								},
							},
						},
					},
					ExtractedAt: extractedAt,
				},
			},
			ExtractedAt: extractedAt,
			DatabaseInfo: DatabaseInfo{
				Type:    "postgresql",
				Version: "14.2",
				Name:    "testdb",
				Charset: "UTF8",
			},
			Errors: []error{},
		}

		assert.Equal(t, 1, len(result.Schemas))
		assert.Equal(t, "public", result.Schemas[0].Name)
		assert.Equal(t, 1, len(result.Schemas[0].Tables))
		assert.Equal(t, "users", result.Schemas[0].Tables[0].Name)
		assert.Equal(t, extractedAt, result.ExtractedAt)
		assert.Equal(t, "postgresql", result.DatabaseInfo.Type)
		assert.Equal(t, 0, len(result.Errors))
	})

	t.Run("CreatePullResultWithErrors", func(t *testing.T) {
		result := PullResult{
			Schemas:     []DatabaseSchema{},
			ExtractedAt: time.Now(),
			DatabaseInfo: DatabaseInfo{
				Type: "postgresql",
			},
			Errors: []error{
				ErrConnectionFailed,
				ErrTableNotFound,
			},
		}

		assert.Equal(t, 0, len(result.Schemas))
		assert.Equal(t, 2, len(result.Errors))
		assert.Equal(t, ErrConnectionFailed, result.Errors[0])
		assert.Equal(t, ErrTableNotFound, result.Errors[1])
	})

	t.Run("CreateMultiSchemaPullResult", func(t *testing.T) {
		result := PullResult{
			Schemas: []DatabaseSchema{
				{
					Name: "public",
					Tables: []TableSchema{
						{Name: "users", Schema: "public"},
						{Name: "posts", Schema: "public"},
					},
				},
				{
					Name: "auth",
					Tables: []TableSchema{
						{Name: "sessions", Schema: "auth"},
						{Name: "permissions", Schema: "auth"},
					},
				},
			},
			ExtractedAt: time.Now(),
			DatabaseInfo: DatabaseInfo{
				Type: "postgresql",
			},
		}

		assert.Equal(t, 2, len(result.Schemas))
		assert.Equal(t, "public", result.Schemas[0].Name)
		assert.Equal(t, "auth", result.Schemas[1].Name)
		assert.Equal(t, 2, len(result.Schemas[0].Tables))
		assert.Equal(t, 2, len(result.Schemas[1].Tables))
	})
}

func TestOutputFormat(t *testing.T) {
	t.Run("OutputFormatConstants", func(t *testing.T) {
		assert.Equal(t, OutputFormat("single"), OutputSingleFile)
		assert.Equal(t, OutputFormat("per_table"), OutputPerTable)
		assert.Equal(t, OutputFormat("per_schema"), OutputPerSchema)
	})

	t.Run("OutputFormatString", func(t *testing.T) {
		assert.Equal(t, "single", string(OutputSingleFile))
		assert.Equal(t, "per_table", string(OutputPerTable))
		assert.Equal(t, "per_schema", string(OutputPerSchema))
	})
}

func TestExtractConfig(t *testing.T) {
	t.Run("CreateBasicExtractConfig", func(t *testing.T) {
		config := ExtractConfig{
			IncludeViews:   true,
			IncludeIndexes: true,
		}

		assert.Equal(t, true, config.IncludeViews)
		assert.Equal(t, true, config.IncludeIndexes)
		assert.Equal(t, 0, len(config.IncludeSchemas))
		assert.Equal(t, 0, len(config.ExcludeSchemas))
	})

	t.Run("CreateExtractConfigWithSchemaFilters", func(t *testing.T) {
		config := ExtractConfig{
			IncludeSchemas: []string{"public", "auth"},
			ExcludeSchemas: []string{"information_schema", "pg_catalog"},
			IncludeTables:  []string{"users", "posts"},
			ExcludeTables:  []string{"migrations", "temp_*"},
			IncludeViews:   false,
			IncludeIndexes: true,
		}

		assert.Equal(t, []string{"public", "auth"}, config.IncludeSchemas)
		assert.Equal(t, []string{"information_schema", "pg_catalog"}, config.ExcludeSchemas)
		assert.Equal(t, []string{"users", "posts"}, config.IncludeTables)
		assert.Equal(t, []string{"migrations", "temp_*"}, config.ExcludeTables)
		assert.Equal(t, false, config.IncludeViews)
		assert.Equal(t, true, config.IncludeIndexes)
	})

	t.Run("CreateExtractConfigForSQLite", func(t *testing.T) {
		config := ExtractConfig{
			// SQLite doesn't have schemas, so schema filters should be empty
			IncludeSchemas: []string{},
			ExcludeSchemas: []string{},
			IncludeTables:  []string{"users", "posts", "comments"},
			ExcludeTables:  []string{"sqlite_*"},
			IncludeViews:   true,
			IncludeIndexes: true,
		}

		assert.Equal(t, 0, len(config.IncludeSchemas))
		assert.Equal(t, 0, len(config.ExcludeSchemas))
		assert.Equal(t, []string{"users", "posts", "comments"}, config.IncludeTables)
		assert.Equal(t, []string{"sqlite_*"}, config.ExcludeTables)
	})
}

// Test helper functions for creating test data
func TestCreateTestData(t *testing.T) {
	t.Run("CreateCompleteTableSchema", func(t *testing.T) {
		table := createTestTableSchema()

		assert.Equal(t, "users", table.Name)
		assert.Equal(t, "public", table.Schema)
		assert.Equal(t, 4, len(table.Columns))
		assert.Equal(t, 3, len(table.Constraints))
		assert.Equal(t, 2, len(table.Indexes))
	})

	t.Run("CreateCompleteSchemaWithMultipleTables", func(t *testing.T) {
		schema := createTestDatabaseSchema()

		assert.Equal(t, "public", schema.Name)
		assert.Equal(t, 2, len(schema.Tables))
		assert.Equal(t, "users", schema.Tables[0].Name)
		assert.Equal(t, "posts", schema.Tables[1].Name)
		assert.Equal(t, "postgresql", schema.DatabaseInfo.Type)
	})
}

// Helper functions for creating test data
func createTestTableSchema() TableSchema {
	return TableSchema{
		Name:   "users",
		Schema: "public",
		Columns: []ColumnSchema{
			{
				Name:         "id",
				Type:         "integer",
				SnapSQLType:  "int",
				Nullable:     false,
				IsPrimaryKey: true,
			},
			{
				Name:        "email",
				Type:        "character varying(255)",
				SnapSQLType: "string",
				Nullable:    false,
			},
			{
				Name:         "created_at",
				Type:         "timestamp with time zone",
				SnapSQLType:  "datetime",
				Nullable:     false,
				DefaultValue: "now()",
			},
			{
				Name:        "updated_at",
				Type:        "timestamp with time zone",
				SnapSQLType: "datetime",
				Nullable:    true,
			},
		},
		Constraints: []ConstraintSchema{
			{
				Name:    "users_pkey",
				Type:    "PRIMARY_KEY",
				Columns: []string{"id"},
			},
			{
				Name:    "users_email_unique",
				Type:    "UNIQUE",
				Columns: []string{"email"},
			},
			{
				Name:       "users_email_check",
				Type:       "CHECK",
				Columns:    []string{"email"},
				Definition: "email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$'",
			},
		},
		Indexes: []IndexSchema{
			{
				Name:     "idx_users_email",
				Columns:  []string{"email"},
				IsUnique: true,
				Type:     "btree",
			},
			{
				Name:     "idx_users_created_at",
				Columns:  []string{"created_at"},
				IsUnique: false,
				Type:     "btree",
			},
		},
		Comment: "User accounts table",
	}
}

func createTestDatabaseSchema() DatabaseSchema {
	return DatabaseSchema{
		Name: "public",
		Tables: []TableSchema{
			createTestTableSchema(),
			{
				Name:   "posts",
				Schema: "public",
				Columns: []ColumnSchema{
					{
						Name:         "id",
						Type:         "integer",
						SnapSQLType:  "int",
						Nullable:     false,
						IsPrimaryKey: true,
					},
					{
						Name:        "title",
						Type:        "character varying(255)",
						SnapSQLType: "string",
						Nullable:    false,
					},
					{
						Name:        "user_id",
						Type:        "integer",
						SnapSQLType: "int",
						Nullable:    false,
					},
				},
				Constraints: []ConstraintSchema{
					{
						Name:    "posts_pkey",
						Type:    "PRIMARY_KEY",
						Columns: []string{"id"},
					},
					{
						Name:              "posts_user_id_fkey",
						Type:              "FOREIGN_KEY",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
		Views: []ViewSchema{
			{
				Name:       "active_users",
				Schema:     "public",
				Definition: "SELECT id, email FROM users WHERE created_at > now() - interval '30 days'",
				Comment:    "Users active in the last 30 days",
			},
		},
		ExtractedAt: time.Now(),
		DatabaseInfo: DatabaseInfo{
			Type:    "postgresql",
			Version: "14.2",
			Name:    "testdb",
			Charset: "UTF8",
		},
	}
}
