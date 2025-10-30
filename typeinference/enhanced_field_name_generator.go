package typeinference

import (
	"slices"
	"strings"

	"github.com/shibukawa/snapsql/parser"
)

// EnhancedFieldNameGenerator provides advanced field name generation for complex expressions
type EnhancedFieldNameGenerator struct {
	*FieldNameGenerator // Embed basic generator
}

// NewEnhancedFieldNameGenerator creates a new enhanced field name generator
func NewEnhancedFieldNameGenerator() *EnhancedFieldNameGenerator {
	return &EnhancedFieldNameGenerator{
		FieldNameGenerator: NewFieldNameGenerator(),
	}
}

// GenerateComplexFieldName generates meaningful names for complex expressions using AST nodes
func (g *EnhancedFieldNameGenerator) GenerateComplexFieldName(selectField *parser.SelectField) string {
	// Use explicit field name if provided
	if selectField.ExplicitName && selectField.FieldName != "" {
		return g.makeUnique(selectField.FieldName)
	}

	// Use TypeName if available (for CAST expressions)
	if selectField.TypeName != "" && selectField.OriginalField != "" {
		return g.makeUnique(g.extractMainIdentifier(selectField.OriginalField) + "_as_" + strings.ToLower(selectField.TypeName))
	}

	// Generate names based on field kind and content
	switch selectField.FieldKind {
	case parser.SingleField:
		return g.makeUnique(selectField.OriginalField)
	case parser.TableField:
		// Extract field name from table.field
		if selectField.TableName != "" && selectField.OriginalField != "" {
			return g.makeUnique(selectField.OriginalField)
		}

		return g.makeUnique("table_field")
	case parser.FunctionField:
		return g.generateFunctionFieldName(selectField.OriginalField)
	case parser.ComplexField:
		return g.generateComplexFieldName(selectField.OriginalField)
	case parser.LiteralField:
		return g.generateLiteralFieldName(selectField.OriginalField)
	default:
		return g.makeUnique("field")
	}
}

// generateFunctionFieldName generates names for function calls
func (g *EnhancedFieldNameGenerator) generateFunctionFieldName(content string) string {
	if content == "" {
		return g.makeUnique("function_result")
	}

	upper := strings.ToUpper(content)

	// Common aggregate functions
	functions := map[string]string{
		"COUNT":    "count",
		"SUM":      "sum",
		"AVG":      "average",
		"MIN":      "minimum",
		"MAX":      "maximum",
		"COALESCE": "coalesced",
		"CONCAT":   "concatenated",
		"UPPER":    "uppercased",
		"LOWER":    "lowercased",
		"LENGTH":   "length",
		"TRIM":     "trimmed",
	}

	for fnName, baseName := range functions {
		if strings.Contains(upper, fnName+"(") {
			// Try to extract the argument
			start := strings.Index(upper, fnName+"(") + len(fnName) + 1

			end := strings.LastIndex(content, ")")
			if end > start {
				arg := strings.TrimSpace(content[start:end])

				argName := g.extractMainIdentifier(arg)
				if argName != "" {
					return g.makeUnique(baseName + "_" + argName)
				}
			}

			return g.makeUnique(baseName + "_value")
		}
	}

	// Fallback: extract first identifier
	identifier := g.extractMainIdentifier(content)
	if identifier != "" {
		return g.makeUnique(identifier + "_result")
	}

	return g.makeUnique("function_result")
}

// generateComplexFieldName generates names for complex expressions
func (g *EnhancedFieldNameGenerator) generateComplexFieldName(content string) string {
	if content == "" {
		return g.makeUnique("expression")
	}

	// Handle JSON operations FIRST (before arithmetic which might conflict with -)
	if name := g.parseJSONExpression(content); name != "" {
		return g.makeUnique(name)
	}

	// Handle CASE expressions
	if name := g.parseCaseExpression(content); name != "" {
		return g.makeUnique(name)
	}

	// Handle CAST expressions
	if name := g.parseCastExpression(content); name != "" {
		return g.makeUnique(name)
	}

	// Handle arithmetic expressions
	if name := g.parseArithmeticExpression(content); name != "" {
		return g.makeUnique(name)
	}

	// Handle string operations
	if name := g.parseStringExpression(content); name != "" {
		return g.makeUnique(name)
	}

	// Handle comparison operations
	if name := g.parseComparisonExpression(content); name != "" {
		return g.makeUnique(name)
	}

	// Fallback: extract main identifier
	identifier := g.extractMainIdentifier(content)
	if identifier != "" {
		return g.makeUnique(identifier + "_expr")
	}

	return g.makeUnique("expression")
}

