package markdownparser

import (
	"errors"
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
	ErrInvalidFrontMatter                       = errors.New("invalid front matter")
	ErrMissingRequiredSection                   = errors.New("missing required section")
	ErrInvalidTestCase                          = errors.New("invalid test case")
	ErrDuplicateParameters                      = errors.New("duplicate parameters section")
	ErrDuplicateExpectedResults                 = errors.New("duplicate expected results section")
	ErrInvalidExpectedResultsExternalLinkFormat = errors.New("invalid expected results external file link format")
	ErrInvalidFixturesExternalLinkFormat        = errors.New("invalid fixtures external file link format")
)

// ParseOptions contains options for parsing markdown documents
type ParseOptions struct {
	// DatabaseOverride allows overriding the database connection info
	DatabaseOverride *DatabaseConfig
}

// DatabaseConfig represents database connection configuration
type DatabaseConfig struct {
	Driver     string
	Connection string
	Dialect    string
}

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
	return ParseWithOptions(reader, nil)
}

// ParseWithOptions parses a markdown query file with custom options

func ParseWithOptions(reader io.Reader, options *ParseOptions) (*SnapSQLDocument, error) {
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

	// Apply database override if provided (dialect hint only)
	if options != nil && options.DatabaseOverride != nil {
		if frontMatter == nil {
			frontMatter = make(map[string]any)
		}

		if options.DatabaseOverride.Dialect != "" {
			frontMatter["dialect"] = options.DatabaseOverride.Dialect
		}
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

	contentBytes := []byte(contentWithoutFrontMatter)
	doc := md.Parser().Parse(text.NewReader(contentBytes))
	lineMapper := newIndexToLine(contentBytes)

	// Extract title and sections
	title, sections := extractSectionsFromAST(doc, contentBytes)

	// Validate required sections
	if err := validateRequiredSections(sections); err != nil {
		return nil, err
	}

	// Build SnapSQL document
	document := &SnapSQLDocument{
		Metadata: frontMatter,
	}

	// Set title if available (do not derive function_name from title)
	if title != "" {
		document.Metadata["title"] = title
	}

	// Extract SQL
	if sqlSection, exists := sections["sql"]; exists {
		sql, startIdx := extractSQLFromASTNodes(sqlSection.Content, contentBytes)
		document.SQL = sql
		document.SQLStartLine = lineMapper.lineFor(startIdx)
	}

	// Extract description if present and not already in metadata
	if document.Metadata["description"] == nil {
		if descSection, exists := sections["description"]; exists {
			descText, err := extractTextFromASTNodes(descSection.Content, contentBytes)
			if err == nil && strings.TrimSpace(descText) != "" {
				document.Metadata["description"] = strings.TrimSpace(descText)
			}
		}
	}

	// Extract parameters if present
	parameterSectionNames := []string{"parameters", "params", "parameter"}
	for _, sectionName := range parameterSectionNames {
		if paramSection, exists := sections[sectionName]; exists {
			paramText, paramType, err := extractParameterTextFromASTNodes(paramSection.Content, contentBytes)
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
		testCases, err := parseTestCasesFromAST(testSection.Content, contentBytes, lineMapper)
		if err != nil {
			return nil, fmt.Errorf("failed to parse test cases: %w", err)
		}

		document.TestCases = testCases
	}

	return document, nil
}
