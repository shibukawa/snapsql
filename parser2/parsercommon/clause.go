package parsercommon

import (
	"github.com/shibukawa/snapsql/tokenizer"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// Clause structures

type ClauseNode interface {
	AstNode
	SourceText() string               // Return case sensitive source text of the clause
	ContentTokens() []tokenizer.Token // Returns tokens that make up the clause
	IfDirective() string
	Type() NodeType
}

type clauseBaseNode struct {
	clauseSourceText string
	headingTokens    []tokenizer.Token // Leading tokens before the clause
	bodyTokens       []tokenizer.Token // Raw tokens that make up the clause
}

// SourceText implements ClauseNode.
func (cbn *clauseBaseNode) SourceText() string {
	return cbn.clauseSourceText
}

func (cbn *clauseBaseNode) RawTokens() []tokenizer.Token {
	return append(cbn.headingTokens, cbn.bodyTokens...)
}

func (cbn *clauseBaseNode) ContentTokens() []tokenizer.Token {
	return cbn.bodyTokens
}

func (cbn *clauseBaseNode) Position() tokenizer.Position {
	return cbn.headingTokens[0].Position
}

// WithClause represents WITH clause for CTEs
type WithClause struct {
	clauseBaseNode
	Recursive      bool
	HeadingTokens  []tokenizer.Token // Leading tokens before the WITH clause
	CTEs           []CTEDefinition
	TrailingTokens []tokenizer.Token // Additional tokens that may follow the CTE definitions
}

// SourceText implements ClauseNode.
func (n *WithClause) SourceText() string {
	return n.clauseSourceText
}

func (n *WithClause) IfDirective() string {
	panic("not implemented")
}

func (n *WithClause) Type() NodeType {
	return WITH_CLAUSE
}

func (n *WithClause) String() string {
	return "WITH"
}

var _ ClauseNode = (*WithClause)(nil)

// SelectClause represents SELECT clause
type SelectClause struct {
	clauseBaseNode
	Distinct   bool
	DistinctOn []FieldName // DISTINCT ON fields
	Fields     []SelectField
}

func NewSelectClause(srcText string, heading, body []tokenizer.Token) *SelectClause {
	return &SelectClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *SelectClause) IfDirective() string {
	panic("not implemented")
}

func (n *SelectClause) Type() NodeType {
	return SELECT_CLAUSE
}

func (n *SelectClause) String() string {
	return "SELECT"
}

var _ ClauseNode = (*SelectClause)(nil)

// FromClause represents FROM clause
type FromClause struct {
	clauseBaseNode
	Tables []TableReferenceForFrom
}

func NewFromClause(srcText string, heading, body []tokenizer.Token) *FromClause {
	return &FromClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *FromClause) IfDirective() string {
	panic("not implemented")
}
func (n *FromClause) Type() NodeType {
	return FROM_CLAUSE
}
func (n *FromClause) String() string {
	return "FROM"
}

var _ ClauseNode = (*FromClause)(nil)

// WhereClause represents WHERE clause
type WhereClause struct {
	clauseBaseNode
	Condition AstNode // Expression
}

func NewWhereClause(srcText string, heading, body []tokenizer.Token) *WhereClause {
	return &WhereClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *WhereClause) IfDirective() string {
	panic("not implemented")
}
func (n *WhereClause) Type() NodeType {
	return WHERE_CLAUSE
}
func (n WhereClause) String() string {
	return "WHERE"
}

var _ ClauseNode = (*WhereClause)(nil)

// GroupByClause represents GROUP BY clause
type GroupByClause struct {
	clauseBaseNode
	Null             bool // Indicates if NULL is used in GROUP BY
	AdvancedGrouping bool // Indicates if advanced grouping features like ROLLUP, CUBE, GROUPING SETS are used
	Fields           []FieldName
}

func NewGroupByClause(srcText string, heading, body []tokenizer.Token) *GroupByClause {
	return &GroupByClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *GroupByClause) IfDirective() string {
	panic("not implemented")
}
func (n *GroupByClause) Type() NodeType {
	return GROUP_BY_CLAUSE
}
func (n *GroupByClause) String() string {
	return "GROUP BY"
}

var _ ClauseNode = (*GroupByClause)(nil)

// HavingClause represents HAVING clause
type HavingClause struct {
	clauseBaseNode
	Condition AstNode // Expression
}

func NewHavingClause(srcText string, heading, body []tokenizer.Token) *HavingClause {
	return &HavingClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *HavingClause) IfDirective() string {
	panic("not implemented")
}
func (n *HavingClause) Type() NodeType {
	return HAVING_CLAUSE
}
func (n HavingClause) String() string {
	return "HAVING"
}

var _ ClauseNode = (*HavingClause)(nil)

// OrderByClause represents ORDER BY clause
type OrderByClause struct {
	clauseBaseNode
	Fields []OrderByField
}

func NewOrderByClause(srcText string, heading, body []tokenizer.Token) *OrderByClause {
	return &OrderByClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *OrderByClause) IfDirective() string {
	panic("not implemented")
}
func (n *OrderByClause) Type() NodeType {
	return ORDER_BY_CLAUSE
}
func (n OrderByClause) String() string {
	return "ORDER BY"
}

var _ ClauseNode = (*OrderByClause)(nil)

// LimitClause represents LIMIT clause
type LimitClause struct {
	clauseBaseNode
	Count int // Expression
}

func NewLimitClause(srcText string, heading, body []tokenizer.Token) *LimitClause {
	return &LimitClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *LimitClause) IfDirective() string {
	panic("not implemented")
}
func (n *LimitClause) Type() NodeType {
	return LIMIT_CLAUSE
}
func (n *LimitClause) String() string {
	return "LIMIT"
}

var _ ClauseNode = (*LimitClause)(nil)

// OffsetClause represents OFFSET clause
type OffsetClause struct {
	clauseBaseNode
	Count int // Expression
}

func NewOffsetClause(srcText string, heading, body []tokenizer.Token) *OffsetClause {
	return &OffsetClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *OffsetClause) IfDirective() string {
	panic("not implemented")
}
func (n *OffsetClause) Type() NodeType {
	return OFFSET_CLAUSE
}
func (n *OffsetClause) String() string {
	return "OFFSET"
}

var _ ClauseNode = (*OffsetClause)(nil)

// ReturningClause represents RETURNING clause
type ReturningClause struct {
	clauseBaseNode
	Fields []SelectField
}

func NewReturningClause(srcText string, heading, body []tokenizer.Token) *ReturningClause {
	return &ReturningClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

func (n *ReturningClause) IfDirective() string {
	panic("not implemented")
}
func (n *ReturningClause) Type() NodeType {
	return RETURNING_CLAUSE
}
func (n *ReturningClause) String() string {
	return "RETURNING"
}

var _ ClauseNode = (*ReturningClause)(nil)

// Helper structures

// CTEDefinition represents a Common Table Expression definition
type CTEDefinition struct {
	Name           string
	Select         AstNode
	TrailingTokens []tokenizer.Token
}

func (n CTEDefinition) String() string {
	return "CTE"
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

type ForClause struct {
	clauseBaseNode
	TableName TableReference
}

func NewForClause(srcText string, heading, body []tokenizer.Token) *ForClause {
	return &ForClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

// IfDirective implements ClauseNode.
func (f *ForClause) IfDirective() string {
	panic("unimplemented")
}

// String implements ClauseNode.
func (f *ForClause) String() string {
	panic("unimplemented")
}

// Type implements ClauseNode.
func (f *ForClause) Type() NodeType {
	panic("unimplemented")
}

var _ ClauseNode = (*ForClause)(nil)

type InsertIntoClause struct {
	clauseBaseNode
	Table   TableReference
	Columns []string
}

func NewInsertIntoClause(srcText string, heading, body []tokenizer.Token) *InsertIntoClause {
	return &InsertIntoClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

// IfDirective implements ClauseNode.
func (i *InsertIntoClause) IfDirective() string {
	panic("unimplemented")
}

// String implements ClauseNode.
func (i *InsertIntoClause) String() string {
	panic("unimplemented")
}

// Type implements ClauseNode.
func (i *InsertIntoClause) Type() NodeType {
	return INSERT_INTO_CLAUSE
}

var _ ClauseNode = (*InsertIntoClause)(nil)

type OnConflictClause struct {
	clauseBaseNode
	Target []FieldName
	Action []SetClause
}

func NewOnConflictClause(srcText string, heading, body []tokenizer.Token) *OnConflictClause {
	return &OnConflictClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

// IfDirective implements ClauseNode.
func (n *OnConflictClause) IfDirective() string {
	panic("unimplemented")
}

// Type implements ClauseNode.
func (n *OnConflictClause) Type() NodeType {
	return ON_CONFLICT_CLAUSE
}

func (n OnConflictClause) String() string {
	return "ON_CONFLICT_CLAUSE"
}

var _ ClauseNode = (*OnConflictClause)(nil)

type ValuesClause struct {
	clauseBaseNode
	Rows [][]AstNode // Expression
}

func NewValuesClause(srcText string, heading, body []tokenizer.Token) *ValuesClause {
	return &ValuesClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

// IfDirective implements ClauseNode.
func (n *ValuesClause) IfDirective() string {
	panic("unimplemented")
}

// Type implements ClauseNode.
func (n *ValuesClause) Type() NodeType {
	return VALUES_CLAUSE
}

func (n ValuesClause) String() string {
	return "VALUES_CLAUSE"
}

var _ ClauseNode = (*ValuesClause)(nil)

type UpdateClause struct {
	clauseBaseNode
	Table TableReference
}

func NewUpdateClause(srcText string, heading, body []tokenizer.Token) *UpdateClause {
	return &UpdateClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

// IfDirective implements ClauseNode.
func (u *UpdateClause) IfDirective() string {
	panic("unimplemented")
}

// String implements ClauseNode.
func (u *UpdateClause) String() string {
	panic("unimplemented")
}

// Type implements ClauseNode.
func (u *UpdateClause) Type() NodeType {
	return UPDATE_CLAUSE
}

var _ ClauseNode = (*UpdateClause)(nil)

// SetClause represents a SET clause in UPDATE statement
type SetClause struct {
	clauseBaseNode
	Assigns []SetAssign // List of field assignments
}

func NewSetClause(srcText string, heading, body []tokenizer.Token) *SetClause {
	return &SetClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

// IfDirective implements ClauseNode.
func (n *SetClause) IfDirective() string {
	panic("unimplemented")
}

// Type implements ClauseNode.
func (n *SetClause) Type() NodeType {
	return SET_CLAUSE
}

func (n *SetClause) String() string {
	return "SET"
}

var _ ClauseNode = (*SetClause)(nil)

type DeleteFromClause struct {
	clauseBaseNode
	Table TableReference
}

func NewDeleteFromClause(srcText string, heading, body []tokenizer.Token) *DeleteFromClause {
	return &DeleteFromClause{
		clauseBaseNode: clauseBaseNode{
			clauseSourceText: srcText,
			headingTokens:    heading,
			bodyTokens:       body,
		},
	}
}

// IfDirective implements ClauseNode.
func (d *DeleteFromClause) IfDirective() string {
	panic("unimplemented")
}

// String implements ClauseNode.
func (d *DeleteFromClause) String() string {
	panic("unimplemented")
}

// Type implements ClauseNode.
func (d *DeleteFromClause) Type() NodeType {
	return DELETE_FROM_CLAUSE
}

var _ ClauseNode = (*DeleteFromClause)(nil)
