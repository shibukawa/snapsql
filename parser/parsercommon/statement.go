package parsercommon

import (
	"github.com/shibukawa/snapsql/tokenizer"
)

// Types from parserstep7 now available in parsercommon

// DML Statement structures

type StatementNode interface {
	AstNode
	CTE() *WithClause                 // Returns CTE definitions in the block
	LeadingTokens() []tokenizer.Token // Returns leading tokens of the block
	Clauses() []ClauseNode            // Returns clauses in the block

	// New methods for parserstep7 field source information
	GetFieldSources() map[string]*SQFieldSource
	GetTableReferences() map[string]*SQTableReference
	GetSubqueryDependencies() *SQDependencyGraph
	GetProcessingOrder() []string

	// Convenience methods for field and table lookup
	FindFieldReference(tableOrAlias, fieldOrReference string) *SQFieldSource
	FindTableReference(tableOrAlias string) *SQTableReference

	// Subquery analysis information access
	GetSubqueryAnalysis() *SubqueryAnalysisResult
	HasSubqueryAnalysis() bool
}

type baseStatement struct {
	leadingTokens []tokenizer.Token // Leading tokens before the SELECT statement
	with          *WithClause       // CTE definitions in the SELECT statement
	clauses       []ClauseNode      // All clauses in the statement

	// New fields for parserstep7
	fieldSources         map[string]*SQFieldSource
	tableReferences      map[string]*SQTableReference
	subqueryDependencies *SQDependencyGraph
	processingOrder      []string                // Processing order for subqueries
	subqueryAnalysis     *SubqueryAnalysisResult // Subquery analysis information
}

func (bs *baseStatement) LeadingTokens() []tokenizer.Token {
	return bs.leadingTokens
}

// GetFieldSources implements StatementNode
func (bs *baseStatement) GetFieldSources() map[string]*SQFieldSource {
	if bs.fieldSources == nil {
		bs.fieldSources = make(map[string]*SQFieldSource)
	}
	return bs.fieldSources
}

// GetTableReferences implements StatementNode
func (bs *baseStatement) GetTableReferences() map[string]*SQTableReference {
	if bs.tableReferences == nil {
		bs.tableReferences = make(map[string]*SQTableReference)
	}
	return bs.tableReferences
}

// GetSubqueryDependencies implements StatementNode
func (bs *baseStatement) GetSubqueryDependencies() *SQDependencyGraph {
	return bs.subqueryDependencies
}

// GetProcessingOrder implements StatementNode
func (bs *baseStatement) GetProcessingOrder() []string {
	return bs.processingOrder
}

// GetSubqueryAnalysis implements StatementNode
func (bs *baseStatement) GetSubqueryAnalysis() *SubqueryAnalysisResult {
	return bs.subqueryAnalysis
}

// HasSubqueryAnalysis implements StatementNode
func (bs *baseStatement) HasSubqueryAnalysis() bool {
	return bs.subqueryAnalysis != nil && bs.subqueryAnalysis.HasSubqueries
}

// FindFieldReference implements StatementNode
func (bs *baseStatement) FindFieldReference(tableOrAlias, fieldOrReference string) *SQFieldSource {
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
func (bs *baseStatement) FindTableReference(tableOrAlias string) *SQTableReference {
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
	stmt := &InsertIntoStatement{
		baseStatement: baseStatement{
			leadingTokens: leadingTokens,
			with:          with,
			clauses:       clauses,
		},
	}

	// Set clause references (Columns will be set later in parserstep4)
	for _, clause := range clauses {
		if insertIntoClause, ok := clause.(*InsertIntoClause); ok {
			stmt.Into = insertIntoClause
		} else if valuesClause, ok := clause.(*ValuesClause); ok {
			stmt.ValuesList = valuesClause
		}
	}

	return stmt
}

// Clauses implements BlockNode.
func (n *InsertIntoStatement) Clauses() []ClauseNode {
	return n.clauses
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

// External setter functions for StatementNode

// SetFieldSources sets field sources for a statement
func SetFieldSources(stmt StatementNode, sources map[string]*SQFieldSource) {
	if bs, ok := stmt.(*SelectStatement); ok {
		bs.fieldSources = sources
	} else if bs, ok := stmt.(*InsertIntoStatement); ok {
		bs.fieldSources = sources
	} else if bs, ok := stmt.(*UpdateStatement); ok {
		bs.fieldSources = sources
	} else if bs, ok := stmt.(*DeleteFromStatement); ok {
		bs.fieldSources = sources
	}
}

// SetTableReferences sets table references for a statement
func SetTableReferences(stmt StatementNode, refs map[string]*SQTableReference) {
	if bs, ok := stmt.(*SelectStatement); ok {
		bs.tableReferences = refs
	} else if bs, ok := stmt.(*InsertIntoStatement); ok {
		bs.tableReferences = refs
	} else if bs, ok := stmt.(*UpdateStatement); ok {
		bs.tableReferences = refs
	} else if bs, ok := stmt.(*DeleteFromStatement); ok {
		bs.tableReferences = refs
	}
}

// SetSubqueryDependencies sets subquery dependencies for a statement
func SetSubqueryDependencies(stmt StatementNode, deps *SQDependencyGraph) {
	if bs, ok := stmt.(*SelectStatement); ok {
		bs.subqueryDependencies = deps
	} else if bs, ok := stmt.(*InsertIntoStatement); ok {
		bs.subqueryDependencies = deps
	} else if bs, ok := stmt.(*UpdateStatement); ok {
		bs.subqueryDependencies = deps
	} else if bs, ok := stmt.(*DeleteFromStatement); ok {
		bs.subqueryDependencies = deps
	}
}

// SetProcessingOrder sets processing order for a statement
func SetProcessingOrder(stmt StatementNode, order []string) {
	if bs, ok := stmt.(*SelectStatement); ok {
		bs.processingOrder = order
	} else if bs, ok := stmt.(*InsertIntoStatement); ok {
		bs.processingOrder = order
	} else if bs, ok := stmt.(*UpdateStatement); ok {
		bs.processingOrder = order
	} else if bs, ok := stmt.(*DeleteFromStatement); ok {
		bs.processingOrder = order
	}
}

// SetSubqueryAnalysis sets subquery analysis for a statement
func SetSubqueryAnalysis(stmt StatementNode, analysis *SubqueryAnalysisResult) {
	if bs, ok := stmt.(*SelectStatement); ok {
		bs.subqueryAnalysis = analysis
	} else if bs, ok := stmt.(*InsertIntoStatement); ok {
		bs.subqueryAnalysis = analysis
	} else if bs, ok := stmt.(*UpdateStatement); ok {
		bs.subqueryAnalysis = analysis
	} else if bs, ok := stmt.(*DeleteFromStatement); ok {
		bs.subqueryAnalysis = analysis
	}
}
