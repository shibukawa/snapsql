package parser

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

// AstNode represents AST (Abstract Syntax Tree) node interface
type AstNode interface {
	Type() NodeType
	Position() tokenizer.Position
	String() string
}

// NodeType represents the type of AST node
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

	// CTE (Common Table Expression) support
	WITH_CLAUSE
	CTE_DEFINITION

	// SnapSQL extensions
	TEMPLATE_IF_BLOCK
	TEMPLATE_ELSEIF_BLOCK
	TEMPLATE_ELSE_BLOCK
	TEMPLATE_FOR_BLOCK
	VARIABLE_SUBSTITUTION
	DEFERRED_VARIABLE_SUBSTITUTION // deferred variable substitution
	BULK_VARIABLE_SUBSTITUTION     // bulk variable substitution (for map arrays)
	ENVIRONMENT_REFERENCE
	IMPLICIT_CONDITIONAL // Automatically generated conditional block

	// Expressions and literals
	IDENTIFIER
	LITERAL
	EXPRESSION

	// Others
	OTHER_NODE

	// RETURNING_CLAUSE: RETURNING句
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
	default:
		return "UNKNOWN"
	}
}

// BaseAstNode is the base implementation of AST nodes
type BaseAstNode struct {
	nodeType NodeType
	position tokenizer.Position
}

func (n *BaseAstNode) Type() NodeType {
	return n.nodeType
}

func (n *BaseAstNode) Position() tokenizer.Position {
	return n.position
}

func joinNodes(nodes []AstNode) string {
	parts := make([]string, len(nodes))
	for i, field := range nodes {
		parts[i] = field.String()
	}
	return strings.Join(parts, ", ")
}

// SelectStatement represents SELECT statement AST node
type SelectStatement struct {
	BaseAstNode
	WithClause    *WithClause
	SelectClause  *SelectClause
	FromClause    *FromClause
	WhereClause   *WhereClause
	OrderByClause *OrderByClause
	GroupByClause *GroupByClause
	HavingClause  *HavingClause
	LimitClause   *LimitClause
	OffsetClause  *OffsetClause
}

func (s *SelectStatement) String() string {
	var parts []string

	if s.WithClause != nil {
		parts = append(parts, s.WithClause.String())
	}

	parts = append(parts, s.SelectClause.String())

	if s.FromClause != nil {
		parts = append(parts, s.FromClause.String())
	}
	if s.WhereClause != nil {
		parts = append(parts, s.WhereClause.String())
	}
	if s.GroupByClause != nil {
		parts = append(parts, s.GroupByClause.String())
	}
	if s.HavingClause != nil {
		parts = append(parts, s.HavingClause.String())
	}
	if s.OrderByClause != nil {
		parts = append(parts, s.OrderByClause.String())
	}
	if s.LimitClause != nil {
		parts = append(parts, s.LimitClause.String())
	}
	if s.OffsetClause != nil {
		parts = append(parts, s.OffsetClause.String())
	}

	return strings.Join(parts, " ")
}

// SelectClause represents SELECT clause AST node
type SelectClause struct {
	BaseAstNode
	Fields []AstNode
}

func (s *SelectClause) String() string {
	return "SELECT " + joinNodes(s.Fields)
}

// FromClause represents FROM clause AST node
type FromClause struct {
	BaseAstNode
	Tables []AstNode
}

func (f *FromClause) String() string {
	return "FROM " + joinNodes(f.Tables)
}

// WhereClause represents WHERE clause AST node
type WhereClause struct {
	BaseAstNode
	Condition AstNode
}

func (w *WhereClause) String() string {
	return "WHERE " + w.Condition.String()
}

// OrderByClause represents ORDER BY clause AST node
type OrderByClause struct {
	BaseAstNode
	Fields []AstNode
}

func (o *OrderByClause) String() string {
	return "ORDER BY " + joinNodes(o.Fields)
}

// GroupByClause represents GROUP BY clause AST node
type GroupByClause struct {
	BaseAstNode
	Fields []AstNode
}

func (g *GroupByClause) String() string {
	return "GROUP BY " + joinNodes(g.Fields)
}

// HavingClause represents HAVING clause AST node
type HavingClause struct {
	BaseAstNode
	Condition AstNode
}

func (h *HavingClause) String() string {
	return "HAVING " + h.Condition.String()
}

// LimitClause represents LIMIT clause AST node
type LimitClause struct {
	BaseAstNode
	Value AstNode
}

func (l *LimitClause) String() string {
	return "LIMIT " + l.Value.String()
}

