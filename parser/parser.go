package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

// Sentinel errors
var (
	ErrUnexpectedToken     = errors.New("unexpected token")
	ErrUnexpectedEOF       = errors.New("unexpected end of file")
	ErrMismatchedParens    = errors.New("mismatched parentheses")
	ErrMismatchedQuotes    = errors.New("mismatched quotes")
	ErrInvalidSyntax       = errors.New("invalid syntax")
	ErrMismatchedDirective = errors.New("mismatched SnapSQL directive")
	ErrConstraintViolation = errors.New("SnapSQL directive clause constraint violation")
	ErrCTEExpectedParen    = errors.New("expected '(' after 'AS' in CTE definition")
	ErrCTEExpectedColumn   = errors.New("expected column name in CTE column list")
)

// SqlParser represents SQL parser
type SqlParser struct {
	tokens          []tokenizer.Token
	current         int
	errors          []ParseError
	namespace       *Namespace       // integrated Namespace (including CEL functionality)
	interfaceSchema *InterfaceSchema // Updated to use unified InterfaceSchema
}

// ParseError represents parse error
type ParseError struct {
	Message  string
	Position tokenizer.Position
	Token    tokenizer.Token
	Severity ErrorSeverity
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("[%s] %s at line %d, column %d (token: %s)",
		e.Severity, e.Message, e.Position.Line, e.Position.Column, e.Token.Value)
}

// ErrorSeverity represents error severity level
type ErrorSeverity int

const (
	WARNING ErrorSeverity = iota
	ERROR
	FATAL
)

