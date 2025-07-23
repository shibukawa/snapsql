package intermediate

import (
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// ResultCardinality represents the expected number of rows returned by a query
type ResultCardinality string

const (
	// CardinalityNone means the query doesn't return any rows (e.g., INSERT without RETURNING)
	CardinalityNone ResultCardinality = "none"
	
	// CardinalityOne means the query returns exactly one row (e.g., SELECT with LIMIT 1)
	CardinalityOne ResultCardinality = "one"
	
	// CardinalityOptional means the query returns zero or one row (e.g., SELECT with unique key)
	CardinalityOptional ResultCardinality = "optional"
	
	// CardinalityMany means the query returns multiple rows
	CardinalityMany ResultCardinality = "many"
)

// DetermineResultCardinality analyzes a SQL statement and determines its result cardinality
func DetermineResultCardinality(stmt parser.StatementNode) ResultCardinality {
	// For now, we'll use a simple approach - assume all queries return multiple rows
	// In a real implementation, we would analyze the statement structure
	
	// Check for LIMIT 1 in clauses
	for _, clause := range stmt.Clauses() {
		if limitClause, ok := clause.(*parser.LimitClause); ok {
			// Check if the limit value is 1 by examining the tokens
			if hasLimitOne(limitClause) {
				return CardinalityOne
			}
		}
	}
	
	// Default to many
	return CardinalityMany
}

// hasLimitOne checks if a LimitClause has a value of 1
func hasLimitOne(limitClause *parser.LimitClause) bool {
	// Get content tokens
	tokens := limitClause.ContentTokens()
	
	// Look for a NUMBER token with value "1"
	for i, token := range tokens {
		if token.Type == tokenizer.NUMBER && token.Value == "1" {
			// Check if this is preceded by a directive comment
			if i > 0 && isPrecedingDirectiveComment(tokens, i) {
				// This is a variable substitution, not a literal 1
				return false
			}
			return true
		}
	}
	
	return false
}

// isPrecedingDirectiveComment checks if the token at the given index is preceded by a directive comment
func isPrecedingDirectiveComment(tokens []tokenizer.Token, index int) bool {
	// Check for /*= */ pattern before the current token
	for i := index - 1; i >= 0; i-- {
		token := tokens[i]
		
		// Skip whitespace
		if token.Type == tokenizer.WHITESPACE {
			continue
		}
		
		// Check for directive comment
		if (token.Type == tokenizer.BLOCK_COMMENT || token.Type == tokenizer.LINE_COMMENT) && isDirectiveComment(token.Value) {
			return true
		}
		
		// Any other token means no directive comment
		return false
	}
	
	return false
}

// isDirectiveComment checks if a comment is a directive comment (/*= */)
func isDirectiveComment(comment string) bool {
	// Check for /*= */ pattern
	if len(comment) >= 4 && comment[:3] == "/*=" && comment[len(comment)-2:] == "*/" {
		return true
	}
	return false
}