// OffsetClause represents OFFSET clause AST node
type OffsetClause struct {
	BaseAstNode
	Value AstNode
}

func (o *OffsetClause) String() string {
	return "OFFSET " + o.Value.String()
}

// Identifier represents identifier AST node
type Identifier struct {
	BaseAstNode
	Name string
}

func (i *Identifier) String() string {
	return i.Name
}

// Literal represents literal value AST node
type Literal struct {
	BaseAstNode
	Value string
}

func (l *Literal) String() string {
	return l.Value
}

// Expression represents expression AST node (does not parse OTHER token content)
type Expression struct {
	BaseAstNode
	Tokens []tokenizer.Token
}

func (e *Expression) String() string {
	parts := make([]string, len(e.Tokens))
	for i, token := range e.Tokens {
		parts[i] = token.Value
	}
	return strings.Join(parts, " ")
}

// TemplateIfBlock represents SnapSQL if statement AST node (supports nesting)
type TemplateIfBlock struct {
	BaseAstNode
	Condition    string                 // CEL expression
	Content      []AstNode              // if block content (nestable)
	ElseIfBlocks []*TemplateElseIfBlock // elseif block list
	ElseBlock    *TemplateElseBlock     // else block (optional)
}

func (t *TemplateIfBlock) String() string {
	result := fmt.Sprintf("/*# if %s */", t.Condition)
	for _, elseif := range t.ElseIfBlocks {
		result += elseif.String()
	}
	if t.ElseBlock != nil {
		result += t.ElseBlock.String()
	}
	result += "/*# end */"
	return result
}

// TemplateElseIfBlock represents SnapSQL elseif statement AST node
type TemplateElseIfBlock struct {
	BaseAstNode
	Condition string    // CEL expression
	Content   []AstNode // elseif block content (nestable)
}

func (t *TemplateElseIfBlock) String() string {
	return fmt.Sprintf("/*# elseif %s */", t.Condition)
}

// TemplateElseBlock represents SnapSQL else statement AST node
type TemplateElseBlock struct {
	BaseAstNode
	Content []AstNode // else block content (nestable)
}

func (t *TemplateElseBlock) String() string {
	return "/*# else */"
}

// TemplateForBlock represents SnapSQL for statement AST node (supports nesting)
type TemplateForBlock struct {
	BaseAstNode
	Variable string    // loop variable name
	ListExpr string    // list variable CEL expression
	Content  []AstNode // for block content (nestable)
}

func (t *TemplateForBlock) String() string {
	return fmt.Sprintf("/*# for %s : %s */.../*# end */", t.Variable, t.ListExpr)
}

// VariableSubstitution represents SnapSQL variable substitution AST node
type VariableSubstitution struct {
	BaseAstNode
	Expression string // CEL expression
	DummyValue string // dummy literal
}

func (v *VariableSubstitution) String() string {
	if v.DummyValue != "" {
		return fmt.Sprintf("/*= %s */%s", v.Expression, v.DummyValue)
	}
	return fmt.Sprintf("/*= %s */", v.Expression)
}

// BulkVariableSubstitution represents map array variable substitution AST node for bulk insert
type BulkVariableSubstitution struct {
	BaseAstNode
	Expression string   // CEL expression (map array variable)
	Columns    []string // column name list
	DummyValue string   // dummy value for development
}

func (bv *BulkVariableSubstitution) String() string {
	if bv.DummyValue != "" {
		return fmt.Sprintf("/*= %s */%s", bv.Expression, bv.DummyValue)
	}
	return fmt.Sprintf("/*= %s */", bv.Expression)
}

// EnvironmentReference represents a SnapSQL environment reference AST node
type EnvironmentReference struct {
	BaseAstNode
	Expression    string // CEL expression
	ResolvedValue string // resolved literal value (set at build time)
}

func (e *EnvironmentReference) String() string {
	if e.ResolvedValue != "" {
		return e.ResolvedValue
	}
	return fmt.Sprintf("/*@ %s */", e.Expression)
}

// ImplicitConditional represents an automatically generated conditional block
type ImplicitConditional struct {
	BaseAstNode
	Variable   string  // Variable to check for null/empty
	Content    AstNode // Content to conditionally include
	ClauseType string  // Type of clause (WHERE, ORDER_BY, LIMIT, etc.)
	Operator   string  // Operator type (AND, OR, etc.)
	Condition  string  // Generated CEL condition
}

func (ic *ImplicitConditional) String() string {
	return fmt.Sprintf("ImplicitIf(%s: %s)", ic.Variable, ic.Content.String())
}

