package typeinference

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
)

// SchemaResolver resolves type information from snapsql.DatabaseSchema
type SchemaResolver struct {
	schemas     []snapsql.DatabaseSchema
	schemaIndex map[string]map[string]*snapsql.TableInfo // [schema][table] -> TableInfo
	columnIndex map[string]*snapsql.ColumnInfo           // [schema.table.column] -> ColumnInfo
}

// NewSchemaResolver creates a new schema resolver from DatabaseSchema
func NewSchemaResolver(schemas []snapsql.DatabaseSchema) *SchemaResolver {
	resolver := &SchemaResolver{
		schemas:     schemas,
		schemaIndex: make(map[string]map[string]*snapsql.TableInfo),
		columnIndex: make(map[string]*snapsql.ColumnInfo),
	}

	// Build indexes for fast lookup
	resolver.buildIndexes()
	return resolver
}

// buildIndexes builds internal indexes for fast schema lookup
func (r *SchemaResolver) buildIndexes() {
	for _, dbSchema := range r.schemas {
		if r.schemaIndex[dbSchema.Name] == nil {
			r.schemaIndex[dbSchema.Name] = make(map[string]*snapsql.TableInfo)
		}

		for _, table := range dbSchema.Tables {
			// Add table to schema index
			r.schemaIndex[dbSchema.Name][table.Name] = table

			// Add columns to column index
			for columnName, column := range table.Columns {
				columnKey := fmt.Sprintf("%s.%s.%s", dbSchema.Name, table.Name, columnName)
				r.columnIndex[columnKey] = column
			}
		}
	}
}

// ResolveTableColumn resolves a table column and returns column information
func (r *SchemaResolver) ResolveTableColumn(schemaName, tableName, columnName string) (*snapsql.ColumnInfo, error) {
	// Check if table exists
	if err := r.ValidateTableExists(schemaName, tableName); err != nil {
		return nil, err
	}

	// Look up column
	columnKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, columnName)
	column, exists := r.columnIndex[columnKey]
	if !exists {
		return nil, fmt.Errorf("%w '%s' in table '%s.%s'", snapsql.ErrColumnDoesNotExist, columnName, schemaName, tableName)
	}

	return column, nil
}

// ValidateTableExists validates that a table exists in the given schema
func (r *SchemaResolver) ValidateTableExists(schemaName, tableName string) error {
	tables, exists := r.schemaIndex[schemaName]
	if !exists {
		return fmt.Errorf("%w: %s", snapsql.ErrSchemaDoesNotExist, schemaName)
	}

	if _, exists := tables[tableName]; !exists {
		return fmt.Errorf("%w '%s' in schema '%s'", snapsql.ErrTableDoesNotExist, tableName, schemaName)
	}

	return nil
}

// GetTableColumns returns all columns for a given table
func (r *SchemaResolver) GetTableColumns(schemaName, tableName string) ([]*snapsql.ColumnInfo, error) {
	if err := r.ValidateTableExists(schemaName, tableName); err != nil {
		return nil, err
	}

	table := r.schemaIndex[schemaName][tableName]
	columns := make([]*snapsql.ColumnInfo, 0, len(table.Columns))
	for _, column := range table.Columns {
		columns = append(columns, column)
	}

	return columns, nil
}

// ConvertToFieldType converts a snapsql.ColumnInfo to a TypeInfo
func (r *SchemaResolver) ConvertToFieldType(column *snapsql.ColumnInfo) *TypeInfo {
	return &TypeInfo{
		BaseType:   column.DataType, // Use normalized SnapSQL type
		IsNullable: column.Nullable,
		MaxLength:  column.MaxLength,
		Precision:  column.Precision,
		Scale:      column.Scale,
	}
}

// FindColumnInTables finds a column across all available tables and returns matching table names
func (r *SchemaResolver) FindColumnInTables(columnName string, availableTables []string) []string {
	var matches []string

	for _, tableRef := range availableTables {
		parts := strings.Split(tableRef, ".")
		var schemaName, tableName string

		if len(parts) == 2 {
			schemaName, tableName = parts[0], parts[1]
		} else {
			// If no schema specified, search in all schemas
			for schema := range r.schemaIndex {
				if tables, exists := r.schemaIndex[schema]; exists {
					if _, exists := tables[tableRef]; exists {
						schemaName, tableName = schema, tableRef
						break
					}
				}
			}
		}

		if schemaName == "" || tableName == "" {
			continue
		}

		// Check if column exists in this table
		columnKey := fmt.Sprintf("%s.%s.%s", schemaName, tableName, columnName)
		if _, exists := r.columnIndex[columnKey]; exists {
			matches = append(matches, fmt.Sprintf("%s.%s", schemaName, tableName))
		}
	}

	return matches
}

// GetAllSchemas returns all available schema names
func (r *SchemaResolver) GetAllSchemas() []string {
	var schemas []string
	for schemaName := range r.schemaIndex {
		schemas = append(schemas, schemaName)
	}
	return schemas
}

// GetTablesInSchema returns all table names in a given schema
func (r *SchemaResolver) GetTablesInSchema(schemaName string) []string {
	var tables []string
	if schemaMap, exists := r.schemaIndex[schemaName]; exists {
		for tableName := range schemaMap {
			tables = append(tables, tableName)
		}
	}
	return tables
}
