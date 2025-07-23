package parsercommon

import (
	"github.com/shibukawa/snapsql/tokenizer"
)

// Clause structures

type ClauseNode interface {
	AstNode
	SourceText() string                                                            // Return case sensitive source text of the clause
	ContentTokens() []tokenizer.Token                                              // Returns tokens that make up the clause
	IfCondition() string                                                           // Returns implicit if condition for the clause
	SetIfCondition(condition string, ifIndex, endIndex int, prevClause ClauseNode) // Sets implicit if condition for the clause and removes if directive from previous clause
	Type() NodeType
	InsertTokensAfterIndex(index int, tokens []tokenizer.Token)    // Inserts tokens after the specified index
	ReplaceTokens(startIndex, endIndex int, token tokenizer.Token) // Replaces tokens from startIndex to endIndex with the specified token
	baseNode() *clauseBaseNode                                     // Returns the base node for common functionality
}

type clauseBaseNode struct {
	clauseSourceText string
	headingTokens    []tokenizer.Token // Leading tokens before the clause
	bodyTokens       []tokenizer.Token // Raw tokens that make up the clause
	ifCondition      string
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

func (cbn *clauseBaseNode) IfCondition() string {
	return cbn.ifCondition
}

func (cbn *clauseBaseNode) baseNode() *clauseBaseNode {
	return cbn
}

func (cbn *clauseBaseNode) SetIfCondition(condition string, ifIndex, endIndex int, prevClause ClauseNode) {
	cbn.ifCondition = condition
	if endIndex != -1 { // for implicit if condition
		cbn.bodyTokens = cbn.bodyTokens[:endIndex]
		pbn := prevClause.baseNode()
		pbn.bodyTokens = cbn.bodyTokens[:ifIndex]
	}
}

// InsertTokensAfterIndex は指定されたインデックスの後にトークンを挿入します
// インデックスはheadingTokensとbodyTokensの両方を考慮した全体のインデックスです
func (cbn *clauseBaseNode) InsertTokensAfterIndex(index int, newTokens []tokenizer.Token) {
	headingLen := len(cbn.headingTokens)

	// インデックスがheadingTokens内にある場合
	if index < headingLen {
		// headingTokensを分割して挿入
		result := make([]tokenizer.Token, 0, len(cbn.headingTokens)+len(newTokens))
		result = append(result, cbn.headingTokens[:index+1]...)
		result = append(result, newTokens...)
		result = append(result, cbn.headingTokens[index+1:]...)
		cbn.headingTokens = result
	} else {
		// bodyTokens内のインデックスに変換
		bodyIndex := index - headingLen
		if bodyIndex >= len(cbn.bodyTokens) {
			bodyIndex = len(cbn.bodyTokens) - 1
		}

		// bodyTokensを分割して挿入
		result := make([]tokenizer.Token, 0, len(cbn.bodyTokens)+len(newTokens))
		result = append(result, cbn.bodyTokens[:bodyIndex+1]...)
		result = append(result, newTokens...)
		result = append(result, cbn.bodyTokens[bodyIndex+1:]...)
		cbn.bodyTokens = result
	}
}

// ReplaceTokens は指定された範囲のトークンを新しいトークンに置き換えます
// startIndexからendIndex-1までのトークンが置き換えられます
func (cbn *clauseBaseNode) ReplaceTokens(startIndex, endIndex int, newToken tokenizer.Token) {
	headingLen := len(cbn.headingTokens)

	// 置換範囲がheadingTokens内にある場合
	if startIndex < headingLen && endIndex <= headingLen {
		// headingTokensを分割して置換
		result := make([]tokenizer.Token, 0, len(cbn.headingTokens)-(endIndex-startIndex)+1)
		result = append(result, cbn.headingTokens[:startIndex]...)
		result = append(result, newToken)
		result = append(result, cbn.headingTokens[endIndex:]...)
		cbn.headingTokens = result
	} else if startIndex < headingLen && endIndex > headingLen {
		// 置換範囲がheadingTokensとbodyTokensにまたがる場合
		headingResult := make([]tokenizer.Token, 0, startIndex+1)
		headingResult = append(headingResult, cbn.headingTokens[:startIndex]...)
		headingResult = append(headingResult, newToken)
		cbn.headingTokens = headingResult

		bodyResult := make([]tokenizer.Token, 0, len(cbn.bodyTokens)-(endIndex-headingLen))
		bodyResult = append(bodyResult, cbn.bodyTokens[endIndex-headingLen:]...)
		cbn.bodyTokens = bodyResult
	} else {
		// 置換範囲がbodyTokens内にある場合
		bodyStartIndex := startIndex - headingLen
		bodyEndIndex := endIndex - headingLen

		// bodyTokensを分割して置換
		result := make([]tokenizer.Token, 0, len(cbn.bodyTokens)-(bodyEndIndex-bodyStartIndex)+1)
		result = append(result, cbn.bodyTokens[:bodyStartIndex]...)
		result = append(result, newToken)
		result = append(result, cbn.bodyTokens[bodyEndIndex:]...)
		cbn.bodyTokens = result
	}
}

type WithClause struct {
	clauseBaseNode
	Recursive      bool
	HeadingTokens  []tokenizer.Token // Leading tokens before the WITH clause
	CTEs           []CTEDefinition
	TrailingTokens []tokenizer.Token // Additional tokens that may follow the CTE definitions
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

func (n *WhereClause) Type() NodeType {
	return WHERE_CLAUSE
}
func (n *WhereClause) String() string {
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

func (n *HavingClause) Type() NodeType {
	return HAVING_CLAUSE
}
func (n *HavingClause) String() string {
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

func (n *OrderByClause) Type() NodeType {
	return ORDER_BY_CLAUSE
}
func (n *OrderByClause) String() string {
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

// String implements ClauseNode.
func (f *ForClause) String() string {
	return "FOR CLAUSE"
}

// Type implements ClauseNode.
func (f *ForClause) Type() NodeType {
	return FOR_CLAUSE
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

// Type implements ClauseNode.
func (n *OnConflictClause) Type() NodeType {
	return ON_CONFLICT_CLAUSE
}

func (n *OnConflictClause) String() string {
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

// Type implements ClauseNode.
func (n *ValuesClause) Type() NodeType {
	return VALUES_CLAUSE
}

func (n *ValuesClause) String() string {
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

// String implements ClauseNode.
func (d *DeleteFromClause) String() string {
	panic("unimplemented")
}

// Type implements ClauseNode.
func (d *DeleteFromClause) Type() NodeType {
	return DELETE_FROM_CLAUSE
}

var _ ClauseNode = (*DeleteFromClause)(nil)
