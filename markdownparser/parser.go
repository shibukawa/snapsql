package markdownparser

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var (
	ErrInvalidFrontMatter     = errors.New("invalid front matter")
	ErrMissingRequiredSection = errors.New("missing required section")
	ErrInvalidYAML            = errors.New("invalid YAML content")
)

// SnapSQLDocument represents the parsed SnapSQL markdown document
type SnapSQLDocument struct {
	Metadata       map[string]any
	ParameterBlock string
	SQL            string
	TestCases      []TestCase
	MockData       map[string]map[string]any
}

// Section represents a markdown section
type Section struct {
	Heading string
	Content string
}

// TestCase represents a single test case
type TestCase struct {
	Name           string
	Fixture        map[string]any
	Parameters     map[string]any
	ExpectedResult any
}

// Parse parses a markdown query file and returns a SnapSQLDocument
func Parse(reader io.Reader) (*SnapSQLDocument, error) {
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Create parser
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.Meta,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	// Create parser context
	context := parser.NewContext()

	// Parse markdown document
	doc := md.Parser().Parse(text.NewReader(content), parser.WithContext(context))

	// Extract metadata (front matter)
	metaData := meta.Get(context)
	frontMatter := fixFrontmatter(metaData)

	// Extract title and sections
	title, sections := extractSectionsFromAST(doc, content)

	// Validate required sections (only description and sql)
	if err := validateRequiredSections(sections); err != nil {
		return nil, err
	}

	// Build SnapSQL document
	document := &SnapSQLDocument{
		Metadata: frontMatter,
	}

	// Extract parameter block if present
	if paramSection, exists := sections["parameters"]; exists {
		document.ParameterBlock = extractYAMLFromCodeBlock(paramSection.Content)
	}

	// Extract SQL
	if sqlSection, exists := sections["sql"]; exists {
		document.SQL = extractSQLFromCodeBlock(sqlSection.Content)
	}

	// Generate function_name if not present in metadata
	if document.Metadata["function_name"] == nil && title != "" {
		document.Metadata["function_name"] = generateFunctionNameFromTitle(title)
	}

	// Extract test cases if present
	if testSection, exists := sections["test"]; exists {
		testCases, err := parseTestCases(testSection.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse test cases: %w", err)
		}
		document.TestCases = testCases
	}

	// Extract mock data if present
	if mockSection, exists := sections["mock"]; exists {
		mockData, err := parseMockData(mockSection.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mock data: %w", err)
		}
		document.MockData = mockData
	}

	return document, nil
}

// fixFrontmatter fixes front matter data
func fixFrontmatter(metaData map[string]any) map[string]any {
	if metaData == nil {
		return make(map[string]any)
	}
	return metaData
}

// generateFunctionNameFromTitle generates a function name from title
func generateFunctionNameFromTitle(title string) string {
	// Convert to camelCase
	words := strings.Fields(title)
	if len(words) == 0 {
		return "query"
	}

	var result strings.Builder
	for i, word := range words {
		// Remove non-alphanumeric characters
		cleanWord := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(word, "")
		if cleanWord == "" {
			continue
		}

		if i == 0 {
			// First word is lowercase
			result.WriteString(strings.ToLower(cleanWord))
		} else {
			// Subsequent words are title case
			result.WriteString(strings.Title(cleanWord))
		}
	}

	functionName := result.String()
	if functionName == "" {
		return "query"
	}
	return functionName
}

