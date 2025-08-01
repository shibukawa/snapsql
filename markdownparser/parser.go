package markdownparser

import (
	"fmt"
	"io"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Sentinel errors
var (
	ErrInvalidFrontMatter       = fmt.Errorf("invalid front matter")
	ErrMissingRequiredSection   = fmt.Errorf("missing required section")
	ErrInvalidTestCase          = fmt.Errorf("invalid test case")
	ErrDuplicateParameters      = fmt.Errorf("duplicate parameters section")
	ErrDuplicateExpectedResults = fmt.Errorf("duplicate expected results section")
)

// SnapSQLDocument represents the parsed SnapSQL markdown document
type SnapSQLDocument struct {
	Metadata       map[string]any
	ParametersText string // Raw parameter text (YAML/JSON)
	ParametersType string // "yaml", "json", etc.
	SQL            string
	SQLStartLine   int // Line number where SQL code block starts
	TestCases      []TestCase
}

// Parse parses a markdown query file and returns a SnapSQLDocument
func Parse(reader io.Reader) (*SnapSQLDocument, error) {
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Parse front matter manually
	frontMatter, contentWithoutFrontMatter, err := parseFrontMatter(string(content))
	if err != nil {
		return nil, err
	}

	// Create parser
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	// Parse markdown document
	doc := md.Parser().Parse(text.NewReader([]byte(contentWithoutFrontMatter)))

	// Extract title and sections
	title, sections := extractSectionsFromAST(doc, []byte(contentWithoutFrontMatter))

	// Validate required sections
	if err := validateRequiredSections(sections); err != nil {
		return nil, err
	}

	// Build SnapSQL document
	document := &SnapSQLDocument{
		Metadata: frontMatter,
	}

	// Set title if available
	if title != "" {
		document.Metadata["title"] = title
		// Generate function_name if not present in metadata
		if document.Metadata["function_name"] == nil {
			document.Metadata["function_name"] = generateFunctionNameFromTitle(title)
		}
	}

	// Extract SQL
	if sqlSection, exists := sections["sql"]; exists {
		sql, startLine := extractSQLFromASTNodes(sqlSection.Content, []byte(contentWithoutFrontMatter))
		document.SQL = sql
		document.SQLStartLine = startLine
	}

	// Extract description if present and not already in metadata
	if document.Metadata["description"] == nil {
		if descSection, exists := sections["description"]; exists {
			descText, err := extractTextFromASTNodes(descSection.Content, []byte(contentWithoutFrontMatter))
			if err == nil && strings.TrimSpace(descText) != "" {
				document.Metadata["description"] = strings.TrimSpace(descText)
			}
		}
	}

	// Extract parameters if present
	parameterSectionNames := []string{"parameters", "params", "parameter"}
	for _, sectionName := range parameterSectionNames {
		if paramSection, exists := sections[sectionName]; exists {
			paramText, paramType, err := extractParameterTextFromASTNodes(paramSection.Content, []byte(contentWithoutFrontMatter))
			if err != nil {
				return nil, fmt.Errorf("failed to extract parameters: %w", err)
			}
			document.ParametersText = paramText
			document.ParametersType = paramType
			break
		}
	}

	// Parse test cases
	if testSection, exists := sections["test cases"]; exists {
		testCases, err := parseTestCasesFromAST(testSection.Content, []byte(contentWithoutFrontMatter))
		if err != nil {
			return nil, fmt.Errorf("failed to parse test cases: %w", err)
		}
		document.TestCases = testCases
	}

	return document, nil
}