func (s ErrorSeverity) String() string {
	switch s {
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// NewSqlParser creates a new SQL parser
// If ns is nil, creates an empty Namespace
func NewSqlParser(tokens []tokenizer.Token, ns *Namespace) *SqlParser {
	// Create empty Namespace if nil
	if ns == nil {
		ns = NewNamespace(nil)
	}

	return &SqlParser{
		tokens:    tokens,
		current:   0,
		errors:    make([]ParseError, 0),
		namespace: ns,
	}
}

// Parse parses SQL and generates AST
func (p *SqlParser) Parse() (AstNode, error) {
	// Basic syntax validation
	if err := p.validateBasicSyntax(); err != nil {
		return nil, err
	}

	// Phase 1: Structure analysis (variable validation is deferred)
	// fmt.Printf("DEBUG: Phase 1 - Structure analysis started\n")
	stmt, err := p.parseStatement(p.namespace)
	if err != nil {
		return nil, err
	}

	// Phase 2: Deferred variable validation
	// fmt.Printf("DEBUG: Phase 2 - Deferred variable validation started\n")
	if selectStmt, ok := stmt.(*SelectStatement); ok {
		if err := ValidateDeferredAST(selectStmt); err != nil {
			return nil, err
		}

		// Generate implicit conditionals (SELECT文のみ)
		processedStmt := GenerateImplicitConditionals(selectStmt, p.interfaceSchema, p.namespace)
		if processedStmt, ok := processedStmt.(*SelectStatement); ok {
			stmt = processedStmt
		}
	}

	return stmt, nil
}

// parseStatement はSQL文の種類を判定してパースする
func (p *SqlParser) parseStatement(ns *Namespace) (AstNode, error) {
	// 先頭のコメントをスキップ
	p.skipLeadingComments()

	// SQL文の種類を判定
	if p.check(tokenizer.SELECT) || p.check(tokenizer.WITH) {
		return p.parseSelectStatement(ns)
	} else if p.check(tokenizer.INSERT) {
		return p.parseInsertStatement(ns)
	} else if p.check(tokenizer.UPDATE) {
		return p.parseUpdateStatement(ns)
	} else if p.check(tokenizer.DELETE) {
		return p.parseDeleteStatement(ns)
	}

	return nil, fmt.Errorf("%w: expected SELECT, INSERT, UPDATE, or DELETE", ErrUnexpectedToken)
}

// GetErrors returns list of parse errors
func (p *SqlParser) GetErrors() []ParseError {
	return p.errors
}

// validateBasicSyntax validates basic syntax
func (p *SqlParser) validateBasicSyntax() error {
	// Validate parentheses matching
	if err := p.validateParentheses(); err != nil {
		return err
	}

	// 引用符の対応チェック
	if err := p.validateQuotes(); err != nil {
		return err
	}

	// SnapSQL拡張の対応チェック
	if err := p.validateSnapSQLDirectives(); err != nil {
		return err
	}

	return nil
}

// validateParentheses は括弧の対応をチェックする
func (p *SqlParser) validateParentheses() error {
	stack := 0
	for _, token := range p.tokens {
		switch token.Type {
		case tokenizer.OPENED_PARENS:
			stack++
		case tokenizer.CLOSED_PARENS:
			stack--
			if stack < 0 {
				return fmt.Errorf("%w: unexpected closing parenthesis at line %d, column %d",
					ErrMismatchedParens, token.Position.Line, token.Position.Column)
			}
		}
	}

	if stack > 0 {
		return fmt.Errorf("%w: %d unclosed parentheses", ErrMismatchedParens, stack)
	}

	return nil
}

// validateQuotes は引用符の対応をチェックする
func (p *SqlParser) validateQuotes() error {
	for _, token := range p.tokens {
		if token.Type == tokenizer.QUOTE {
			// 引用符で始まって引用符で終わっているかチェック
			value := token.Value
			if len(value) < 2 {
				return fmt.Errorf("%w: invalid quote at line %d, column %d",
					ErrMismatchedQuotes, token.Position.Line, token.Position.Column)
			}

			first := value[0]
			last := value[len(value)-1]
			if first != last || (first != '\'' && first != '"') {
				return fmt.Errorf("%w: mismatched quotes at line %d, column %d",
					ErrMismatchedQuotes, token.Position.Line, token.Position.Column)
			}
		}
	}

	return nil
}

// validateSnapSQLDirectives はSnapSQL拡張の対応をチェックする
func (p *SqlParser) validateSnapSQLDirectives() error {
	// Check basic directive matching
	if err := p.validateBasicDirectiveMatching(); err != nil {
		return err
	}

	// Validate clause constraints
	clauseErrors := ValidateDirectiveClauseConstraints(p.tokens)

	// 節制約エラーをパーサーエラーに追加
	p.errors = append(p.errors, clauseErrors...)

	// Stop processing if there are fatal errors
	for _, err := range clauseErrors {
		if err.Severity == FATAL || err.Severity == ERROR {
			return fmt.Errorf("%w: %s", ErrConstraintViolation, err.Message)
		}
	}

	return nil
}

// validateBasicDirectiveMatching checks basic directive matching
func (p *SqlParser) validateBasicDirectiveMatching() error {
	stack := make([]string, 0)

	for _, token := range p.tokens {
		if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
			switch token.DirectiveType {
			case "if":
				stack = append(stack, "if")
			case "elseif":
				if len(stack) == 0 || stack[len(stack)-1] != "if" {
					return fmt.Errorf("%w: elseif without matching if at line %d, column %d",
						ErrMismatchedDirective, token.Position.Line, token.Position.Column)
				}
			case "else":
				if len(stack) == 0 || stack[len(stack)-1] != "if" {
					return fmt.Errorf("%w: else without matching if at line %d, column %d",
						ErrMismatchedDirective, token.Position.Line, token.Position.Column)
				}
			case "for":
				stack = append(stack, "for")
			case "end":
				if len(stack) == 0 {
					return fmt.Errorf("%w: end without matching if/for at line %d, column %d",
						ErrMismatchedDirective, token.Position.Line, token.Position.Column)
				}
				stack = stack[:len(stack)-1]
			}
		}
	}

	if len(stack) > 0 {
		return fmt.Errorf("%w: %d unclosed SnapSQL directives", ErrMismatchedDirective, len(stack))
	}

	return nil
}

func (p *SqlParser) parseSelectStatement(ns *Namespace) (*SelectStatement, error) {
	// 先頭のコメントをスキップ
	p.skipLeadingComments()

	stmt := &SelectStatement{
		BaseAstNode: BaseAstNode{
			nodeType: SELECT_STATEMENT,
			position: p.currentToken().Position,
		},
	}

	// SnapSQLディレクティブをスキップ
	p.skipSnapSQLDirectives()

	// WITH句（オプション）
	if p.match(tokenizer.WITH) {
		withClause, err := p.parseWithClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.WithClause = withClause

		// WITH句の後のSnapSQLディレクティブをスキップ
		p.skipSnapSQLDirectives()
	}

	// SELECT句
	selectClause, err := p.parseSelectClause(ns)
	if err != nil {
		return nil, err
	}
	stmt.SelectClause = selectClause

	// FROM句（オプション）
	if p.match(tokenizer.FROM) {
		fromClause, err := p.parseFromClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.FromClause = fromClause
	}

	// WHERE句（オプション）
	if p.match(tokenizer.WHERE) {
		whereClause, err := p.parseWhereClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.WhereClause = whereClause
	}

	// GROUP BY句（オプション）
	if p.match(tokenizer.GROUP) {
		if !p.match(tokenizer.BY) {
			return nil, fmt.Errorf("%w: expected BY after GROUP", ErrExpectedBy)
		}
		groupByClause, err := p.parseGroupByClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.GroupByClause = groupByClause
	}

	// HAVING句（オプション）
	if p.match(tokenizer.HAVING) {
		havingClause, err := p.parseHavingClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.HavingClause = havingClause
	}

	// ORDER BY句（オプション）
	if p.match(tokenizer.ORDER) {
		if !p.match(tokenizer.BY) {
			return nil, fmt.Errorf("%w: expected BY after ORDER", ErrExpectedBy)
		}
		orderByClause, err := p.parseOrderByClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.OrderByClause = orderByClause
	}

	// LIMIT句（オプション）
	if p.matchWord("LIMIT") {
		limitClause, err := p.parseLimitClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.LimitClause = limitClause
	}

	// OFFSET句（オプション）
	if p.matchWord("OFFSET") {
		offsetClause, err := p.parseOffsetClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.OffsetClause = offsetClause
	}

	return stmt, nil
}

func (p *SqlParser) parseSelectClause(ns *Namespace) (*SelectClause, error) {
	if !p.match(tokenizer.SELECT) {
		return nil, ErrExpectedSelect
	}

	clause := &SelectClause{
		BaseAstNode: BaseAstNode{
			nodeType: SELECT_CLAUSE,
			position: p.previousToken().Position,
		},
		Fields: make([]AstNode, 0),
	}

	// フィールドリストをパース
	for {
		// SnapSQLディレクティブをスキップ（構造系は保持）
		p.skipSnapSQLDirectives()

		// 式の終端チェック
		if p.isAtEnd() || p.isExpressionBoundary() {
			break
		}

		// Process structural SnapSQL directives
		if p.currentToken().Type == tokenizer.BLOCK_COMMENT && p.currentToken().IsSnapSQLDirective {
			token := p.advance() // consume token
			switch token.DirectiveType {
			case "for":
				// fmt.Printf("DEBUG: Starting for statement processing in SELECT clause\n")
				field, err := p.parseTemplateForBlock(ns, token)
				if err != nil {
					return nil, err
				}
				clause.Fields = append(clause.Fields, field)
				continue
			case "if":
				// fmt.Printf("DEBUG: Starting if statement processing in SELECT clause\n")
				field, err := p.parseTemplateIfBlock(ns, token)
				if err != nil {
					return nil, err
				}
				clause.Fields = append(clause.Fields, field)
				continue
			}
		}

		field, err := p.parseExpression(ns)
		if err != nil {
			return nil, err
		}
		clause.Fields = append(clause.Fields, field)

		if !p.match(tokenizer.COMMA) {
			break
		}
	}

	// Error if fields are empty
	if len(clause.Fields) == 0 {
		return nil, ErrSelectMustHaveFields
	}

	return clause, nil
}

// parseFromClause はFROM句をパースする
func (p *SqlParser) parseFromClause(ns *Namespace) (*FromClause, error) {
	clause := &FromClause{
		BaseAstNode: BaseAstNode{
			nodeType: FROM_CLAUSE,
			position: p.previousToken().Position,
		},
		Tables: make([]AstNode, 0),
	}

	// テーブルリストをパース
	for {
		table, err := p.parseExpression(ns)
		if err != nil {
			return nil, err
		}
		clause.Tables = append(clause.Tables, table)

		if !p.match(tokenizer.COMMA) {
			break
		}
	}

	return clause, nil
}

// parseWhereClause はWHERE句をパースする
func (p *SqlParser) parseWhereClause(ns *Namespace) (*WhereClause, error) {
	clause := &WhereClause{
		BaseAstNode: BaseAstNode{
			nodeType: WHERE_CLAUSE,
			position: p.previousToken().Position,
		},
	}

	condition, err := p.parseExpression(ns)
	if err != nil {
		return nil, err
	}
	clause.Condition = condition

	return clause, nil
}

// parseGroupByClause はGROUP BY句をパースする
func (p *SqlParser) parseGroupByClause(ns *Namespace) (*GroupByClause, error) {
	clause := &GroupByClause{
		BaseAstNode: BaseAstNode{
			nodeType: GROUP_BY_CLAUSE,
			position: p.currentToken().Position,
		},
		Fields: make([]AstNode, 0),
	}

	// フィールドリストをパース
	for {
		field, err := p.parseExpression(ns)
		if err != nil {
			return nil, err
		}
		clause.Fields = append(clause.Fields, field)

		if !p.match(tokenizer.COMMA) {
			break
		}
	}

	return clause, nil
}

// parseHavingClause はHAVING句をパースする
func (p *SqlParser) parseHavingClause(ns *Namespace) (*HavingClause, error) {
	clause := &HavingClause{
		BaseAstNode: BaseAstNode{
			nodeType: HAVING_CLAUSE,
			position: p.previousToken().Position,
		},
	}

	condition, err := p.parseExpression(ns)
	if err != nil {
		return nil, err
	}
	clause.Condition = condition

	return clause, nil
}

// parseOrderByClause はORDER BY句をパースする
func (p *SqlParser) parseOrderByClause(ns *Namespace) (*OrderByClause, error) {
	clause := &OrderByClause{
		BaseAstNode: BaseAstNode{
			nodeType: ORDER_BY_CLAUSE,
			position: p.currentToken().Position,
		},
		Fields: make([]AstNode, 0),
	}

	// フィールドリストをパース
	for {
		field, err := p.parseExpression(ns)
		if err != nil {
			return nil, err
		}
		clause.Fields = append(clause.Fields, field)

		if !p.match(tokenizer.COMMA) {
			break
		}
	}

	return clause, nil
}

// parseLimitClause はLIMIT句をパースする
func (p *SqlParser) parseLimitClause(ns *Namespace) (*LimitClause, error) {
	clause := &LimitClause{
		BaseAstNode: BaseAstNode{
			nodeType: LIMIT_CLAUSE,
			position: p.previousToken().Position,
		},
	}

	value, err := p.parseExpression(ns)
	if err != nil {
		return nil, err
	}
	clause.Value = value

	return clause, nil
}

// parseOffsetClause はOFFSET句をパースする
func (p *SqlParser) parseOffsetClause(ns *Namespace) (*OffsetClause, error) {
	clause := &OffsetClause{
		BaseAstNode: BaseAstNode{
			nodeType: OFFSET_CLAUSE,
			position: p.previousToken().Position,
		},
	}

	value, err := p.parseExpression(ns)
	if err != nil {
		return nil, err
	}
	clause.Value = value

	return clause, nil
}

// parseExpression は指定された名前空間で式をパースする
func (p *SqlParser) parseExpression(ns *Namespace) (AstNode, error) {
	var tokens []tokenizer.Token
	startPos := p.currentToken().Position

	// fmt.Printf("DEBUG: parseExpression開始, 現在のトークン: %s\n", p.currentToken().Value)

	// 式の終端まで読み取る
	for !p.isAtEnd() && !p.isExpressionBoundary() {
		token := p.advance()
		// fmt.Printf("DEBUG: Token being processed in parseExpression: %s (type: %s, directive: %t)\n",
		//	token.Value, token.Type, token.IsSnapSQLDirective)
		tokens = append(tokens, token)

		// Process SnapSQL extensions
		if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
			// fmt.Printf("DEBUG: Processing SnapSQL directive: %s\n", token.DirectiveType)
			switch token.DirectiveType {
			case "variable":
				return p.parseVariableSubstitution(ns, token)
			case "env":
				return p.parseEnvironmentReference(ns, token)
			case "if":
				// fmt.Printf("DEBUG: Starting if statement processing\n")
				return p.parseTemplateIfBlock(ns, token)
			case "for":
				// fmt.Printf("DEBUG: Starting for statement processing\n")
				return p.parseTemplateForBlock(ns, token)
			}
		}
	}

	if len(tokens) == 0 {
		return nil, ErrExpectedExpression
	}

	// Special handling for single token case
	if len(tokens) == 1 {
		token := tokens[0]
		switch token.Type {
		case tokenizer.WORD:
			return &Identifier{
				BaseAstNode: BaseAstNode{nodeType: IDENTIFIER, position: token.Position},
				Name:        token.Value,
			}, nil
		case tokenizer.QUOTE, tokenizer.NUMBER:
			return &Literal{
				BaseAstNode: BaseAstNode{nodeType: LITERAL, position: token.Position},
				Value:       token.Value,
			}, nil
		}
	}

	// Process complex expressions as Expression
	return &Expression{
		BaseAstNode: BaseAstNode{nodeType: EXPRESSION, position: startPos},
		Tokens:      tokens,
	}, nil
}

