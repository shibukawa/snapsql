package parsercommon

import (
	"github.com/shibukawa/snapsql/tokenizer"
)

// AstNode represents AST (Abstract Syntax Tree) node interface
// All AST nodes must implement this interface.
type AstNode interface {
	Type() NodeType
	Position() tokenizer.Position
	String() string
	RawTokens() []tokenizer.Token // Returns the original token sequence
}

// NodeType represents the type of AST node
// This is used for type discrimination and debugging.
type NodeType int

const (
	// SQL statement structures
	SELECT_STATEMENT NodeType = iota
	INSERT_STATEMENT
	UPDATE_STATEMENT
	DELETE_STATEMENT

	// SQL clauses
	SELECT_CLAUSE
	FROM_CLAUSE
	WHERE_CLAUSE
	ORDER_BY_CLAUSE
	GROUP_BY_CLAUSE
	HAVING_CLAUSE
	LIMIT_CLAUSE
	OFFSET_CLAUSE
	SET_CLAUSE
	WITH_CLAUSE
	CTE_DEFINITION

	// SnapSQL extensions
	TEMPLATE_IF_BLOCK
	TEMPLATE_ELSEIF_BLOCK
	TEMPLATE_ELSE_BLOCK
	TEMPLATE_FOR_BLOCK
	VARIABLE_SUBSTITUTION
	DEFERRED_VARIABLE_SUBSTITUTION
	BULK_VARIABLE_SUBSTITUTION
	ENVIRONMENT_REFERENCE
	IMPLICIT_CONDITIONAL

	// Expressions and literals
	IDENTIFIER
	LITERAL
	EXPRESSION

	// Others
	OTHER_NODE
	RETURNING_CLAUSE
)

// String returns string representation of NodeType
func (n NodeType) String() string {
	switch n {
	case SELECT_STATEMENT:
		return "SELECT_STATEMENT"
	case INSERT_STATEMENT:
		return "INSERT_STATEMENT"
	case UPDATE_STATEMENT:
		return "UPDATE_STATEMENT"
	case DELETE_STATEMENT:
		return "DELETE_STATEMENT"
	case SELECT_CLAUSE:
		return "SELECT_CLAUSE"
	case FROM_CLAUSE:
		return "FROM_CLAUSE"
	case WHERE_CLAUSE:
		return "WHERE_CLAUSE"
	case ORDER_BY_CLAUSE:
		return "ORDER_BY_CLAUSE"
	case GROUP_BY_CLAUSE:
		return "GROUP_BY_CLAUSE"
	case HAVING_CLAUSE:
		return "HAVING_CLAUSE"
	case LIMIT_CLAUSE:
		return "LIMIT_CLAUSE"
	case OFFSET_CLAUSE:
		return "OFFSET_CLAUSE"
	case WITH_CLAUSE:
		return "WITH_CLAUSE"
	case CTE_DEFINITION:
		return "CTE_DEFINITION"
	case TEMPLATE_IF_BLOCK:
		return "TEMPLATE_IF_BLOCK"
	case TEMPLATE_ELSEIF_BLOCK:
		return "TEMPLATE_ELSEIF_BLOCK"
	case TEMPLATE_ELSE_BLOCK:
		return "TEMPLATE_ELSE_BLOCK"
	case TEMPLATE_FOR_BLOCK:
		return "TEMPLATE_FOR_BLOCK"
	case VARIABLE_SUBSTITUTION:
		return "VARIABLE_SUBSTITUTION"
	case BULK_VARIABLE_SUBSTITUTION:
		return "BULK_VARIABLE_SUBSTITUTION"
	case ENVIRONMENT_REFERENCE:
		return "ENVIRONMENT_REFERENCE"
	case IMPLICIT_CONDITIONAL:
		return "IMPLICIT_CONDITIONAL"
	case IDENTIFIER:
		return "IDENTIFIER"
	case LITERAL:
		return "LITERAL"
	case EXPRESSION:
		return "EXPRESSION"
	case OTHER_NODE:
		return "OTHER_NODE"
	case RETURNING_CLAUSE:
		return "RETURNING_CLAUSE"
	default:
		return "UNKNOWN"
	}
}

// BaseAstNode is the base implementation of AST nodes
// Embedding this struct is recommended for all AST node types.
type BaseAstNode struct {
	nodeType NodeType
	position tokenizer.Position
	tokens   []tokenizer.Token // Original token sequence
}

func (n BaseAstNode) Type() NodeType {
	return n.nodeType
}

func (n BaseAstNode) Position() tokenizer.Position {
	return n.position
}

func (n BaseAstNode) RawTokens() []tokenizer.Token {
	return n.tokens
}

// OptionalClause represents a clause that can be optionally included with if/end directive
// This is used for GROUP BY, HAVING, ORDER BY, LIMIT, OFFSET clauses that can be conditionally included.
type OptionalClause[T AstNode] struct {
	BaseAstNode
	Condition AstNode // The if condition expression (nil if not conditional)
	Clause    T       // The actual clause content
}

func (n OptionalClause[T]) String() string {
	if n.Condition != nil {
		return "Optional(" + n.Condition.String() + "): " + n.Clause.String()
	}
	return n.Clause.String()
}

// DML Statement structures

// SelectStatement represents SELECT statement
type SelectStatement struct {
	BaseAstNode
	WithClause    *OptionalClause[WithClause]
	SelectClause  SelectClause
	FromClause    *OptionalClause[FromClause]
	WhereClause   *OptionalClause[WhereClause]
	GroupByClause *OptionalClause[GroupByClause]
	HavingClause  *OptionalClause[HavingClause]
	OrderByClause *OptionalClause[OrderByClause]
	LimitClause   *OptionalClause[LimitClause]
	OffsetClause  *OptionalClause[OffsetClause]
}

