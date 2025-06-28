package pull

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/shibukawa/snapsql"
)

// Extractor interface for database-agnostic schema extraction
// snapsqlパッケージの型で統一
// ExtractTables/ExtractColumns/ExtractConstraints/ExtractIndexes/ExtractViewsもsnapsql型で返す
// DatabaseInfoもsnapsql型で返す
type Extractor interface {
	ExtractSchemas(db *sql.DB, config ExtractConfig) ([]snapsql.DatabaseSchema, error)
	ExtractTables(db *sql.DB, schemaName string) ([]*snapsql.TableInfo, error)
	ExtractColumns(db *sql.DB, schemaName, tableName string) (map[string]*snapsql.ColumnInfo, error)
	ExtractConstraints(db *sql.DB, schemaName, tableName string) ([]snapsql.ConstraintInfo, error)
	ExtractIndexes(db *sql.DB, schemaName, tableName string) ([]snapsql.IndexInfo, error)
	ExtractViews(db *sql.DB, schemaName string) ([]*snapsql.ViewInfo, error)
	GetDatabaseInfo(db *sql.DB) (snapsql.DatabaseInfo, error)
}

// NewExtractor creates a new extractor for the specified database type
func NewExtractor(databaseType string) (Extractor, error) {
	if databaseType == "" {
		return nil, ErrEmptyDatabaseType
	}

	switch strings.ToLower(databaseType) {
	case "postgresql", "postgres":
		return NewPostgreSQLExtractor(), nil
	case "mysql":
		return NewMySQLExtractor(), nil
	case "sqlite", "sqlite3":
		return NewSQLiteExtractor(), nil
	default:
		return nil, ErrUnsupportedDatabase
	}
}

// ValidateExtractConfig validates the extraction configuration
func ValidateExtractConfig(config ExtractConfig) error {
	// Check for conflicting schema filters
	for _, includeSchema := range config.IncludeSchemas {
		for _, excludeSchema := range config.ExcludeSchemas {
			if includeSchema == excludeSchema {
				return ErrConflictingSchemaFilters
			}
		}
	}

	// Check for conflicting table filters
	for _, includeTable := range config.IncludeTables {
		for _, excludeTable := range config.ExcludeTables {
			if includeTable == excludeTable {
				return ErrConflictingTableFilters
			}
		}
	}

	return nil
}

// ShouldIncludeSchema determines if a schema should be included based on filters
func ShouldIncludeSchema(schemaName string, includeSchemas, excludeSchemas []string) bool {
	// If explicitly excluded, don't include
	for _, excludeSchema := range excludeSchemas {
		if schemaName == excludeSchema {
			return false
		}
	}

	// If include list is specified, only include if in the list
	if len(includeSchemas) > 0 {
		for _, includeSchema := range includeSchemas {
			if schemaName == includeSchema {
				return true
			}
		}
		return false
	}

	// If no include list and not excluded, include
	return true
}

// ShouldIncludeTable determines if a table should be included based on filters
func ShouldIncludeTable(tableName string, includeTables, excludeTables []string) bool {
	// If explicitly excluded, don't include
	for _, excludeTable := range excludeTables {
		if MatchWildcard(excludeTable, tableName) {
			return false
		}
	}

	// If include list is specified, only include if in the list
	if len(includeTables) > 0 {
		for _, includeTable := range includeTables {
			if MatchWildcard(includeTable, tableName) {
				return true
			}
		}
		return false
	}

	// If no include list and not excluded, include
	return true
}

// MatchWildcard performs simple wildcard matching with * character
func MatchWildcard(pattern, text string) bool {
	// If no wildcard, do exact match
	if !strings.Contains(pattern, "*") {
		return pattern == text
	}

	// Use filepath.Match for wildcard matching
	matched, err := filepath.Match(pattern, text)
	if err != nil {
		// If pattern is invalid, fall back to exact match
		return pattern == text
	}
	return matched
}

// BaseExtractor provides common functionality for all extractors
type BaseExtractor struct {
	typeMapper TypeMapper
}

// NewBaseExtractor creates a new base extractor
func NewBaseExtractor(databaseType string) (*BaseExtractor, error) {
	typeMapper, err := NewTypeMapper(databaseType)
	if err != nil {
		return nil, err
	}

	return &BaseExtractor{
		typeMapper: typeMapper,
	}, nil
}

// MapColumnType maps a database type to SnapSQL type using the type mapper
func (e *BaseExtractor) MapColumnType(dbType string) string {
	return e.typeMapper.MapType(dbType)
}

// FilterSchemas filters schemas based on the configuration
func (e *BaseExtractor) FilterSchemas(schemas []string, config ExtractConfig) []string {
	var filtered []string
	for _, schema := range schemas {
		if ShouldIncludeSchema(schema, config.IncludeSchemas, config.ExcludeSchemas) {
			filtered = append(filtered, schema)
		}
	}
	return filtered
}

// FilterTables filters tables based on the configuration
func (e *BaseExtractor) FilterTables(tables []string, config ExtractConfig) []string {
	var filtered []string
	for _, table := range tables {
		if ShouldIncludeTable(table, config.IncludeTables, config.ExcludeTables) {
			filtered = append(filtered, table)
		}
	}
	return filtered
}

// Common SQL queries and patterns used across different database types
const (
	// Common system schema names to exclude by default
	PostgreSQLSystemSchemas = "information_schema,pg_catalog,pg_toast"
	MySQLSystemSchemas      = "information_schema,mysql,performance_schema,sys"
	SQLiteSystemTables      = "sqlite_%"
)

// GetDefaultExcludeSchemas returns default schemas to exclude for each database type
func GetDefaultExcludeSchemas(databaseType string) []string {
	switch strings.ToLower(databaseType) {
	case "postgresql", "postgres":
		return strings.Split(PostgreSQLSystemSchemas, ",")
	case "mysql":
		return strings.Split(MySQLSystemSchemas, ",")
	case "sqlite", "sqlite3":
		return []string{} // SQLite doesn't have schemas, but has system tables
	default:
		return []string{}
	}
}

// GetDefaultExcludeTables returns default tables to exclude for each database type
func GetDefaultExcludeTables(databaseType string) []string {
	switch strings.ToLower(databaseType) {
	case "sqlite", "sqlite3":
		return []string{SQLiteSystemTables}
	default:
		return []string{}
	}
}