// generateLiteralFieldName generates names for literal values
func (g *EnhancedFieldNameGenerator) generateLiteralFieldName(content string) string {
	content = strings.TrimSpace(content)

	if content == "" {
		return g.makeUnique("literal")
	}

	// Handle different literal types
	if content == "NULL" {
		return g.makeUnique("null_value")
	}

	if strings.HasPrefix(content, "'") && strings.HasSuffix(content, "'") {
		// String literal
		return g.makeUnique("string_literal")
	}

	if strings.Contains(content, ".") {
		// Decimal literal
		return g.makeUnique("decimal_literal")
	}

	// Integer literal or other
	return g.makeUnique("literal")
}

// parseCaseExpression generates names for CASE expressions
func (g *EnhancedFieldNameGenerator) parseCaseExpression(content string) string {
	upper := strings.ToUpper(content)
	if !strings.Contains(upper, "CASE") {
		return ""
	}

	// Extract context from conditions and results
	if strings.Contains(upper, "STATUS") {
		return "status_category"
	}

	if strings.Contains(upper, "SCORE") || strings.Contains(upper, "GRADE") {
		return "grade_level"
	}

	if strings.Contains(upper, "AMOUNT") {
		return "price_range"
	}

	if strings.Contains(upper, "PRICE") {
		return "price_range"
	}

	if strings.Contains(upper, "AGE") {
		return "age_group"
	}

	if strings.Contains(upper, "'HIGH'") && strings.Contains(upper, "'LOW'") {
		return "priority_level"
	}

	if strings.Contains(upper, "'ACTIVE'") || strings.Contains(upper, "'INACTIVE'") {
		return "status_flag"
	}

	return "case_result"
}

// parseCastExpression generates names for CAST expressions
func (g *EnhancedFieldNameGenerator) parseCastExpression(content string) string {
	upper := strings.ToUpper(content)

	// Handle CAST(expr AS type) syntax
	if strings.Contains(upper, "CAST(") && strings.Contains(upper, " AS ") {
		start := strings.Index(upper, "CAST(") + 5

		asIndex := strings.Index(upper, " AS ")
		if asIndex > start {
			expr := strings.TrimSpace(content[start:asIndex])
			baseName := g.extractMainIdentifier(expr)

			// Extract target type
			typeStart := asIndex + 4

			parenIndex := strings.Index(content[typeStart:], ")")
			if parenIndex > 0 {
				targetType := strings.TrimSpace(strings.ToLower(content[typeStart : typeStart+parenIndex]))
				if baseName != "" {
					return baseName + "_as_" + targetType
				}

				return "value_as_" + targetType
			}
		}
	}

	// Handle PostgreSQL :: cast syntax
	if strings.Contains(content, "::") {
		parts := strings.Split(content, "::")
		if len(parts) == 2 {
			expr := strings.TrimSpace(parts[0])
			targetType := strings.TrimSpace(strings.ToLower(parts[1]))

			baseName := g.extractMainIdentifier(expr)
			if baseName != "" {
				return baseName + "_as_" + targetType
			}

			return "value_as_" + targetType
		}
	}

	return ""
}

// parseArithmeticExpression generates names for arithmetic expressions
func (g *EnhancedFieldNameGenerator) parseArithmeticExpression(content string) string {
	operators := []struct {
		symbol string
		word   string
	}{
		{"+", "plus"},
		{"-", "minus"},
		{"*", "times"},
		{"/", "divided_by"},
	}

	for _, op := range operators {
		if strings.Contains(content, op.symbol) {
			parts := strings.Split(content, op.symbol)
			if len(parts) == 2 {
				left := g.extractMainIdentifier(strings.TrimSpace(parts[0]))

				right := g.extractMainIdentifier(strings.TrimSpace(parts[1]))
				if left != "" && right != "" {
					return left + "_" + op.word + "_" + right
				}

				if left != "" {
					return left + "_" + op.word + "_value"
				}
			}
		}
	}

	return ""
}

