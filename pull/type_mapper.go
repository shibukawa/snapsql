package pull

import (
	"regexp"
	"strings"
)

// Standard SnapSQL types
const (
	TypeString   = "string"
	TypeInt      = "int"
	TypeFloat    = "float"
	TypeBool     = "bool"
	TypeDate     = "date"
	TypeTime     = "time"
	TypeDateTime = "datetime"
	TypeJSON     = "json"
	TypeArray    = "array"
	TypeBinary   = "binary"
)

// TypeMapper interface for mapping database-specific types to SnapSQL standard types
type TypeMapper interface {
	MapType(dbType string) string
	GetSnapSQLType(dbType string) string
}

// NewTypeMapper creates a new type mapper for the specified database type
func NewTypeMapper(databaseType string) (TypeMapper, error) {
	if databaseType == "" {
		return nil, ErrEmptyDatabaseType
	}

	switch strings.ToLower(databaseType) {
	case "postgresql", "postgres":
		return NewPostgreSQLTypeMapper(), nil
	case "mysql":
		return NewMySQLTypeMapper(), nil
	case "sqlite", "sqlite3":
		return NewSQLiteTypeMapper(), nil
	default:
		return nil, ErrUnsupportedDatabase
	}
}

// PostgreSQLTypeMapper handles PostgreSQL type mapping
type PostgreSQLTypeMapper struct {
	typeMap map[string]string
}

// NewPostgreSQLTypeMapper creates a new PostgreSQL type mapper
func NewPostgreSQLTypeMapper() *PostgreSQLTypeMapper {
	return &PostgreSQLTypeMapper{
		typeMap: map[string]string{
			// Integer types
			"integer":     TypeInt,
			"int":         TypeInt,
			"int4":        TypeInt,
			"bigint":      TypeInt,
			"int8":        TypeInt,
			"smallint":    TypeInt,
			"int2":        TypeInt,
			"serial":      TypeInt,
			"bigserial":   TypeInt,
			"smallserial": TypeInt,

			// String types
			"text":      TypeString,
			"varchar":   TypeString,
			"character": TypeString,
			"char":      TypeString,
			"bpchar":    TypeString,

			// Float types
			"numeric":          TypeFloat,
			"decimal":          TypeFloat,
			"real":             TypeFloat,
			"float4":           TypeFloat,
			"double precision": TypeFloat,
			"float8":           TypeFloat,
			"float":            TypeFloat,

			// Boolean types
			"boolean": TypeBool,
			"bool":    TypeBool,

			// Date/Time types
			"date":                        TypeDate,
			"time":                        TypeTime,
			"time with time zone":         TypeTime,
			"time without time zone":      TypeTime,
			"timetz":                      TypeTime,
			"timestamp":                   TypeDateTime,
			"timestamp with time zone":    TypeDateTime,
			"timestamp without time zone": TypeDateTime,
			"timestamptz":                 TypeDateTime,

			// JSON types
			"json":  TypeJSON,
			"jsonb": TypeJSON,

			// Binary types
			"bytea": TypeBinary,

			// Other types that map to string
			"uuid":     TypeString,
			"inet":     TypeString,
			"cidr":     TypeString,
			"macaddr":  TypeString,
			"interval": TypeString,
			"bit":      TypeString,
			"varbit":   TypeString,
		},
	}
}

// MapType maps a PostgreSQL type to a SnapSQL type
func (m *PostgreSQLTypeMapper) MapType(dbType string) string {
	return m.GetSnapSQLType(dbType)
}

// GetSnapSQLType maps a PostgreSQL type to a SnapSQL type
func (m *PostgreSQLTypeMapper) GetSnapSQLType(dbType string) string {
	// Normalize the type string
	normalized := strings.ToLower(strings.TrimSpace(dbType))

	// Check for array types
	if strings.HasSuffix(normalized, "[]") {
		return TypeArray
	}

	// Handle types with parameters (e.g., varchar(255), numeric(10,2))
	if strings.Contains(normalized, "(") {
		baseType := strings.Split(normalized, "(")[0]
		if mappedType, exists := m.typeMap[baseType]; exists {
			return mappedType
		}
	}

	// Direct mapping
	if mappedType, exists := m.typeMap[normalized]; exists {
		return mappedType
	}

	// Default fallback
	return TypeString
}

// MySQLTypeMapper handles MySQL type mapping
type MySQLTypeMapper struct {
	typeMap map[string]string
}

