package pull

import (
	"database/sql"
	"fmt"
	"strings"

	snapsql "github.com/shibukawa/snapsql"
)

// MySQLExtractor handles MySQL-specific schema extraction
type MySQLExtractor struct {
	*BaseExtractor
}

// NewMySQLExtractor creates a new MySQL extractor
func NewMySQLExtractor() *MySQLExtractor {
	baseExtractor, _ := NewBaseExtractor("mysql")
	return &MySQLExtractor{
		BaseExtractor: baseExtractor,
	}
}

// ExtractSchemas extracts all schemas from the database
func (e *MySQLExtractor) ExtractSchemas(db *sql.DB, config ExtractConfig) ([]snapsql.DatabaseSchema, error) {
	// Get database info
	dbInfo, err := e.GetDatabaseInfo(db)
	if err != nil {
		return nil, err
	}

	// MySQL uses the database name as schema name
	schema := snapsql.DatabaseSchema{
		Name:         dbInfo.Name,
		DatabaseInfo: dbInfo,
	}

	// Extract tables
	tables, err := e.ExtractTables(db, dbInfo.Name)
	if err != nil {
		return nil, err
	}

	// Apply table filtering
	var filteredTables []*snapsql.TableInfo
	for _, table := range tables {
		if ShouldIncludeTable(table.Name, config.IncludeTables, config.ExcludeTables) {
			filteredTables = append(filteredTables, table)
		}
	}
	schema.Tables = filteredTables

	// Extract views if requested
	if config.IncludeViews {
		views, err := e.ExtractViews(db, dbInfo.Name)
		if err != nil {
			return nil, err
		}
		schema.Views = views
	}

	return []snapsql.DatabaseSchema{schema}, nil
}

