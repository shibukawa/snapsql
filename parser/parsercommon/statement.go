package parsercommon

import (
	"github.com/shibukawa/snapsql/tokenizer"
)

// Forward declarations for parserstep7 types to avoid circular imports
type FieldSourceInterface interface{}
type TableReferenceInterface interface{}
type DependencyGraphInterface interface{}

// DML Statement structures

type StatementNode interface {
	AstNode
	CTE() *WithClause                 // Returns CTE definitions in the block
	LeadingTokens() []tokenizer.Token // Returns leading tokens of the block
	Clauses() []ClauseNode            // Returns clauses in the block

	// New methods for parserstep7 field source information
	GetFieldSources() map[string]FieldSourceInterface
	GetTableReferences() map[string]TableReferenceInterface
	GetSubqueryDependencies() DependencyGraphInterface
	SetFieldSources(map[string]FieldSourceInterface)
	SetTableReferences(map[string]TableReferenceInterface)
	SetSubqueryDependencies(DependencyGraphInterface)

	// Convenience methods for field and table lookup
	FindFieldReference(tableOrAlias, fieldOrReference string) FieldSourceInterface
	FindTableReference(tableOrAlias string) TableReferenceInterface

	// Subquery analysis information access
	GetSubqueryAnalysis() *SubqueryAnalysisResult
	SetSubqueryAnalysis(*SubqueryAnalysisResult)
	HasSubqueryAnalysis() bool
}

type baseStatement struct {
	leadingTokens []tokenizer.Token // Leading tokens before the SELECT statement
	with          *WithClause       // CTE definitions in the SELECT statement
	clauses       []ClauseNode      // All clauses in the statement

	// New fields for parserstep7
	fieldSources         map[string]FieldSourceInterface
	tableReferences      map[string]TableReferenceInterface
	subqueryDependencies DependencyGraphInterface
	subqueryAnalysis     *SubqueryAnalysisResult // Subquery analysis information
}

// GetFieldSources implements StatementNode
func (bs *baseStatement) GetFieldSources() map[string]FieldSourceInterface {
	if bs.fieldSources == nil {
		bs.fieldSources = make(map[string]FieldSourceInterface)
	}
	return bs.fieldSources
}

// GetTableReferences implements StatementNode
func (bs *baseStatement) GetTableReferences() map[string]TableReferenceInterface {
	if bs.tableReferences == nil {
		bs.tableReferences = make(map[string]TableReferenceInterface)
	}
	return bs.tableReferences
}

// GetSubqueryDependencies implements StatementNode
func (bs *baseStatement) GetSubqueryDependencies() DependencyGraphInterface {
	return bs.subqueryDependencies
}

// SetFieldSources implements StatementNode
func (bs *baseStatement) SetFieldSources(sources map[string]FieldSourceInterface) {
	bs.fieldSources = sources
}

// SetTableReferences implements StatementNode
func (bs *baseStatement) SetTableReferences(refs map[string]TableReferenceInterface) {
	bs.tableReferences = refs
}

// SetSubqueryDependencies implements StatementNode
func (bs *baseStatement) SetSubqueryDependencies(deps DependencyGraphInterface) {
	bs.subqueryDependencies = deps
}

// GetSubqueryAnalysis implements StatementNode
func (bs *baseStatement) GetSubqueryAnalysis() *SubqueryAnalysisResult {
	return bs.subqueryAnalysis
}

// SetSubqueryAnalysis implements StatementNode
func (bs *baseStatement) SetSubqueryAnalysis(analysis *SubqueryAnalysisResult) {
	bs.subqueryAnalysis = analysis
}

// HasSubqueryAnalysis implements StatementNode
func (bs *baseStatement) HasSubqueryAnalysis() bool {
	return bs.subqueryAnalysis != nil && bs.subqueryAnalysis.HasSubqueries
}

