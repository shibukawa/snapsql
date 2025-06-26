package pull

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SQLiteExtractor handles SQLite-specific schema extraction
type SQLiteExtractor struct {
	*BaseExtractor
}

// NewSQLiteExtractor creates a new SQLite extractor
func NewSQLiteExtractor() *SQLiteExtractor {
	baseExtractor, _ := NewBaseExtractor("sqlite")
	return &SQLiteExtractor{
		BaseExtractor: baseExtractor,
	}
}

// ExtractSchemas extracts all schemas from the database
func (e *SQLiteExtractor) ExtractSchemas(db *sql.DB, config ExtractConfig) ([]DatabaseSchema, error) {
	// Get database info
	dbInfo, err := e.GetDatabaseInfo(db)
	if err != nil {
		return nil, err
	}

	// SQLite uses 'main' as default schema, map to 'global'
	schema := DatabaseSchema{
		Name:         "global",
		ExtractedAt:  time.Now(),
		DatabaseInfo: dbInfo,
	}

	// Extract tables
	tables, err := e.ExtractTables(db, "main")
	if err != nil {
		return nil, err
	}
	schema.Tables = tables

	// Extract views if requested
	if config.IncludeViews {
		views, err := e.ExtractViews(db, "main")
		if err != nil {
			return nil, err
		}
		schema.Views = views
	}

	return []DatabaseSchema{schema}, nil
}

// ExtractTables extracts all tables from a specific schema
func (e *SQLiteExtractor) ExtractTables(db *sql.DB, schemaName string) ([]TableSchema, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableSchema
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			return nil, err
		}

		table := TableSchema{
			Name:   tableName,
			Schema: "global", // SQLite uses global schema
		}

		// Extract columns
		columns, err := e.ExtractColumns(db, schemaName, tableName)
		if err != nil {
			return nil, err
		}
		table.Columns = columns

		// Extract constraints
		constraints, err := e.ExtractConstraints(db, schemaName, tableName)
		if err != nil {
			return nil, err
		}
		table.Constraints = constraints

		// Extract indexes
		indexes, err := e.ExtractIndexes(db, schemaName, tableName)
		if err != nil {
			return nil, err
		}
		table.Indexes = indexes

		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

// ExtractColumns extracts all columns from a specific table
func (e *SQLiteExtractor) ExtractColumns(db *sql.DB, schemaName, tableName string) ([]ColumnSchema, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnSchema
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			return nil, err
		}

		column := ColumnSchema{
			Name:         name,
			Type:         dataType,
			SnapSQLType:  e.MapColumnType(dataType),
			Nullable:     notNull == 0,
			IsPrimaryKey: pk == 1,
		}

		if defaultValue.Valid {
			column.DefaultValue = defaultValue.String
		}

		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// ExtractConstraints extracts all constraints from a specific table
func (e *SQLiteExtractor) ExtractConstraints(db *sql.DB, schemaName, tableName string) ([]ConstraintSchema, error) {
	// SQLite constraint extraction is limited, we'll extract what we can from table_info
	var constraints []ConstraintSchema

	// Get primary key constraints from table_info
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pkColumns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			return nil, err
		}

		if pk == 1 {
			pkColumns = append(pkColumns, name)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Create primary key constraint if exists
	if len(pkColumns) > 0 {
		constraints = append(constraints, ConstraintSchema{
			Name:    fmt.Sprintf("%s_pkey", tableName),
			Type:    "PRIMARY_KEY",
			Columns: pkColumns,
		})
	}

	return constraints, nil
}

// ExtractIndexes extracts all indexes from a specific table
func (e *SQLiteExtractor) ExtractIndexes(db *sql.DB, schemaName, tableName string) ([]IndexSchema, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexSchema
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin, partial string

		err := rows.Scan(&seq, &name, &unique, &origin, &partial)
		if err != nil {
			return nil, err
		}

		// Skip auto-created indexes for primary keys
		if strings.HasPrefix(name, "sqlite_autoindex_") {
			continue
		}

		// Get index columns
		columnsQuery := fmt.Sprintf("PRAGMA index_info(%s)", name)
		colRows, err := db.Query(columnsQuery)
		if err != nil {
			continue
		}
		defer colRows.Close()

		var columns []string
		for colRows.Next() {
			var seqno, cid int
			var colName string
			err := colRows.Scan(&seqno, &cid, &colName)
			if err != nil {
				continue
			}
			columns = append(columns, colName)
		}

		if err := colRows.Err(); err != nil {
			continue
		}

		index := IndexSchema{
			Name:     name,
			Columns:  columns,
			IsUnique: unique == 1,
			Type:     "btree", // SQLite uses B-tree indexes
		}

		indexes = append(indexes, index)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexes, nil
}

// ExtractViews extracts all views from a specific schema
func (e *SQLiteExtractor) ExtractViews(db *sql.DB, schemaName string) ([]ViewSchema, error) {
	query := `SELECT name, sql FROM sqlite_master WHERE type='view' ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []ViewSchema
	for rows.Next() {
		var name, definition string
		err := rows.Scan(&name, &definition)
		if err != nil {
			return nil, err
		}

		view := ViewSchema{
			Name:       name,
			Schema:     "global",
			Definition: definition,
		}

		views = append(views, view)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return views, nil
}

// GetDatabaseInfo extracts database information
func (e *SQLiteExtractor) GetDatabaseInfo(db *sql.DB) (DatabaseInfo, error) {
	var version string
	err := db.QueryRow("SELECT sqlite_version()").Scan(&version)
	if err != nil {
		return DatabaseInfo{}, err
	}

	return DatabaseInfo{
		Type:    "sqlite",
		Version: version,
		Name:    "sqlite_database",
		Charset: "",
	}, nil
}