// ExtractTables extracts all tables from a specific schema
func (e *MySQLExtractor) ExtractTables(db *sql.DB, schemaName string) ([]*snapsql.TableInfo, error) {
	query := e.BuildTablesQuery(schemaName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, e.HandleDatabaseError(err)
	}
	defer rows.Close()

	var tables []*snapsql.TableInfo
	for rows.Next() {
		var tableName, tableType, engine string
		var comment sql.NullString

		err := rows.Scan(&tableName, &tableType, &engine, &comment)
		if err != nil {
			return nil, e.HandleDatabaseError(err)
		}

		table := &snapsql.TableInfo{
			Name:    tableName,
			Schema:  schemaName,
			Columns: map[string]*snapsql.ColumnInfo{},
		}

		if comment.Valid {
			table.Comment = comment.String
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
		return nil, e.HandleDatabaseError(err)
	}

	return tables, nil
}

// ExtractColumns extracts all columns from a specific table
func (e *MySQLExtractor) ExtractColumns(db *sql.DB, schemaName, tableName string) (map[string]*snapsql.ColumnInfo, error) {
	query := e.BuildColumnsQuery(schemaName, tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, e.HandleDatabaseError(err)
	}
	defer rows.Close()

	columns := map[string]*snapsql.ColumnInfo{}
	for rows.Next() {
		var columnName, dataType, isNullable, columnKey string
		var columnDefault, extra, comment sql.NullString
		var characterMaxLength, numericPrecision, numericScale sql.NullInt64

		err := rows.Scan(&columnName, &dataType, &isNullable, &columnKey,
			&columnDefault, &extra, &characterMaxLength, &numericPrecision,
			&numericScale, &comment)
		if err != nil {
			return nil, e.HandleDatabaseError(err)
		}

		col := &snapsql.ColumnInfo{
			Name:         columnName,
			DataType:     e.MapColumnType(dataType),
			Nullable:     isNullable == "YES",
			IsPrimaryKey: columnKey == "PRI",
		}
		if columnDefault.Valid {
			col.DefaultValue = e.ParseDefaultValue(columnDefault.String)
		}
		if comment.Valid {
			col.Comment = comment.String
		}
		if characterMaxLength.Valid {
			v := int(characterMaxLength.Int64)
			col.MaxLength = &v
		}
		if numericPrecision.Valid {
			v := int(numericPrecision.Int64)
			col.Precision = &v
		}
		if numericScale.Valid {
			v := int(numericScale.Int64)
			col.Scale = &v
		}
		columns[columnName] = col
	}

	if err := rows.Err(); err != nil {
		return nil, e.HandleDatabaseError(err)
	}

	return columns, nil
}

// ExtractConstraints extracts all constraints from a specific table
func (e *MySQLExtractor) ExtractConstraints(db *sql.DB, schemaName, tableName string) ([]snapsql.ConstraintInfo, error) {
	query := e.BuildConstraintsQuery(schemaName, tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, e.HandleDatabaseError(err)
	}
	defer rows.Close()

	constraintMap := make(map[string]*snapsql.ConstraintInfo)

	for rows.Next() {
		var constraintName, constraintType string
		var columnName sql.NullString
		var referencedSchema, referencedTable, referencedColumn sql.NullString

		err := rows.Scan(&constraintName, &constraintType, &columnName,
			&referencedSchema, &referencedTable, &referencedColumn)
		if err != nil {
			return nil, e.HandleDatabaseError(err)
		}

		if constraint, exists := constraintMap[constraintName]; exists {
			// Add column to existing constraint
			if columnName.Valid {
				constraint.Columns = append(constraint.Columns, columnName.String)
			}
			if referencedColumn.Valid {
				constraint.ReferencedColumns = append(constraint.ReferencedColumns, referencedColumn.String)
			}
		} else {
			// Create new constraint
			constraint := &snapsql.ConstraintInfo{
				Name: constraintName,
				Type: e.ParseConstraintType(constraintType),
			}

			if columnName.Valid {
				constraint.Columns = []string{columnName.String}
			} else {
				constraint.Columns = []string{}
			}

			if referencedTable.Valid {
				constraint.ReferencedTable = referencedTable.String
			}
			if referencedColumn.Valid {
				constraint.ReferencedColumns = []string{referencedColumn.String}
			}

			constraintMap[constraintName] = constraint
		}
	}

	if err := rows.Err(); err != nil {
		return nil, e.HandleDatabaseError(err)
	}

	// Convert map to slice
	constraints := make([]snapsql.ConstraintInfo, 0, len(constraintMap))
	for _, constraint := range constraintMap {
		constraints = append(constraints, *constraint)
	}

	return constraints, nil
}

// ExtractIndexes extracts all indexes from a specific table
func (e *MySQLExtractor) ExtractIndexes(db *sql.DB, schemaName, tableName string) ([]snapsql.IndexInfo, error) {
	query := e.BuildIndexesQuery(schemaName, tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, e.HandleDatabaseError(err)
	}
	defer rows.Close()

	indexMap := make(map[string]*snapsql.IndexInfo)

	for rows.Next() {
		var indexName, columnName, indexType string
		var nonUnique int
		var seqInIndex int

		err := rows.Scan(&indexName, &nonUnique, &seqInIndex, &columnName, &indexType)
		if err != nil {
			return nil, e.HandleDatabaseError(err)
		}

		// Skip primary key index (handled as constraint)
		if indexName == "PRIMARY" {
			continue
		}

		if index, exists := indexMap[indexName]; exists {
			// Add column to existing index (maintain order)
			if seqInIndex <= len(index.Columns) {
				// Insert at correct position
				newColumns := make([]string, len(index.Columns)+1)
				copy(newColumns[:seqInIndex-1], index.Columns[:seqInIndex-1])
				newColumns[seqInIndex-1] = columnName
				copy(newColumns[seqInIndex:], index.Columns[seqInIndex-1:])
				index.Columns = newColumns
			} else {
				index.Columns = append(index.Columns, columnName)
			}
		} else {
			// Create new index
			index := &snapsql.IndexInfo{
				Name:     indexName,
				Columns:  []string{columnName},
				IsUnique: nonUnique == 0,
				Type:     strings.ToLower(indexType),
			}

			indexMap[indexName] = index
		}
	}

	if err := rows.Err(); err != nil {
		return nil, e.HandleDatabaseError(err)
	}

	// Convert map to slice
	indexes := make([]snapsql.IndexInfo, 0, len(indexMap))
	for _, index := range indexMap {
		indexes = append(indexes, *index)
	}

	return indexes, nil
}

// ExtractViews extracts all views from a specific schema
func (e *MySQLExtractor) ExtractViews(db *sql.DB, schemaName string) ([]*snapsql.ViewInfo, error) {
	query := e.BuildViewsQuery(schemaName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, e.HandleDatabaseError(err)
	}
	defer rows.Close()

	var views []*snapsql.ViewInfo
	for rows.Next() {
		var viewName, viewDefinition string

		err := rows.Scan(&viewName, &viewDefinition)
		if err != nil {
			return nil, e.HandleDatabaseError(err)
		}

		view := &snapsql.ViewInfo{
			Name:       viewName,
			Schema:     schemaName,
			Definition: viewDefinition,
		}

		views = append(views, view)
	}

	if err := rows.Err(); err != nil {
		return nil, e.HandleDatabaseError(err)
	}

	return views, nil
}

// GetDatabaseInfo extracts database information
func (e *MySQLExtractor) GetDatabaseInfo(db *sql.DB) (snapsql.DatabaseInfo, error) {
	var version, dbName, charset string

	// Get version
	err := db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return snapsql.DatabaseInfo{}, e.HandleDatabaseError(err)
	}

	// Get database name
	err = db.QueryRow("SELECT DATABASE()").Scan(&dbName)
	if err != nil {
		return snapsql.DatabaseInfo{}, e.HandleDatabaseError(err)
	}

	// Get default charset
	err = db.QueryRow("SELECT @@character_set_database").Scan(&charset)
	if err != nil {
		// If charset query fails, use default
		charset = "utf8mb4"
	}

	return snapsql.DatabaseInfo{
		Type:    "mysql",
		Version: version,
		Name:    dbName,
		Charset: charset,
	}, nil
}

