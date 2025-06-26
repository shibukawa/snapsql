package pull

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
)

// YAMLGenerator generates YAML schema files from database schemas
type YAMLGenerator struct {
	Format      OutputFormat
	Pretty      bool
	SchemaAware bool // Enable schema-aware directory structure
	FlowStyle   bool // Use flow style for columns, constraints, indexes
}

// NewYAMLGenerator creates a new YAML generator
func NewYAMLGenerator(format OutputFormat, pretty, schemaAware, flowStyle bool) *YAMLGenerator {
	return &YAMLGenerator{
		Format:      format,
		Pretty:      pretty,
		SchemaAware: schemaAware,
		FlowStyle:   flowStyle,
	}
}

// Generate generates YAML files from database schemas
func (g *YAMLGenerator) Generate(schemas []DatabaseSchema, outputPath string) error {
	switch g.Format {
	case OutputSingleFile:
		return g.generateSingleFile(schemas, outputPath)
	case OutputPerTable:
		return g.generatePerTable(schemas, outputPath)
	case OutputPerSchema:
		return g.generatePerSchema(schemas, outputPath)
	default:
		return ErrInvalidOutputFormat
	}
}

// generateSingleFile generates a single YAML file containing all schemas
func (g *YAMLGenerator) generateSingleFile(schemas []DatabaseSchema, outputPath string) error {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return ErrDirectoryCreateFailed
	}

	filename := filepath.Join(outputPath, "database_schema.yaml")
	file, err := os.Create(filename)
	if err != nil {
		return ErrFileWriteFailed
	}
	defer file.Close()

	// Create single file structure
	singleFileData := SingleFileSchema{
		DatabaseInfo: schemas[0].DatabaseInfo, // Use first schema's database info
		ExtractedAt:  time.Now(),
		Schemas:      schemas,
	}

	return g.writeYAML(file, singleFileData)
}

// generatePerTable generates separate YAML files for each table
func (g *YAMLGenerator) generatePerTable(schemas []DatabaseSchema, outputPath string) error {
	for _, schema := range schemas {
		schemaPath := g.getSchemaPath(outputPath, schema.Name)
		if err := os.MkdirAll(schemaPath, 0755); err != nil {
			return ErrDirectoryCreateFailed
		}

		for _, table := range schema.Tables {
			filename := filepath.Join(schemaPath, g.getTableFileName(table.Name))
			file, err := os.Create(filename)
			if err != nil {
				return ErrFileWriteFailed
			}

			tableData := PerTableSchema{
				Table: table,
				Metadata: SchemaMetadata{
					ExtractedAt:  schema.ExtractedAt,
					DatabaseInfo: schema.DatabaseInfo,
				},
			}

			if err := g.writeYAML(file, tableData); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
	}

	return nil
}

// generatePerSchema generates separate YAML files for each schema
func (g *YAMLGenerator) generatePerSchema(schemas []DatabaseSchema, outputPath string) error {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return ErrDirectoryCreateFailed
	}

	for _, schema := range schemas {
		filename := filepath.Join(outputPath, g.getSchemaFileName(schema.Name))
		file, err := os.Create(filename)
		if err != nil {
			return ErrFileWriteFailed
		}

		schemaData := PerSchemaSchema{
			Schema: SchemaInfo{
				Name:   schema.Name,
				Tables: schema.Tables,
				Views:  schema.Views,
			},
			Metadata: SchemaMetadata{
				ExtractedAt:  schema.ExtractedAt,
				DatabaseInfo: schema.DatabaseInfo,
			},
		}

		if err := g.writeYAML(file, schemaData); err != nil {
			file.Close()
			return err
		}
		file.Close()
	}

	return nil
}

// writeYAML writes data to YAML format with appropriate styling
func (g *YAMLGenerator) writeYAML(writer io.Writer, data interface{}) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()

	// Apply flow style transformation if enabled
	if g.FlowStyle {
		data = g.applyFlowStyle(data)
	}

	if err := encoder.Encode(data); err != nil {
		return ErrYAMLGenerationFailed
	}

	return nil
}

