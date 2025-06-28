package pull

import (
	"github.com/shibukawa/snapsql"
)

// YAML output structure for a single file (database_schema.yaml)
type YAMLSingleFileSchema struct {
	DatabaseInfo snapsql.DatabaseInfo `yaml:"database_info"`
	ExtractedAt  string               `yaml:"extracted_at"`
	Schemas      []YAMLSchema         `yaml:"schemas"`
}

type YAMLSchema struct {
	Name         string               `yaml:"name"`
	Tables       []YAMLTable          `yaml:"tables"`
	Views        []YAMLView           `yaml:"views,omitempty"`
	ExtractedAt  string               `yaml:"extracted_at"`
	DatabaseInfo snapsql.DatabaseInfo `yaml:"database_info"`
}

type YAMLTable struct {
	Name        string           `yaml:"name"`
	Schema      string           `yaml:"schema,omitempty"`
	Columns     []YAMLColumn     `yaml:"columns"`
	Constraints []YAMLConstraint `yaml:"constraints,omitempty"`
	Indexes     []YAMLIndex      `yaml:"indexes,omitempty"`
	Comment     string           `yaml:"comment,omitempty"`
}

type YAMLColumn struct {
	Name         string `yaml:"name"`
	DataType     string `yaml:"data_type"`
	Nullable     bool   `yaml:"nullable"`
	DefaultValue string `yaml:"default_value,omitempty"`
	Comment      string `yaml:"comment,omitempty"`
	IsPrimaryKey bool   `yaml:"is_primary_key,omitempty"`
	MaxLength    *int   `yaml:"max_length,omitempty"`
	Precision    *int   `yaml:"precision,omitempty"`
	Scale        *int   `yaml:"scale,omitempty"`
}

type YAMLConstraint struct {
	Name              string   `yaml:"name"`
	Type              string   `yaml:"type"`
	Columns           []string `yaml:"columns"`
	ReferencedTable   string   `yaml:"referenced_table,omitempty"`
	ReferencedColumns []string `yaml:"referenced_columns,omitempty"`
	Definition        string   `yaml:"definition,omitempty"`
}

type YAMLIndex struct {
	Name     string   `yaml:"name"`
	Columns  []string `yaml:"columns"`
	IsUnique bool     `yaml:"is_unique"`
	Type     string   `yaml:"type,omitempty"`
}

type YAMLView struct {
	Name       string `yaml:"name"`
	Schema     string `yaml:"schema,omitempty"`
	Definition string `yaml:"definition"`
	Comment    string `yaml:"comment,omitempty"`
}
