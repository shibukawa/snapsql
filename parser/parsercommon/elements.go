package parsercommon

import tok "github.com/shibukawa/snapsql/tokenizer"

// FieldName represents a field/column name
type FieldName struct {
	Name       string
	TableName  string // Optional table qualifier
	Pos        tok.Position
	Expression []tok.Token // Optional expression for complex fields (e.g., "table.field->'key'")
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
	InvalidField                       // Invalid field (e.g., "*")
	DummyField                         // Dummy field (e.g., "/*= user.name */")
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
	case InvalidField:
		return "InvalidField"
	case DummyField:
		return "DummyField"
	default:
		return "UnknownFieldType"
	}
}

// SelectField represents an item in SELECT clause
type SelectField struct {
	FieldKind     FieldType
	OriginalField string
	TableName     string
	Expression    []tok.Token // For function fields or complex expressions
	TypeName      string      // Optional cast type
	ExplicitType  bool
	FieldName     string
	ExplicitName  bool
	Pos           tok.Position
}

func (n SelectField) String() string {
	return "SELECT_ITEM"
}

// JoinType constants for TableReference
type JoinType int

const (
	JoinNone JoinType = iota
	JoinInner
	JoinLeft
	JoinRight
	JoinFull
	JoinCross
	// Natural join family (inspect mode only; generation mode treats it as invalid)
	JoinNatural
	JoinNaturalLeft
	JoinNaturalRight
	JoinNaturalFull
	JoinInvalid
)

func (jt JoinType) String() string {
	switch jt {
	case JoinNone:
		return "NO JOIN"
	case JoinInner:
		return "INNER JOIN"
	case JoinLeft:
		return "LEFT OUTER JOIN"
	case JoinRight:
		return "RIGHT OUTER JOIN"
	case JoinFull:
		return "FULL OUTER JOIN"
	case JoinCross:
		return "CROSS JOIN"
	case JoinNatural:
		return "NATURAL JOIN"
	case JoinNaturalLeft:
		return "NATURAL LEFT JOIN"
	case JoinNaturalRight:
		return "NATURAL RIGHT JOIN"
	case JoinNaturalFull:
		return "NATURAL FULL JOIN"
	case JoinInvalid:
		return "invalid JOIN"
	default:
		return "unknown JOIN"
	}
}

// TableReferenceForFrom represents a table or join in FROM clause
type TableReferenceForFrom struct {
	TableReference

	JoinType      JoinType    // Join type (see constants)
	JoinCondition []tok.Token // ON/USING clause tokens
	IsSubquery    bool        // true if this is a subquery
	Expression    []tok.Token // Optional expression for complex references
}

func (n TableReferenceForFrom) String() string {
	return "TABLE_REF"
}

type TableReference struct {
	Name         string // Table name or alias (if present, otherwise original name)
	SchemaName   string // Optional schema name (e.g., "schema.table")
	TableName    string // Original table name before alias (empty for subquery)
	ExplicitName bool   // true if table name is explicitly specified (e.g., "AS alias")
}

type SetAssign struct {
	FieldName string
	Value     []tok.Token
}

// OrderByField represents a field in ORDER BY clause
type OrderByField struct {
	Field      FieldName
	Cast       string // Optional cast type
	Desc       bool   // true for DESC, false for ASC
	Extras     []tok.Token
	Expression []tok.Token // Expression for ORDER BY
}

func (n OrderByField) String() string {
	return "ORDER_FIELD"
}
