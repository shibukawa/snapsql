package pull

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	snapsql "github.com/shibukawa/snapsql"
)

// PostgreSQLExtractor handles PostgreSQL-specific schema extraction
type PostgreSQLExtractor struct {
	*BaseExtractor
}

// NewPostgreSQLExtractor creates a new PostgreSQL extractor
func NewPostgreSQLExtractor() *PostgreSQLExtractor {
	baseExtractor, _ := NewBaseExtractor("postgresql")

	return &PostgreSQLExtractor{
		BaseExtractor: baseExtractor,
	}
}

// GetDatabaseType returns the database type
func (e *PostgreSQLExtractor) GetDatabaseType() string {
	return "postgresql"
}

// GetSystemSchemas returns PostgreSQL system schemas to exclude by default
func (e *PostgreSQLExtractor) GetSystemSchemas() []string {
	return []string{
		"information_schema",
		"pg_catalog",
		"pg_toast",
		"pg_temp_1",
		"pg_toast_temp_1",
	}
}

// ExtractSchemas extracts all schemas from the database
func (e *PostgreSQLExtractor) ExtractSchemas(ctx context.Context, db *sql.DB, config ExtractConfig) ([]snapsql.DatabaseSchema, error) {
	// Get database info
	dbInfo, err := e.GetDatabaseInfo(ctx, db)
	if err != nil {
		return nil, err
	}

	// Get all schema names
	schemaNames, err := e.getSchemaNames(db)
	if err != nil {
		return nil, err
	}

	// Filter schemas
	filteredSchemas := e.FilterSchemas(schemaNames, config)

	schemas := make([]snapsql.DatabaseSchema, 0, len(filteredSchemas))

	for _, schemaName := range filteredSchemas {
		schema := snapsql.DatabaseSchema{
			Name:         schemaName,
			DatabaseInfo: dbInfo,
		}

		// Extract tables
		tables, err := e.ExtractTables(ctx, db, schemaName)
		if err != nil {
			return nil, err
		}

		var filteredTables []*snapsql.TableInfo

		for _, table := range tables {
			if ShouldIncludeTable(table.Name, config.IncludeTables, config.ExcludeTables) {
				filteredTables = append(filteredTables, table)
			}
		}

		schema.Tables = filteredTables

		// Extract views if requested
		if config.IncludeViews {
			views, err := e.ExtractViews(ctx, db, schemaName)
			if err != nil {
				return nil, err
			}

			schema.Views = views
		}

		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// ExtractTables extracts all tables from a specific schema
func (e *PostgreSQLExtractor) ExtractTables(ctx context.Context, db *sql.DB, schemaName string) ([]*snapsql.TableInfo, error) {
	query := e.BuildTablesQuery(schemaName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []*snapsql.TableInfo

	for rows.Next() {
		var (
			tableName string
			comment   sql.NullString
		)

		err := rows.Scan(&tableName, &comment)
		if err != nil {
			return nil, err
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
func (e *PostgreSQLExtractor) ExtractColumns(ctx context.Context, db *sql.DB, schemaName, tableName string) (map[string]*snapsql.ColumnInfo, error) {
	query := e.BuildColumnsQuery(schemaName, tableName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := map[string]*snapsql.ColumnInfo{}

	for rows.Next() {
		var (
			columnName, dataType, isNullable string
			defaultValue, comment            sql.NullString
		)

		err := rows.Scan(&columnName, &dataType, &isNullable, &defaultValue, &comment)
		if err != nil {
			return nil, err
		}

		col := &snapsql.ColumnInfo{
			Name:     columnName,
			DataType: e.MapPostgreSQLType(dataType),
			Nullable: isNullable == "YES",
		}
		if defaultValue.Valid {
			col.DefaultValue = e.ParseDefaultValue(defaultValue.String)
		}

		if comment.Valid {
			col.Comment = comment.String
		}

		columns[columnName] = col
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

// ExtractConstraints extracts all constraints from a specific table
func (e *PostgreSQLExtractor) ExtractConstraints(ctx context.Context, db *sql.DB, schemaName, tableName string) ([]snapsql.ConstraintInfo, error) {
	query := e.BuildConstraintsQuery(schemaName, tableName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []snapsql.ConstraintInfo

	for rows.Next() {
		var name, typ, columnsStr string

		err := rows.Scan(&name, &typ, &columnsStr)
		if err != nil {
			return nil, err
		}

		// Parse column names
		columns := strings.Split(columnsStr, ",")
		for i, col := range columns {
			columns[i] = strings.TrimSpace(col)
		}

		constraint := snapsql.ConstraintInfo{
			Name:    name,
			Type:    e.ParseConstraintType(typ),
			Columns: columns,
		}
		constraints = append(constraints, constraint)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return constraints, nil
}

// ExtractIndexes extracts all indexes from a specific table
func (e *PostgreSQLExtractor) ExtractIndexes(ctx context.Context, db *sql.DB, schemaName, tableName string) ([]snapsql.IndexInfo, error) {
	query := e.BuildIndexesQuery(schemaName, tableName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []snapsql.IndexInfo

	for rows.Next() {
		var (
			name, columnsStr, typ string
			isUnique              bool
		)

		err := rows.Scan(&name, &columnsStr, &isUnique, &typ)
		if err != nil {
			return nil, err
		}

		// Parse column names
		columns := strings.Split(columnsStr, ",")
		for i, col := range columns {
			columns[i] = strings.TrimSpace(col)
		}

		index := snapsql.IndexInfo{
			Name:     name,
			Columns:  columns,
			IsUnique: isUnique,
			Type:     typ,
		}
		indexes = append(indexes, index)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return indexes, nil
}

// ExtractViews extracts all views from a specific schema
func (e *PostgreSQLExtractor) ExtractViews(ctx context.Context, db *sql.DB, schemaName string) ([]*snapsql.ViewInfo, error) {
	query := e.BuildViewsQuery(schemaName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []*snapsql.ViewInfo

	for rows.Next() {
		var (
			viewName, viewDefinition string
			comment                  sql.NullString
		)

		err := rows.Scan(&viewName, &viewDefinition, &comment)
		if err != nil {
			return nil, err
		}

		view := &snapsql.ViewInfo{
			Name:       viewName,
			Schema:     schemaName,
			Definition: viewDefinition,
		}
		if comment.Valid {
			view.Comment = comment.String
		}

		views = append(views, view)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return views, nil
}

// GetDatabaseInfo extracts database information
func (e *PostgreSQLExtractor) GetDatabaseInfo(ctx context.Context, db *sql.DB) (snapsql.DatabaseInfo, error) {
	query := e.BuildDatabaseInfoQuery()
	row := db.QueryRowContext(ctx, query)

	var version, dbName string

	err := row.Scan(&version, &dbName)
	if err != nil {
		return snapsql.DatabaseInfo{}, err
	}

	return snapsql.DatabaseInfo{
		Type:    "postgresql",
		Version: version,
		Name:    dbName,
	}, nil
}

// Helper methods
func (e *PostgreSQLExtractor) getSchemaNames(db *sql.DB) ([]string, error) {
	query := e.BuildSchemasQuery()
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string

	for rows.Next() {
		var schema string

		err := rows.Scan(&schema)
		if err != nil {
			return nil, err
		}

		schemas = append(schemas, schema)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return schemas, nil
}

// Query builders
func (e *PostgreSQLExtractor) BuildSchemasQuery() string {
	return `
		SELECT schema_name 
		FROM information_schema.schemata 
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		ORDER BY schema_name
	`
}

func (e *PostgreSQLExtractor) BuildTablesQuery(schemaName string) string {
	return fmt.Sprintf(`
		SELECT 
			table_name,
			obj_description(c.oid) as comment
		FROM information_schema.tables t
		LEFT JOIN pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		WHERE table_schema = '%s' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`, schemaName)
}

func (e *PostgreSQLExtractor) BuildColumnsQuery(schemaName, tableName string) string {
	return fmt.Sprintf(`
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			col_description(c.oid, ordinal_position) as comment
		FROM information_schema.columns col
		LEFT JOIN pg_class c ON c.relname = col.table_name
		LEFT JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = col.table_schema
		WHERE table_schema = '%s' 
		AND table_name = '%s'
		ORDER BY ordinal_position
	`, schemaName, tableName)
}

func (e *PostgreSQLExtractor) BuildConstraintsQuery(schemaName, tableName string) string {
	return fmt.Sprintf(`
		SELECT 
			tc.constraint_name,
			tc.constraint_type,
			string_agg(kcu.column_name, ',' ORDER BY kcu.ordinal_position) as columns
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name 
			AND tc.table_schema = kcu.table_schema
		WHERE tc.table_schema = '%s' 
		AND tc.table_name = '%s'
		GROUP BY tc.constraint_name, tc.constraint_type
		ORDER BY tc.constraint_name
	`, schemaName, tableName)
}

func (e *PostgreSQLExtractor) BuildIndexesQuery(schemaName, tableName string) string {
	return fmt.Sprintf(`
		SELECT 
			i.relname as index_name,
			string_agg(a.attname, ',' ORDER BY a.attnum) as columns,
			ix.indisunique as is_unique,
			am.amname as index_type
		FROM pg_class t
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_am am ON i.relam = am.oid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = '%s' 
		AND t.relname = '%s'
		AND NOT ix.indisprimary
		GROUP BY i.relname, ix.indisunique, am.amname
		ORDER BY i.relname
	`, schemaName, tableName)
}

func (e *PostgreSQLExtractor) BuildViewsQuery(schemaName string) string {
	return fmt.Sprintf(`
		SELECT 
			table_name,
			view_definition,
			obj_description(c.oid) as comment
		FROM information_schema.views v
		LEFT JOIN pg_class c ON c.relname = v.table_name
		LEFT JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = v.table_schema
		WHERE table_schema = '%s'
		ORDER BY table_name
	`, schemaName)
}

func (e *PostgreSQLExtractor) BuildDatabaseInfoQuery() string {
	return `SELECT version(), current_database()`
}

// MapPostgreSQLType maps PostgreSQL data types to SnapSQL types
func (e *PostgreSQLExtractor) MapPostgreSQLType(pgType string) string {
	// Remove precision/scale information
	baseType := regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(pgType, "")
	baseType = strings.ToLower(strings.TrimSpace(baseType))

	switch baseType {
	case "smallint", "integer", "bigint", "serial", "bigserial":
		return "int"
	case "decimal", "numeric", "real", "double precision", "money":
		return "float"
	case "boolean":
		return "bool"
	case "character", "character varying", "varchar", "text", "char":
		return "string"
	case "date":
		return "date"
	case "time", "time without time zone", "time with time zone":
		return "time"
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
		return "datetime"
	case "interval":
		return "duration"
	case "uuid":
		return "uuid"
	case "json", "jsonb":
		return "json"
	case "bytea":
		return "bytes"
	case "inet", "cidr", "macaddr":
		return "string"
	case "point", "line", "lseg", "box", "path", "polygon", "circle":
		return "geometry"
	case "array":
		return "array"
	default:
		return "string"
	}
}

// ParseConstraintType parses PostgreSQL constraint types
func (e *PostgreSQLExtractor) ParseConstraintType(constraintType string) string {
	switch strings.ToUpper(strings.TrimSpace(constraintType)) {
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

// ParseIndexUnique parses PostgreSQL index uniqueness
func (e *PostgreSQLExtractor) ParseIndexUnique(unique any) bool {
	switch v := unique.(type) {
	case string:
		// Handle CREATE INDEX statements
		if strings.Contains(strings.ToUpper(v), "CREATE UNIQUE INDEX") {
			return true
		}
		// Handle boolean string values
		return strings.ToLower(strings.TrimSpace(v)) == "t" ||
			strings.ToLower(strings.TrimSpace(v)) == "true"
	case bool:
		return v
	default:
		return false
	}
}

// ParseIndexType parses PostgreSQL index types
func (e *PostgreSQLExtractor) ParseIndexType(indexType string) string {
	// Handle CREATE INDEX statements
	if strings.Contains(strings.ToUpper(indexType), "USING") {
		parts := strings.Split(strings.ToUpper(indexType), "USING")
		if len(parts) > 1 {
			typePart := strings.TrimSpace(parts[1])
			// Extract type before opening parenthesis
			if idx := strings.Index(typePart, "("); idx > 0 {
				typePart = strings.TrimSpace(typePart[:idx])
			}

			return e.normalizeIndexType(typePart)
		}
	}

	// Handle direct type values
	return e.normalizeIndexType(indexType)
}

// normalizeIndexType normalizes index type names
func (e *PostgreSQLExtractor) normalizeIndexType(indexType string) string {
	switch strings.ToLower(strings.TrimSpace(indexType)) {
	case "btree":
		return "BTREE"
	case "hash":
		return "HASH"
	case "gist":
		return "GIST"
	case "gin":
		return "GIN"
	case "spgist":
		return "SPGIST"
	case "brin":
		return "BRIN"
	default:
		return strings.ToUpper(indexType)
	}
}

// ParseIndexColumns parses PostgreSQL index column definitions
func (e *PostgreSQLExtractor) ParseIndexColumns(columns string) []string {
	if columns == "" {
		return []string{}
	}

	// Handle CREATE INDEX statements
	if strings.Contains(strings.ToUpper(columns), "CREATE") && strings.Contains(columns, "INDEX") {
		// Extract column names from CREATE INDEX statement
		start := strings.Index(columns, "(")

		end := strings.LastIndex(columns, ")")
		if start > 0 && end > start {
			columnPart := columns[start+1 : end]
			parts := strings.Split(columnPart, ",")

			result := make([]string, len(parts))
			for i, part := range parts {
				result[i] = strings.TrimSpace(part)
			}

			return result
		}
		// If no parentheses found, return the whole string as single column
		return []string{columns}
	}

	// Handle comma-separated column list
	parts := strings.Split(columns, ",")

	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
	}

	return result
}

// ParseDefaultValue parses PostgreSQL default values
func (e *PostgreSQLExtractor) ParseDefaultValue(defaultValue string) string {
	if defaultValue == "" {
		return ""
	}

	// Remove common PostgreSQL default value prefixes
	value := strings.TrimSpace(defaultValue)

	// Handle nextval() for sequences (including regclass casting)
	if strings.HasPrefix(value, "nextval(") {
		return "AUTO_INCREMENT"
	}

	// Handle string literals with type casting (e.g., 'value'::character varying)
	if strings.Contains(value, "::") {
		parts := strings.Split(value, "::")
		if len(parts) > 0 {
			literalPart := strings.TrimSpace(parts[0])
			if strings.HasPrefix(literalPart, "'") && strings.HasSuffix(literalPart, "'") {
				return literalPart[1 : len(literalPart)-1]
			}
		}
	}

	// Handle string literals
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value[1 : len(value)-1]
	}

	// Handle boolean values
	switch strings.ToLower(value) {
	case "true", "false":
		return strings.ToLower(value)
	}

	// Handle NULL
	if strings.ToUpper(value) == "NULL" {
		return ""
	}

	return value
}

// FilterSystemSchemas filters out PostgreSQL system schemas
func (e *PostgreSQLExtractor) FilterSystemSchemas(schemas []string, config ExtractConfig) []string {
	systemSchemas := e.GetSystemSchemas()

	var filtered []string

	for _, schema := range schemas {
		isSystem := false

		for _, sysSchema := range systemSchemas {
			if schema == sysSchema {
				isSystem = true
				break
			}
		}

		if !isSystem {
			filtered = append(filtered, schema)
		}
	}

	return e.FilterSchemas(filtered, config)
}

// HandleDatabaseError handles PostgreSQL-specific database errors
func (e *PostgreSQLExtractor) HandleDatabaseError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Handle common PostgreSQL errors
	if strings.Contains(errStr, "connection refused") {
		return fmt.Errorf("PostgreSQL connection refused: %w", ErrConnectionFailed)
	}

	if strings.Contains(errStr, "authentication failed") {
		return fmt.Errorf("PostgreSQL authentication failed: %w", ErrConnectionFailed)
	}

	if strings.Contains(errStr, "database") && strings.Contains(errStr, "does not exist") {
		return fmt.Errorf("PostgreSQL database does not exist: %w", ErrSchemaNotFound)
	}

	if strings.Contains(errStr, "relation") && strings.Contains(errStr, "does not exist") {
		return fmt.Errorf("PostgreSQL table/view does not exist: %w", ErrTableNotFound)
	}

	// Return original error if no specific handling
	return err
}
