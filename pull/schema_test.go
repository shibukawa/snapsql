package pull

import (
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func TestDatabaseSchema(t *testing.T) {
	t.Run("CreateBasicDatabaseSchema", func(t *testing.T) {
		schema := DatabaseSchema{
			Name:        "public",
			ExtractedAt: time.Now(),
			DatabaseInfo: DatabaseInfo{
				Type:    "postgresql",
				Version: "14.2",
				Name:    "testdb",
				Charset: "UTF8",
			},
		}

		assert.Equal(t, "public", schema.Name)
		assert.Equal(t, "postgresql", schema.DatabaseInfo.Type)
		assert.NotZero(t, schema.ExtractedAt)
	})

	t.Run("AddTableToSchema", func(t *testing.T) {
		schema := DatabaseSchema{
			Name:   "public",
			Tables: []TableSchema{},
		}

		table := TableSchema{
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
			},
		}

		schema.Tables = append(schema.Tables, table)

		assert.Equal(t, 1, len(schema.Tables))
		assert.Equal(t, "users", schema.Tables[0].Name)
		assert.Equal(t, 2, len(schema.Tables[0].Columns))
	})
}

func TestTableSchema(t *testing.T) {
	t.Run("CreateTableWithColumns", func(t *testing.T) {
		table := TableSchema{
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
					Name:         "email",
					Type:         "character varying(255)",
					SnapSQLType:  "string",
					Nullable:     false,
					DefaultValue: "",
				},
				{
					Name:         "created_at",
					Type:         "timestamp with time zone",
					SnapSQLType:  "datetime",
					Nullable:     false,
					DefaultValue: "now()",
				},
			},
		}

		assert.Equal(t, "users", table.Name)
		assert.Equal(t, "public", table.Schema)
		assert.Equal(t, 3, len(table.Columns))

		// Check primary key column
		idColumn := table.Columns[0]
		assert.Equal(t, "id", idColumn.Name)
		assert.Equal(t, true, idColumn.IsPrimaryKey)
		assert.Equal(t, false, idColumn.Nullable)

		// Check string column
		emailColumn := table.Columns[1]
		assert.Equal(t, "email", emailColumn.Name)
		assert.Equal(t, "string", emailColumn.SnapSQLType)
		assert.Equal(t, false, emailColumn.IsPrimaryKey)

		// Check datetime column with default
		createdAtColumn := table.Columns[2]
		assert.Equal(t, "created_at", createdAtColumn.Name)
		assert.Equal(t, "datetime", createdAtColumn.SnapSQLType)
		assert.Equal(t, "now()", createdAtColumn.DefaultValue)
	})

	t.Run("AddConstraintsToTable", func(t *testing.T) {
		table := TableSchema{
			Name:   "users",
			Schema: "public",
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
					Name:              "users_profile_id_fkey",
					Type:              "FOREIGN_KEY",
					Columns:           []string{"profile_id"},
					ReferencedTable:   "profiles",
					ReferencedColumns: []string{"id"},
				},
			},
		}

		assert.Equal(t, 3, len(table.Constraints))

		// Check primary key constraint
		pkConstraint := table.Constraints[0]
		assert.Equal(t, "users_pkey", pkConstraint.Name)
		assert.Equal(t, "PRIMARY_KEY", pkConstraint.Type)
		assert.Equal(t, []string{"id"}, pkConstraint.Columns)

		// Check unique constraint
		uniqueConstraint := table.Constraints[1]
		assert.Equal(t, "users_email_unique", uniqueConstraint.Name)
		assert.Equal(t, "UNIQUE", uniqueConstraint.Type)
		assert.Equal(t, []string{"email"}, uniqueConstraint.Columns)

		// Check foreign key constraint
		fkConstraint := table.Constraints[2]
		assert.Equal(t, "users_profile_id_fkey", fkConstraint.Name)
		assert.Equal(t, "FOREIGN_KEY", fkConstraint.Type)
		assert.Equal(t, []string{"profile_id"}, fkConstraint.Columns)
		assert.Equal(t, "profiles", fkConstraint.ReferencedTable)
		assert.Equal(t, []string{"id"}, fkConstraint.ReferencedColumns)
	})

	t.Run("AddIndexesToTable", func(t *testing.T) {
		table := TableSchema{
			Name:   "users",
			Schema: "public",
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
				{
					Name:     "idx_users_name_email",
					Columns:  []string{"name", "email"},
					IsUnique: false,
					Type:     "btree",
				},
			},
		}

		assert.Equal(t, 3, len(table.Indexes))

		// Check unique index
		uniqueIndex := table.Indexes[0]
		assert.Equal(t, "idx_users_email", uniqueIndex.Name)
		assert.Equal(t, []string{"email"}, uniqueIndex.Columns)
		assert.Equal(t, true, uniqueIndex.IsUnique)
		assert.Equal(t, "btree", uniqueIndex.Type)

		// Check non-unique index
		nonUniqueIndex := table.Indexes[1]
		assert.Equal(t, "idx_users_created_at", nonUniqueIndex.Name)
		assert.Equal(t, []string{"created_at"}, nonUniqueIndex.Columns)
		assert.Equal(t, false, nonUniqueIndex.IsUnique)

		// Check composite index
		compositeIndex := table.Indexes[2]
		assert.Equal(t, "idx_users_name_email", compositeIndex.Name)
		assert.Equal(t, []string{"name", "email"}, compositeIndex.Columns)
		assert.Equal(t, false, compositeIndex.IsUnique)
	})
}

