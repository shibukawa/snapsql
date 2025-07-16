package typeinference

import (
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

// CaseExpressionAnalyzer handles CASE expression type inference
type CaseExpressionAnalyzer struct {
	tokens []tokenizer.Token
	engine *TypeInferenceEngine2
}

// CaseWhenClause represents a WHEN clause in a CASE expression
type CaseWhenClause struct {
	Condition []tokenizer.Token // Tokens for the condition
	Result    []tokenizer.Token // Tokens for the result
}

// CaseAnalysisResult contains the analysis result of a CASE expression
type CaseAnalysisResult struct {
	WhenClauses  []CaseWhenClause
	ElseClause   []tokenizer.Token // Tokens for the ELSE clause (if present)
	InferredType *TypeInfo         // Inferred result type
}

// NewCaseExpressionAnalyzer creates a new CASE expression analyzer
func NewCaseExpressionAnalyzer(tokens []tokenizer.Token, engine *TypeInferenceEngine2) *CaseExpressionAnalyzer {
	return &CaseExpressionAnalyzer{
		tokens: tokens,
		engine: engine,
	}
}

// AnalyzeCaseExpression analyzes a CASE expression and infers its type
func (a *CaseExpressionAnalyzer) AnalyzeCaseExpression() (*CaseAnalysisResult, error) {
	if len(a.tokens) == 0 {
		return nil, nil
	}

	// Check if this is a CASE expression
	if a.tokens[0].Type != tokenizer.IDENTIFIER || strings.ToUpper(a.tokens[0].Value) != "CASE" {
		return nil, nil
	}

	result := &CaseAnalysisResult{
		WhenClauses: make([]CaseWhenClause, 0),
	}

	position := 1 // Skip CASE keyword

	// Parse WHEN clauses
	for position < len(a.tokens) {
		if position >= len(a.tokens) {
			break
		}

		token := a.tokens[position]

		// Check for WHEN keyword
		if token.Type == tokenizer.IDENTIFIER && strings.ToUpper(token.Value) == "WHEN" {
			whenClause, newPos := a.parseWhenClause(position)
			if whenClause != nil {
				result.WhenClauses = append(result.WhenClauses, *whenClause)
				position = newPos
			} else {
				break
			}
		} else if token.Type == tokenizer.IDENTIFIER && strings.ToUpper(token.Value) == "ELSE" {
			// Parse ELSE clause
			elseClause, newPos := a.parseElseClause(position)
			if elseClause != nil {
				result.ElseClause = elseClause
				position = newPos
			}
			break
		} else if token.Type == tokenizer.IDENTIFIER && strings.ToUpper(token.Value) == "END" {
			// End of CASE expression
			break
		} else {
			position++
		}
	}

	// Infer the result type based on WHEN/ELSE clauses
	result.InferredType = a.inferCaseResultType(result)

	return result, nil
}

// parseWhenClause parses a WHEN condition THEN result clause
func (a *CaseExpressionAnalyzer) parseWhenClause(startPos int) (*CaseWhenClause, int) {
	if startPos >= len(a.tokens) ||
		a.tokens[startPos].Type != tokenizer.IDENTIFIER ||
		strings.ToUpper(a.tokens[startPos].Value) != "WHEN" {
		return nil, startPos
	}

	position := startPos + 1 // Skip WHEN

	// Parse condition until THEN
	var condition []tokenizer.Token
	for position < len(a.tokens) {
		token := a.tokens[position]
		if token.Type == tokenizer.IDENTIFIER && strings.ToUpper(token.Value) == "THEN" {
			position++ // Skip THEN
			break
		}
		condition = append(condition, token)
		position++
	}

	// Parse result until next WHEN, ELSE, or END
	var result []tokenizer.Token
	for position < len(a.tokens) {
		token := a.tokens[position]
		if token.Type == tokenizer.IDENTIFIER {
			upperValue := strings.ToUpper(token.Value)
			if upperValue == "WHEN" || upperValue == "ELSE" || upperValue == "END" {
				break
			}
		}
		result = append(result, token)
		position++
	}

	return &CaseWhenClause{
		Condition: condition,
		Result:    result,
	}, position
}

// parseElseClause parses the ELSE clause
func (a *CaseExpressionAnalyzer) parseElseClause(startPos int) ([]tokenizer.Token, int) {
	if startPos >= len(a.tokens) ||
		a.tokens[startPos].Type != tokenizer.IDENTIFIER ||
		strings.ToUpper(a.tokens[startPos].Value) != "ELSE" {
		return nil, startPos
	}

	position := startPos + 1 // Skip ELSE

	// Parse result until END
	var result []tokenizer.Token
	for position < len(a.tokens) {
		token := a.tokens[position]
		if token.Type == tokenizer.IDENTIFIER && strings.ToUpper(token.Value) == "END" {
			break
		}
		result = append(result, token)
		position++
	}

	return result, position
}

// inferCaseResultType infers the result type of a CASE expression
func (a *CaseExpressionAnalyzer) inferCaseResultType(caseResult *CaseAnalysisResult) *TypeInfo {
	var resultTypes []*TypeInfo

	// Collect types from all WHEN clauses
	for _, whenClause := range caseResult.WhenClauses {
		if len(whenClause.Result) > 0 {
			analyzer := NewExpressionCastAnalyzer(whenClause.Result, a.engine)
			if resultType, err := analyzer.InferExpressionType(); err == nil && resultType != nil {
				resultTypes = append(resultTypes, resultType)
			}
		}
	}

	// Collect type from ELSE clause
	if len(caseResult.ElseClause) > 0 {
		analyzer := NewExpressionCastAnalyzer(caseResult.ElseClause, a.engine)
		if resultType, err := analyzer.InferExpressionType(); err == nil && resultType != nil {
			resultTypes = append(resultTypes, resultType)
		}
	}

	// If no types found, default to any
	if len(resultTypes) == 0 {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	// Apply type promotion rules
	return a.promoteTypes(resultTypes)
}

// promoteTypes applies type promotion rules to find a common type
func (a *CaseExpressionAnalyzer) promoteTypes(types []*TypeInfo) *TypeInfo {
	if len(types) == 0 {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	if len(types) == 1 {
		return types[0]
	}

	// Start with the first type
	result := &TypeInfo{
		BaseType:   types[0].BaseType,
		IsNullable: types[0].IsNullable,
	}

	// Apply promotion rules
	for i := 1; i < len(types); i++ {
		result = a.promoteTypePair(result, types[i])
	}

	return result
}

// promoteTypePair promotes two types to a common type
func (a *CaseExpressionAnalyzer) promoteTypePair(type1, type2 *TypeInfo) *TypeInfo {
	// If either type is nullable, result is nullable
	isNullable := type1.IsNullable || type2.IsNullable

	// If types are the same, return that type
	if type1.BaseType == type2.BaseType {
		return &TypeInfo{
			BaseType:   type1.BaseType,
			IsNullable: isNullable,
		}
	}

	// Numeric type promotion rules
	if isNumericType(type1.BaseType) && isNumericType(type2.BaseType) {
		promotedType := promoteNumericTypes(type1.BaseType, type2.BaseType)
		return &TypeInfo{
			BaseType:   promotedType,
			IsNullable: isNullable,
		}
	}

	// String type promotion
	if isStringType(type1.BaseType) && isStringType(type2.BaseType) {
		return &TypeInfo{
			BaseType:   "string",
			IsNullable: isNullable,
		}
	}

	// Date/time type promotion
	if isDateTimeType(type1.BaseType) && isDateTimeType(type2.BaseType) {
		return &TypeInfo{
			BaseType:   "timestamp", // Most general date/time type
			IsNullable: isNullable,
		}
	}

	// If no specific promotion rule applies, default to string
	// (most databases convert incompatible types to string in CASE expressions)
	return &TypeInfo{
		BaseType:   "string",
		IsNullable: isNullable,
	}
}

// isNumericType checks if a type is numeric
func isNumericType(baseType string) bool {
	return baseType == "int" || baseType == "float" || baseType == "decimal"
}

// isStringType checks if a type is string-like
func isStringType(baseType string) bool {
	return baseType == "string"
}

// isDateTimeType checks if a type is date/time related
func isDateTimeType(baseType string) bool {
	return baseType == "date" || baseType == "time" || baseType == "timestamp"
}