// parseJSONExpression generates names for JSON operations
func (g *EnhancedFieldNameGenerator) parseJSONExpression(content string) string {
	jsonOps := []struct {
		symbol string
		suffix string
	}{
		{"->>", "_text"}, // Must check ->> before ->
		{"->", "_field"},
		{"#>>", "_path_text"}, // Must check #>> before #>
		{"#>", "_path_field"},
	}

	for _, op := range jsonOps {
		if strings.Contains(content, op.symbol) {
			parts := strings.Split(content, op.symbol)
			if len(parts) == 2 {
				left := g.extractMainIdentifier(strings.TrimSpace(parts[0]))
				right := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				right = strings.ToLower(strings.ReplaceAll(right, " ", "_"))
				right = strings.ReplaceAll(right, "{", "")
				right = strings.ReplaceAll(right, "}", "")

				right = strings.ReplaceAll(right, ",", "_")
				if left != "" && right != "" && !strings.Contains(right, "_") {
					// Only use right part if it's a simple key name
					return left + "_" + right + op.suffix
				}

				if left != "" {
					return left + "_json" + op.suffix
				}
			}
		}
	}

	return ""
}

// parseStringExpression generates names for string operations
func (g *EnhancedFieldNameGenerator) parseStringExpression(content string) string {
	upper := strings.ToUpper(content)

	// String concatenation with ||
	if strings.Contains(content, "||") {
		parts := strings.Split(content, "||")
		if len(parts) >= 2 {
			var identifiers []string

			for _, part := range parts {
				if id := g.extractMainIdentifier(strings.TrimSpace(part)); id != "" {
					identifiers = append(identifiers, id)
				}
			}

			if len(identifiers) >= 2 {
				return strings.Join(identifiers[:2], "_") + "_concat"
			}

			if len(identifiers) == 1 {
				return identifiers[0] + "_concat"
			}
		}

		return "string_concat"
	}

	// CONCAT function
	if strings.Contains(upper, "CONCAT(") {
		return "concat_result"
	}

	return ""
}

// parseComparisonExpression generates names for comparison expressions
func (g *EnhancedFieldNameGenerator) parseComparisonExpression(content string) string {
	operators := []struct {
		symbol string
		word   string
	}{
		{">=", "gte"},
		{"<=", "lte"},
		{"<>", "not_equal"},
		{"!=", "not_equal"},
		{"=", "equals"},
		{">", "greater"},
		{"<", "less"},
	}

	for _, op := range operators {
		if strings.Contains(content, op.symbol) {
			parts := strings.Split(content, op.symbol)
			if len(parts) == 2 {
				left := g.extractMainIdentifier(strings.TrimSpace(parts[0]))
				if left != "" {
					return left + "_" + op.word
				}
			}
		}
	}

	return ""
}

// extractMainIdentifier extracts the main identifier from an expression
func (g *EnhancedFieldNameGenerator) extractMainIdentifier(expr string) string {
	if expr == "" {
		return ""
	}

	expr = strings.Trim(expr, " ()")
	words := strings.FieldsSeq(expr)

	for word := range words {
		word = strings.Trim(word, "(),;")
		if g.isValidIdentifier(word) {
			return strings.ToLower(word)
		}
	}

	return ""
}

// isValidIdentifier checks if a word is a valid identifier (not a keyword or operator)
func (g *EnhancedFieldNameGenerator) isValidIdentifier(word string) bool {
	if word == "" {
		return false
	}

	upper := strings.ToUpper(word)

	// Skip SQL keywords
	keywords := []string{
		"SELECT", "FROM", "WHERE", "GROUP", "ORDER", "BY", "HAVING", "LIMIT",
		"INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "TABLE",
		"AND", "OR", "NOT", "IS", "NULL", "IN", "EXISTS", "BETWEEN", "LIKE",
		"CASE", "WHEN", "THEN", "ELSE", "END", "AS", "DISTINCT", "ALL",
		"COUNT", "SUM", "AVG", "MIN", "MAX", "COALESCE", "CAST", "CONCAT",
		"UPPER", "LOWER", "LENGTH", "TRIM", "SUBSTRING", "LEFT", "RIGHT",
	}

	if slices.Contains(keywords, upper) {
		return false
	}

	// Must start with letter or underscore
	if len(word) > 0 && (word[0] >= 'a' && word[0] <= 'z' || word[0] >= 'A' && word[0] <= 'Z' || word[0] == '_') {
		return true
	}

	return false
}

// makeUnique ensures the generated name is unique
func (g *EnhancedFieldNameGenerator) makeUnique(baseName string) string {
	return g.GenerateFieldName(baseName, "", "")
}
