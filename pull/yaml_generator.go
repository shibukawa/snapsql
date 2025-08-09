package pull

import (
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	snapsql "github.com/shibukawa/snapsql"
)

// YAMLGenerator generates YAML schema files from database schemas
type YAMLGenerator struct {
	SchemaAware bool // Enable schema-aware directory structure
}

// NewYAMLGenerator creates a new YAML generator
func NewYAMLGenerator(schemaAware bool) *YAMLGenerator {
	return &YAMLGenerator{
		SchemaAware: schemaAware,
	}
}

// Generate generates YAML files from database schemas (PerTableのみ)
func (g *YAMLGenerator) Generate(schemas []snapsql.DatabaseSchema, outputPath string) error {
	return g.generatePerTable(schemas, outputPath)
}

// generatePerTable generates separate YAML files for each table
func (g *YAMLGenerator) generatePerTable(schemas []snapsql.DatabaseSchema, outputPath string) error {
	for _, schema := range schemas {
		yamlSchema := toYAMLSchema(schema)

		schemaPath := g.getSchemaPath(outputPath, schema.Name)

		err := os.MkdirAll(schemaPath, 0755)
		if err != nil {
			return ErrDirectoryCreateFailed
		}

		for _, table := range yamlSchema.Tables {
			filename := filepath.Join(schemaPath, g.getTableFileName(table.Name))

			file, err := os.Create(filename)
			if err != nil {
				return ErrFileWriteFailed
			}

			// metadataラッパー+tableラッパー
			out := map[string]interface{}{
				"metadata": yamlSchema,
				"table":    table,
			}
			if err := g.writeYAML(file, out); err != nil {
				file.Close()
				return err
			}

			file.Close()
		}
	}

	return nil
}

// writeYAML writes data to YAML format with appropriate styling
func (g *YAMLGenerator) writeYAML(writer io.Writer, data interface{}) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()

	// ラッパー付与: テーブル単位の場合はtable:、スキーマ単位はmetadata:等
	switch v := data.(type) {
	case YAMLTable:
		return encoder.Encode(map[string]interface{}{"table": v})
	case YAMLSchema:
		return encoder.Encode(map[string]interface{}{"metadata": v})
	default:
		err := encoder.Encode(data)
		if err != nil {
			return ErrYAMLGenerationFailed
		}
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
	return tableName + ".yaml"
}

// getSchemaFileName returns the filename for a schema YAML file
func (g *YAMLGenerator) getSchemaFileName(schemaName string) string {
	if schemaName == "" || schemaName == "main" {
		schemaName = "global"
	}

	return schemaName + ".yaml"
}

// --- 変換関数 ---
func toYAMLSchema(s snapsql.DatabaseSchema) YAMLSchema {
	tables := make([]YAMLTable, len(s.Tables))
	for i, t := range s.Tables {
		tables[i] = toYAMLTable(t, s.DatabaseInfo.Type)
	}

	views := make([]YAMLView, len(s.Views))
	for i, v := range s.Views {
		views[i] = toYAMLView(v)
	}

	return YAMLSchema{
		Name:         s.Name,
		Tables:       tables,
		Views:        views,
		DatabaseInfo: s.DatabaseInfo,
	}
}

func toYAMLTable(t *snapsql.TableInfo, dbType string) YAMLTable {
	columns := make([]YAMLColumn, 0, len(t.Columns))
	for _, c := range t.Columns {
		columns = append(columns, toYAMLColumn(c, dbType))
	}

	constraints := make([]YAMLConstraint, len(t.Constraints))
	for i, c := range t.Constraints {
		constraints[i] = toYAMLConstraint(c)
	}

	indexes := make([]YAMLIndex, len(t.Indexes))
	for i, idx := range t.Indexes {
		indexes[i] = toYAMLIndex(idx)
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

func toYAMLColumn(c *snapsql.ColumnInfo, dbType string) YAMLColumn {
	mapper, err := NewTypeMapper(dbType)

	var snapType string
	if err == nil {
		snapType = mapper.GetSnapSQLType(c.DataType)
	} else {
		snapType = "string"
	}

	return YAMLColumn{
		Name:         c.Name,
		DataType:     c.DataType,
		SnapSQLType:  snapType,
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
