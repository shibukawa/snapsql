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

// Common boundary patterns (leading)
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

// Trailing boundary patterns (for loop contexts)
// These patterns match tokens that END with a boundary delimiter
// but also contain other content before it (e.g., "),")
var trailingBoundaryPatterns = []BoundaryPattern{
	{
		Pattern:     regexp.MustCompile(`.+,$`),
		Description: "Trailing comma (for VALUES clauses in loops, e.g., '),')",
	},
	{
		Pattern:     regexp.MustCompile(`.+\s+AND$`),
		Description: "Trailing AND keyword (for WHERE clauses in loops)",
	},
	{
		Pattern:     regexp.MustCompile(`.+\s+OR$`),
		Description: "Trailing OR keyword (for WHERE clauses in loops)",
	},
}

// detectBoundaryDelimiter checks if a token value starts or ends with a boundary delimiter
func detectBoundaryDelimiter(value string) bool {
	// Check leading boundary patterns
	for _, pattern := range boundaryPatterns {
		if pattern.Pattern.MatchString(value) {
			return true
		}
	}

	// Check trailing boundary patterns
	for _, pattern := range trailingBoundaryPatterns {
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

	// Check for SQL clause keywords using token types
	switch token.Type {
	case tokenizer.FROM, tokenizer.WHERE, tokenizer.GROUP,
		tokenizer.HAVING, tokenizer.LIMIT, tokenizer.OFFSET, tokenizer.UNION,
		tokenizer.CLOSED_PARENS:
		return true
	}

	// Check for keywords that don't have dedicated token types or need special handling
	value := strings.TrimSpace(strings.ToUpper(token.Value))
	stringOnlyKeywords := []string{
		"EXCEPT", "INTERSECT", "ORDER BY", "GROUP BY",
	}

	for _, keyword := range stringOnlyKeywords {
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
			(token.Type != tokenizer.BLOCK_COMMENT || token.Directive == nil) &&
			isInConditionalBlock(tokens, i) {
			if detectBoundaryDelimiter(token.Value) {
				// Check if we're in a FOR loop
				inForLoop := isInForLoop(tokens, i)

				if inForLoop {
					// In a FOR loop: only mark as EMIT_UNLESS_BOUNDARY if it's the last boundary delimiter before LOOP_END
					if isLastBoundaryInLoop(tokens, i) {
						boundaries[i] = "EMIT_UNLESS_BOUNDARY"
					}
					// Otherwise, leave it as regular EMIT_STATIC (don't add to boundaries map)
				} else {
					// In an IF block: always mark as EMIT_UNLESS_BOUNDARY
					boundaries[i] = "EMIT_UNLESS_BOUNDARY"
				}
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

// isInConditionalBlock checks if a token at the given index is inside a conditional block (IF/ELSEIF/FOR)
func isInConditionalBlock(tokens []tokenizer.Token, index int) bool {
	// Use a stack to track directive nesting accurately
	// When traversing backwards, END means we need to skip a block,
	// and IF/ELSEIF/FOR means we're entering (from the perspective of going backwards)
	depth := 0

	for i := index - 1; i >= 0; i-- {
		token := tokens[i]
		if token.Type == tokenizer.BLOCK_COMMENT && token.Directive != nil {
			switch token.Directive.Type {
			case "if", "elseif", "for":
				// When going backwards, if we hit an opening directive
				if depth == 0 {
					// We found an opening directive without a matching END, so we're inside it
					return true
				}
				// Otherwise, this opening matches an END we saw earlier
				depth--
			case "end":
				// When going backwards, END means we need to skip over a block
				depth++
			}
		}
	}

	// If we didn't find any opening directive at depth 0, we're not inside a block
	return false
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

// isInForLoop checks if a token at the given index is inside a FOR loop
func isInForLoop(tokens []tokenizer.Token, index int) bool {
	depth := 0

	for i := index - 1; i >= 0; i-- {
		token := tokens[i]
		if token.Type == tokenizer.BLOCK_COMMENT && token.Directive != nil {
			switch token.Directive.Type {
			case "for":
				if depth == 0 {
					return true
				}

				depth--
			case "if", "elseif":
				if depth == 0 {
					return false
				}

				depth--
			case "end":
				depth++
			}
		}
	}

	return false
}

// isLastBoundaryInLoop checks if a boundary delimiter token is the last one before the **innermost** loop ends
func isLastBoundaryInLoop(tokens []tokenizer.Token, index int) bool {
	// Find the matching END directive for the innermost loop containing this token
	depth := 0
	foundEnd := false
	endIndex := -1

	// Look forward to find the next END directive at depth 0 (innermost loop's end)
	for i := index + 1; i < len(tokens); i++ {
		token := tokens[i]
		if token.Type == tokenizer.BLOCK_COMMENT && token.Directive != nil {
			switch token.Directive.Type {
			case "for", "if", "elseif":
				depth++
			case "end":
				if depth == 0 {
					foundEnd = true
					endIndex = i

					break
				}

				depth--
			}
		}
	}

	if !foundEnd {
		return false
	}

	// Check if there are any other boundary delimiters between this token and the END
	// We need to check at the same nesting level (depth 0)
	checkDepth := 0

	for i := index + 1; i < endIndex; i++ {
		token := tokens[i]

		// Track nesting depth
		if token.Type == tokenizer.BLOCK_COMMENT && token.Directive != nil {
			switch token.Directive.Type {
			case "for", "if", "elseif":
				checkDepth++
			case "end":
				checkDepth--
			}
		}

		// Only check for boundary delimiters at the same nesting level (depth 0)
		if checkDepth == 0 &&
			token.Type != tokenizer.WHITESPACE && token.Type != tokenizer.LINE_COMMENT &&
			(token.Type != tokenizer.BLOCK_COMMENT || token.Directive == nil) {
			if detectBoundaryDelimiter(token.Value) {
				// Found another boundary delimiter after this one at the same level
				return false
			}
		}
	}

	// This is the last boundary delimiter before the innermost loop ends
	return true
}