// parseEnvironmentReference はSnapSQL環境参照をパースする
func (p *SqlParser) parseEnvironmentReference(ns *Namespace, token tokenizer.Token) (AstNode, error) {
	expression := p.extractCELExpression(token.Value, "@")

	// Validate CEL expression validity
	if err := ns.ValidateExpression(expression); err != nil {
		return nil, fmt.Errorf("invalid environment reference: %w", err)
	}

	// 環境参照を解決してリテラル値に変換（CELエンジン使用）
	resolvedValue := ""
	if result, err := ns.EvaluateEnvironmentExpression(expression); err == nil {
		resolvedValue = ns.valueToLiteral(result)
	}

	return &EnvironmentReference{
		BaseAstNode:   BaseAstNode{nodeType: ENVIRONMENT_REFERENCE, position: token.Position},
		Expression:    expression,
		ResolvedValue: resolvedValue,
	}, nil
}

// parseVariableSubstitution parses SnapSQL variable substitution with specified namespace
func (p *SqlParser) parseVariableSubstitution(ns *Namespace, token tokenizer.Token) (AstNode, error) {
	expression := p.extractCELExpression(token.Value, "=")

	// Debug output (test only)
	// fmt.Printf("DEBUG: Creating variable substitution '%s' as deferred processing\n", expression)

	dummyValue := ""
	// Detect dummy literals
	if !p.isAtEnd() && p.isDummyLiteral(p.currentToken()) {
		dummyValue = p.advance().Value
	}

	// Create as deferred variable substitution (no immediate validation)
	return &DeferredVariableSubstitution{
		BaseAstNode: BaseAstNode{nodeType: DEFERRED_VARIABLE_SUBSTITUTION, position: token.Position},
		Expression:  expression,
		DummyValue:  dummyValue,
		Namespace:   ns, // 現在の名前空間を保存
	}, nil
}

