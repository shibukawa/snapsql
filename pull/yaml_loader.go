package pull

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/goccy/go-yaml"
	snapsql "github.com/shibukawa/snapsql"
)

// LoadTableFromYAMLFile loads a TableInfo from a per-table YAML file
func LoadTableFromYAMLFile(path string) (*snapsql.TableInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Metadata YAMLSchema `yaml:"metadata"`
		Table    YAMLTable  `yaml:"table"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	return yamlTableToTableInfo(&wrapper.Table), nil
}

// LoadDatabaseSchemaFromDir loads a DatabaseSchema from a schema directory (per-table YAMLs)
func LoadDatabaseSchemaFromDir(dir string) (*snapsql.DatabaseSchema, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	dbSchema := &snapsql.DatabaseSchema{}
	tables := []*snapsql.TableInfo{}
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".yaml" {
			continue
		}
		table, err := LoadTableFromYAMLFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	// テーブル名でソート
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})
	if len(tables) > 0 {
		dbSchema.Name = tables[0].Schema
	}
	dbSchema.Tables = tables
	return dbSchema, nil
}

// --- 型変換 ---
func yamlTableToTableInfo(y *YAMLTable) *snapsql.TableInfo {
	columns := map[string]*snapsql.ColumnInfo{}
	for _, c := range y.Columns {
		columns[c.Name] = &snapsql.ColumnInfo{
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
	constraints := []snapsql.ConstraintInfo{}
	for _, c := range y.Constraints {
		constraints = append(constraints, snapsql.ConstraintInfo{
			Name:              c.Name,
			Type:              c.Type,
			Columns:           c.Columns,
			ReferencedTable:   c.ReferencedTable,
			ReferencedColumns: c.ReferencedColumns,
			Definition:        c.Definition,
		})
	}
	indexes := []snapsql.IndexInfo{}
	for _, i := range y.Indexes {
		indexes = append(indexes, snapsql.IndexInfo{
			Name:     i.Name,
			Columns:  i.Columns,
			IsUnique: i.IsUnique,
			Type:     i.Type,
		})
	}
	return &snapsql.TableInfo{
		Name:        y.Name,
		Schema:      y.Schema,
		Columns:     columns,
		Constraints: constraints,
		Indexes:     indexes,
		Comment:     y.Comment,
	}
}
