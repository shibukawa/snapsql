package pull

import "time"

// DatabaseSchema represents the complete schema information for a database
type DatabaseSchema struct {
	Name         string        `yaml:"name"`
	Tables       []TableSchema `yaml:"tables"`
	Views        []ViewSchema  `yaml:"views,omitempty"`
	ExtractedAt  time.Time     `yaml:"extracted_at"`
	DatabaseInfo DatabaseInfo  `yaml:"database_info"`
}

// TableSchema represents the schema information for a single table
type TableSchema struct {
	Name        string             `yaml:"name"`
	Schema      string             `yaml:"schema,omitempty"`
	Columns     []ColumnSchema     `yaml:"columns,flow"`
	Constraints []ConstraintSchema `yaml:"constraints,omitempty,flow"`
	Indexes     []IndexSchema      `yaml:"indexes,omitempty,flow"`
	Comment     string             `yaml:"comment,omitempty"`
}

// ColumnSchema represents the schema information for a single column
type ColumnSchema struct {
	Name         string `yaml:"name"`
	Type         string `yaml:"type"`
	SnapSQLType  string `yaml:"snapsql_type"`
	Nullable     bool   `yaml:"nullable"`
	DefaultValue string `yaml:"default_value,omitempty"`
	Comment      string `yaml:"comment,omitempty"`
	IsPrimaryKey bool   `yaml:"is_primary_key,omitempty"`
}

// ConstraintSchema represents the schema information for a table constraint
type ConstraintSchema struct {
	Name              string   `yaml:"name"`
	Type              string   `yaml:"type"` // PRIMARY_KEY, FOREIGN_KEY, UNIQUE, CHECK
	Columns           []string `yaml:"columns"`
	ReferencedTable   string   `yaml:"referenced_table,omitempty"`
	ReferencedColumns []string `yaml:"referenced_columns,omitempty"`
	Definition        string   `yaml:"definition,omitempty"`
}

// IndexSchema represents the schema information for a table index
type IndexSchema struct {
	Name     string   `yaml:"name"`
	Columns  []string `yaml:"columns"`
	IsUnique bool     `yaml:"is_unique"`
	Type     string   `yaml:"type,omitempty"`
}

// ViewSchema represents the schema information for a database view
type ViewSchema struct {
	Name       string `yaml:"name"`
	Schema     string `yaml:"schema,omitempty"`
	Definition string `yaml:"definition"`
	Comment    string `yaml:"comment,omitempty"`
}

// DatabaseInfo represents basic information about the database
type DatabaseInfo struct {
	Type    string `yaml:"type"`
	Version string `yaml:"version"`
	Name    string `yaml:"name"`
	Charset string `yaml:"charset,omitempty"`
}
