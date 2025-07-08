package parsercommon

import tok "github.com/shibukawa/snapsql/tokenizer"

// TableName represents a table name
type TableName struct {
	Name   string
	Schema string // Optional schema name
}

func (n TableName) String() string {
	return "TABLE"
}

// FieldName represents a field/column name
type FieldName struct {
	Name      string
	TableName string // Optional table qualifier
	Pos       tok.Position
}

func (n FieldName) String() string {
	return "FIELD"
}

type FieldType int

const (
	SingleField   FieldType = iota + 1 // Single field (e.g., "field")
	TableField                         // Field with table name (e.g., "table.field")
	FunctionField                      // Function field (e.g., "COUNT(field)")
	ComplexField                       // Complex field (e.g., "table.field->'key'")
	LiteralField                       // Literal field (e.g., "123", "'string'", "NULL")
)

func (ft FieldType) String() string {
	switch ft {
	case SingleField:
		return "SingleField"
	case TableField:
		return "TableField"
	case FunctionField:
		return "FunctionField"
	case ComplexField:
		return "ComplexField"
	case LiteralField:
		return "LiteralField"
	default:
		return "UnknownFieldType"
	}
}

// SelectField represents an item in SELECT clause
type SelectField struct {
	FieldKind    FieldType
	Expression   []tok.Token // Expression, FieldName, etc
	TypeName     string      // Optional cast type
	ExplicitType bool
	FieldName    string
	ExplicitName bool
	Pos          tok.Position
}

func (n SelectField) String() string {
	return "SELECT_ITEM"
}

// TableReference represents a table reference in FROM clause
type TableReference struct {
	Table AstNode // TableName, SubQuery, JoinClause, etc.
	Alias string  // Optional alias
}

func (n TableReference) String() string {
	return "TABLE_REF"
}
