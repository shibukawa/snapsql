package pull

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	snapsql "github.com/shibukawa/snapsql"
)

func TestPullConfig(t *testing.T) {
	t.Run("CreateBasicPullConfig", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:  "postgres://user:pass@localhost/testdb",
			DatabaseType: "postgresql",
			OutputPath:   ".snapsql/schema",
			SchemaAware:  true,
		}

		assert.Equal(t, "postgres://user:pass@localhost/testdb", config.DatabaseURL)
		assert.Equal(t, "postgresql", config.DatabaseType)
		assert.Equal(t, ".snapsql/schema", config.OutputPath)
		assert.Equal(t, true, config.SchemaAware)
	})

	t.Run("CreatePullConfigWithFilters", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:    "mysql://user:pass@localhost/testdb",
			DatabaseType:   "mysql",
			OutputPath:     ".snapsql/schema",
			SchemaAware:    true,
			IncludeSchemas: []string{"public", "auth"},
			ExcludeSchemas: []string{"information_schema", "mysql"},
			IncludeTables:  []string{"users", "posts", "comments"},
			ExcludeTables:  []string{"migrations", "temp_*"},
			IncludeViews:   true,
			IncludeIndexes: true,
		}

		assert.Equal(t, "mysql", config.DatabaseType)
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
			SchemaAware:  false, // SQLite uses global schema
		}

		assert.Equal(t, "sqlite:///path/to/database.db", config.DatabaseURL)
		assert.Equal(t, "sqlite", config.DatabaseType)
		assert.Equal(t, false, config.SchemaAware)
	})
}

func TestPullResult(t *testing.T) {
	t.Run("CreateBasicPullResult", func(t *testing.T) {
		result := struct {
			Schemas      []snapsql.DatabaseSchema
			DatabaseInfo snapsql.DatabaseInfo
			Errors       []error
		}{
			Schemas: []snapsql.DatabaseSchema{
				createTestDatabaseSchema(),
			},
			DatabaseInfo: snapsql.DatabaseInfo{Type: "postgresql", Version: "14.2", Name: "testdb", Charset: "UTF8"},
			Errors:       []error{},
		}

		assert.Equal(t, 1, len(result.Schemas))
		assert.Equal(t, "public", result.Schemas[0].Name)
		assert.Equal(t, 2, len(result.Schemas[0].Tables))
		assert.Equal(t, "postgresql", result.DatabaseInfo.Type)
		assert.Equal(t, 0, len(result.Errors))
	})

	t.Run("CreatePullResultWithErrors", func(t *testing.T) {
		result := struct {
			Schemas      []snapsql.DatabaseSchema
			DatabaseInfo snapsql.DatabaseInfo
			Errors       []error
		}{
			Schemas: []snapsql.DatabaseSchema{},
			DatabaseInfo: snapsql.DatabaseInfo{
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
		result := struct {
			Schemas      []snapsql.DatabaseSchema
			DatabaseInfo snapsql.DatabaseInfo
		}{
			Schemas: []snapsql.DatabaseSchema{
				{
					Name: "public",
					Tables: []*snapsql.TableInfo{
						&snapsql.TableInfo{Name: "users", Schema: "public", Columns: map[string]*snapsql.ColumnInfo{}},
						&snapsql.TableInfo{Name: "posts", Schema: "public", Columns: map[string]*snapsql.ColumnInfo{}},
					},
				},
				{
					Name: "auth",
					Tables: []*snapsql.TableInfo{
						&snapsql.TableInfo{Name: "sessions", Schema: "auth", Columns: map[string]*snapsql.ColumnInfo{}},
						&snapsql.TableInfo{Name: "permissions", Schema: "auth", Columns: map[string]*snapsql.ColumnInfo{}},
					},
				},
			},
			DatabaseInfo: snapsql.DatabaseInfo{Type: "postgresql"},
		}

		assert.Equal(t, 2, len(result.Schemas))
		assert.Equal(t, "public", result.Schemas[0].Name)
		assert.Equal(t, "auth", result.Schemas[1].Name)
		assert.Equal(t, 2, len(result.Schemas[0].Tables))
		assert.Equal(t, 2, len(result.Schemas[1].Tables))
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

// テストデータ生成ヘルパー
func createTestTableInfo() *snapsql.TableInfo {
	columns := map[string]*snapsql.ColumnInfo{
		"id": {
			Name:         "id",
			DataType:     "int",
			Nullable:     false,
			IsPrimaryKey: true,
		},
		"email": {
			Name:     "email",
			DataType: "string",
			Nullable: false,
		},
		"created_at": {
			Name:         "created_at",
			DataType:     "datetime",
			Nullable:     false,
			DefaultValue: "now()",
		},
		"updated_at": {
			Name:     "updated_at",
			DataType: "datetime",
			Nullable: true,
		},
	}

	return &snapsql.TableInfo{
		Name:    "users",
		Schema:  "public",
		Columns: columns,
		Constraints: []snapsql.ConstraintInfo{
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
		Indexes: []snapsql.IndexInfo{
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

func createTestDatabaseSchema() snapsql.DatabaseSchema {
	tables := []*snapsql.TableInfo{
		{
			Name:   "posts",
			Schema: "public",
			Columns: map[string]*snapsql.ColumnInfo{
				"id": {
					Name:         "id",
					DataType:     "int",
					Nullable:     false,
					IsPrimaryKey: true,
				},
				"title": {
					Name:     "title",
					DataType: "string",
					Nullable: false,
				},
				"user_id": {
					Name:     "user_id",
					DataType: "int",
					Nullable: false,
				},
			},
			Constraints: []snapsql.ConstraintInfo{
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
		createTestTableInfo(),
	}

	return snapsql.DatabaseSchema{
		Name:   "public",
		Tables: tables,
		Views: []*snapsql.ViewInfo{
			{
				Name:       "active_users",
				Schema:     "public",
				Definition: "SELECT id, email FROM users WHERE created_at > now() - interval '30 days'",
				Comment:    "Users active in the last 30 days",
			},
		},
		DatabaseInfo: snapsql.DatabaseInfo{Type: "postgresql", Version: "14.2", Name: "testdb", Charset: "UTF8"},
	}
}
