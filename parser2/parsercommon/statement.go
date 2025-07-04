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

// DML Statement structures

type StatementNode interface {
	AstNode
	CTEs() []CTEDefinition            // Returns CTE definitions in the block
	LeadingTokens() []tokenizer.Token // Returns leading tokens of the block
	Clauses() []ClauseNode            // Returns clauses in the block
}

// SelectStatement represents SELECT statement
type SelectStatement struct {
	WithClause    *WithClause
	SelectClause  *SelectClause
	FromClause    *FromClause
	WhereClause   *WhereClause
	GroupByClause *GroupByClause
	HavingClause  *HavingClause
	OrderByClause *OrderByClause
	LimitClause   *LimitClause
	OffsetClause  *OffsetClause
	leadingTokens []tokenizer.Token // Leading tokens before the SELECT statement
	cteClauses    []CTEDefinition   // CTE definitions in the SELECT statement
	clauses       []ClauseNode      // All clauses in the statement
}

func NewSelectStatement(leadingTokens []tokenizer.Token, cteClauses []CTEDefinition, clauses []ClauseNode) *SelectStatement {
	return &SelectStatement{
		leadingTokens: leadingTokens,
		cteClauses:    cteClauses,
		clauses:       clauses,
	}
}

// Clauses implements BlockNode.
func (s *SelectStatement) Clauses() []ClauseNode {
	return s.clauses
}

// LeadingTokens implements BlockNode.
func (s *SelectStatement) LeadingTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Position implements BlockNode.
func (s *SelectStatement) Position() tokenizer.Position {
	panic("unimplemented")
}

// RawTokens implements BlockNode.
func (s *SelectStatement) RawTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Type implements BlockNode.
func (s *SelectStatement) Type() NodeType {
	return SELECT_STATEMENT
}

// CTEs implements BlockNode.
func (s *SelectStatement) CTEs() []CTEDefinition {
	return s.cteClauses
}

var _ StatementNode = (*SelectStatement)(nil)

func (s SelectStatement) String() string {
	return "SELECT"
}

// InsertStatement represents INSERT statement
type InsertStatement struct {
	WithClause       *WithClause
	Table            TableName
	Columns          []FieldName
	ValuesList       *Values
	SelectStmt       *AstNode // Expression or SelectStatement
	OnConflictClause *OnConflictClause
	ReturningClause  *ReturningClause
}

// Clauses implements BlockNode.
func (n *InsertStatement) Clauses() []ClauseNode {
	panic("unimplemented")
}

// LeadingTokens implements BlockNode.
func (n *InsertStatement) LeadingTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Position implements BlockNode.
func (n *InsertStatement) Position() tokenizer.Position {
	panic("unimplemented")
}

// RawTokens implements BlockNode.
func (n *InsertStatement) RawTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Type implements BlockNode.
func (n *InsertStatement) Type() NodeType {
	panic("unimplemented")
}

func (n InsertStatement) String() string {
	return "INSERT"
}

// CTEs implements BlockNode.
func (n *InsertStatement) CTEs() []CTEDefinition {
	panic("unimplemented")
}

var _ StatementNode = (*InsertStatement)(nil)

// UpdateStatement represents UPDATE statement
type UpdateStatement struct {
	WithClause      *WithClause
	Table           TableName
	SetClauses      []SetClause
	WhereClause     *WhereClause
	ReturningClause *ReturningClause
}

func (n UpdateStatement) String() string {
	return "UPDATE"
}

// CTEs implements BlockNode.
func (n *UpdateStatement) CTEs() []CTEDefinition {
	panic("unimplemented")
}

var _ StatementNode = (*InsertStatement)(nil)

// DeleteStatement represents DELETE statement
type DeleteStatement struct {
	WithClause      *WithClause
	Table           TableName
	WhereClause     *WhereClause
	ReturningClause *ReturningClause
}

// Clauses implements BlockNode.
func (n *DeleteStatement) Clauses() []ClauseNode {
	panic("unimplemented")
}

// LeadingTokens implements BlockNode.
func (n *DeleteStatement) LeadingTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Position implements BlockNode.
func (n *DeleteStatement) Position() tokenizer.Position {
	panic("unimplemented")
}

// RawTokens implements BlockNode.
func (n *DeleteStatement) RawTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Type implements BlockNode.
func (n *DeleteStatement) Type() NodeType {
	panic("unimplemented")
}

func (n DeleteStatement) String() string {
	return "DELETE"
}

// CTEs implements BlockNode.
func (n *DeleteStatement) CTEs() []CTEDefinition {
	panic("unimplemented")
}

var _ StatementNode = (*DeleteStatement)(nil)