func TestColumnSchema(t *testing.T) {
	t.Run("CreateBasicColumn", func(t *testing.T) {
		column := ColumnSchema{
			Name:        "id",
			Type:        "integer",
			SnapSQLType: "int",
			Nullable:    false,
		}

		assert.Equal(t, "id", column.Name)
		assert.Equal(t, "integer", column.Type)
		assert.Equal(t, "int", column.SnapSQLType)
		assert.Equal(t, false, column.Nullable)
	})

	t.Run("CreateColumnWithDefault", func(t *testing.T) {
		column := ColumnSchema{
			Name:         "created_at",
			Type:         "timestamp with time zone",
			SnapSQLType:  "datetime",
			Nullable:     false,
			DefaultValue: "now()",
		}

		assert.Equal(t, "created_at", column.Name)
		assert.Equal(t, "timestamp with time zone", column.Type)
		assert.Equal(t, "datetime", column.SnapSQLType)
		assert.Equal(t, "now()", column.DefaultValue)
	})

	t.Run("CreateNullableColumn", func(t *testing.T) {
		column := ColumnSchema{
			Name:        "description",
			Type:        "text",
			SnapSQLType: "string",
			Nullable:    true,
		}

		assert.Equal(t, "description", column.Name)
		assert.Equal(t, true, column.Nullable)
	})

	t.Run("CreatePrimaryKeyColumn", func(t *testing.T) {
		column := ColumnSchema{
			Name:         "id",
			Type:         "integer",
			SnapSQLType:  "int",
			Nullable:     false,
			IsPrimaryKey: true,
		}

		assert.Equal(t, "id", column.Name)
		assert.Equal(t, true, column.IsPrimaryKey)
		assert.Equal(t, false, column.Nullable)
	})

	t.Run("CreateColumnWithComment", func(t *testing.T) {
		column := ColumnSchema{
			Name:        "user_id",
			Type:        "integer",
			SnapSQLType: "int",
			Nullable:    false,
			Comment:     "Foreign key to users table",
		}

		assert.Equal(t, "user_id", column.Name)
		assert.Equal(t, "Foreign key to users table", column.Comment)
	})
}

func TestConstraintSchema(t *testing.T) {
	t.Run("CreatePrimaryKeyConstraint", func(t *testing.T) {
		constraint := ConstraintSchema{
			Name:    "users_pkey",
			Type:    "PRIMARY_KEY",
			Columns: []string{"id"},
		}

		assert.Equal(t, "users_pkey", constraint.Name)
		assert.Equal(t, "PRIMARY_KEY", constraint.Type)
		assert.Equal(t, []string{"id"}, constraint.Columns)
	})

	t.Run("CreateForeignKeyConstraint", func(t *testing.T) {
		constraint := ConstraintSchema{
			Name:              "posts_user_id_fkey",
			Type:              "FOREIGN_KEY",
			Columns:           []string{"user_id"},
			ReferencedTable:   "users",
			ReferencedColumns: []string{"id"},
		}

		assert.Equal(t, "posts_user_id_fkey", constraint.Name)
		assert.Equal(t, "FOREIGN_KEY", constraint.Type)
		assert.Equal(t, []string{"user_id"}, constraint.Columns)
		assert.Equal(t, "users", constraint.ReferencedTable)
		assert.Equal(t, []string{"id"}, constraint.ReferencedColumns)
	})

	t.Run("CreateUniqueConstraint", func(t *testing.T) {
		constraint := ConstraintSchema{
			Name:    "users_email_unique",
			Type:    "UNIQUE",
			Columns: []string{"email"},
		}

		assert.Equal(t, "users_email_unique", constraint.Name)
		assert.Equal(t, "UNIQUE", constraint.Type)
		assert.Equal(t, []string{"email"}, constraint.Columns)
	})

	t.Run("CreateCheckConstraint", func(t *testing.T) {
		constraint := ConstraintSchema{
			Name:       "users_age_check",
			Type:       "CHECK",
			Columns:    []string{"age"},
			Definition: "age >= 0 AND age <= 150",
		}

		assert.Equal(t, "users_age_check", constraint.Name)
		assert.Equal(t, "CHECK", constraint.Type)
		assert.Equal(t, []string{"age"}, constraint.Columns)
		assert.Equal(t, "age >= 0 AND age <= 150", constraint.Definition)
	})

	t.Run("CreateCompositeUniqueConstraint", func(t *testing.T) {
		constraint := ConstraintSchema{
			Name:    "users_name_email_unique",
			Type:    "UNIQUE",
			Columns: []string{"name", "email"},
		}

		assert.Equal(t, "users_name_email_unique", constraint.Name)
		assert.Equal(t, "UNIQUE", constraint.Type)
		assert.Equal(t, []string{"name", "email"}, constraint.Columns)
	})
}