// parseTemplateIfBlock は指定された名前空間でSnapSQL if文をパースする
func (p *SqlParser) parseTemplateIfBlock(ns *Namespace, token tokenizer.Token) (AstNode, error) {
	condition := p.extractCELExpression(token.Value, "if")

	// Validate CEL expression validity in parameter context
	if err := ns.ValidateParameterExpression(condition); err != nil {
		return nil, fmt.Errorf("invalid if condition: %w", err)
	}

	ifBlock := &TemplateIfBlock{
		BaseAstNode:  BaseAstNode{nodeType: TEMPLATE_IF_BLOCK, position: token.Position},
		Condition:    condition,
		Content:      make([]AstNode, 0),
		ElseIfBlocks: make([]*TemplateElseIfBlock, 0),
	}

	// if文の内容をパース（指定された名前空間を使用）
	for !p.isAtEnd() {
		if p.check(tokenizer.BLOCK_COMMENT) && p.currentToken().IsSnapSQLDirective {
			directive := p.currentToken().DirectiveType
			switch directive {
			case "elseif":
				elseifToken := p.advance()
				elseifCondition := p.extractCELExpression(elseifToken.Value, "elseif")
				if err := ns.ValidateParameterExpression(elseifCondition); err != nil {
					return nil, fmt.Errorf("invalid elseif condition: %w", err)
				}

				elseifBlock := &TemplateElseIfBlock{
					BaseAstNode: BaseAstNode{nodeType: TEMPLATE_ELSEIF_BLOCK, position: elseifToken.Position},
					Condition:   elseifCondition,
					Content:     make([]AstNode, 0),
				}

				// elseif文の内容をパース
				for !p.isAtEnd() {
					if p.checkSnapSQLDirective("else") || p.checkSnapSQLDirective("elseif") || p.checkSnapSQLDirective("end") {
						break
					}
					expr, err := p.parseExpression(ns)
					if err != nil {
						return nil, err
					}
					elseifBlock.Content = append(elseifBlock.Content, expr)
				}

				ifBlock.ElseIfBlocks = append(ifBlock.ElseIfBlocks, elseifBlock)

			case "else":
				p.advance() // else ディレクティブを消費
				elseBlock := &TemplateElseBlock{
					BaseAstNode: BaseAstNode{nodeType: TEMPLATE_ELSE_BLOCK, position: p.previousToken().Position},
					Content:     make([]AstNode, 0),
				}

				// else文の内容をパース
				for !p.isAtEnd() {
					if p.checkSnapSQLDirective("end") {
						break
					}
					expr, err := p.parseExpression(ns)
					if err != nil {
						return nil, err
					}
					elseBlock.Content = append(elseBlock.Content, expr)
				}

				ifBlock.ElseBlock = elseBlock

			case "end":
				p.advance() // end ディレクティブを消費
				return ifBlock, nil
			}
		} else {
			expr, err := p.parseExpression(ns)
			if err != nil {
				return nil, err
			}
			ifBlock.Content = append(ifBlock.Content, expr)
		}
	}

	return nil, fmt.Errorf("%w: expected end directive for if block", ErrExpectedEndDirective)
}

// parseTemplateForBlock は指定された名前空間でSnapSQL for文をパースする
func (p *SqlParser) parseTemplateForBlock(ns *Namespace, token tokenizer.Token) (AstNode, error) {
	// fmt.Printf("DEBUG: parseTemplateForBlock開始\n")
	variable, listExpr := p.parseForDirective(token.Value)
	// fmt.Printf("DEBUG: variable='%s', listExpr='%s'\n", variable, listExpr)

	// Validate CEL expression validity in parameter context
	if err := ns.ValidateParameterExpression(listExpr); err != nil {
		return nil, fmt.Errorf("invalid for list expression: %w", err)
	}

	forBlock := &TemplateForBlock{
		BaseAstNode: BaseAstNode{nodeType: TEMPLATE_FOR_BLOCK, position: token.Position},
		Variable:    variable,
		ListExpr:    listExpr,
		Content:     make([]AstNode, 0),
	}

	// Create new namespace with added loop variable
	// Actual type inference through CEL evaluation
	loopNamespace, err := ns.AddLoopVariableWithEvaluation(variable, listExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate loop expression: %w", err)
	}

	// Debug output (test only)
	// fmt.Printf("DEBUG: Adding loop variable '%s' in for statement, new schema: %+v\n", variable, loopNamespace.Schema.Parameters)

	// Parse for statement content (using namespace containing loop variable)
	for !p.isAtEnd() {
		if p.checkSnapSQLDirective("end") {
			p.advance() // end ディレクティブを消費

			// Update namespace of deferred variables within for statement
			p.updateDeferredVariableNamespaces(forBlock.Content, loopNamespace)

			return forBlock, nil
		}

		expr, err := p.parseExpression(loopNamespace)
		if err != nil {
			return nil, err
		}
		forBlock.Content = append(forBlock.Content, expr)
	}

	return nil, fmt.Errorf("%w: expected end directive for 'for' block", ErrExpectedEndDirective)
}

// updateDeferredVariableNamespaces updates the namespace of deferred variables
func (p *SqlParser) updateDeferredVariableNamespaces(nodes []AstNode, namespace *Namespace) {
	for _, node := range nodes {
		p.updateDeferredVariableNamespace(node, namespace)
	}
}

// updateDeferredVariableNamespace updates the deferred variable namespace of a single node
func (p *SqlParser) updateDeferredVariableNamespace(node AstNode, namespace *Namespace) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *DeferredVariableSubstitution:
		// fmt.Printf("DEBUG: Updating namespace of deferred variable '%s': %+v\n", n.Expression, namespace.Schema.Parameters)
		n.Namespace = namespace
	case *TemplateIfBlock:
		p.updateDeferredVariableNamespaces(n.Content, namespace)
		for _, elseif := range n.ElseIfBlocks {
			p.updateDeferredVariableNamespaces(elseif.Content, namespace)
		}
		if n.ElseBlock != nil {
			p.updateDeferredVariableNamespaces(n.ElseBlock.Content, namespace)
		}
	case *TemplateForBlock:
		// In case of nested for statements, each for statement has its own namespace
		p.updateDeferredVariableNamespaces(n.Content, namespace)
	}
}