func (n SelectStatement) String() string {
	return "SELECT"
}

// InsertStatement represents INSERT statement
type InsertStatement struct {
	BaseAstNode
	WithClause       *OptionalClause[WithClause]
	Table            TableName
	Columns          []FieldName
	ValuesList       Values
	SelectStmt       AstNode // Expression or SelectStatement
	OnConflictClause *OptionalClause[OnConflictClause]
	ReturningClause  *OptionalClause[ReturningClause]
}

func (n InsertStatement) String() string {
	return "INSERT"
}

// UpdateStatement represents UPDATE statement
type UpdateStatement struct {
	BaseAstNode
	WithClause      *OptionalClause[WithClause]
	Table           TableName
	SetClauses      []SetClause
	WhereClause     *OptionalClause[WhereClause]
	ReturningClause *OptionalClause[ReturningClause]
}

func (n UpdateStatement) String() string {
	return "UPDATE"
}

// DeleteStatement represents DELETE statement
type DeleteStatement struct {
	BaseAstNode
	WithClause      *OptionalClause[WithClause]
	Table           TableName
	WhereClause     *OptionalClause[WhereClause]
	ReturningClause *OptionalClause[ReturningClause]
}

func (n DeleteStatement) String() string {
	return "DELETE"
}

// Clause structures

// WithClause represents WITH clause for CTEs
type WithClause struct {
	BaseAstNode
	CTEs []CTEDefinition
}

func (n WithClause) String() string {
	return "WITH"
}

// SelectClause represents SELECT clause
type SelectClause struct {
	BaseAstNode
	Items []SelectItem
}

func (n SelectClause) String() string {
	return "SELECT"
}

// FromClause represents FROM clause
type FromClause struct {
	BaseAstNode
	Tables []TableReference
}

func (n FromClause) String() string {
	return "FROM"
}

// WhereClause represents WHERE clause
type WhereClause struct {
	BaseAstNode
	Condition AstNode // Expression
}

func (n WhereClause) String() string {
	return "WHERE"
}

// GroupByClause represents GROUP BY clause
type GroupByClause struct {
	BaseAstNode
	Fields []FieldName
}

func (n GroupByClause) String() string {
	return "GROUP BY"
}

// HavingClause represents HAVING clause
type HavingClause struct {
	BaseAstNode
	Condition AstNode // Expression
}

func (n HavingClause) String() string {
	return "HAVING"
}

// OrderByClause represents ORDER BY clause
type OrderByClause struct {
	BaseAstNode
	Fields []OrderByField
}

func (n OrderByClause) String() string {
	return "ORDER BY"
}

// LimitClause represents LIMIT clause
type LimitClause struct {
	BaseAstNode
	Count AstNode // Expression
}

func (n LimitClause) String() string {
	return "LIMIT"
}

// OffsetClause represents OFFSET clause
type OffsetClause struct {
	BaseAstNode
	Count AstNode // Expression
}

func (n OffsetClause) String() string {
	return "OFFSET"
}

// ReturningClause represents RETURNING clause
type ReturningClause struct {
	BaseAstNode
	Fields []FieldName
}

func (n ReturningClause) String() string {
	return "RETURNING"
}

// Helper structures

// CTEDefinition represents a Common Table Expression definition
type CTEDefinition struct {
	BaseAstNode
	Name      string
	Recursive bool
	Query     AstNode // SelectStatement
	Columns   []string
}

func (n CTEDefinition) String() string {
	return "CTE"
}

// TableName represents a table name
type TableName struct {
	BaseAstNode
	Name   string
	Schema string // Optional schema name
}

func (n TableName) String() string {
	return "TABLE"
}

// FieldName represents a field/column name
type FieldName struct {
	BaseAstNode
	Name      string
	TableName string // Optional table qualifier
}

func (n FieldName) String() string {
	return "FIELD"
}

// SelectItem represents an item in SELECT clause
type SelectItem struct {
	BaseAstNode
	Expression AstNode // Expression, FieldName, etc.
	Alias      string  // Optional alias
}

func (n SelectItem) String() string {
	return "SELECT_ITEM"
}

// TableReference represents a table reference in FROM clause
type TableReference struct {
	BaseAstNode
	Table AstNode // TableName, SubQuery, JoinClause, etc.
	Alias string  // Optional alias
}

func (n TableReference) String() string {
	return "TABLE_REF"
}

// OrderByField represents a field in ORDER BY clause
type OrderByField struct {
	BaseAstNode
	Field FieldName
	Desc  bool // true for DESC, false for ASC
}

func (n OrderByField) String() string {
	return "ORDER_FIELD"
}

// SetClause represents a SET clause in UPDATE statement
type SetClause struct {
	BaseAstNode
	Field FieldName
	Value AstNode // Expression
}

func (n SetClause) String() string {
	return "SET"
}

// OnConflictClause represents ON CONFLICT clause
type OnConflictClause struct {
	BaseAstNode
	Target []FieldName
	Action []SetClause
}

func (n OnConflictClause) String() string {
	return "ON_CONFLICT"
}

// Values represents VALUES in INSERT statement
type Values struct {
	BaseAstNode
	Rows [][]AstNode // Expression
}

func (n Values) String() string {
	return "VALUES"
}