func TestIndexSchema(t *testing.T) {
	t.Run("CreateBasicIndex", func(t *testing.T) {
		index := IndexSchema{
			Name:     "idx_users_email",
			Columns:  []string{"email"},
			IsUnique: false,
			Type:     "btree",
		}

		assert.Equal(t, "idx_users_email", index.Name)
		assert.Equal(t, []string{"email"}, index.Columns)
		assert.Equal(t, false, index.IsUnique)
		assert.Equal(t, "btree", index.Type)
	})

	t.Run("CreateUniqueIndex", func(t *testing.T) {
		index := IndexSchema{
			Name:     "idx_users_username_unique",
			Columns:  []string{"username"},
			IsUnique: true,
			Type:     "btree",
		}

		assert.Equal(t, "idx_users_username_unique", index.Name)
		assert.Equal(t, []string{"username"}, index.Columns)
		assert.Equal(t, true, index.IsUnique)
	})

	t.Run("CreateCompositeIndex", func(t *testing.T) {
		index := IndexSchema{
			Name:     "idx_users_name_created_at",
			Columns:  []string{"name", "created_at"},
			IsUnique: false,
			Type:     "btree",
		}

		assert.Equal(t, "idx_users_name_created_at", index.Name)
		assert.Equal(t, []string{"name", "created_at"}, index.Columns)
		assert.Equal(t, false, index.IsUnique)
	})

	t.Run("CreateHashIndex", func(t *testing.T) {
		index := IndexSchema{
			Name:     "idx_users_email_hash",
			Columns:  []string{"email"},
			IsUnique: false,
			Type:     "hash",
		}

		assert.Equal(t, "idx_users_email_hash", index.Name)
		assert.Equal(t, "hash", index.Type)
	})
}

func TestViewSchema(t *testing.T) {
	t.Run("CreateBasicView", func(t *testing.T) {
		view := ViewSchema{
			Name:       "active_users",
			Schema:     "public",
			Definition: "SELECT id, name, email FROM users WHERE active = true",
		}

		assert.Equal(t, "active_users", view.Name)
		assert.Equal(t, "public", view.Schema)
		assert.Equal(t, "SELECT id, name, email FROM users WHERE active = true", view.Definition)
	})

	t.Run("CreateViewWithComment", func(t *testing.T) {
		view := ViewSchema{
			Name:       "user_stats",
			Schema:     "public",
			Definition: "SELECT user_id, COUNT(*) as post_count FROM posts GROUP BY user_id",
			Comment:    "Statistics view for user post counts",
		}

		assert.Equal(t, "user_stats", view.Name)
		assert.Equal(t, "Statistics view for user post counts", view.Comment)
	})
}

func TestDatabaseInfo(t *testing.T) {
	t.Run("CreatePostgreSQLInfo", func(t *testing.T) {
		info := DatabaseInfo{
			Type:    "postgresql",
			Version: "14.2",
			Name:    "myapp_production",
			Charset: "UTF8",
		}

		assert.Equal(t, "postgresql", info.Type)
		assert.Equal(t, "14.2", info.Version)
		assert.Equal(t, "myapp_production", info.Name)
		assert.Equal(t, "UTF8", info.Charset)
	})

	t.Run("CreateMySQLInfo", func(t *testing.T) {
		info := DatabaseInfo{
			Type:    "mysql",
			Version: "8.0.28",
			Name:    "myapp_dev",
			Charset: "utf8mb4",
		}

		assert.Equal(t, "mysql", info.Type)
		assert.Equal(t, "8.0.28", info.Version)
		assert.Equal(t, "myapp_dev", info.Name)
		assert.Equal(t, "utf8mb4", info.Charset)
	})

	t.Run("CreateSQLiteInfo", func(t *testing.T) {
		info := DatabaseInfo{
			Type:    "sqlite",
			Version: "3.36.0",
			Name:    "myapp.db",
			Charset: "",
		}

		assert.Equal(t, "sqlite", info.Type)
		assert.Equal(t, "3.36.0", info.Version)
		assert.Equal(t, "myapp.db", info.Name)
		assert.Equal(t, "", info.Charset)
	})
}