func (p *SqlParser) isAtEnd() bool {
	return p.current >= len(p.tokens)
}

func (p *SqlParser) currentToken() tokenizer.Token {
	// 空白トークンをスキップ
	for p.current < len(p.tokens) && p.tokens[p.current].Type == tokenizer.WHITESPACE {
		p.current++
	}

	if p.current >= len(p.tokens) {
		return tokenizer.Token{Type: tokenizer.EOF}
	}
	return p.tokens[p.current]
}

func (p *SqlParser) previousToken() tokenizer.Token {
	if p.current == 0 {
		return tokenizer.Token{Type: tokenizer.EOF}
	}
	return p.tokens[p.current-1]
}

func (p *SqlParser) advance() tokenizer.Token {
	if !p.isAtEnd() {
		p.current++
	}
	// 空白トークンをスキップ
	for !p.isAtEnd() && p.currentToken().Type == tokenizer.WHITESPACE {
		p.current++
	}
	return p.previousToken()
}

func (p *SqlParser) check(tokenType tokenizer.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	// 空白トークンをスキップ
	for !p.isAtEnd() && p.currentToken().Type == tokenizer.WHITESPACE {
		p.current++
	}
	return p.currentToken().Type == tokenType
}

func (p *SqlParser) match(tokenType tokenizer.TokenType) bool {
	if p.check(tokenType) {
		p.advance()
		return true
	}
	return false
}

func (p *SqlParser) matchWord(word string) bool {
	// 空白トークンをスキップ
	for p.current < len(p.tokens) && p.tokens[p.current].Type == tokenizer.WHITESPACE {
		p.current++
	}

	if p.check(tokenizer.WORD) && strings.EqualFold(p.currentToken().Value, word) {
		p.advance()
		return true
	}
	return false
}

func (p *SqlParser) checkSnapSQLDirective(directiveType string) bool {
	return p.check(tokenizer.BLOCK_COMMENT) &&
		p.currentToken().IsSnapSQLDirective &&
		p.currentToken().DirectiveType == directiveType
}

func (p *SqlParser) isExpressionBoundary() bool {
	token := p.currentToken()

	// 構造系SnapSQLディレクティブは式の境界とする
	if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
		if token.DirectiveType == "for" || token.DirectiveType == "if" ||
			token.DirectiveType == "elseif" || token.DirectiveType == "else" ||
			token.DirectiveType == "end" {
			// fmt.Printf("DEBUG: 構造ディレクティブ'%s'を式の境界として認識\n", token.DirectiveType)
			return true
		}
	}

	return token.Type == tokenizer.COMMA ||
		token.Type == tokenizer.SEMICOLON ||
		token.Type == tokenizer.FROM ||
		token.Type == tokenizer.WHERE ||
		token.Type == tokenizer.GROUP ||
		token.Type == tokenizer.HAVING ||
		token.Type == tokenizer.ORDER ||
		(token.Type == tokenizer.WORD && strings.ToUpper(token.Value) == "LIMIT") ||
		(token.Type == tokenizer.WORD && strings.ToUpper(token.Value) == "OFFSET")
}

func (p *SqlParser) isDummyLiteral(token tokenizer.Token) bool {
	// ダミーリテラルの判定ロジック（簡易版）
	return token.Type == tokenizer.WORD ||
		token.Type == tokenizer.QUOTE ||
		token.Type == tokenizer.NUMBER
}

func (p *SqlParser) extractCELExpression(comment, directive string) string {
	// Extract expression from /*= expression */, /*@ expression */, or /*# if expression */
	trimmed := strings.TrimSpace(comment)
	if strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
		content := strings.TrimSpace(trimmed[2 : len(trimmed)-2])

		switch directive {
		case "=":
			if strings.HasPrefix(content, "=") {
				return strings.TrimSpace(content[1:])
			}
		case "@":
			if strings.HasPrefix(content, "@") {
				return strings.TrimSpace(content[1:])
			}
		case "if":
			if strings.HasPrefix(content, "#") {
				content = strings.TrimSpace(content[1:])
				if strings.HasPrefix(content, "if ") {
					return strings.TrimSpace(content[3:])
				}
			}
		case "elseif":
			if strings.HasPrefix(content, "#") {
				content = strings.TrimSpace(content[1:])
				if strings.HasPrefix(content, "elseif ") {
					return strings.TrimSpace(content[7:])
				}
			}
		}
	}
	return ""
}

func (p *SqlParser) parseForDirective(comment string) (variable, listExpr string) {
	// Extract variable and list_expression from /*# for variable : list_expression */
	trimmed := strings.TrimSpace(comment)
	if strings.HasPrefix(trimmed, "/*#") && strings.HasSuffix(trimmed, "*/") {
		content := strings.TrimSpace(trimmed[3 : len(trimmed)-2])
		if strings.HasPrefix(content, "for ") {
			forContent := strings.TrimSpace(content[4:])
			parts := strings.Split(forContent, ":")
			if len(parts) == 2 {
				variable = strings.TrimSpace(parts[0])
				listExpr = strings.TrimSpace(parts[1])
			}
		}
	}
	return
}

// skipLeadingComments skips leading comments (parameter schema, etc.)
// Does not skip SnapSQL directives
func (p *SqlParser) skipLeadingComments() {
	for !p.isAtEnd() {
		token := p.currentToken()
		if token.Type == tokenizer.BLOCK_COMMENT && !token.IsSnapSQLDirective {
			p.advance()
		} else if token.Type == tokenizer.LINE_COMMENT {
			p.advance()
		} else if token.Type == tokenizer.WHITESPACE {
			p.advance()
		} else {
			break
		}
	}
}

