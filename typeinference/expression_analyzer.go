package typeinference

import (
	"strconv"
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

// CastInfo represents information about a CAST expression
type CastInfo struct {
	StartPos   int               // CAST start position in token stream
	EndPos     int               // CAST end position in token stream
	TargetType *TypeInfo         // Target type of the CAST
	Expression []tokenizer.Token // Tokens within the CAST expression
}

// ExpressionCastAnalyzer analyzes expressions for CAST operations
type ExpressionCastAnalyzer struct {
	tokens    []tokenizer.Token
	position  int
	castStack []CastInfo // Stack for nested CAST expressions
	engine    *TypeInferenceEngine2
}

// NewExpressionCastAnalyzer creates a new expression cast analyzer
func NewExpressionCastAnalyzer(tokens []tokenizer.Token, engine *TypeInferenceEngine2) *ExpressionCastAnalyzer {
	return &ExpressionCastAnalyzer{
		tokens:    tokens,
		position:  0,
		castStack: make([]CastInfo, 0),
		engine:    engine,
	}
}

// DetectCasts finds all CAST expressions in the token stream
func (a *ExpressionCastAnalyzer) DetectCasts() []CastInfo {
	var casts []CastInfo

	a.position = 0

	for a.position < len(a.tokens) {
		token := a.tokens[a.position]

		// CAST(expr AS type) pattern
		if token.Type == tokenizer.CAST ||
			(token.Type == tokenizer.IDENTIFIER && strings.ToUpper(token.Value) == "CAST") {
			if cast := a.parseCastExpression(); cast != nil {
				casts = append(casts, *cast)
			}
		}

		// PostgreSQL type conversion expr::type pattern
		if token.Type == tokenizer.DOUBLE_COLON {
			if cast := a.parsePostgreSQLCast(); cast != nil {
				casts = append(casts, *cast)
			}
		}

		a.position++
	}

	return casts
}

// InferExpressionType infers the type of an expression with CAST analysis
func (a *ExpressionCastAnalyzer) InferExpressionType() (*TypeInfo, error) {
	// First, check for CAST expressions
	casts := a.DetectCasts()

	// If we have CAST expressions, use the target type of the outermost cast
	if len(casts) > 0 {
		return casts[len(casts)-1].TargetType, nil
	}

	// No CAST found, perform regular type inference
	return a.inferRegularExpression()
}

// parseCastExpression parses a CAST(expr AS type) expression
func (a *ExpressionCastAnalyzer) parseCastExpression() *CastInfo {
	startPos := a.position

	// Expect CAST
	if a.position >= len(a.tokens) ||
		(a.tokens[a.position].Type != tokenizer.CAST &&
			(a.tokens[a.position].Type != tokenizer.IDENTIFIER || strings.ToUpper(a.tokens[a.position].Value) != "CAST")) {
		return nil
	}

	a.position++

	// Expect opening parenthesis
	if a.position >= len(a.tokens) ||
		a.tokens[a.position].Type != tokenizer.OPENED_PARENS {
		return nil
	}

	a.position++

	// Parse expression until AS keyword
	var expression []tokenizer.Token

	parenLevel := 1

	for a.position < len(a.tokens) {
		token := a.tokens[a.position]

		if token.Type == tokenizer.OPENED_PARENS {
			parenLevel++
		} else if token.Type == tokenizer.CLOSED_PARENS {
			parenLevel--
			if parenLevel == 0 {
				break
			}
		} else if token.Type == tokenizer.AS && parenLevel == 1 {
			// Found AS keyword at the right level
			a.position++
			break
		}

		expression = append(expression, token)
		a.position++
	}

	// Parse target type
	targetType := a.parseTypeSpecification()
	if targetType == nil {
		return nil
	}

	// Expect closing parenthesis
	if a.position >= len(a.tokens) ||
		a.tokens[a.position].Type != tokenizer.CLOSED_PARENS {
		return nil
	}

	endPos := a.position

	return &CastInfo{
		StartPos:   startPos,
		EndPos:     endPos,
		TargetType: targetType,
		Expression: expression,
	}
}

// parsePostgreSQLCast parses PostgreSQL-style expr::type cast
func (a *ExpressionCastAnalyzer) parsePostgreSQLCast() *CastInfo {
	if a.position == 0 {
		return nil // No expression before ::
	}

	startPos := a.position - 1 // Include the expression before ::

	// Skip :: operator
	a.position++

	// Parse target type
	targetType := a.parseTypeSpecification()
	if targetType == nil {
		return nil
	}

	// Extract the expression before :: (simple heuristic)
	var expression []tokenizer.Token
	if startPos >= 0 {
		expression = append(expression, a.tokens[startPos])
	}

	return &CastInfo{
		StartPos:   startPos,
		EndPos:     a.position - 1,
		TargetType: targetType,
		Expression: expression,
	}
}

// parseTypeSpecification parses a type specification (e.g., VARCHAR(255), INTEGER)
func (a *ExpressionCastAnalyzer) parseTypeSpecification() *TypeInfo {
	if a.position >= len(a.tokens) {
		return nil
	}

	token := a.tokens[a.position]
	if token.Type != tokenizer.IDENTIFIER {
		return nil
	}

	typeName := strings.ToUpper(token.Value)
	a.position++

	// Check for type parameters (e.g., VARCHAR(255), DECIMAL(10,2))
	var maxLength, precision, scale *int

	if a.position < len(a.tokens) && a.tokens[a.position].Type == tokenizer.OPENED_PARENS {
		a.position++ // Skip (

		// Parse first parameter
		if a.position < len(a.tokens) && a.tokens[a.position].Type == tokenizer.NUMBER {
			if num, err := strconv.Atoi(a.tokens[a.position].Value); err == nil {
				switch {
				case strings.Contains(typeName, "VARCHAR") || strings.Contains(typeName, "CHAR"):
					maxLength = &num
				case strings.Contains(typeName, "DECIMAL") || strings.Contains(typeName, "NUMERIC"):
					precision = &num
				default:
					maxLength = &num
				}
			}

			a.position++

			// Check for comma and second parameter (for DECIMAL(p,s))
			if a.position < len(a.tokens) && a.tokens[a.position].Type == tokenizer.COMMA {
				a.position++ // Skip comma
				if a.position < len(a.tokens) && a.tokens[a.position].Type == tokenizer.NUMBER {
					if num, err := strconv.Atoi(a.tokens[a.position].Value); err == nil {
						scale = &num
					}

					a.position++
				}
			}
		}

		// Skip closing parenthesis
		if a.position < len(a.tokens) && a.tokens[a.position].Type == tokenizer.CLOSED_PARENS {
			a.position++
		}
	}

	// Normalize type name
	baseType := a.normalizeTypeName(typeName)

	return &TypeInfo{
		BaseType:   baseType,
		IsNullable: true, // CAST results can generally be null
		MaxLength:  maxLength,
		Precision:  precision,
		Scale:      scale,
	}
}

// normalizeTypeName normalizes database type names to standard types
func (a *ExpressionCastAnalyzer) normalizeTypeName(typeName string) string {
	typeName = strings.ToUpper(typeName)

	switch {
	case strings.Contains(typeName, "VARCHAR") || strings.Contains(typeName, "TEXT") || strings.Contains(typeName, "CHAR"):
		return "string"
	case typeName == "INTEGER" || typeName == "INT" || typeName == "BIGINT" || typeName == "SMALLINT":
		return "int"
	case typeName == "DECIMAL" || typeName == "NUMERIC":
		return "decimal"
	case typeName == "FLOAT" || typeName == "DOUBLE" || typeName == "REAL":
		return "float"
	case typeName == "BOOLEAN" || typeName == "BOOL":
		return "bool"
	case typeName == "TIMESTAMP" || typeName == "DATETIME":
		return "timestamp"
	case typeName == "DATE":
		return "date"
	case typeName == "TIME":
		return "time"
	case typeName == "JSON" || typeName == "JSONB":
		return "json"
	default:
		return "any"
	}
}

// inferRegularExpression performs type inference for expressions without CAST
func (a *ExpressionCastAnalyzer) inferRegularExpression() (*TypeInfo, error) {
	if len(a.tokens) == 0 {
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	}

	// Simple single token cases
	if len(a.tokens) == 1 {
		return a.inferSingleToken(a.tokens[0])
	}

	// Multi-token expressions
	return a.inferComplexExpression()
}

// inferSingleToken infers type for a single token
func (a *ExpressionCastAnalyzer) inferSingleToken(token tokenizer.Token) (*TypeInfo, error) {
	switch token.Type {
	case tokenizer.NUMBER:
		if strings.Contains(token.Value, ".") {
			return &TypeInfo{BaseType: "float", IsNullable: false}, nil
		}

		return &TypeInfo{BaseType: "int", IsNullable: false}, nil

	case tokenizer.STRING:
		return &TypeInfo{BaseType: "string", IsNullable: false}, nil

	case tokenizer.IDENTIFIER:
		// Try to resolve as column reference
		if a.engine != nil && a.engine.schemaResolver != nil {
			// Simple column resolution - this is a placeholder
			// In reality, we'd need more context about tables
			return &TypeInfo{BaseType: "any", IsNullable: true}, nil
		}

		return &TypeInfo{BaseType: "any", IsNullable: true}, nil

	default:
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	}
}

// inferComplexExpression infers type for complex expressions
func (a *ExpressionCastAnalyzer) inferComplexExpression() (*TypeInfo, error) {
	// Look for function calls
	if len(a.tokens) >= 2 &&
		a.tokens[0].Type == tokenizer.IDENTIFIER &&
		a.tokens[1].Type == tokenizer.OPENED_PARENS {
		return a.inferFunctionCall()
	}

	// Look for binary operators
	for i, token := range a.tokens {
		if isOperatorToken(token) {
			return a.inferBinaryOperation(i)
		}
	}

	// Default to any type for complex expressions
	return &TypeInfo{BaseType: "any", IsNullable: true}, nil
}

// inferFunctionCall infers type for function calls
func (a *ExpressionCastAnalyzer) inferFunctionCall() (*TypeInfo, error) {
	if len(a.tokens) == 0 {
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	}

	funcName := strings.ToUpper(a.tokens[0].Value)

	return a.applyFunctionTypeRule(funcName)
}

// applyFunctionTypeRule applies type inference rules for functions
func (a *ExpressionCastAnalyzer) applyFunctionTypeRule(funcName string) (*TypeInfo, error) {
	switch funcName {
	case "COUNT":
		return &TypeInfo{BaseType: "int", IsNullable: false}, nil
	case "SUM", "AVG":
		return &TypeInfo{BaseType: "decimal", IsNullable: true}, nil
	case "MIN", "MAX":
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	case "LENGTH", "CHAR_LENGTH":
		return &TypeInfo{BaseType: "int", IsNullable: true}, nil
	case "UPPER", "LOWER", "TRIM", "CONCAT":
		return &TypeInfo{BaseType: "string", IsNullable: true}, nil
	default:
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	}
}

// inferBinaryOperation infers type for binary operations
func (a *ExpressionCastAnalyzer) inferBinaryOperation(operatorPos int) (*TypeInfo, error) {
	if operatorPos == 0 || operatorPos >= len(a.tokens)-1 {
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	}

	operator := a.tokens[operatorPos].Value

	// Get left and right operand types
	leftTokens := a.tokens[:operatorPos]
	rightTokens := a.tokens[operatorPos+1:]

	leftAnalyzer := NewExpressionCastAnalyzer(leftTokens, a.engine)
	rightAnalyzer := NewExpressionCastAnalyzer(rightTokens, a.engine)

	leftType, _ := leftAnalyzer.InferExpressionType()
	rightType, _ := rightAnalyzer.InferExpressionType()

	return a.applyOperatorTypeRule(operator, leftType, rightType)
}

// isOperatorToken checks if a token represents an operator
func isOperatorToken(token tokenizer.Token) bool {
	return token.Type == tokenizer.PLUS ||
		token.Type == tokenizer.MINUS ||
		token.Type == tokenizer.MULTIPLY ||
		token.Type == tokenizer.DIVIDE ||
		token.Type == tokenizer.EQUAL ||
		token.Type == tokenizer.NOT_EQUAL ||
		token.Type == tokenizer.LESS_THAN ||
		token.Type == tokenizer.GREATER_THAN ||
		token.Type == tokenizer.LESS_EQUAL ||
		token.Type == tokenizer.GREATER_EQUAL ||
		token.Type == tokenizer.JSON_OPERATOR ||
		token.Value == "||" // String concatenation
}

// applyOperatorTypeRule applies type inference rules for operators
func (a *ExpressionCastAnalyzer) applyOperatorTypeRule(operator string, leftType, rightType *TypeInfo) (*TypeInfo, error) {
	switch operator {
	case "+", "-", "*", "/":
		// Arithmetic operators
		if leftType.BaseType == "int" && rightType.BaseType == "int" {
			if operator == "/" {
				return &TypeInfo{BaseType: "float", IsNullable: leftType.IsNullable || rightType.IsNullable}, nil
			}

			return &TypeInfo{BaseType: "int", IsNullable: leftType.IsNullable || rightType.IsNullable}, nil
		}

		return &TypeInfo{BaseType: "float", IsNullable: leftType.IsNullable || rightType.IsNullable}, nil

	case "=", "<>", "!=", "<", ">", "<=", ">=":
		// Comparison operators always return boolean
		return &TypeInfo{BaseType: "bool", IsNullable: leftType.IsNullable || rightType.IsNullable}, nil

	case "AND", "OR":
		// Logical operators return boolean
		return &TypeInfo{BaseType: "bool", IsNullable: leftType.IsNullable || rightType.IsNullable}, nil

	case "||":
		// String concatenation (PostgreSQL style)
		return &TypeInfo{BaseType: "string", IsNullable: leftType.IsNullable || rightType.IsNullable}, nil

	case "->", "->>", "#>", "#>>":
		// JSON operators (PostgreSQL)
		return a.applyJSONOperatorRule(operator, leftType, rightType)

	default:
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	}
}

// applyJSONOperatorRule applies type inference rules for JSON operators
func (a *ExpressionCastAnalyzer) applyJSONOperatorRule(operator string, leftType, rightType *TypeInfo) (*TypeInfo, error) {
	switch operator {
	case "->":
		// JSON -> key: returns JSON value
		return &TypeInfo{BaseType: "json", IsNullable: true}, nil
	case "->>":
		// JSON ->> key: returns text value
		return &TypeInfo{BaseType: "string", IsNullable: true}, nil
	case "#>":
		// JSON #> path: returns JSON value
		return &TypeInfo{BaseType: "json", IsNullable: true}, nil
	case "#>>":
		// JSON #>> path: returns text value
		return &TypeInfo{BaseType: "string", IsNullable: true}, nil
	default:
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	}
}
