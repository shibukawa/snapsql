package markdownparser

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-meta"
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

// FrontMatter represents the YAML front matter of a markdown file
type FrontMatter struct {
	Name    string `yaml:"name"`
	Dialect string `yaml:"dialect"`
}

// Section represents a markdown section
type Section struct {
	Heading string
	Content string
}

// ParsedMarkdown represents a parsed markdown query file
type ParsedMarkdown struct {
	FrontMatter FrontMatter
	Title       string
	Sections    map[string]Section
}

// Parameters represents the parsed parameters section
type Parameters map[string]interface{}

// TestCase represents a single test case
type TestCase struct {
	Name           string
	Fixture        map[string]interface{}
	Parameters     map[string]interface{}
	ExpectedResult interface{}
}

// Parser handles markdown parsing using goldmark
type Parser struct {
	markdown goldmark.Markdown
}

// NewParser creates a new markdown parser with goldmark
func NewParser() *Parser {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.Meta,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	return &Parser{
		markdown: md,
	}
}

// Parse parses a markdown query file using goldmark
func (p *Parser) Parse(reader io.Reader) (*ParsedMarkdown, error) {
	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Create parser context
	context := parser.NewContext()

	// Parse markdown document
	doc := p.markdown.Parser().Parse(text.NewReader(content), parser.WithContext(context))

	// Extract metadata (front matter)
	metaData := meta.Get(context)
	frontMatter, err := p.extractFrontMatter(metaData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract front matter: %w", err)
	}

	// Extract title and sections
	title, sections := p.extractSectionsFromAST(doc, content)

	// Validate required sections
	if err := p.validateRequiredSections(sections); err != nil {
		return nil, err
	}

	return &ParsedMarkdown{
		FrontMatter: *frontMatter,
		Title:       title,
		Sections:    sections,
	}, nil
}

// extractFrontMatter extracts front matter from metadata
func (p *Parser) extractFrontMatter(metaData map[string]interface{}) (*FrontMatter, error) {
	if metaData == nil {
		return nil, ErrInvalidFrontMatter
	}

	name, nameExists := metaData["name"]
	dialect, dialectExists := metaData["dialect"]

	if !nameExists || !dialectExists {
		return nil, fmt.Errorf("%w: name and dialect are required", ErrInvalidFrontMatter)
	}

	nameStr, ok := name.(string)
	if !ok {
		return nil, fmt.Errorf("%w: name must be a string", ErrInvalidFrontMatter)
	}

	dialectStr, ok := dialect.(string)
	if !ok {
		return nil, fmt.Errorf("%w: dialect must be a string", ErrInvalidFrontMatter)
	}

	return &FrontMatter{
		Name:    nameStr,
		Dialect: dialectStr,
	}, nil
}

// extractSectionsFromAST extracts title and sections from the AST
func (p *Parser) extractSectionsFromAST(_ ast.Node, source []byte) (string, map[string]Section) {
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
				keyword := p.extractKeywordFromHeading(headingText)
				if keyword != "" && p.isKnownSection(keyword) {
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
				customKeyword := p.extractFirstWord(headingText)
				if customKeyword != "" && !p.isKnownSection(customKeyword) {
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
func (p *Parser) isKnownSection(keyword string) bool {
	knownSections := map[string]bool{
		"overview":    true,
		"parameters":  true,
		"sql":         true,
		"test":        true,
		"tests":       true, // Also accept "tests"
		"mock":        true,
		"performance": true,
		"security":    true,
		"change":      true,
	}
	return knownSections[strings.ToLower(keyword)]
}

// extractFirstWord extracts the first word from text (for custom sections)
func (p *Parser) extractFirstWord(text string) string {
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
func (p *Parser) extractKeywordFromHeading(heading string) string {
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
func (p *Parser) validateRequiredSections(sections map[string]Section) error {
	requiredSections := []string{"overview", "parameters", "sql", "test"}

	for _, required := range requiredSections {
		if _, exists := sections[required]; !exists {
			return fmt.Errorf("%w: %s", ErrMissingRequiredSection, required)
		}
	}

	return nil
}

// ParseParameters parses the parameters section as YAML
func (p *Parser) ParseParameters(content string) (Parameters, error) {
	// Extract YAML content from markdown code block
	yamlContent := p.extractYAMLFromCodeBlock(content)

	var params Parameters
	if err := yaml.Unmarshal([]byte(yamlContent), &params); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidYAML, err)
	}

	return params, nil
}

// ParseTestCases parses the test cases section
func (p *Parser) ParseTestCases(content string) ([]TestCase, error) {
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

		testCase, err := p.parseTestCase(caseName, cases[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse test case %s: %w", caseName, err)
		}

		testCases = append(testCases, *testCase)
	}

	return testCases, nil
}

// parseTestCase parses a single test case
func (p *Parser) parseTestCase(name, content string) (*TestCase, error) {
	testCase := &TestCase{Name: name}

	// Extract Fixture
	if fixture := p.extractSectionContent(content, "Fixture"); fixture != "" {
		var fixtureData map[string]interface{}
		if err := yaml.Unmarshal([]byte(fixture), &fixtureData); err != nil {
			return nil, fmt.Errorf("invalid fixture YAML: %w", err)
		}
		testCase.Fixture = fixtureData
	}

	// Extract Parameters
	if params := p.extractSectionContent(content, "Parameters"); params != "" {
		var paramData map[string]interface{}
		if err := yaml.Unmarshal([]byte(params), &paramData); err != nil {
			return nil, fmt.Errorf("invalid parameters YAML: %w", err)
		}
		testCase.Parameters = paramData
	}

	// Extract Expected Result
	if result := p.extractSectionContent(content, "Expected Result"); result != "" {
		var resultData interface{}
		if err := yaml.Unmarshal([]byte(result), &resultData); err != nil {
			return nil, fmt.Errorf("invalid expected result YAML: %w", err)
		}
		testCase.ExpectedResult = resultData
	}

	return testCase, nil
}

// ExtractYAMLFromCodeBlock extracts YAML content from markdown code block (public method)
func (p *Parser) ExtractYAMLFromCodeBlock(content string) string {
	return p.extractYAMLFromCodeBlock(content)
}

// ExtractSQLFromCodeBlock extracts SQL content from markdown code block
func (p *Parser) ExtractSQLFromCodeBlock(content string) string {
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
func (p *Parser) extractYAMLFromCodeBlock(content string) string {
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

// extractSectionContent extracts content from a subsection like **Fixture:**
func (p *Parser) extractSectionContent(content, sectionName string) string {
	pattern := regexp.MustCompile(`(?s)\*\*` + sectionName + `[:\s]*\*\*\s*\n` + "```[^`\n]*\n(.*?)\n```")
	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