// skipSnapSQLDirectives skips comment-based SnapSQL directives and whitespace
// Structural directives (for statements, if statements) are kept as processing targets
func (p *SqlParser) skipSnapSQLDirectives() {
	for !p.isAtEnd() {
		token := p.currentToken()
		if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
			// Keep structural directives as processing targets
			if token.DirectiveType == "for" || token.DirectiveType == "if" ||
				token.DirectiveType == "elseif" || token.DirectiveType == "else" ||
				token.DirectiveType == "end" {
				// fmt.Printf("DEBUG: Keeping structural directive '%s' as processing target\n", token.DirectiveType)
				break
			}
			// Skip comment-based directives
			// fmt.Printf("DEBUG: Skipping comment-based directive '%s'\n", token.DirectiveType)
			p.advance()
		} else if token.Type == tokenizer.WHITESPACE {
			p.advance()
		} else {
			break
		}
	}
}

// parseWithClause parses WITH clause containing CTEs
func (p *SqlParser) parseWithClause(ns *Namespace) (*WithClause, error) {
	withClause := &WithClause{
		BaseAstNode: BaseAstNode{
			nodeType: WITH_CLAUSE,
			position: p.previousToken().Position,
		},
	}

	// RECURSIVE keyword (optional)
	recursive := false
	if p.check(tokenizer.WORD) && p.currentToken().Value == "RECURSIVE" {
		recursive = true
		p.advance()
	}

	// Parse first CTE
	cte, err := p.parseCTEDefinition(ns, recursive)
	if err != nil {
		return nil, err
	}
	withClause.CTEs = append(withClause.CTEs, *cte)

	// Parse additional CTEs (comma-separated)
	for p.match(tokenizer.COMMA) {
		// Skip whitespace
		for p.check(tokenizer.WHITESPACE) {
			p.advance()
		}

		// Check for RECURSIVE keyword for subsequent CTEs
		recursive = false
		if p.check(tokenizer.WORD) && p.currentToken().Value == "RECURSIVE" {
			recursive = true
			p.advance()
		}

		cte, err := p.parseCTEDefinition(ns, recursive)
		if err != nil {
			return nil, err
		}
		withClause.CTEs = append(withClause.CTEs, *cte)
	}

	return withClause, nil
}

// parseCTEDefinition parses individual CTE definition
func (p *SqlParser) parseCTEDefinition(ns *Namespace, recursive bool) (*CTEDefinition, error) {
	// CTE name
	if !p.check(tokenizer.WORD) {
		return nil, fmt.Errorf("%w at line %d, column %d", ErrExpectedCTEName,
			p.currentToken().Position.Line, p.currentToken().Position.Column)
	}

	cteName := p.currentToken().Value
	p.advance()

	cte := &CTEDefinition{
		BaseAstNode: BaseAstNode{
			nodeType: CTE_DEFINITION,
			position: p.previousToken().Position,
		},
		Name:      cteName,
		Recursive: recursive,
	}

	// Optional column list
	if p.match(tokenizer.OPENED_PARENS) {
		columns, err := p.parseCTEColumnList(ns)
		if err != nil {
			return nil, err
		}
		cte.Columns = columns

		if !p.match(tokenizer.CLOSED_PARENS) {
			return nil, fmt.Errorf("%w at line %d, column %d", ErrExpectedCloseParenAfterCTEList,
				p.currentToken().Position.Line, p.currentToken().Position.Column)
		}
	}

	// AS keyword
	if !p.match(tokenizer.AS) {
		return nil, fmt.Errorf("%w at line %d, column %d", ErrExpectedAsAfterCTEName,
			p.currentToken().Position.Line, p.currentToken().Position.Column)
	}

	// Opening parenthesis for subquery
	if !p.match(tokenizer.OPENED_PARENS) {
		return nil, fmt.Errorf("%w at line %d, column %d", ErrCTEExpectedParen,
			p.currentToken().Position.Line, p.currentToken().Position.Column)
	}

	// Parse subquery (SELECT statement)
	// Subqueries in CTE are enclosed in parentheses, so parse only tokens within parentheses
	subqueryTokens := p.extractParenthesesContent()

	// Create parser for subquery
	subParser := NewSqlParser(subqueryTokens, ns)
	// interfaceSchema is inherited from environment, but set explicitly
	if p.interfaceSchema != nil {
		subParser.interfaceSchema = p.interfaceSchema
	}
	subquery, err := subParser.parseSelectStatementWithoutWith(ns)
	if err != nil {
		return nil, fmt.Errorf("error parsing CTE subquery: %w", err)
	}
	cte.Query = subquery

	return cte, nil
}

// parseCTEColumnList parses optional column list in CTE definition
func (p *SqlParser) parseCTEColumnList(_ *Namespace) ([]string, error) {
	var columns []string

	// First column
	if !p.check(tokenizer.WORD) {
		return nil, fmt.Errorf("%w at line %d, column %d", ErrCTEExpectedColumn,
			p.currentToken().Position.Line, p.currentToken().Position.Column)
	}
	columns = append(columns, p.currentToken().Value)
	p.advance()

	// Additional columns (comma-separated)
	for p.match(tokenizer.COMMA) {
		// Skip whitespace
		for p.check(tokenizer.WHITESPACE) {
			p.advance()
		}

		if !p.check(tokenizer.WORD) {
			return nil, fmt.Errorf("%w after ',' at line %d, column %d", ErrCTEExpectedColumn,
				p.currentToken().Position.Line, p.currentToken().Position.Column)
		}
		columns = append(columns, p.currentToken().Value)
		p.advance()
	}

	return columns, nil
}

// parseSelectStatementWithoutWith parses SELECT statement without checking for WITH clause
func (p *SqlParser) parseSelectStatementWithoutWith(ns *Namespace) (*SelectStatement, error) {
	stmt := &SelectStatement{
		BaseAstNode: BaseAstNode{
			nodeType: SELECT_STATEMENT,
			position: p.currentToken().Position,
		},
	}

	// SELECT句
	selectClause, err := p.parseSelectClause(ns)
	if err != nil {
		return nil, err
	}
	stmt.SelectClause = selectClause

	// FROM句（オプション）
	if p.match(tokenizer.FROM) {
		fromClause, err := p.parseFromClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.FromClause = fromClause
	}

	// WHERE句（オプション）
	if p.match(tokenizer.WHERE) {
		whereClause, err := p.parseWhereClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.WhereClause = whereClause
	}

	// GROUP BY句（オプション）
	if p.match(tokenizer.GROUP) && p.match(tokenizer.BY) {
		groupByClause, err := p.parseGroupByClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.GroupByClause = groupByClause
	}

	// HAVING句（オプション）
	if p.match(tokenizer.HAVING) {
		havingClause, err := p.parseHavingClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.HavingClause = havingClause
	}

	// ORDER BY句（オプション）
	if p.match(tokenizer.ORDER) && p.match(tokenizer.BY) {
		orderByClause, err := p.parseOrderByClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.OrderByClause = orderByClause
	}

	// LIMIT句（オプション）
	if p.matchWord("LIMIT") {
		limitClause, err := p.parseLimitClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.LimitClause = limitClause
	}

	// OFFSET句（オプション）
	if p.matchWord("OFFSET") {
		offsetClause, err := p.parseOffsetClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.OffsetClause = offsetClause
	}

	return stmt, nil
}

