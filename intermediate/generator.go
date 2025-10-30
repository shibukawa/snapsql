package intermediate

import (
	"fmt"
	"io"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// GenerateFromSQL generates the intermediate format for a SQL template
func GenerateFromSQL(reader io.Reader, constants map[string]any, basePath string, projectRootPath string, tableInfo map[string]*snapsql.TableInfo, config *snapsql.Config) (*IntermediateFormat, error) {
	// Parse the SQL
	stmt, typeInfoMap, funcDef, err := parser.ParseSQLFile(reader, constants, basePath, projectRootPath, parser.DefaultOptions)
	if err != nil {
		return nil, err
	}

	// Generate intermediate format
	return generateIntermediateFormat(stmt, typeInfoMap, funcDef, basePath, tableInfo, config)
}

// GenerateFromMarkdown generates the intermediate format for a Markdown file containing SQL
func GenerateFromMarkdown(doc *markdownparser.SnapSQLDocument, basePath string, projectRootPath string, constants map[string]any, tableInfo map[string]*snapsql.TableInfo, config *snapsql.Config) (*IntermediateFormat, error) {
	// Parse the Markdown
	stmt, typeInfoMap, funcDef, err := parser.ParseMarkdownFile(doc, basePath, projectRootPath, constants, parser.DefaultOptions)
	if err != nil {
		return nil, err
	}
	// Generate intermediate format
	format, err := generateIntermediateFormat(stmt, typeInfoMap, funcDef, basePath, tableInfo, config)
	if err != nil {
		return nil, err
	}

	if len(doc.TestCases) > 0 {
		format.MockTestCases = markdownparser.ExtractMockTestCases(doc)
	}

	return format, nil
}

// generateIntermediateFormat is the common implementation using the new pipeline approach
func generateIntermediateFormat(stmt parsercommon.StatementNode, typeInfoMap map[string]any, funcDef *parsercommon.FunctionDefinition, filePath string, tableInfo map[string]*snapsql.TableInfo, config *snapsql.Config) (*IntermediateFormat, error) {
	_ = filePath // File path not currently used in pipeline processing
	// Create and execute the token processing pipeline
	pipeline := CreateDefaultPipeline(stmt, funcDef, config, tableInfo, typeInfoMap)

	result, err := pipeline.Execute()
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	result.HasOrderedResult = statementHasOrderBy(stmt)

	return result, nil
}

func statementHasOrderBy(stmt parsercommon.StatementNode) bool {
	switch s := stmt.(type) {
	case *parsercommon.SelectStatement:
		return s.OrderBy != nil && len(s.OrderBy.Fields) > 0
	case *parsercommon.InsertIntoStatement:
		return s.OrderBy != nil && len(s.OrderBy.Fields) > 0
	default:
		return false
	}
}

// extractParameterType extracts the type from a parameter value
func extractParameterType(paramValue any) string {
	// The parameter value could be a string (simple type) or a map (complex type definition)
	switch v := paramValue.(type) {
	case string:
		// Simple type like "int", "string", "bool", etc.
		return v
	case map[string]any:
		// Complex type definition with "type" field
		if typeVal, ok := v["type"]; ok {
			if typeStr, ok := typeVal.(string); ok {
				return typeStr
			}
		}
		// Fallback to "any" if type field is not found or not a string
		return "any"
	default:
		// Unknown type, fallback to "any"
		return "any"
	}
}

// extractTokensFromStatement extracts all tokens from a statement
func extractTokensFromStatement(stmt parsercommon.StatementNode) []tokenizer.Token {
	tokens := []tokenizer.Token{}

	// WITH clause must appear before the main statement.
	// Ensure CTE tokens come first if present.
	if cte := stmt.CTE(); cte != nil {
		cteTokens := cte.RawTokens()
		tokens = append(tokens, cteTokens...)

		if len(cteTokens) > 0 {
			last := cteTokens[len(cteTokens)-1]
			switch last.Type {
			case tokenizer.WHITESPACE, tokenizer.LINE_COMMENT, tokenizer.BLOCK_COMMENT:
				// trailing whitespace already present; no-op
			default:
				tokens = append(tokens, tokenizer.Token{Type: tokenizer.WHITESPACE, Value: " "})
			}
		}
	}

	// Then process tokens from each clause
	for _, clause := range stmt.Clauses() {
		if clause.Type() == parsercommon.WITH_CLAUSE {
			continue
		}

		tokens = append(tokens, clause.RawTokens()...)
	}

	return tokens
}

// extractParameterTypeFromOriginal extracts type string from original parameter value (for common types)
func extractParameterTypeFromOriginal(value any) string {
	switch v := value.(type) {
	case string:
		return v // Common type names like "User", "Department[]", "api/users/User"
	default:
		// Fallback to regular extraction
		return extractParameterType(value)
	}
}
