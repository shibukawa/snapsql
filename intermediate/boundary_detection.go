package intermediate

import (
	"regexp"
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

// BoundaryPattern represents a pattern that should use EMIT_UNLESS_BOUNDARY
type BoundaryPattern struct {
	Pattern     *regexp.Regexp
	Description string
}

// Common boundary patterns
var boundaryPatterns = []BoundaryPattern{
	{
		Pattern:     regexp.MustCompile(`^,`),
		Description: "Leading comma (for field lists, SET clauses, etc.)",
	},
	{
		Pattern:     regexp.MustCompile(`^AND$`),
		Description: "AND keyword (for WHERE clauses)",
	},
	{
		Pattern:     regexp.MustCompile(`^OR$`),
		Description: "OR keyword (for WHERE clauses)",
	},
}

// detectBoundaryDelimiter checks if a token value starts with a boundary delimiter
func detectBoundaryDelimiter(value string) bool {
	for _, pattern := range boundaryPatterns {
		if pattern.Pattern.MatchString(value) {
			return true
		}
	}
	return false
}

// isClauseBoundary checks if a token represents a clause boundary (WHERE, FROM, ORDER BY, etc.)
func isClauseBoundary(token tokenizer.Token) bool {
	// For now, we'll use a simple approach - check if it's not a directive or whitespace
	if token.Type == tokenizer.WHITESPACE || token.Type == tokenizer.LINE_COMMENT {
		return false
	}

	// Check if it's a directive (conditional block)
	if token.Type == tokenizer.BLOCK_COMMENT && token.Directive != nil {
		return false
	}

	value := strings.TrimSpace(strings.ToUpper(token.Value))

	// Check for SQL clause keywords
	clauseKeywords := []string{
		"FROM", "WHERE", "ORDER BY", "GROUP BY", "HAVING",
		"LIMIT", "OFFSET", "UNION", "EXCEPT", "INTERSECT",
		")", // Closing parenthesis can also be a boundary
	}

	for _, keyword := range clauseKeywords {
		if strings.HasPrefix(value, keyword) || strings.Contains(value, keyword) {
			return true
		}
	}

	return false
}

// findConditionalBoundaries analyzes tokens to find where EMIT_UNLESS_BOUNDARY and BOUNDARY should be placed
func findConditionalBoundaries(tokens []tokenizer.Token) map[int]string {
	boundaries := make(map[int]string)

	for i, token := range tokens {
		// Check if this token is inside a conditional block and starts with a delimiter
		if token.Type != tokenizer.WHITESPACE && token.Type != tokenizer.LINE_COMMENT &&
			!(token.Type == tokenizer.BLOCK_COMMENT && token.Directive != nil) &&
			isInConditionalBlock(tokens, i) {
			if detectBoundaryDelimiter(token.Value) {
				boundaries[i] = "EMIT_UNLESS_BOUNDARY"
			}
		}

		// Check if this token represents a clause boundary
		if isClauseBoundary(token) {
			// Look back to see if there are any conditional blocks before this
			if hasConditionalBlockBefore(tokens, i) {
				boundaries[i] = "BOUNDARY"
			}
		}
	}

	return boundaries
}

// isInConditionalBlock checks if a token at the given index is inside a conditional block
func isInConditionalBlock(tokens []tokenizer.Token, index int) bool {
	// Look backwards to find the nearest IF/END pair
	ifCount := 0
	for i := index - 1; i >= 0; i-- {
		token := tokens[i]
		if token.Type == tokenizer.BLOCK_COMMENT && token.Directive != nil {
			switch token.Directive.Type {
			case "if", "elseif":
				ifCount++
			case "end":
				ifCount--
			}
		}
	}

	// If ifCount > 0, we're inside a conditional block
	return ifCount > 0
}

// hasConditionalBlockBefore checks if there are any conditional blocks before the given index
func hasConditionalBlockBefore(tokens []tokenizer.Token, index int) bool {
	// Look backwards for IF tokens
	for i := index - 1; i >= 0; i-- {
		token := tokens[i]
		if token.Type == tokenizer.BLOCK_COMMENT && token.Directive != nil &&
			(token.Directive.Type == "if" || token.Directive.Type == "elseif") {
			return true
		}
		// Stop looking if we hit a major clause boundary
		if isClauseBoundary(token) {
			break
		}
	}
	return false
}
