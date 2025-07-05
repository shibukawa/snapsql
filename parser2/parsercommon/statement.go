package parsercommon

import (
	"github.com/shibukawa/snapsql/tokenizer"
)

// DML Statement structures

type StatementNode interface {
	AstNode
	CTEs() []CTEDefinition            // Returns CTE definitions in the block
	LeadingTokens() []tokenizer.Token // Returns leading tokens of the block
	Clauses() []ClauseNode            // Returns clauses in the block
}

type baseStatement struct {
	leadingTokens []tokenizer.Token // Leading tokens before the SELECT statement
	cteClauses    []CTEDefinition   // CTE definitions in the SELECT statement
	clauses       []ClauseNode      // All clauses in the statement
}

// SelectStatement represents SELECT statement
type SelectStatement struct {
	baseStatement
	WithClause    *WithClause
	SelectClause  *SelectClause
	FromClause    *FromClause
	WhereClause   *WhereClause
	GroupByClause *GroupByClause
	HavingClause  *HavingClause
	OrderByClause *OrderByClause
	LimitClause   *LimitClause
	OffsetClause  *OffsetClause
}

func NewSelectStatement(leadingTokens []tokenizer.Token, cteClauses []CTEDefinition, clauses []ClauseNode) *SelectStatement {
	return &SelectStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			cteClauses:    cteClauses,
			clauses:       clauses,
		},
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
type InsertIntoStatement struct {
	baseStatement
	WithClause       *WithClause
	Table            TableName
	Columns          []FieldName
	ValuesList       *ValuesClause
	SelectStmt       *AstNode // Expression or SelectStatement
	OnConflictClause *OnConflictClause
	ReturningClause  *ReturningClause
}

func NewInsertIntoStatement(leadingTokens []tokenizer.Token, cteClauses []CTEDefinition, clauses []ClauseNode) *InsertIntoStatement {
	return &InsertIntoStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			cteClauses:    cteClauses,
			clauses:       clauses,
		},
	}
}

// Clauses implements BlockNode.
func (n *InsertIntoStatement) Clauses() []ClauseNode {
	return n.clauses
}

// LeadingTokens implements BlockNode.
func (n *InsertIntoStatement) LeadingTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Position implements BlockNode.
func (n *InsertIntoStatement) Position() tokenizer.Position {
	panic("unimplemented")
}

// RawTokens implements BlockNode.
func (n *InsertIntoStatement) RawTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Type implements BlockNode.
func (n *InsertIntoStatement) Type() NodeType {
	return INSERT_INTO_STATEMENT
}

func (n InsertIntoStatement) String() string {
	return "INSERT INTO"
}

// CTEs implements BlockNode.
func (n *InsertIntoStatement) CTEs() []CTEDefinition {
	panic("unimplemented")
}

var _ StatementNode = (*InsertIntoStatement)(nil)

// UpdateStatement represents UPDATE statement
type UpdateStatement struct {
	baseStatement
	WithClause      *WithClause
	Table           TableName
	SetClauses      []SetClause
	WhereClause     *WhereClause
	ReturningClause *ReturningClause
}

// NewUpdateStatement creates a new UpdateStatement node.
func NewUpdateStatement(leadingTokens []tokenizer.Token, cteClauses []CTEDefinition, clauses []ClauseNode) *UpdateStatement {
	return &UpdateStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			cteClauses:    cteClauses,
			clauses:       clauses,
		},
	}
}

// Clauses implements StatementNode.
func (n *UpdateStatement) Clauses() []ClauseNode {
	return n.clauses
}

// LeadingTokens implements StatementNode.
func (n *UpdateStatement) LeadingTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Position implements StatementNode.
func (n *UpdateStatement) Position() tokenizer.Position {
	panic("unimplemented")
}

// RawTokens implements StatementNode.
func (n *UpdateStatement) RawTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Type implements StatementNode.
func (n *UpdateStatement) Type() NodeType {
	return UPDATE_STATEMENT
}

func (n UpdateStatement) String() string {
	return "UPDATE"
}

// CTEs implements BlockNode.
func (n *UpdateStatement) CTEs() []CTEDefinition {
	panic("unimplemented")
}

var _ StatementNode = (*UpdateStatement)(nil)

// DeleteStatement represents DELETE statement
type DeleteFromStatement struct {
	baseStatement
	WithClause      *WithClause
	Table           TableName
	WhereClause     *WhereClause
	ReturningClause *ReturningClause
}

// NewDeleteFromStatement creates a new DeleteFromStatement node.
func NewDeleteFromStatement(leadingTokens []tokenizer.Token, cteClauses []CTEDefinition, clauses []ClauseNode) *DeleteFromStatement {
	return &DeleteFromStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			cteClauses:    cteClauses,
			clauses:       clauses,
		},
	}
}

// Clauses implements BlockNode.
func (n *DeleteFromStatement) Clauses() []ClauseNode {
	return n.clauses
}

// LeadingTokens implements BlockNode.
func (n *DeleteFromStatement) LeadingTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Position implements BlockNode.
func (n *DeleteFromStatement) Position() tokenizer.Position {
	panic("unimplemented")
}

// RawTokens implements BlockNode.
func (n *DeleteFromStatement) RawTokens() []tokenizer.Token {
	panic("unimplemented")
}

// Type implements BlockNode.
func (n *DeleteFromStatement) Type() NodeType {
	return DELETE_FROM_STATEMENT
}

func (n DeleteFromStatement) String() string {
	return "DELETE_FROM"
}

// CTEs implements BlockNode.
func (n *DeleteFromStatement) CTEs() []CTEDefinition {
	panic("unimplemented")
}

var _ StatementNode = (*DeleteFromStatement)(nil)