// extractSectionsFromAST extracts title and sections from the AST
func extractSectionsFromAST(_ ast.Node, source []byte) (string, map[string]Section) {
	sections := make(map[string]Section)
	var title string

	// Convert source to string for easier processing
	sourceStr := string(source)
	lines := strings.Split(sourceStr, "\n")

	var currentSection string
	var currentContent []string
	inSection := false
	inFrontMatter := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Handle front matter
		if trimmedLine == "---" {
			if !inFrontMatter {
				inFrontMatter = true
				continue
			} else {
				inFrontMatter = false
				continue
			}
		}

		// Skip front matter content
		if inFrontMatter {
			continue
		}

		// Check for level 2 headings only (##) for sections
		if strings.HasPrefix(trimmedLine, "##") && !strings.HasPrefix(trimmedLine, "###") {
			// Extract heading level and text
			headingMatch := regexp.MustCompile(`^(#+)\s+(.*)$`).FindStringSubmatch(trimmedLine)
			if len(headingMatch) == 3 {
				headingText := headingMatch[2]

				// Check if this heading matches a known section keyword
				keyword := extractKeywordFromHeading(headingText)
				if keyword != "" && isKnownSection(keyword) {
					// Save previous section
					if currentSection != "" && inSection {
						sections[currentSection] = Section{
							Heading: currentSection,
							Content: strings.Join(currentContent, "\n"),
						}
					}

					// Start new section
					currentSection = strings.ToLower(keyword)
					currentContent = []string{}
					inSection = true
					continue
				}

				// Check for custom sections (unknown keywords)
				customKeyword := extractFirstWord(headingText)
				if customKeyword != "" && !isKnownSection(customKeyword) {
					// Save previous section
					if currentSection != "" && inSection {
						sections[currentSection] = Section{
							Heading: currentSection,
							Content: strings.Join(currentContent, "\n"),
						}
					}

					// Start new custom section
					currentSection = strings.ToLower(customKeyword)
					currentContent = []string{}
					inSection = true
					continue
				}

				// If it's not a recognized section, treat as content
				if inSection && currentSection != "" {
					currentContent = append(currentContent, line)
				}
				continue
			}
		}

		// Check for level 1 headings (#) for title
		if strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "##") {
			// Extract heading level and text
			headingMatch := regexp.MustCompile(`^(#+)\s+(.*)$`).FindStringSubmatch(trimmedLine)
			if len(headingMatch) == 3 {
				headingText := headingMatch[2]

				// First heading becomes title (regardless of level)
				if title == "" {
					title = headingText
					continue
				}

				// If it's not the first heading, treat as content
				if inSection && currentSection != "" {
					currentContent = append(currentContent, line)
				}
				continue
			}
		}

		// Add content to current section
		if inSection && currentSection != "" {
			currentContent = append(currentContent, line)
		}
	}

	// Save last section
	if currentSection != "" && inSection {
		sections[currentSection] = Section{
			Heading: currentSection,
			Content: strings.Join(currentContent, "\n"),
		}
	}

	return title, sections
}

// isKnownSection checks if a keyword is a known section type
func isKnownSection(keyword string) bool {
	knownSections := map[string]bool{
		"overview":    true,
		"description": true, // Accept both overview and description
		"parameters":  true,
		"sql":         true,
		"test":        true,
		"tests":       true, // Also accept "tests"
		"mock":        true,
		"change":      true,
	}
	return knownSections[strings.ToLower(keyword)]
}

// extractFirstWord extracts the first word from text (for custom sections)
func extractFirstWord(text string) string {
	words := strings.Fields(text)
	if len(words) > 0 {
		// Remove any non-alphanumeric characters from the first word
		word := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(words[0], "")
		if len(word) > 0 {
			return word
		}
	}
	return ""
}