// NewMySQLTypeMapper creates a new MySQL type mapper
func NewMySQLTypeMapper() *MySQLTypeMapper {
	return &MySQLTypeMapper{
		typeMap: map[string]string{
			// Integer types
			"int":       TypeInt,
			"integer":   TypeInt,
			"bigint":    TypeInt,
			"smallint":  TypeInt,
			"tinyint":   TypeInt,
			"mediumint": TypeInt,

			// String types
			"varchar":    TypeString,
			"char":       TypeString,
			"text":       TypeString,
			"tinytext":   TypeString,
			"mediumtext": TypeString,
			"longtext":   TypeString,

			// Float types
			"decimal": TypeFloat,
			"numeric": TypeFloat,
			"float":   TypeFloat,
			"double":  TypeFloat,
			"real":    TypeFloat,

			// Boolean types
			"boolean": TypeBool,
			"bool":    TypeBool,

			// Date/Time types
			"date":      TypeDate,
			"time":      TypeTime,
			"datetime":  TypeDateTime,
			"timestamp": TypeDateTime,
			"year":      TypeInt,

			// JSON types
			"json": TypeJSON,

			// Binary types
			"blob":       TypeBinary,
			"tinyblob":   TypeBinary,
			"mediumblob": TypeBinary,
			"longblob":   TypeBinary,
			"binary":     TypeBinary,
			"varbinary":  TypeBinary,

			// Other types
			"enum": TypeString,
			"set":  TypeString,
		},
	}
}

// MapType maps a MySQL type to a SnapSQL type
func (m *MySQLTypeMapper) MapType(dbType string) string {
	return m.GetSnapSQLType(dbType)
}

// GetSnapSQLType maps a MySQL type to a SnapSQL type
func (m *MySQLTypeMapper) GetSnapSQLType(dbType string) string {
	// Normalize the type string
	normalized := strings.ToLower(strings.TrimSpace(dbType))

	// Special case for tinyint(1) which is boolean in MySQL
	if matched, _ := regexp.MatchString(`^tinyint\s*\(\s*1\s*\)`, normalized); matched {
		return TypeBool
	}

	// Handle types with parameters
	if strings.Contains(normalized, "(") {
		baseType := strings.Split(normalized, "(")[0]
		if mappedType, exists := m.typeMap[baseType]; exists {
			return mappedType
		}
	}

	// Handle unsigned types
	if strings.Contains(normalized, "unsigned") {
		baseType := strings.Fields(normalized)[0]
		if mappedType, exists := m.typeMap[baseType]; exists {
			return mappedType
		}
	}

	// Direct mapping
	if mappedType, exists := m.typeMap[normalized]; exists {
		return mappedType
	}

	// Default fallback
	return TypeString
}

// SQLiteTypeMapper handles SQLite type mapping
type SQLiteTypeMapper struct {
	typeMap map[string]string
}

// NewSQLiteTypeMapper creates a new SQLite type mapper
func NewSQLiteTypeMapper() *SQLiteTypeMapper {
	return &SQLiteTypeMapper{
		typeMap: map[string]string{
			// Integer types
			"integer":  TypeInt,
			"int":      TypeInt,
			"bigint":   TypeInt,
			"smallint": TypeInt,
			"tinyint":  TypeInt,

			// String types
			"text":      TypeString,
			"varchar":   TypeString,
			"char":      TypeString,
			"character": TypeString,
			"clob":      TypeString,
			"nchar":     TypeString,
			"nvarchar":  TypeString,

			// Float types
			"real":    TypeFloat,
			"double":  TypeFloat,
			"float":   TypeFloat,
			"numeric": TypeFloat,
			"decimal": TypeFloat,

			// Boolean types
			"boolean": TypeBool,
			"bool":    TypeBool,

			// Date/Time types
			"date":      TypeDate,
			"time":      TypeTime,
			"datetime":  TypeDateTime,
			"timestamp": TypeDateTime,

			// Binary types
			"blob": TypeBinary,
		},
	}
}

// MapType maps a SQLite type to a SnapSQL type
func (m *SQLiteTypeMapper) MapType(dbType string) string {
	return m.GetSnapSQLType(dbType)
}

// GetSnapSQLType maps a SQLite type to a SnapSQL type
func (m *SQLiteTypeMapper) GetSnapSQLType(dbType string) string {
	// Normalize the type string
	normalized := strings.ToLower(strings.TrimSpace(dbType))

	// Handle empty type (SQLite allows this)
	if normalized == "" {
		return TypeString
	}

	// Handle types with parameters
	if strings.Contains(normalized, "(") {
		baseType := strings.Split(normalized, "(")[0]
		if mappedType, exists := m.typeMap[baseType]; exists {
			return mappedType
		}
	}

	// Handle compound types (e.g., "unsigned big int", "varying character")
	words := strings.Fields(normalized)
	for _, word := range words {
		if mappedType, exists := m.typeMap[word]; exists {
			return mappedType
		}
	}

	// Direct mapping
	if mappedType, exists := m.typeMap[normalized]; exists {
		return mappedType
	}

	// SQLite is very flexible with types, default to string
	return TypeString
}
