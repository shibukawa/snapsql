package parsercommon

import "github.com/shibukawa/snapsql/tokenizer"

// Clause structures

type ClauseNode interface {
	AstNode
	SourceText() string               // Return case sensitive source text of the clause
	ContentTokens() []tokenizer.Token // Returns tokens that make up the clause
	IfDirective() string
	Type() NodeType
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

// Position implements ClauseNode.
func (n *WithClause) Position() tokenizer.Position {
	panic("unimplemented")
}

// RawTokens implements ClauseNode.
func (n *WithClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *WithClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
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

type clauseBaseNode struct {
	clauseSourceText string
	headingTokens    []tokenizer.Token // Leading tokens before the clause
	bodyTokens       []tokenizer.Token // Raw tokens that make up the clause
}

func (n *clauseBaseNode) rawTokens() []tokenizer.Token {
	return append(n.headingTokens, n.bodyTokens...)
}

var _ ClauseNode = (*WithClause)(nil)

// SelectClause represents SELECT clause
type SelectClause struct {
	clauseBaseNode
	Items []SelectItem
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

// SourceText implements ClauseNode.
func (n *SelectClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *SelectClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *SelectClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *SelectClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
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
	Tables []TableReference
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

// SourceText implements ClauseNode.
func (n *FromClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *FromClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *FromClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *FromClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
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

// SourceText implements ClauseNode.
func (n *WhereClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *WhereClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *WhereClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *WhereClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
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
	Fields []FieldName
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

// SourceText implements ClauseNode.
func (n *GroupByClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *GroupByClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *GroupByClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *GroupByClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
}
func (n *GroupByClause) IfDirective() string {
	panic("not implemented")
}
func (n *GroupByClause) Type() NodeType {
	return GROUP_BY_CLAUSE
}
func (n GroupByClause) String() string {
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

// SourceText implements ClauseNode.
func (n *HavingClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *HavingClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *HavingClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *HavingClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
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

// SourceText implements ClauseNode.
func (n *OrderByClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *OrderByClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *OrderByClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *OrderByClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
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
	Count AstNode // Expression
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

// SourceText implements ClauseNode.
func (n *LimitClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *LimitClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *LimitClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *LimitClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
}
func (n *LimitClause) IfDirective() string {
	panic("not implemented")
}
func (n *LimitClause) Type() NodeType {
	return LIMIT_CLAUSE
}
func (n LimitClause) String() string {
	return "LIMIT"
}

var _ ClauseNode = (*LimitClause)(nil)

// OffsetClause represents OFFSET clause
type OffsetClause struct {
	clauseBaseNode
	Count AstNode // Expression
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

// SourceText implements ClauseNode.
func (n *OffsetClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *OffsetClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *OffsetClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *OffsetClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
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
	Fields []FieldName
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

// SourceText implements ClauseNode.
func (n *ReturningClause) SourceText() string {
	return n.clauseSourceText
}

// Position implements ClauseNode.
func (n *ReturningClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *ReturningClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

func (n *ReturningClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
}
func (n *ReturningClause) IfDirective() string {
	panic("not implemented")
}
func (n *ReturningClause) Type() NodeType {
	return RETURNING_CLAUSE
}
func (n ReturningClause) String() string {
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
	Field FieldName
	Desc  bool // true for DESC, false for ASC
}

func (n OrderByField) String() string {
	return "ORDER_FIELD"
}

type ForClause struct {
	clauseBaseNode
	TableName TableName
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

// SourceText implements ClauseNode.
func (f *ForClause) SourceText() string {
	return f.clauseSourceText
}

// ContentTokens implements ClauseNode.
func (f *ForClause) ContentTokens() []tokenizer.Token {
	return f.bodyTokens
}

// IfDirective implements ClauseNode.
func (f *ForClause) IfDirective() string {
	panic("unimplemented")
}

// Position implements ClauseNode.
func (f *ForClause) Position() tokenizer.Position {
	return f.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (f *ForClause) RawTokens() []tokenizer.Token {
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
	TableName TableName
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

// SourceText implements ClauseNode.
func (i *InsertIntoClause) SourceText() string {
	return i.clauseSourceText
}

// ContentTokens implements ClauseNode.
func (i *InsertIntoClause) ContentTokens() []tokenizer.Token {
	return i.bodyTokens
}

// IfDirective implements ClauseNode.
func (i *InsertIntoClause) IfDirective() string {
	panic("unimplemented")
}

// Position implements ClauseNode.
func (i *InsertIntoClause) Position() tokenizer.Position {
	return i.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (i *InsertIntoClause) RawTokens() []tokenizer.Token {
	return i.rawTokens()
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

// SourceText implements ClauseNode.
func (n *OnConflictClause) SourceText() string {
	return n.clauseSourceText
}

// ContentTokens implements ClauseNode.
func (n *OnConflictClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
}

// IfDirective implements ClauseNode.
func (n *OnConflictClause) IfDirective() string {
	panic("unimplemented")
}

// Position implements ClauseNode.
func (n *OnConflictClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *OnConflictClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
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

// SourceText implements ClauseNode.
func (n *ValuesClause) SourceText() string {
	return n.clauseSourceText
}

// ContentTokens implements ClauseNode.
func (n *ValuesClause) ContentTokens() []tokenizer.Token {
	return n.bodyTokens
}

// IfDirective implements ClauseNode.
func (n *ValuesClause) IfDirective() string {
	panic("unimplemented")
}

// Position implements ClauseNode.
func (n *ValuesClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *ValuesClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
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
	TableName TableName
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

// SourceText implements ClauseNode.
func (u *UpdateClause) SourceText() string {
	return u.clauseSourceText
}

// ContentTokens implements ClauseNode.
func (u *UpdateClause) ContentTokens() []tokenizer.Token {
	return u.bodyTokens
}

// IfDirective implements ClauseNode.
func (u *UpdateClause) IfDirective() string {
	panic("unimplemented")
}

// Position implements ClauseNode.
func (u *UpdateClause) Position() tokenizer.Position {
	return u.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (u *UpdateClause) RawTokens() []tokenizer.Token {
	return u.rawTokens()
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
	Field FieldName
	Value AstNode // Expression
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

// SourceText implements ClauseNode.
func (n *SetClause) SourceText() string {
	return n.clauseSourceText
}

// ContentTokens implements ClauseNode.
func (n *SetClause) ContentTokens() []tokenizer.Token {
	panic("unimplemented")
}

// IfDirective implements ClauseNode.
func (n *SetClause) IfDirective() string {
	panic("unimplemented")
}

// Position implements ClauseNode.
func (n *SetClause) Position() tokenizer.Position {
	return n.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (n *SetClause) RawTokens() []tokenizer.Token {
	return n.rawTokens()
}

// Type implements ClauseNode.
func (n *SetClause) Type() NodeType {
	return SET_CLAUSE
}

func (n SetClause) String() string {
	return "SET"
}

var _ ClauseNode = (*SetClause)(nil)

type DeleteFromClause struct {
	clauseBaseNode
	TableName TableName
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

// SourceText implements ClauseNode.
func (d *DeleteFromClause) SourceText() string {
	return d.clauseSourceText
}

// ContentTokens implements ClauseNode.
func (d *DeleteFromClause) ContentTokens() []tokenizer.Token {
	return d.bodyTokens
}

// IfDirective implements ClauseNode.
func (d *DeleteFromClause) IfDirective() string {
	panic("unimplemented")
}

// Position implements ClauseNode.
func (d *DeleteFromClause) Position() tokenizer.Position {
	return d.headingTokens[0].Position
}

// RawTokens implements ClauseNode.
func (d *DeleteFromClause) RawTokens() []tokenizer.Token {
	return d.rawTokens()
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
