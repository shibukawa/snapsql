package snapsql

// ColumnInfo is a unified column definition for schema, type inference, etc.
type ColumnInfo struct {
	Name         string // Column name
	DataType     string // Normalized type (snapsql type)
	Nullable     bool   // Is nullable
	DefaultValue string // Default value (optional)
	Comment      string // Comment (optional)
	IsPrimaryKey bool   // Is primary key (optional)
	MaxLength    *int   // For string types (optional)
	Precision    *int   // For numeric types (optional)
	Scale        *int   // For numeric types (optional)
}

// TableInfo is a unified table definition
type TableInfo struct {
	Name        string                 // Table name
	Schema      string                 // Schema name (optional)
	Columns     map[string]*ColumnInfo // Columns by name
	Constraints []ConstraintInfo       // Constraints (optional)
	Indexes     []IndexInfo            // Indexes (optional)
	Comment     string                 // Table comment (optional)
}

// DatabaseSchema is a unified database schema definition
type DatabaseSchema struct {
	Name         string       // Schema/database name
	Tables       []*TableInfo // Tables
	Views        []*ViewInfo  // Views (optional)
	DatabaseInfo DatabaseInfo // DB info
}

type ConstraintInfo struct {
	Name              string
	Type              string // PRIMARY_KEY, FOREIGN_KEY, UNIQUE, CHECK
	Columns           []string
	ReferencedTable   string
	ReferencedColumns []string
	Definition        string
}

type IndexInfo struct {
	Name     string
	Columns  []string
	IsUnique bool
	Type     string
}

type ViewInfo struct {
	Name       string
	Schema     string
	Definition string
	Comment    string
}

type DatabaseInfo struct {
	Type    string
	Version string
	Name    string
	Charset string
}