// Query builders for MySQL

// BuildTablesQuery builds a query to get all tables in a schema
func (e *MySQLExtractor) BuildTablesQuery(schemaName string) string {
	return `
		SELECT 
			TABLE_NAME,
			TABLE_TYPE,
			ENGINE,
			TABLE_COMMENT
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = '` + schemaName + `'
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME`
}

// BuildColumnsQuery builds a query to get all columns in a table
func (e *MySQLExtractor) BuildColumnsQuery(schemaName, tableName string) string {
	return `
		SELECT 
			COLUMN_NAME,
			DATA_TYPE,
			IS_NULLABLE,
			COLUMN_KEY,
			COLUMN_DEFAULT,
			EXTRA,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE,
			COLUMN_COMMENT
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = '` + schemaName + `' 
		  AND TABLE_NAME = '` + tableName + `'
		ORDER BY ORDINAL_POSITION`
}

// BuildConstraintsQuery builds a query to get all constraints in a table
func (e *MySQLExtractor) BuildConstraintsQuery(schemaName, tableName string) string {
	return `
		SELECT 
			tc.CONSTRAINT_NAME,
			tc.CONSTRAINT_TYPE,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_SCHEMA,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME
		FROM information_schema.TABLE_CONSTRAINTS tc
		LEFT JOIN information_schema.KEY_COLUMN_USAGE kcu 
			ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME 
			AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA
			AND tc.TABLE_NAME = kcu.TABLE_NAME
		WHERE tc.TABLE_SCHEMA = '` + schemaName + `' 
		  AND tc.TABLE_NAME = '` + tableName + `'
		ORDER BY tc.CONSTRAINT_NAME, kcu.ORDINAL_POSITION`
}

// BuildIndexesQuery builds a query to get all indexes in a table
func (e *MySQLExtractor) BuildIndexesQuery(schemaName, tableName string) string {
	return `
		SELECT 
			INDEX_NAME,
			NON_UNIQUE,
			SEQ_IN_INDEX,
			COLUMN_NAME,
			INDEX_TYPE
		FROM information_schema.STATISTICS 
		WHERE TABLE_SCHEMA = '` + schemaName + `' 
		  AND TABLE_NAME = '` + tableName + `'
		ORDER BY INDEX_NAME, SEQ_IN_INDEX`
}

// BuildViewsQuery builds a query to get all views in a schema
func (e *MySQLExtractor) BuildViewsQuery(schemaName string) string {
	return `
		SELECT 
			TABLE_NAME,
			VIEW_DEFINITION
		FROM information_schema.VIEWS 
		WHERE TABLE_SCHEMA = '` + schemaName + `'
		ORDER BY TABLE_NAME`
}

// Helper methods for MySQL

// ParseDefaultValue parses MySQL default values
func (e *MySQLExtractor) ParseDefaultValue(defaultValue string) string {
	// Handle MySQL-specific default values
	switch strings.ToUpper(defaultValue) {
	case "CURRENT_TIMESTAMP", "NOW()":
		return "CURRENT_TIMESTAMP"
	case "NULL":
		return ""
	default:
		// Remove quotes if present
		if strings.HasPrefix(defaultValue, "'") && strings.HasSuffix(defaultValue, "'") {
			return strings.Trim(defaultValue, "'")
		}
		return defaultValue
	}
}

// ParseConstraintType converts MySQL constraint types to standard types
func (e *MySQLExtractor) ParseConstraintType(constraintType string) string {
	switch strings.ToUpper(constraintType) {
	case "PRIMARY KEY":
		return "PRIMARY_KEY"
	case "FOREIGN KEY":
		return "FOREIGN_KEY"
	case "UNIQUE":
		return "UNIQUE"
	case "CHECK":
		return "CHECK"
	default:
		return strings.ToUpper(constraintType)
	}
}

// HandleDatabaseError converts MySQL-specific errors to standard errors
func (e *MySQLExtractor) HandleDatabaseError(err error) error {
	if err == nil {
		return nil
	}

	// For debugging, return the original error wrapped
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "connection"):
		return fmt.Errorf("connection error: %w", err)
	case strings.Contains(errStr, "doesn't exist"):
		return fmt.Errorf("schema not found: %w", err)
	case strings.Contains(errStr, "Access denied"):
		return fmt.Errorf("permission denied: %w", err)
	default:
		return fmt.Errorf("query execution failed: %w", err)
	}
}
