package snapsql

// ColumnInfo is a unified column definition for schema, type inference, etc.
type ColumnInfo struct {
	Name         string `json:"name" yaml:"name"`                 // Column name
	DataType     string `json:"dataType" yaml:"dataType"`         // Normalized type (snapsql type)
	Nullable     bool   `json:"nullable" yaml:"nullable"`         // Is nullable
	DefaultValue string `json:"defaultValue" yaml:"defaultValue"` // Default value (optional)
	Comment      string `json:"comment" yaml:"comment"`           // Comment (optional)
	IsPrimaryKey bool   `json:"isPrimaryKey" yaml:"isPrimaryKey"` // Is primary key (optional)
	MaxLength    *int   `json:"maxLength" yaml:"maxLength"`       // For string types (optional)
	Precision    *int   `json:"precision" yaml:"precision"`       // For numeric types (optional)
	Scale        *int   `json:"scale" yaml:"scale"`               // For numeric types (optional)
}

// TableInfo is a unified table definition
type TableInfo struct {
	Name        string                 `json:"name" yaml:"name"`               // Table name
	Schema      string                 `json:"schema" yaml:"schema"`           // Schema name (optional)
	Columns     map[string]*ColumnInfo `json:"columns" yaml:"columns"`         // Columns by name
	Constraints []ConstraintInfo       `json:"constraints" yaml:"constraints"` // Constraints (optional)
	Indexes     []IndexInfo            `json:"indexes" yaml:"indexes"`         // Indexes (optional)
	Comment     string                 `json:"comment" yaml:"comment"`         // Table comment (optional)
}

// DatabaseSchema is a unified database schema definition
type DatabaseSchema struct {
	Name         string       `json:"name" yaml:"name"`                 // Schema/database name
	Tables       []*TableInfo `json:"tables" yaml:"tables"`             // Tables
	Views        []*ViewInfo  `json:"views" yaml:"views"`               // Views (optional)
	DatabaseInfo DatabaseInfo `json:"databaseInfo" yaml:"databaseInfo"` // DB info
}

type ConstraintInfo struct {
	Name              string   `json:"name" yaml:"name"`
	Type              string   `json:"type" yaml:"type"` // PRIMARY_KEY, FOREIGN_KEY, UNIQUE, CHECK
	Columns           []string `json:"columns" yaml:"columns"`
	ReferencedTable   string   `json:"referencedTable" yaml:"referencedTable"`
	ReferencedColumns []string `json:"referencedColumns" yaml:"referencedColumns"`
	Definition        string   `json:"definition" yaml:"definition"`
}

type IndexInfo struct {
	Name     string   `json:"name" yaml:"name"`
	Columns  []string `json:"columns" yaml:"columns"`
	IsUnique bool     `json:"isUnique" yaml:"isUnique"`
	Type     string   `json:"type" yaml:"type"`
}

type ViewInfo struct {
	Name       string `json:"name" yaml:"name"`
	Schema     string `json:"schema" yaml:"schema"`
	Definition string `json:"definition" yaml:"definition"`
	Comment    string `json:"comment" yaml:"comment"`
}

type DatabaseInfo struct {
	Type    string `json:"type" yaml:"type"`
	Version string `json:"version" yaml:"version"`
	Name    string `json:"name" yaml:"name"`
	Charset string `json:"charset" yaml:"charset"`
}
