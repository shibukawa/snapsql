package parsercommon

import "github.com/shibukawa/snapsql/tokenizer"

// Clause structures

type ClauseNode interface {
	AstNode
	ContentTokens() []tokenizer.Token // Returns tokens that make up the clause
	IfDirective() string
	Type() NodeType
}

// WithClause represents WITH clause for CTEs
type WithClause struct {
	CTEs []CTEDefinition
}

// Position implements ClauseNode.
func (n *WithClause) Position() tokenizer.Position {
	panic("unimplemented")
}

// RawTokens implements ClauseNode.
func (n *WithClause) RawTokens() []tokenizer.Token {
	panic("unimplemented")
}

func (n *WithClause) ContentTokens() []tokenizer.Token {
	panic("not implemented")
}
func (n *WithClause) IfDirective() string {
	panic("not implemented")
}
func (n *WithClause) Type() NodeType {
	return WITH_CLAUSE
}
func (n WithClause) String() string {
	return "WITH"
}

type clauseBaseNode struct {
	headingTokens []tokenizer.Token // Leading tokens before the clause
	bodyTokens    []tokenizer.Token // Raw tokens that make up the clause
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

// Position implements ClauseNode.
func (n *SelectClause) Position() tokenizer.Position {
	panic("unimplemented")
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
func (n SelectClause) String() string {
	return "SELECT"
}

func NewSelectClause(heading, body []tokenizer.Token) *SelectClause {
	return &SelectClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

var _ ClauseNode = (*SelectClause)(nil)

// FromClause represents FROM clause
type FromClause struct {
	clauseBaseNode
	Tables []TableReference
}

// Position implements ClauseNode.
func (n *FromClause) Position() tokenizer.Position {
	panic("unimplemented")
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
func (n FromClause) String() string {
	return "FROM"
}

func NewFromClause(heading, body []tokenizer.Token) *FromClause {
	return &FromClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

var _ ClauseNode = (*FromClause)(nil)

// WhereClause represents WHERE clause
type WhereClause struct {
	clauseBaseNode
	Condition AstNode // Expression
}

func NewWhereClause(heading, body []tokenizer.Token) *WhereClause {
	return &WhereClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

// Position implements ClauseNode.
func (n *WhereClause) Position() tokenizer.Position {
	panic("unimplemented")
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

func NewGroupByClause(heading, body []tokenizer.Token) *GroupByClause {
	return &GroupByClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

// Position implements ClauseNode.
func (n *GroupByClause) Position() tokenizer.Position {
	panic("unimplemented")
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

func NewHavingClause(heading, body []tokenizer.Token) *HavingClause {
	return &HavingClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

// Position implements ClauseNode.
func (n *HavingClause) Position() tokenizer.Position {
	panic("unimplemented")
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

func NewOrderByClause(heading, body []tokenizer.Token) *OrderByClause {
	return &OrderByClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

// Position implements ClauseNode.
func (n *OrderByClause) Position() tokenizer.Position {
	panic("unimplemented")
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

func NewLimitClause(heading, body []tokenizer.Token) *LimitClause {
	return &LimitClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

// Position implements ClauseNode.
func (n *LimitClause) Position() tokenizer.Position {
	panic("unimplemented")
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

func NewOffsetClause(heading, body []tokenizer.Token) *OffsetClause {
	return &OffsetClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

// Position implements ClauseNode.
func (n *OffsetClause) Position() tokenizer.Position {
	panic("unimplemented")
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
func (n OffsetClause) String() string {
	return "OFFSET"
}

var _ ClauseNode = (*OffsetClause)(nil)

// ReturningClause represents RETURNING clause
type ReturningClause struct {
	clauseBaseNode
	Fields []FieldName
}

func NewReturningClause(heading, body []tokenizer.Token) *ReturningClause {
	return &ReturningClause{
		clauseBaseNode: clauseBaseNode{
			headingTokens: heading,
			bodyTokens:    body,
		},
	}
}

// Position implements ClauseNode.
func (n *ReturningClause) Position() tokenizer.Position {
	panic("unimplemented")
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
	Name      string
	Recursive bool
	Query     *SelectStatement // SelectStatement

	TrailingTokens []tokenizer.Token

	Columns []string
}

func (n CTEDefinition) String() string {
	return "CTE"
}

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
}

func (n FieldName) String() string {
	return "FIELD"
}

// SelectItem represents an item in SELECT clause
type SelectItem struct {
	Expression AstNode // Expression, FieldName, etc.
	Alias      string  // Optional alias
}

func (n SelectItem) String() string {
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

// OrderByField represents a field in ORDER BY clause
type OrderByField struct {
	Field FieldName
	Desc  bool // true for DESC, false for ASC
}

func (n OrderByField) String() string {
	return "ORDER_FIELD"
}

// SetClause represents a SET clause in UPDATE statement
type SetClause struct {
	Field FieldName
	Value AstNode // Expression
}

func (n SetClause) String() string {
	return "SET"
}

// OnConflictClause represents ON CONFLICT clause
type OnConflictClause struct {
	Target []FieldName
	Action []SetClause
}

func (n OnConflictClause) String() string {
	return "ON_CONFLICT"
}

// Values represents VALUES in INSERT statement
type Values struct {
	Rows [][]AstNode // Expression
}

func (n Values) String() string {
	return "VALUES"
}