// extractParenthesesContent extracts tokens within parentheses for subquery parsing
func (p *SqlParser) extractParenthesesContent() []tokenizer.Token {
	var tokens []tokenizer.Token
	parenCount := 0

	// 開始括弧は既に消費されているので、内容から開始
	for !p.isAtEnd() {
		token := p.currentToken()

		if token.Type == tokenizer.OPENED_PARENS {
			parenCount++
			tokens = append(tokens, token)
		} else if token.Type == tokenizer.CLOSED_PARENS {
			if parenCount == 0 {
				// 対応する閉じ括弧に到達
				p.advance() // 閉じ括弧を消費
				break
			}
			parenCount--
			tokens = append(tokens, token)
		} else {
			tokens = append(tokens, token)
		}

		p.advance()
	}

	// EOFトークンを追加
	tokens = append(tokens, tokenizer.Token{
		Type:     tokenizer.EOF,
		Value:    "",
		Position: p.currentToken().Position,
	})

	return tokens
}

// parseInsertStatement はINSERT文をパースする
func (p *SqlParser) parseInsertStatement(ns *Namespace) (*InsertStatement, error) {
	if !p.match(tokenizer.INSERT) {
		return nil, ErrExpectedInsert
	}

	if !p.matchWord("INTO") {
		return nil, ErrExpectedInto
	}

	stmt := &InsertStatement{
		BaseAstNode: BaseAstNode{
			nodeType: INSERT_STATEMENT,
			position: p.previousToken().Position,
		},
	}

	// テーブル名をパース（単純な識別子のみ）
	table, err := p.parseTableName(ns)
	if err != nil {
		return nil, err
	}
	stmt.Table = table

	// カラムリスト（オプション）
	if p.match(tokenizer.OPENED_PARENS) {
		for {
			column, err := p.parseSimpleIdentifier(ns)
			if err != nil {
				return nil, err
			}
			stmt.Columns = append(stmt.Columns, column)

			if !p.match(tokenizer.COMMA) {
				break
			}
		}

		if !p.match(tokenizer.CLOSED_PARENS) {
			return nil, ErrExpectedCloseParen
		}
	}

	// VALUES句またはSELECT文
	if p.matchWord("VALUES") {
		// バルク変数（map配列）の検出を試行
		if p.check(tokenizer.BLOCK_COMMENT) && strings.HasPrefix(p.currentToken().Value, "/*=") {
			// 現在位置を保存
			savedPosition := p.current

			// バルク変数の可能性をチェック
			bulkVar := p.parseBulkVariableSubstitution(ns, stmt.Columns)
			if bulkVar != nil {
				stmt.BulkVariable = bulkVar
				return stmt, nil
			}
			// If not a bulk variable, restore position and return to normal processing
			p.current = savedPosition
		}

		// 複数のVALUES句をパース（バルクインサート対応）
		for {
			if !p.match(tokenizer.OPENED_PARENS) {
				return nil, ErrExpectedValuesAfter
			}

			var values []AstNode
			for {
				value, err := p.parseSimpleValue(ns)
				if err != nil {
					return nil, err
				}
				values = append(values, value)

				if !p.match(tokenizer.COMMA) {
					break
				}
			}

			if !p.match(tokenizer.CLOSED_PARENS) {
				return nil, ErrExpectedCloseParen
			}

			stmt.ValuesList = append(stmt.ValuesList, values)

			// 次のVALUES句があるかチェック
			if !p.match(tokenizer.COMMA) {
				break
			}
		}
	} else if p.check(tokenizer.SELECT) {
		// INSERT INTO ... SELECT の場合
		selectStmt, err := p.parseSelectStatement(ns)
		if err != nil {
			return nil, err
		}
		stmt.SelectStmt = selectStmt
	} else {
		return nil, ErrExpectedValues
	}

	return stmt, nil
}

// parseBulkVariableSubstitution はバルク変数置換をパースする
func (p *SqlParser) parseBulkVariableSubstitution(ns *Namespace, columns []AstNode) *BulkVariableSubstitution {
	if !p.check(tokenizer.BLOCK_COMMENT) {
		return nil
	}

	comment := p.advance()
	if !strings.HasPrefix(comment.Value, "/*=") || !strings.HasSuffix(comment.Value, "*/") {
		return nil
	}

	// /*= expression */の形式から式を抽出
	content := strings.TrimSpace(comment.Value[3 : len(comment.Value)-2])

	// 名前空間でバルク変数かどうかを確認
	if ns != nil {
		paramType := ns.Schema.Parameters[content]
		if !isBulkVariableType(paramType) {
			return nil
		}
	}

	// カラム名を抽出
	var columnNames []string
	for _, col := range columns {
		if ident, ok := col.(*Identifier); ok {
			columnNames = append(columnNames, ident.Name)
		}
	}

	// ダミー値を取得（次のトークンがダミー値の場合）
	var dummyValue string
	if p.current < len(p.tokens) {
		nextToken := p.currentToken()
		if nextToken.Type == tokenizer.OPENED_PARENS {
			// ダミー値のVALUES句をパース
			p.advance() // '('を消費
			var dummyValues []string
			for {
				if p.check(tokenizer.QUOTE) || p.check(tokenizer.NUMBER) {
					dummyValues = append(dummyValues, p.advance().Value)
				} else {
					break
				}
				if !p.match(tokenizer.COMMA) {
					break
				}
			}
			if p.match(tokenizer.CLOSED_PARENS) {
				dummyValue = "(" + strings.Join(dummyValues, ", ") + ")"
			}
		}
	}

	return &BulkVariableSubstitution{
		BaseAstNode: BaseAstNode{nodeType: BULK_VARIABLE_SUBSTITUTION},
		Expression:  content,
		Columns:     columnNames,
		DummyValue:  dummyValue,
	}
}