// extractKeywordFromHeading extracts the first word from a heading
func extractKeywordFromHeading(heading string) string {
	// Handle special cases first
	lowerHeading := strings.ToLower(heading)
	if strings.HasPrefix(lowerHeading, "test case") || strings.HasPrefix(lowerHeading, "test") {
		return "Test"
	}
	if strings.HasPrefix(lowerHeading, "mock data") || strings.HasPrefix(lowerHeading, "mock") {
		return "Mock"
	}
	if strings.HasPrefix(lowerHeading, "change log") || strings.HasPrefix(lowerHeading, "changelog") {
		return "Change"
	}

	// Use regex to extract the first alphabetic word
	re := regexp.MustCompile(`^([A-Za-z]\w*)`)
	matches := re.FindStringSubmatch(heading)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// validateRequiredSections checks if all required sections are present
func validateRequiredSections(sections map[string]Section) error {
	// Only description/overview and sql are required
	descriptionFound := false
	sqlFound := false

	for sectionName := range sections {
		switch sectionName {
		case "overview", "description":
			descriptionFound = true
		case "sql":
			sqlFound = true
		}
	}

	if !descriptionFound {
		return fmt.Errorf("%w: description or overview", ErrMissingRequiredSection)
	}
	if !sqlFound {
		return fmt.Errorf("%w: sql", ErrMissingRequiredSection)
	}

	return nil
}

// extractYAMLFromCodeBlock extracts YAML content from markdown code block
func extractYAMLFromCodeBlock(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var yamlLines []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				inCodeBlock = true
			} else {
				break
			}
			continue
		}

		if inCodeBlock {
			yamlLines = append(yamlLines, line)
		}
	}

	return strings.Join(yamlLines, "\n")
}

// extractSQLFromCodeBlock extracts SQL content from markdown code block
func extractSQLFromCodeBlock(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var sqlLines []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				inCodeBlock = true
			} else {
				break
			}
			continue
		}

		if inCodeBlock {
			sqlLines = append(sqlLines, line)
		}
	}

	return strings.Join(sqlLines, "\n")
}

// parseTestCases parses the test cases section
func parseTestCases(content string) ([]TestCase, error) {
	var testCases []TestCase

	// Split content by case headers
	casePattern := regexp.MustCompile(`(?m)^### Case \d+: (.+)$`)
	cases := casePattern.Split(content, -1)
	caseNames := casePattern.FindAllStringSubmatch(content, -1)

	for i := 1; i < len(cases); i++ {
		caseName := ""
		if i-1 < len(caseNames) {
			caseName = caseNames[i-1][1]
		}

		testCase, err := parseTestCase(caseName, cases[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse test case %s: %w", caseName, err)
		}

		testCases = append(testCases, *testCase)
	}

	return testCases, nil
}

// parseTestCase parses a single test case
func parseTestCase(name, content string) (*TestCase, error) {
	testCase := &TestCase{Name: name}

	// Extract Fixture
	if fixture := extractSectionContent(content, "Fixture"); fixture != "" {
		var fixtureData map[string]any
		if err := yaml.Unmarshal([]byte(fixture), &fixtureData); err != nil {
			return nil, fmt.Errorf("invalid fixture YAML: %w", err)
		}
		testCase.Fixture = fixtureData
	}

	// Extract Parameters
	if params := extractSectionContent(content, "Parameters"); params != "" {
		var paramData map[string]any
		if err := yaml.Unmarshal([]byte(params), &paramData); err != nil {
			return nil, fmt.Errorf("invalid parameters YAML: %w", err)
		}
		testCase.Parameters = paramData
	}

	// Extract Expected Result
	if result := extractSectionContent(content, "Expected Result"); result != "" {
		var resultData any
		if err := yaml.Unmarshal([]byte(result), &resultData); err != nil {
			return nil, fmt.Errorf("invalid expected result YAML: %w", err)
		}
		testCase.ExpectedResult = resultData
	}

	return testCase, nil
}

// extractSectionContent extracts content from a subsection like **Fixture:**
func extractSectionContent(content, sectionName string) string {
	pattern := regexp.MustCompile(`(?s)\*\*` + sectionName + `[:\s]*\*\*\s*\n` + "```[^`\n]*\n(.*?)\n```")
	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseMockData parses the mock data section
func parseMockData(content string) (map[string]map[string]any, error) {
	mockData := make(map[string]map[string]any)

	// Extract YAML content from markdown code block
	yamlContent := extractYAMLFromCodeBlock(content)

	if yamlContent == "" {
		return mockData, nil
	}

	var data map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &data); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidYAML, err)
	}

	// Convert to the expected format
	for key, value := range data {
		if tableData, ok := value.(map[string]any); ok {
			mockData[key] = tableData
		}
	}

	return mockData, nil
}
