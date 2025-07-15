package parserstep5

import (
	"strings"

	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// applyImplicitIfConditions applies implicit if conditions to LIMIT and OFFSET clauses
// when they contain a single variable parameter and don't already have an explicit if condition.
func applyImplicitIfConditions(statement parsercommon.StatementNode) {
	if statement == nil {
		return
	}

	for _, clause := range statement.Clauses() {
		switch typedClause := clause.(type) {
		case *parsercommon.LimitClause:
			if typedClause.IfCondition() == "" {
				if condition := extractImplicitCondition(typedClause.RawTokens()); condition != "" {
					typedClause.SetIfCondition(condition)
				}
			}
		case *parsercommon.OffsetClause:
			if typedClause.IfCondition() == "" {
				if condition := extractImplicitCondition(typedClause.RawTokens()); condition != "" {
					typedClause.SetIfCondition(condition)
				}
			}
		}
	}
}

// extractImplicitCondition extracts implicit condition from tokens containing a single variable directive
func extractImplicitCondition(tokens []tokenizer.Token) string {
	var variableDirectives []string

	for _, token := range tokens {
		if token.Directive != nil && token.Directive.Type == "variable" {
			// Extract variable name from directive content like "/*= limit */"
			content := strings.TrimSpace(token.Value)
			if strings.HasPrefix(content, "/*=") && strings.HasSuffix(content, "*/") {
				varContent := strings.TrimSpace(content[3 : len(content)-2])
				if varContent != "" {
					variableDirectives = append(variableDirectives, varContent)
				}
			}
		}
	}

	// Only apply implicit condition if there's exactly one variable directive
	if len(variableDirectives) == 1 {
		return variableDirectives[0] + " != null"
	}

	return ""
}