// isBulkVariableType はパラメータタイプがバルク変数（map配列）かどうかを判定する
func isBulkVariableType(paramType any) bool {
	if paramType == nil {
		return false
	}

	// []map[string]any または []any の形式をチェック
	switch t := paramType.(type) {
	case []any:
		if len(t) > 0 {
			if _, ok := t[0].(map[string]any); ok {
				return true
			}
		}
		return true // 空配列でも一応バルク変数として扱う
	case []map[string]any:
		return true
	case map[string]any:
		// 単一mapはバルク変数ではない
		return false
	default:
		return false
	}
}

// parseUpdateStatement はUPDATE文をパースする
func (p *SqlParser) parseUpdateStatement(ns *Namespace) (*UpdateStatement, error) {
	if !p.match(tokenizer.UPDATE) {
		return nil, ErrExpectedUpdate
	}

	stmt := &UpdateStatement{
		BaseAstNode: BaseAstNode{
			nodeType: UPDATE_STATEMENT,
			position: p.previousToken().Position,
		},
	}

	// テーブル名をパース（単純な識別子のみ）
	table, err := p.parseTableName(ns)
	if err != nil {
		return nil, err
	}
	stmt.Table = table

	// SET句
	if !p.matchWord("SET") {
		return nil, ErrExpectedSet
	}

	for {
		// カラム名
		column, err := p.parseSimpleIdentifier(ns)
		if err != nil {
			return nil, err
		}

		// =
		if !p.match(tokenizer.EQUAL) {
			return nil, ErrExpectedEquals
		}

		// 値
		value, err := p.parseSimpleValue(ns)
		if err != nil {
			return nil, err
		}

		setClause := &SetClause{
			BaseAstNode: BaseAstNode{
				nodeType: SET_CLAUSE,
				position: column.Position(),
			},
			Column: column,
			Value:  value,
		}
		stmt.SetClauses = append(stmt.SetClauses, setClause)

		if !p.match(tokenizer.COMMA) {
			break
		}
	}

	// WHERE句（オプション）
	if p.match(tokenizer.WHERE) {
		whereClause, err := p.parseWhereClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.WhereClause = whereClause
	}

	return stmt, nil
}

// parseDeleteStatement はDELETE文をパースする
func (p *SqlParser) parseDeleteStatement(ns *Namespace) (*DeleteStatement, error) {
	if !p.match(tokenizer.DELETE) {
		return nil, ErrExpectedDelete
	}

	if !p.match(tokenizer.FROM) {
		return nil, ErrExpectedFrom
	}

	stmt := &DeleteStatement{
		BaseAstNode: BaseAstNode{
			nodeType: DELETE_STATEMENT,
			position: p.previousToken().Position,
		},
	}

	// テーブル名をパース（単純な識別子のみ）
	table, err := p.parseTableName(ns)
	if err != nil {
		return nil, err
	}
	stmt.Table = table

	// WHERE句（オプション）
	if p.match(tokenizer.WHERE) {
		whereClause, err := p.parseWhereClause(ns)
		if err != nil {
			return nil, err
		}
		stmt.WhereClause = whereClause
	}

	return stmt, nil
}

// parseTableName はテーブル名をパースする（単純な識別子のみ）
func (p *SqlParser) parseTableName(ns *Namespace) (AstNode, error) {
	var tokens []tokenizer.Token
	startPos := p.currentToken().Position

	// テーブル名は単純な識別子またはSnapSQL変数置換
	for !p.isAtEnd() && !p.isTableNameBoundary() {
		token := p.advance()

		// Process SnapSQL variable substitution
		if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
			switch token.DirectiveType {
			case "variable":
				return p.parseVariableSubstitution(ns, token)
			case "env":
				return p.parseEnvironmentReference(ns, token)
			}
		}

		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		return nil, ErrExpectedTableName
	}

	// Process as simple identifier
	return &Identifier{
		BaseAstNode: BaseAstNode{
			nodeType: IDENTIFIER,
			position: startPos,
		},
		Name: tokens[0].Value,
	}, nil
}

// isTableNameBoundary はテーブル名の終端を判定する
func (p *SqlParser) isTableNameBoundary() bool {
	token := p.currentToken()

	// キーワードで終端
	if token.Type == tokenizer.WORD {
		upperValue := strings.ToUpper(token.Value)
		switch upperValue {
		case "VALUES", "SET", "WHERE", "SELECT":
			return true
		}
	}

	// 括弧で終端
	if token.Type == tokenizer.OPENED_PARENS {
		return true
	}

	// セミコロンやEOFで終端
	if token.Type == tokenizer.SEMICOLON || token.Type == tokenizer.EOF {
		return true
	}

	return false
}

// parseSimpleIdentifier は単純な識別子をパースする
func (p *SqlParser) parseSimpleIdentifier(ns *Namespace) (AstNode, error) {
	token := p.currentToken()

	// SnapSQL変数置換の処理
	if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
		switch token.DirectiveType {
		case "variable":
			p.advance() // トークンを消費
			return p.parseVariableSubstitution(ns, token)
		case "env":
			p.advance() // トークンを消費
			return p.parseEnvironmentReference(ns, token)
		}
	}

	// 通常の識別子
	if token.Type == tokenizer.WORD {
		p.advance()
		return &Identifier{
			BaseAstNode: BaseAstNode{
				nodeType: IDENTIFIER,
				position: token.Position,
			},
			Name: token.Value,
		}, nil
	}

	return nil, ErrExpectedIdentifier
}

// parseSimpleValue は単純な値をパースする
func (p *SqlParser) parseSimpleValue(ns *Namespace) (AstNode, error) {
	token := p.currentToken()

	// SnapSQL変数置換の処理
	if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
		switch token.DirectiveType {
		case "variable":
			p.advance() // トークンを消費
			return p.parseVariableSubstitution(ns, token)
		case "env":
			p.advance() // トークンを消費
			return p.parseEnvironmentReference(ns, token)
		}
	}

	// リテラル値
	switch token.Type {
	case tokenizer.QUOTE, tokenizer.NUMBER, tokenizer.WORD:
		p.advance()
		return &Literal{
			BaseAstNode: BaseAstNode{
				nodeType: LITERAL,
				position: token.Position,
			},
			Value: token.Value,
		}, nil
	}

	return nil, ErrExpectedValue
}
