package parserstep6

import (
	"regexp"
	"strings"

	"github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// DirectiveExpressionExtractor extracts expression from SnapSQL directive tokens
type DirectiveExpressionExtractor struct{}

// NewDirectiveExpressionExtractor creates a new directive expression extractor
func NewDirectiveExpressionExtractor() *DirectiveExpressionExtractor {
	return &DirectiveExpressionExtractor{}
}

// ExtractDirectiveExpressions processes a statement and extracts expressions from directive tokens
func (dee *DirectiveExpressionExtractor) ExtractDirectiveExpressions(statement parsercommon.StatementNode) error {
	if statement == nil {
		return nil
	}

	// Process all clauses in the statement
	for _, clause := range statement.Clauses() {
		if err := dee.processClauseTokens(clause.RawTokens()); err != nil {
			return err
		}
	}

	return nil
}

// processClauseTokens processes tokens in a clause and extracts directive expressions
func (dee *DirectiveExpressionExtractor) processClauseTokens(tokens []tokenizer.Token) error {
	for i := range tokens {
		if tokens[i].Directive != nil {
			if err := dee.extractExpressionFromToken(&tokens[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// extractExpressionFromToken extracts expression from a directive token and updates the Directive
func (dee *DirectiveExpressionExtractor) extractExpressionFromToken(token *tokenizer.Token) error {
	if token.Directive == nil {
		return nil
	}

	directive := token.Directive
	expression, err := dee.parseDirectiveExpression(token.Value, directive.Type)
	if err != nil {
		return err
	}

	// Update the directive with the extracted expression
	directive.Condition = expression

	return nil
}

// parseDirectiveExpression parses directive token value and extracts the expression
func (dee *DirectiveExpressionExtractor) parseDirectiveExpression(tokenValue, directiveType string) (string, error) {
	switch directiveType {
	case "if", "elseif":
		return dee.parseIfExpression(tokenValue)
	case "for":
		return dee.parseForExpression(tokenValue)
	case "variable":
		return dee.parseVariableExpression(tokenValue)
	case "const":
		return dee.parseConstExpression(tokenValue)
	case "else", "end":
		// These directives don't have expressions
		return "", nil
	default:
		return "", nil
	}
}

// parseIfExpression parses if/elseif directive expression
// Format: /*# if condition */ or /*# elseif condition */
func (dee *DirectiveExpressionExtractor) parseIfExpression(tokenValue string) (string, error) {
	// Remove comment markers
	content := strings.TrimSpace(tokenValue)
	if !strings.HasPrefix(content, "/*#") || !strings.HasSuffix(content, "*/") {
		return "", ErrInvalidDirectiveFormat
	}

	// Extract inner content
	inner := strings.TrimSpace(content[3 : len(content)-2])

	// Parse if/elseif pattern
	ifPattern := regexp.MustCompile(`^(if|elseif)\s+(.+)$`)
	matches := ifPattern.FindStringSubmatch(inner)
	if len(matches) != 3 {
		return "", ErrInvalidIfDirectiveFormat
	}

	// Return the condition part
	return strings.TrimSpace(matches[2]), nil
}

// parseForExpression parses for directive expression
// Format: /*# for variable in expression */
func (dee *DirectiveExpressionExtractor) parseForExpression(tokenValue string) (string, error) {
	// Remove comment markers
	content := strings.TrimSpace(tokenValue)
	if !strings.HasPrefix(content, "/*#") || !strings.HasSuffix(content, "*/") {
		return "", ErrInvalidDirectiveFormat
	}

	// Extract inner content
	inner := strings.TrimSpace(content[3 : len(content)-2])

	// Parse for pattern: for variable in expression
	forPattern := regexp.MustCompile(`^for\s+(\w+)\s+in\s+(.+)$`)
	matches := forPattern.FindStringSubmatch(inner)
	if len(matches) != 3 {
		return "", ErrInvalidForDirectiveFormat
	}

	// For 'for' directives, we store the full expression "variable in expression"
	// This allows parserstep6 to extract both variable name and list expression
	return strings.TrimSpace(matches[1]) + " in " + strings.TrimSpace(matches[2]), nil
}

// parseVariableExpression parses variable directive expression
// Format: /*= expression */
func (dee *DirectiveExpressionExtractor) parseVariableExpression(tokenValue string) (string, error) {
	// Remove comment markers
	content := strings.TrimSpace(tokenValue)
	if !strings.HasPrefix(content, "/*=") || !strings.HasSuffix(content, "*/") {
		return "", ErrInvalidDirectiveFormat
	}

	// Extract and return the expression
	expression := strings.TrimSpace(content[3 : len(content)-2])
	return expression, nil
}

// parseConstExpression parses const directive expression
// Format: /*$ expression */
func (dee *DirectiveExpressionExtractor) parseConstExpression(tokenValue string) (string, error) {
	// Remove comment markers
	content := strings.TrimSpace(tokenValue)
	if !strings.HasPrefix(content, "/*$") || !strings.HasSuffix(content, "*/") {
		return "", ErrInvalidDirectiveFormat
	}

	// Extract and return the expression
	expression := strings.TrimSpace(content[3 : len(content)-2])
	return expression, nil
}

// ParseForExpression parses for directive and extracts variable name and list expression
func (dee *DirectiveExpressionExtractor) ParseForExpression(forExpression string) (variable, listExpr string, err error) {
	// Parse "variable in expression" format
	forPattern := regexp.MustCompile(`^(\w+)\s+in\s+(.+)$`)
	matches := forPattern.FindStringSubmatch(forExpression)
	if len(matches) != 3 {
		return "", "", ErrInvalidForDirectiveFormat
	}

	return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2]), nil
}