// WithClause represents WITH clause containing CTEs
type WithClause struct {
	BaseAstNode
	CTEs []CTEDefinition
}

func (w *WithClause) String() string {
	parts := make([]string, len(w.CTEs))
	for i, cte := range w.CTEs {
		parts[i] = cte.String()
	}
	return fmt.Sprintf("WITH %s", strings.Join(parts, ", "))
}

// CTEDefinition represents individual CTE definition
type CTEDefinition struct {
	BaseAstNode
	Name      string
	Recursive bool
	Query     *SelectStatement
	Columns   []string // Optional column list
}

func (c *CTEDefinition) String() string {
	var result strings.Builder

	if c.Recursive {
		result.WriteString("RECURSIVE ")
	}

	result.WriteString(c.Name)

	if len(c.Columns) > 0 {
		result.WriteString("(")
		result.WriteString(strings.Join(c.Columns, ", "))
		result.WriteString(")")
	}

	result.WriteString(" AS (")
	if c.Query != nil {
		result.WriteString(c.Query.String())
	}
	result.WriteString(")")

	return result.String()
}

// InsertStatement represents an INSERT statement AST node
type InsertStatement struct {
	BaseAstNode
	Table           AstNode          // table name
	Columns         []AstNode        // column name list (optional)
	ValuesList      [][]AstNode      // multiple VALUES clauses (bulk insert support)
	SelectStmt      AstNode          // SELECT statement (for INSERT INTO ... SELECT)
	BulkVariable    AstNode          // bulk variable (for map arrays)
	ReturningClause *ReturningClause // RETURNING句（オプション）
}

func (i *InsertStatement) String() string {
	var parts []string
	parts = append(parts, "INSERT INTO")
	parts = append(parts, i.Table.String())

	if len(i.Columns) > 0 {
		parts = append(parts, "("+joinNodes(i.Columns)+")")
	}

	if i.BulkVariable != nil {
		parts = append(parts, "VALUES")
		parts = append(parts, i.BulkVariable.String())
	} else if len(i.ValuesList) > 0 {
		parts = append(parts, "VALUES")
		var valueGroups []string
		for _, values := range i.ValuesList {
			var valueStrings []string
			for _, val := range values {
				valueStrings = append(valueStrings, val.String())
			}
			valueGroups = append(valueGroups, "("+strings.Join(valueStrings, ", ")+")")
		}
		parts = append(parts, strings.Join(valueGroups, ", "))
	} else if i.SelectStmt != nil {
		parts = append(parts, i.SelectStmt.String())
	}

	if i.ReturningClause != nil {
		parts = append(parts, i.ReturningClause.String())
	}

	return strings.Join(parts, " ")
}

// UpdateStatement represents an UPDATE statement AST node
type UpdateStatement struct {
	BaseAstNode
	Table           AstNode          // table name
	SetClauses      []AstNode        // list of SET clauses
	WhereClause     *WhereClause     // WHERE clause (optional)
	ReturningClause *ReturningClause // RETURNING句（オプション）
}

func (u *UpdateStatement) String() string {
	var parts []string
	parts = append(parts, "UPDATE")
	parts = append(parts, u.Table.String())
	parts = append(parts, "SET")

	parts = append(parts, joinNodes(u.SetClauses))

	if u.WhereClause != nil {
		parts = append(parts, u.WhereClause.String())
	}

	if u.ReturningClause != nil {
		parts = append(parts, u.ReturningClause.String())
	}

	return strings.Join(parts, " ")
}

// DeleteStatement represents a DELETE statement AST node
type DeleteStatement struct {
	BaseAstNode
	Table       AstNode      // table name
	WhereClause *WhereClause // WHERE clause (optional)
}

func (d *DeleteStatement) String() string {
	var parts []string
	parts = append(parts, "DELETE FROM")
	parts = append(parts, d.Table.String())

	if d.WhereClause != nil {
		parts = append(parts, d.WhereClause.String())
	}

	return strings.Join(parts, " ")
}

// SetClause represents a SET clause in UPDATE statement AST node
type SetClause struct {
	BaseAstNode
	Column AstNode // column name
	Value  AstNode // assigned value
}

func (s *SetClause) String() string {
	return s.Column.String() + " = " + s.Value.String()
}

// ReturningClause represents a RETURNING clause in INSERT/UPDATE statements
// Fields: list of expressions (AstNode)
type ReturningClause struct {
	BaseAstNode
	Fields []AstNode
}

func (r *ReturningClause) String() string {
	return "RETURNING " + joinNodes(r.Fields)
}
