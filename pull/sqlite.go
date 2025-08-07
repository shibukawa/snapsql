package pull

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	snapsql "github.com/shibukawa/snapsql"
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
func (e *SQLiteExtractor) ExtractSchemas(ctx context.Context, db *sql.DB, config ExtractConfig) ([]snapsql.DatabaseSchema, error) {
	// Get database info
	dbInfo, err := e.GetDatabaseInfo(ctx, db)
	if err != nil {
		return nil, err
	}

	// SQLite uses 'main' as default schema, map to 'global'
	schema := snapsql.DatabaseSchema{
		Name:         "global",
		DatabaseInfo: dbInfo,
	}

	// Extract tables
	tables, err := e.ExtractTables(ctx, db, "main")
	if err != nil {
		return nil, err
	}
	schema.Tables = tables

	// Extract views if requested
	if config.IncludeViews {
		views, err := e.ExtractViews(ctx, db, "main")
		if err != nil {
			return nil, err
		}
		schema.Views = views
	}

	return []snapsql.DatabaseSchema{schema}, nil
}

// ExtractTables extracts all tables from a specific schema
func (e *SQLiteExtractor) ExtractTables(ctx context.Context, db *sql.DB, schemaName string) ([]*snapsql.TableInfo, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []*snapsql.TableInfo
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			return nil, err
		}

		table := &snapsql.TableInfo{
			Name:    tableName,
			Schema:  "global",
			Columns: map[string]*snapsql.ColumnInfo{},
		}

		// Extract columns
		columns, err := e.ExtractColumns(ctx, db, schemaName, tableName)
		if err != nil {
			return nil, err
		}
		table.Columns = columns

		// Extract constraints
		constraints, err := e.ExtractConstraints(ctx, db, schemaName, tableName)
		if err != nil {
			return nil, err
		}
		table.Constraints = constraints

		// Extract indexes
		indexes, err := e.ExtractIndexes(ctx, db, schemaName, tableName)
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
func (e *SQLiteExtractor) ExtractColumns(ctx context.Context, db *sql.DB, schemaName, tableName string) (map[string]*snapsql.ColumnInfo, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := map[string]*snapsql.ColumnInfo{}
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			return nil, err
		}

		col := &snapsql.ColumnInfo{
			Name:         name,
			DataType:     e.MapColumnType(dataType),
			Nullable:     notNull == 0,
			IsPrimaryKey: pk == 1,
		}
		if defaultValue.Valid {
			col.DefaultValue = defaultValue.String
		}
		columns[name] = col
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// ExtractConstraints extracts all constraints from a specific table
func (e *SQLiteExtractor) ExtractConstraints(ctx context.Context, db *sql.DB, schemaName, tableName string) ([]snapsql.ConstraintInfo, error) {
	// SQLite constraint extraction is limited, we'll extract what we can from table_info
	var constraints []snapsql.ConstraintInfo

	// Get primary key constraints from table_info
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.QueryContext(ctx, query)
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
		constraints = append(constraints, snapsql.ConstraintInfo{
			Name:    fmt.Sprintf("%s_pkey", tableName),
			Type:    "PRIMARY_KEY",
			Columns: pkColumns,
		})
	}

	return constraints, nil
}

// ExtractIndexes extracts all indexes from a specific table
func (e *SQLiteExtractor) ExtractIndexes(ctx context.Context, db *sql.DB, schemaName, tableName string) ([]snapsql.IndexInfo, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s)", tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []snapsql.IndexInfo
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
		ctx := context.Background()
		colRows, err := db.QueryContext(ctx, columnsQuery)
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

		index := snapsql.IndexInfo{
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
func (e *SQLiteExtractor) ExtractViews(ctx context.Context, db *sql.DB, schemaName string) ([]*snapsql.ViewInfo, error) {
	query := `SELECT name, sql FROM sqlite_master WHERE type='view' ORDER BY name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []*snapsql.ViewInfo
	for rows.Next() {
		var name, definition string
		err := rows.Scan(&name, &definition)
		if err != nil {
			return nil, err
		}

		view := &snapsql.ViewInfo{
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
func (e *SQLiteExtractor) GetDatabaseInfo(ctx context.Context, db *sql.DB) (snapsql.DatabaseInfo, error) {
	var version string
	err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version)
	if err != nil {
		return snapsql.DatabaseInfo{}, err
	}

	return snapsql.DatabaseInfo{
		Type:    "sqlite",
		Version: version,
		Name:    "sqlite_database",
		Charset: "",
	}, nil
}
