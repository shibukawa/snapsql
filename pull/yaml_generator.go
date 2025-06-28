package pull

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	snapsql "github.com/shibukawa/snapsql"
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
func (g *YAMLGenerator) Generate(schemas []snapsql.DatabaseSchema, outputPath string) error {
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
func (g *YAMLGenerator) generateSingleFile(schemas []snapsql.DatabaseSchema, outputPath string) error {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return ErrDirectoryCreateFailed
	}

	filename := filepath.Join(outputPath, "database_schema.yaml")
	file, err := os.Create(filename)
	if err != nil {
		return ErrFileWriteFailed
	}
	defer file.Close()

	return g.writeYAML(file, toYAMLSingleFileSchema(schemas))
}

// generatePerTable generates separate YAML files for each table
func (g *YAMLGenerator) generatePerTable(schemas []snapsql.DatabaseSchema, outputPath string) error {
	for _, schema := range schemas {
		yamlSchema := toYAMLSchema(schema)
		schemaPath := g.getSchemaPath(outputPath, schema.Name)
		if err := os.MkdirAll(schemaPath, 0755); err != nil {
			return ErrDirectoryCreateFailed
		}

		for _, table := range yamlSchema.Tables {
			filename := filepath.Join(schemaPath, g.getTableFileName(table.Name))
			file, err := os.Create(filename)
			if err != nil {
				return ErrFileWriteFailed
			}

			if err := g.writeYAML(file, table); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
	}

	return nil
}

// generatePerSchema generates separate YAML files for each schema
func (g *YAMLGenerator) generatePerSchema(schemas []snapsql.DatabaseSchema, outputPath string) error {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return ErrDirectoryCreateFailed
	}

	for _, schema := range schemas {
		yamlSchema := toYAMLSchema(schema)
		filename := filepath.Join(outputPath, g.getSchemaFileName(schema.Name))
		file, err := os.Create(filename)
		if err != nil {
			return ErrFileWriteFailed
		}

		if err := g.writeYAML(file, yamlSchema); err != nil {
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

	if err := encoder.Encode(data); err != nil {
		return ErrYAMLGenerationFailed
	}

	return nil
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

// --- 変換関数 ---
func toYAMLSingleFileSchema(schemas []snapsql.DatabaseSchema) YAMLSingleFileSchema {
	var yamlSchemas []YAMLSchema
	for _, s := range schemas {
		yamlSchemas = append(yamlSchemas, toYAMLSchema(s))
	}
	return YAMLSingleFileSchema{
		DatabaseInfo: schemas[0].DatabaseInfo,
		ExtractedAt:  schemas[0].ExtractedAt,
		Schemas:      yamlSchemas,
	}
}

func toYAMLSchema(s snapsql.DatabaseSchema) YAMLSchema {
	var tables []YAMLTable
	for _, t := range s.Tables {
		tables = append(tables, toYAMLTable(t))
	}
	var views []YAMLView
	for _, v := range s.Views {
		views = append(views, toYAMLView(v))
	}
	return YAMLSchema{
		Name:         s.Name,
		Tables:       tables,
		Views:        views,
		ExtractedAt:  s.ExtractedAt,
		DatabaseInfo: s.DatabaseInfo,
	}
}

func toYAMLTable(t *snapsql.TableInfo) YAMLTable {
	var columns []YAMLColumn
	for _, c := range t.Columns {
		columns = append(columns, toYAMLColumn(c))
	}
	var constraints []YAMLConstraint
	for _, c := range t.Constraints {
		constraints = append(constraints, toYAMLConstraint(c))
	}
	var indexes []YAMLIndex
	for _, i := range t.Indexes {
		indexes = append(indexes, toYAMLIndex(i))
	}
	return YAMLTable{
		Name:        t.Name,
		Schema:      t.Schema,
		Columns:     columns,
		Constraints: constraints,
		Indexes:     indexes,
		Comment:     t.Comment,
	}
}

func toYAMLColumn(c *snapsql.ColumnInfo) YAMLColumn {
	return YAMLColumn{
		Name:         c.Name,
		DataType:     c.DataType,
		Nullable:     c.Nullable,
		DefaultValue: c.DefaultValue,
		Comment:      c.Comment,
		IsPrimaryKey: c.IsPrimaryKey,
		MaxLength:    c.MaxLength,
		Precision:    c.Precision,
		Scale:        c.Scale,
	}
}

func toYAMLConstraint(c snapsql.ConstraintInfo) YAMLConstraint {
	return YAMLConstraint{
		Name:              c.Name,
		Type:              c.Type,
		Columns:           c.Columns,
		ReferencedTable:   c.ReferencedTable,
		ReferencedColumns: c.ReferencedColumns,
		Definition:        c.Definition,
	}
}

func toYAMLIndex(i snapsql.IndexInfo) YAMLIndex {
	return YAMLIndex{
		Name:     i.Name,
		Columns:  i.Columns,
		IsUnique: i.IsUnique,
		Type:     i.Type,
	}
}

func toYAMLView(v *snapsql.ViewInfo) YAMLView {
	return YAMLView{
		Name:       v.Name,
		Schema:     v.Schema,
		Definition: v.Definition,
		Comment:    v.Comment,
	}
}