// applyFlowStyle applies flow style to specific fields
func (g *YAMLGenerator) applyFlowStyle(data interface{}) interface{} {
	switch v := data.(type) {
	case SingleFileSchema:
		for i := range v.Schemas {
			if schema, ok := g.applyFlowStyleToSchema(v.Schemas[i]).(DatabaseSchema); ok {
				v.Schemas[i] = schema
			}
		}
		return v
	case PerTableSchema:
		if table, ok := v.Table.(TableSchema); ok {
			v.Table = g.applyFlowStyleToTable(table)
		}
		return v
	case PerSchemaSchema:
		// Create a new schema with flow style tables
		flowSchema := v.Schema
		flowTables := make([]TableSchema, len(v.Schema.Tables))
		for i, table := range v.Schema.Tables {
			if flowTable, ok := g.applyFlowStyleToTable(table).(FlowTableSchema); ok {
				// Convert FlowTableSchema back to TableSchema for per-schema format
				flowTables[i] = TableSchema{
					Name:        flowTable.Name,
					Schema:      flowTable.Schema,
					Columns:     convertFlowColumnsToRegular(flowTable.Columns),
					Constraints: convertFlowConstraintsToRegular(flowTable.Constraints),
					Indexes:     convertFlowIndexesToRegular(flowTable.Indexes),
					Comment:     flowTable.Comment,
				}
			} else {
				flowTables[i] = table
			}
		}
		flowSchema.Tables = flowTables
		return PerSchemaSchema{Schema: flowSchema}
	default:
		return data
	}
}

// applyFlowStyleToSchema applies flow style to a database schema
func (g *YAMLGenerator) applyFlowStyleToSchema(schema DatabaseSchema) interface{} {
	// Create a new schema with flow style tables
	flowSchema := schema
	flowTables := make([]TableSchema, len(schema.Tables))
	for i, table := range schema.Tables {
		if flowTable, ok := g.applyFlowStyleToTable(table).(FlowTableSchema); ok {
			// Convert FlowTableSchema back to TableSchema for database schema format
			flowTables[i] = TableSchema{
				Name:        flowTable.Name,
				Schema:      flowTable.Schema,
				Columns:     convertFlowColumnsToRegular(flowTable.Columns),
				Constraints: convertFlowConstraintsToRegular(flowTable.Constraints),
				Indexes:     convertFlowIndexesToRegular(flowTable.Indexes),
				Comment:     flowTable.Comment,
			}
		} else {
			flowTables[i] = table
		}
	}
	flowSchema.Tables = flowTables
	return flowSchema
}

// applyFlowStyleToTable applies flow style to table elements
func (g *YAMLGenerator) applyFlowStyleToTable(table TableSchema) interface{} {
	// Convert columns to flow style
	flowColumns := make([]FlowColumnSchema, len(table.Columns))
	for i, col := range table.Columns {
		flowColumns[i] = FlowColumnSchema(col)
	}

	// Convert constraints to flow style
	flowConstraints := make([]FlowConstraintSchema, len(table.Constraints))
	for i, constraint := range table.Constraints {
		flowConstraints[i] = FlowConstraintSchema(constraint)
	}

	// Convert indexes to flow style
	flowIndexes := make([]FlowIndexSchema, len(table.Indexes))
	for i, index := range table.Indexes {
		flowIndexes[i] = FlowIndexSchema(index)
	}

	return FlowTableSchema{
		Name:        table.Name,
		Schema:      table.Schema,
		Columns:     flowColumns,
		Constraints: flowConstraints,
		Indexes:     flowIndexes,
		Comment:     table.Comment,
	}
}

// Helper functions to convert flow style back to regular style for mixed formats

// convertFlowColumnsToRegular converts flow columns back to regular columns
func convertFlowColumnsToRegular(flowColumns []FlowColumnSchema) []ColumnSchema {
	columns := make([]ColumnSchema, len(flowColumns))
	for i, flowCol := range flowColumns {
		columns[i] = ColumnSchema(flowCol)
	}
	return columns
}

// convertFlowConstraintsToRegular converts flow constraints back to regular constraints
func convertFlowConstraintsToRegular(flowConstraints []FlowConstraintSchema) []ConstraintSchema {
	constraints := make([]ConstraintSchema, len(flowConstraints))
	for i, flowConstraint := range flowConstraints {
		constraints[i] = ConstraintSchema(flowConstraint)
	}
	return constraints
}

// convertFlowIndexesToRegular converts flow indexes back to regular indexes
func convertFlowIndexesToRegular(flowIndexes []FlowIndexSchema) []IndexSchema {
	indexes := make([]IndexSchema, len(flowIndexes))
	for i, flowIndex := range flowIndexes {
		indexes[i] = IndexSchema(flowIndex)
	}
	return indexes
}

// getSchemaPath returns the appropriate path for schema files
func (g *YAMLGenerator) getSchemaPath(outputPath, schemaName string) string {
	if !g.SchemaAware {
		return outputPath
	}

	// Use 'global' for empty or default schemas
	if schemaName == "" || schemaName == "main" {
		schemaName = "global"
	}

	return filepath.Join(outputPath, schemaName)
}

// getTableFileName returns the filename for a table YAML file
func (g *YAMLGenerator) getTableFileName(tableName string) string {
	return fmt.Sprintf("%s.yaml", tableName)
}

// getSchemaFileName returns the filename for a schema YAML file
func (g *YAMLGenerator) getSchemaFileName(schemaName string) string {
	if schemaName == "" || schemaName == "main" {
		schemaName = "global"
	}
	return fmt.Sprintf("%s.yaml", schemaName)
}