// FindFieldReference implements StatementNode
func (bs *baseStatement) FindFieldReference(tableOrAlias, fieldOrReference string) FieldSourceInterface {
	// First try direct field lookup
	if source, exists := bs.fieldSources[fieldOrReference]; exists {
		return source
	}

	// Then try table.field lookup
	key := tableOrAlias + "." + fieldOrReference
	if source, exists := bs.fieldSources[key]; exists {
		return source
	}

	// Finally try looking up in table references
	if tableRef, exists := bs.tableReferences[tableOrAlias]; exists {
		// This would need to be implemented properly with actual types
		// For now, return nil
		_ = tableRef
	}

	return nil
}

// FindTableReference implements StatementNode
func (bs *baseStatement) FindTableReference(tableOrAlias string) TableReferenceInterface {
	if ref, exists := bs.tableReferences[tableOrAlias]; exists {
		return ref
	}
	return nil
}

// SelectStatement represents SELECT statement
type SelectStatement struct {
	baseStatement
	With    *WithClause
	Select  *SelectClause
	From    *FromClause
	Where   *WhereClause
	GroupBy *GroupByClause
	Having  *HavingClause
	OrderBy *OrderByClause
	Limit   *LimitClause
	Offset  *OffsetClause
	For     *ForClause
}

func NewSelectStatement(leadingTokens []tokenizer.Token, with *WithClause, clauses []ClauseNode) *SelectStatement {
	return &SelectStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			with:          with,
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
func (s *SelectStatement) CTE() *WithClause {
	return s.with
}

var _ StatementNode = (*SelectStatement)(nil)

func (s *SelectStatement) String() string {
	return "SELECT"
}

// InsertStatement represents INSERT statement
type InsertIntoStatement struct {
	baseStatement
	With       *WithClause
	Into       *InsertIntoClause
	Columns    []FieldName
	ValuesList *ValuesClause
	Select     *SelectClause
	From       *FromClause
	Where      *WhereClause
	GroupBy    *GroupByClause
	Having     *HavingClause
	OrderBy    *OrderByClause
	Limit      *LimitClause
	Offset     *OffsetClause

	OnConflict *OnConflictClause
	Returning  *ReturningClause
}

func NewInsertIntoStatement(leadingTokens []tokenizer.Token, with *WithClause, clauses []ClauseNode) *InsertIntoStatement {
	return &InsertIntoStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			with:          with,
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

func (n *InsertIntoStatement) String() string {
	return "INSERT INTO"
}

// CTEs implements BlockNode.
func (n *InsertIntoStatement) CTE() *WithClause {
	return n.with
}

var _ StatementNode = (*InsertIntoStatement)(nil)

// UpdateStatement represents UPDATE statement
type UpdateStatement struct {
	baseStatement
	With      *WithClause
	Update    *UpdateClause
	Set       *SetClause
	Where     *WhereClause
	Returning *ReturningClause
}

// NewUpdateStatement creates a new UpdateStatement node.
func NewUpdateStatement(leadingTokens []tokenizer.Token, with *WithClause, clauses []ClauseNode) *UpdateStatement {
	return &UpdateStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			with:          with,
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

func (n *UpdateStatement) String() string {
	return "UPDATE"
}

// CTEs implements BlockNode.
func (n *UpdateStatement) CTE() *WithClause {
	return n.with
}

var _ StatementNode = (*UpdateStatement)(nil)

// DeleteStatement represents DELETE statement
type DeleteFromStatement struct {
	baseStatement
	With      *WithClause
	From      *DeleteFromClause
	Where     *WhereClause
	Returning *ReturningClause
}

// NewDeleteFromStatement creates a new DeleteFromStatement node.
func NewDeleteFromStatement(leadingTokens []tokenizer.Token, with *WithClause, clauses []ClauseNode) *DeleteFromStatement {
	return &DeleteFromStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			with:          with,
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

func (n *DeleteFromStatement) String() string {
	return "DELETE_FROM"
}

// CTEs implements BlockNode.
func (n *DeleteFromStatement) CTE() *WithClause {
	return n.with
}

var _ StatementNode = (*DeleteFromStatement)(nil)