// Data structures for different output formats

// SingleFileSchema represents the structure for single file output
type SingleFileSchema struct {
	DatabaseInfo DatabaseInfo     `yaml:"database_info"`
	ExtractedAt  time.Time        `yaml:"extracted_at"`
	Schemas      []DatabaseSchema `yaml:"schemas"`
}

// PerTableSchema represents the structure for per-table output
type PerTableSchema struct {
	Table    interface{}    `yaml:"table"`
	Metadata SchemaMetadata `yaml:"metadata"`
}

// PerSchemaSchema represents the structure for per-schema output
type PerSchemaSchema struct {
	Schema   SchemaInfo     `yaml:"schema"`
	Metadata SchemaMetadata `yaml:"metadata"`
}

// SchemaInfo represents schema information for per-schema output
type SchemaInfo struct {
	Name   string        `yaml:"name"`
	Tables []TableSchema `yaml:"tables"`
	Views  []ViewSchema  `yaml:"views,omitempty"`
}

// SchemaMetadata represents metadata for schema files
type SchemaMetadata struct {
	ExtractedAt  time.Time    `yaml:"extracted_at"`
	DatabaseInfo DatabaseInfo `yaml:"database_info"`
}

// Flow style structures for compact YAML representation

// FlowTableSchema represents a table schema with flow style fields
type FlowTableSchema struct {
	Name        string                 `yaml:"name"`
	Schema      string                 `yaml:"schema,omitempty"`
	Columns     []FlowColumnSchema     `yaml:"columns,flow"`
	Constraints []FlowConstraintSchema `yaml:"constraints,omitempty,flow"`
	Indexes     []FlowIndexSchema      `yaml:"indexes,omitempty,flow"`
	Comment     string                 `yaml:"comment,omitempty"`
}

// FlowColumnSchema represents a column schema in flow style
type FlowColumnSchema struct {
	Name         string `yaml:"name" flow:"true"`
	Type         string `yaml:"type" flow:"true"`
	SnapSQLType  string `yaml:"snapsql_type" flow:"true"`
	Nullable     bool   `yaml:"nullable" flow:"true"`
	DefaultValue string `yaml:"default_value,omitempty" flow:"true"`
	Comment      string `yaml:"comment,omitempty" flow:"true"`
	IsPrimaryKey bool   `yaml:"is_primary_key,omitempty" flow:"true"`
}

// FlowConstraintSchema represents a constraint schema in flow style
type FlowConstraintSchema struct {
	Name              string   `yaml:"name" flow:"true"`
	Type              string   `yaml:"type" flow:"true"`
	Columns           []string `yaml:"columns" flow:"true"`
	ReferencedTable   string   `yaml:"referenced_table,omitempty" flow:"true"`
	ReferencedColumns []string `yaml:"referenced_columns,omitempty" flow:"true"`
	Definition        string   `yaml:"definition,omitempty" flow:"true"`
}

// FlowIndexSchema represents an index schema in flow style
type FlowIndexSchema struct {
	Name     string   `yaml:"name" flow:"true"`
	Columns  []string `yaml:"columns" flow:"true"`
	IsUnique bool     `yaml:"is_unique" flow:"true"`
	Type     string   `yaml:"type,omitempty" flow:"true"`
}

// Custom YAML marshaling for flow style
func (f FlowColumnSchema) MarshalYAML() (interface{}, error) {
	// Create a map for flow style representation
	m := make(map[string]interface{})
	m["name"] = f.Name
	m["type"] = f.Type
	m["snapsql_type"] = f.SnapSQLType
	m["nullable"] = f.Nullable

	if f.DefaultValue != "" {
		m["default_value"] = f.DefaultValue
	}
	if f.Comment != "" {
		m["comment"] = f.Comment
	}
	if f.IsPrimaryKey {
		m["is_primary_key"] = f.IsPrimaryKey
	}

	return m, nil
}

func (f FlowConstraintSchema) MarshalYAML() (interface{}, error) {
	m := make(map[string]interface{})
	m["name"] = f.Name
	m["type"] = f.Type
	m["columns"] = f.Columns

	if f.ReferencedTable != "" {
		m["referenced_table"] = f.ReferencedTable
	}
	if len(f.ReferencedColumns) > 0 {
		m["referenced_columns"] = f.ReferencedColumns
	}
	if f.Definition != "" {
		m["definition"] = f.Definition
	}

	return m, nil
}

func (f FlowIndexSchema) MarshalYAML() (interface{}, error) {
	m := make(map[string]interface{})
	m["name"] = f.Name
	m["columns"] = f.Columns
	m["is_unique"] = f.IsUnique

	if f.Type != "" {
		m["type"] = f.Type
	}

	return m, nil
}
