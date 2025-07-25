package markdownparser

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Sentinel errors
var (
	ErrInvalidFrontMatter     = fmt.Errorf("invalid front matter")
	ErrMissingRequiredSection = fmt.Errorf("missing required section")
	ErrInvalidTestCase        = fmt.Errorf("invalid test case")
	ErrInvalidMockData        = fmt.Errorf("invalid mock data")
)

// SnapSQLDocument represents the parsed SnapSQL markdown document
type SnapSQLDocument struct {
	Metadata       map[string]any
	ParameterBlock string
	SQL            string
	SQLStartLine   int // Line number where SQL code block starts
	TestCases      []TestCase
	MockData       map[string]map[string]any
}

// Section represents a markdown section with AST nodes
type Section struct {
	Heading     ast.Node   // The heading node
	HeadingText string     // Extracted heading text
	StartLine   int        // Line number where section starts
	Content     []ast.Node // All nodes between this heading and the next
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

	// Extract title and sections using AST
	title, sections := extractSectionsFromAST(doc, []byte(contentWithoutFrontMatter))

	// Validate required sections (only description and sql)
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
	}

	// Extract SQL from AST nodes
	if sqlSection, exists := sections["sql"]; exists {
		sql, startLine := extractSQLFromASTNodes(sqlSection.Content, []byte(contentWithoutFrontMatter))
		document.SQL = sql
		// Add front matter offset to get correct line number in original document
		frontMatterLines := countFrontMatterLines(string(content))
		document.SQLStartLine = startLine + frontMatterLines
	}

	// Extract parameters if present
	parameterSectionNames := []string{"parameters", "params", "parameter"}
	for _, sectionName := range parameterSectionNames {
		if paramSection, exists := sections[sectionName]; exists {
			parameterBlock := extractParameterBlock(paramSection.Content, []byte(contentWithoutFrontMatter))
			document.ParameterBlock = parameterBlock
			break
		}
	}

	// Generate function_name if not present in metadata
	if document.Metadata["function_name"] == nil && title != "" {
		document.Metadata["function_name"] = generateFunctionNameFromTitle(title)
	}

	// Extract test cases if present
	testSectionNames := []string{"test", "tests", "test cases", "testcases"}
	for _, sectionName := range testSectionNames {
		if testSection, exists := sections[sectionName]; exists {
			testCases, err := parseTestCasesFromAST(testSection.Content, []byte(contentWithoutFrontMatter))
			if err != nil {
				return nil, fmt.Errorf("failed to parse test cases: %w", err)
			}
			document.TestCases = testCases
			break
		}
	}

	// Extract mock data if present
	mockSectionNames := []string{"mock", "mocks", "mock data", "mockdata", "test data", "testdata"}
	for _, sectionName := range mockSectionNames {
		if mockSection, exists := sections[sectionName]; exists {
			mockData, err := parseMockDataFromAST(mockSection.Content, []byte(contentWithoutFrontMatter))
			if err != nil {
				return nil, fmt.Errorf("failed to parse mock data: %w", err)
			}
			document.MockData = mockData
			break
		}
	}

	return document, nil
}

// extractSectionsFromAST extracts sections from markdown AST
func extractSectionsFromAST(doc ast.Node, content []byte) (string, map[string]Section) {
	sections := make(map[string]Section)
	var title string
	var currentSection *Section
	var currentNodes []ast.Node

	// Walk through all nodes
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			// Save previous section if exists
			if currentSection != nil {
				currentSection.Content = currentNodes
				sections[strings.ToLower(currentSection.HeadingText)] = *currentSection
			}

			// Extract heading text from AST
			headingText := extractTextFromHeadingNode(node, content)

			// Get line number from AST
			startLine := getNodeLineNumber(node)

			if node.Level == 1 && title == "" {
				title = headingText
				currentSection = nil
				currentNodes = nil
			} else {
				currentSection = &Section{
					Heading:     node,
					HeadingText: headingText,
					StartLine:   startLine,
					Content:     nil, // Will be filled when we encounter the next heading or end
				}
				currentNodes = make([]ast.Node, 0)
			}

		default:
			// Add all other nodes to current section content
			if currentSection != nil {
				currentNodes = append(currentNodes, node)
			}
		}

		return ast.WalkContinue, nil
	})

	// Save the last section
	if currentSection != nil {
		currentSection.Content = currentNodes
		sections[strings.ToLower(currentSection.HeadingText)] = *currentSection
	}

	return title, sections
}

// extractTextFromHeadingNode extracts text content from a heading AST node
func extractTextFromHeadingNode(heading ast.Node, content []byte) string {
	var result strings.Builder

	// Walk through heading children to extract text
	ast.Walk(heading, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Text:
			segment := node.Segment
			result.Write(content[segment.Start:segment.Stop])
		case *ast.String:
			result.Write(node.Value)
		}

		return ast.WalkContinue, nil
	})

	return strings.TrimSpace(result.String())
}

// getNodeLineNumber extracts line number from AST node
func getNodeLineNumber(node ast.Node) int {
	if node.Lines() != nil && node.Lines().Len() > 0 {
		return node.Lines().At(0).Start
	}
	return 0
}

// extractSQLFromASTNodes extracts SQL content from AST nodes
func extractSQLFromASTNodes(nodes []ast.Node, content []byte) (string, int) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FencedCodeBlock:
			// Check if this is a SQL code block
			if isSQLCodeBlock(n, content) {
				sql := extractCodeBlockContent(n, content)
				startLine := getNodeLineNumber(n) + 1 // +1 because SQL content starts after the ```sql line
				return sql, startLine
			}
		case *ast.CodeBlock:
			// Handle indented code blocks (less common but possible)
			sql := extractCodeBlockContent(n, content)
			startLine := getNodeLineNumber(n)
			return sql, startLine
		}
	}
	return "", 0
}

// isSQLCodeBlock checks if a fenced code block is marked as SQL
func isSQLCodeBlock(codeBlock *ast.FencedCodeBlock, content []byte) bool {
	if codeBlock.Info != nil {
		segment := codeBlock.Info.Segment
		info := string(content[segment.Start:segment.Stop])
		return strings.TrimSpace(strings.ToLower(info)) == "sql"
	}
	return false
}

// extractCodeBlockContent extracts the actual content from a code block AST node
func extractCodeBlockContent(codeBlock ast.Node, content []byte) string {
	var result strings.Builder

	// Extract content from code block lines
	if codeBlock.Lines() != nil {
		for i := 0; i < codeBlock.Lines().Len(); i++ {
			line := codeBlock.Lines().At(i)
			result.Write(content[line.Start:line.Stop])
		}
	}

	return strings.TrimRight(result.String(), "\n")
}

// parseTestCasesFromAST parses test cases from AST nodes
func parseTestCasesFromAST(nodes []ast.Node, content []byte) ([]TestCase, error) {
	var testCases []TestCase
	var currentTestCase *TestCase

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.Heading:
			// Save previous test case if exists
			if currentTestCase != nil {
				testCases = append(testCases, *currentTestCase)
			}

			// Start new test case
			testName := extractTextFromHeadingNode(n, content)
			currentTestCase = &TestCase{
				Name:           testName,
				Fixture:        make(map[string]any),
				Parameters:     make(map[string]any),
				ExpectedResult: nil,
			}

		case *ast.FencedCodeBlock:
			if currentTestCase != nil {
				// Parse YAML/JSON/CSV content from code block
				codeContent := extractCodeBlockContent(n, content)
				info := getCodeBlockInfo(n, content)

				switch strings.ToLower(strings.TrimSpace(info)) {
				case "yaml", "yml":
					if err := parseYAMLIntoTestCase(codeContent, currentTestCase); err != nil {
						return nil, fmt.Errorf("failed to parse YAML in test case '%s': %w", currentTestCase.Name, err)
					}
				case "json":
					if err := parseJSONIntoTestCase(codeContent, currentTestCase); err != nil {
						return nil, fmt.Errorf("failed to parse JSON in test case '%s': %w", currentTestCase.Name, err)
					}
				case "csv":
					if err := parseCSVIntoTestCase(codeContent, currentTestCase); err != nil {
						return nil, fmt.Errorf("failed to parse CSV in test case '%s': %w", currentTestCase.Name, err)
					}
				case "xml":
					if err := parseXMLIntoTestCase(codeContent, currentTestCase); err != nil {
						return nil, fmt.Errorf("failed to parse XML in test case '%s': %w", currentTestCase.Name, err)
					}
				}
			}

		case *ast.List:
			if currentTestCase != nil {
				// Parse list items for test case data
				if err := parseListIntoTestCase(n, content, currentTestCase); err != nil {
					return nil, fmt.Errorf("failed to parse list in test case '%s': %w", currentTestCase.Name, err)
				}
			}

		case *ast.Paragraph:
			if currentTestCase != nil {
				// Parse paragraph text for simple key-value pairs
				paragraphText := extractTextFromNode(n, content)
				if err := parseParagraphIntoTestCase(paragraphText, currentTestCase); err != nil {
					return nil, fmt.Errorf("failed to parse paragraph in test case '%s': %w", currentTestCase.Name, err)
				}
			}
		}
	}

	// Save the last test case
	if currentTestCase != nil {
		testCases = append(testCases, *currentTestCase)
	}

	return testCases, nil
}

// parseMockDataFromAST parses mock data from AST nodes
func parseMockDataFromAST(nodes []ast.Node, content []byte) (map[string]map[string]any, error) {
	mockData := make(map[string]map[string]any)

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FencedCodeBlock:
			// Parse YAML/JSON/CSV content from code block
			codeContent := extractCodeBlockContent(n, content)
			info := getCodeBlockInfo(n, content)

			switch strings.ToLower(strings.TrimSpace(info)) {
			case "yaml", "yml":
				var yamlData map[string]any
				if err := yaml.Unmarshal([]byte(codeContent), &yamlData); err != nil {
					return nil, fmt.Errorf("failed to parse YAML mock data: %w", err)
				}

				// Convert to the expected format
				for tableName, tableData := range yamlData {
					if tableRows, ok := tableData.([]any); ok {
						mockData[tableName] = make(map[string]any)
						for i, row := range tableRows {
							if rowMap, ok := row.(map[string]any); ok {
								mockData[tableName][fmt.Sprintf("row_%d", i)] = rowMap
							}
						}
					} else if tableMap, ok := tableData.(map[string]any); ok {
						mockData[tableName] = tableMap
					}
				}

			case "json":
				var jsonData map[string]any
				if err := yaml.Unmarshal([]byte(codeContent), &jsonData); err != nil {
					return nil, fmt.Errorf("failed to parse JSON mock data: %w", err)
				}

				// Convert to the expected format (similar to YAML)
				for tableName, tableData := range jsonData {
					if tableRows, ok := tableData.([]any); ok {
						mockData[tableName] = make(map[string]any)
						for i, row := range tableRows {
							if rowMap, ok := row.(map[string]any); ok {
								mockData[tableName][fmt.Sprintf("row_%d", i)] = rowMap
							}
						}
					} else if tableMap, ok := tableData.(map[string]any); ok {
						mockData[tableName] = tableMap
					}
				}

			case "csv":
				// Parse CSV content
				csvData, err := parseCSVToMockData(codeContent)
				if err != nil {
					return nil, fmt.Errorf("failed to parse CSV mock data: %w", err)
				}

				// Merge CSV data into mockData
				for tableName, tableData := range csvData {
					if tableMap, ok := tableData.(map[string]any); ok {
						mockData[tableName] = tableMap
					}
				}

			case "xml":
				// Parse XML content (DBUnit format)
				xmlData, err := parseXMLToMockData(codeContent)
				if err != nil {
					return nil, fmt.Errorf("failed to parse XML mock data: %w", err)
				}

				// Merge XML data into mockData
				for tableName, tableData := range xmlData {
					mockData[tableName] = tableData
				}
			}

		case *extast.Table:
			// Parse markdown table as mock data
			tableName := "markdown_table"
			tableData, err := parseTableToMockData(n, content)
			if err != nil {
				return nil, fmt.Errorf("failed to parse table mock data: %w", err)
			}
			mockData[tableName] = tableData
		}
	}

	return mockData, nil
}

// parseCSVIntoTestCase parses CSV content into a test case
func parseCSVIntoTestCase(csvContent string, testCase *TestCase) error {
	// Parse CSV and try to extract parameters and expected results
	records, err := parseCSV(csvContent)
	if err != nil {
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV must have at least header and one data row")
	}

	headers := records[0]

	// Process each data row as a separate test scenario
	for i, record := range records[1:] {
		if len(record) != len(headers) {
			continue // Skip malformed rows
		}

		// Create parameters from CSV row
		params := make(map[string]any)
		var expected any

		for j, value := range record {
			if j < len(headers) {
				header := strings.TrimSpace(headers[j])

				// Special handling for expected result columns
				if strings.ToLower(header) == "expected" ||
					strings.ToLower(header) == "result" ||
					strings.ToLower(header) == "output" {
					expected = parseValue(value)
				} else {
					params[header] = parseValue(value)
				}
			}
		}

		// For the first row, update the current test case
		if i == 0 {
			for k, v := range params {
				testCase.Parameters[k] = v
			}
			if expected != nil {
				testCase.ExpectedResult = expected
			}
		}
		// Additional rows could be handled as separate test cases if needed
	}

	return nil
}

// parseCSVToMockData parses CSV content into mock data format
func parseCSVToMockData(csvContent string) (map[string]any, error) {
	// Check for table name comment at the beginning
	tableName := "csv_data" // Default table name
	lines := strings.Split(strings.TrimSpace(csvContent), "\n")

	var csvLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for table name comment (e.g., "# users" or "// users")
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			comment := strings.TrimSpace(line[1:])
			if comment != "" && !strings.Contains(comment, " ") {
				tableName = comment
			}
			continue
		}

		csvLines = append(csvLines, line)
	}

	if len(csvLines) < 2 {
		return nil, fmt.Errorf("CSV must have at least header and one data row")
	}

	// Parse CSV records
	records, err := parseCSVFromLines(csvLines)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	headers := records[0]
	mockData := make(map[string]any)

	// Process each data row
	for i, record := range records[1:] {
		if len(record) != len(headers) {
			continue // Skip malformed rows
		}

		rowData := make(map[string]any)
		for j, value := range record {
			if j < len(headers) {
				header := strings.TrimSpace(headers[j])
				rowData[header] = parseValue(value)
			}
		}

		mockData[fmt.Sprintf("row_%d", i)] = rowData
	}

	// Return with table name as key
	result := make(map[string]any)
	result[tableName] = mockData
	return result, nil
}

// DBUnit XML structures
type DBUnitDataset struct {
	XMLName xml.Name    `xml:"dataset"`
	Tables  []DBUnitRow `xml:",any"`
}

type DBUnitRow struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
}

// parseXMLIntoTestCase parses XML content into a test case
func parseXMLIntoTestCase(xmlContent string, testCase *TestCase) error {
	// Try to parse as DBUnit dataset first
	dataset, err := parseDBUnitXML(xmlContent)
	if err == nil {
		// Convert DBUnit data to test case parameters
		if len(dataset.Tables) > 0 {
			// Use first table as parameters source
			firstTable := dataset.Tables[0]
			params := make(map[string]any)

			for _, attr := range firstTable.Attrs {
				params[attr.Name.Local] = parseValue(attr.Value)
			}

			for k, v := range params {
				testCase.Parameters[k] = v
			}
		}
		return nil
	}

	// Try to parse as generic XML
	var xmlData map[string]any
	if err := parseGenericXML(xmlContent, &xmlData); err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}

	// Extract parameters and expected results from XML data
	if parameters, ok := xmlData["parameters"]; ok {
		if paramMap, ok := parameters.(map[string]any); ok {
			for k, v := range paramMap {
				testCase.Parameters[k] = v
			}
		}
	}

	if expected, ok := xmlData["expected"]; ok {
		testCase.ExpectedResult = expected
	}

	return nil
}

// parseXMLToMockData parses XML content into mock data format
func parseXMLToMockData(xmlContent string) (map[string]map[string]any, error) {
	// Try to parse as DBUnit dataset
	dataset, err := parseDBUnitXML(xmlContent)
	if err == nil {
		return convertDBUnitToMockData(dataset), nil
	}

	// Try to parse as generic XML with table structure
	mockData, err := parseGenericXMLToMockData(xmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML mock data: %w", err)
	}

	return mockData, nil
}

// parseDBUnitXML parses DBUnit XML format
func parseDBUnitXML(xmlContent string) (*DBUnitDataset, error) {
	var dataset DBUnitDataset

	// Clean up XML content
	xmlContent = strings.TrimSpace(xmlContent)

	// Add dataset wrapper if not present
	if !strings.Contains(xmlContent, "<dataset") {
		xmlContent = "<dataset>" + xmlContent + "</dataset>"
	}

	if err := xml.Unmarshal([]byte(xmlContent), &dataset); err != nil {
		return nil, err
	}

	return &dataset, nil
}

// convertDBUnitToMockData converts DBUnit dataset to mock data format
func convertDBUnitToMockData(dataset *DBUnitDataset) map[string]map[string]any {
	mockData := make(map[string]map[string]any)
	tableRowCounts := make(map[string]int)

	for _, row := range dataset.Tables {
		tableName := row.XMLName.Local

		// Initialize table if not exists
		if _, exists := mockData[tableName]; !exists {
			mockData[tableName] = make(map[string]any)
		}

		// Create row data
		rowData := make(map[string]any)
		for _, attr := range row.Attrs {
			rowData[attr.Name.Local] = parseValue(attr.Value)
		}

		// Add row to table
		rowKey := fmt.Sprintf("row_%d", tableRowCounts[tableName])
		mockData[tableName][rowKey] = rowData
		tableRowCounts[tableName]++
	}

	return mockData
}

// parseGenericXML parses generic XML into map structure
func parseGenericXML(xmlContent string, result *map[string]any) error {
	// This is a simplified XML parser for basic structures
	// In a production system, you might want to use a more sophisticated XML-to-map converter

	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	*result = make(map[string]any)

	var stack []map[string]any
	var current map[string]any = *result

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Create new element
			newElement := make(map[string]any)

			// Add attributes
			for _, attr := range t.Attr {
				newElement["@"+attr.Name.Local] = parseValue(attr.Value)
			}

			// Add to current level
			if existing, exists := current[t.Name.Local]; exists {
				// Convert to array if multiple elements with same name
				if arr, ok := existing.([]any); ok {
					current[t.Name.Local] = append(arr, newElement)
				} else {
					current[t.Name.Local] = []any{existing, newElement}
				}
			} else {
				current[t.Name.Local] = newElement
			}

			// Push to stack
			stack = append(stack, current)
			current = newElement

		case xml.EndElement:
			// Pop from stack
			if len(stack) > 0 {
				current = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			// Add text content
			text := strings.TrimSpace(string(t))
			if text != "" {
				current["#text"] = parseValue(text)
			}
		}
	}

	return nil
}

// parseGenericXMLToMockData parses generic XML to mock data format
func parseGenericXMLToMockData(xmlContent string) (map[string]map[string]any, error) {
	var xmlData map[string]any
	if err := parseGenericXML(xmlContent, &xmlData); err != nil {
		return nil, err
	}

	mockData := make(map[string]map[string]any)

	// Look for table-like structures in XML
	for key, value := range xmlData {
		if valueMap, ok := value.(map[string]any); ok {
			// Single table entry
			mockData[key] = map[string]any{"row_0": valueMap}
		} else if valueArray, ok := value.([]any); ok {
			// Multiple entries for same table
			tableData := make(map[string]any)
			for i, item := range valueArray {
				if itemMap, ok := item.(map[string]any); ok {
					tableData[fmt.Sprintf("row_%d", i)] = itemMap
				}
			}
			mockData[key] = tableData
		}
	}

	return mockData, nil
}

// parseCSVFromLines parses CSV lines into a 2D string array
func parseCSVFromLines(lines []string) ([][]string, error) {
	var records [][]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Simple CSV parsing (handles basic cases)
		record, err := parseCSVLine(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV line '%s': %w", line, err)
		}

		records = append(records, record)
	}

	return records, nil
}

// parseCSV parses CSV content into a 2D string array (legacy function)
func parseCSV(csvContent string) ([][]string, error) {
	lines := strings.Split(strings.TrimSpace(csvContent), "\n")
	var csvLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip comment lines
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		csvLines = append(csvLines, line)
	}

	return parseCSVFromLines(csvLines)
}

// parseCSVLine parses a single CSV line into fields with improved quote handling
func parseCSVLine(line string) ([]string, error) {
	var fields []string
	var current strings.Builder
	inQuotes := false
	i := 0

	for i < len(line) {
		r := rune(line[i])

		switch r {
		case '"':
			if inQuotes {
				// Check for escaped quote (double quote)
				if i+1 < len(line) && line[i+1] == '"' {
					current.WriteRune('"')
					i++ // Skip the next quote
				} else {
					inQuotes = false
				}
			} else {
				inQuotes = true
			}

		case ',':
			if inQuotes {
				current.WriteRune(r)
			} else {
				fields = append(fields, strings.TrimSpace(current.String()))
				current.Reset()
			}

		case '\r':
			// Skip carriage return

		case '\n':
			if inQuotes {
				current.WriteRune(r)
			} else {
				// End of line
				break
			}

		default:
			current.WriteRune(r)
		}

		i++
	}

	// Add the last field
	fields = append(fields, strings.TrimSpace(current.String()))

	return fields, nil
}

// getCodeBlockInfo extracts the info string from a fenced code block
func getCodeBlockInfo(codeBlock *ast.FencedCodeBlock, content []byte) string {
	if codeBlock.Info != nil {
		segment := codeBlock.Info.Segment
		return string(content[segment.Start:segment.Stop])
	}
	return ""
}

// extractTextFromNode extracts text content from any AST node
func extractTextFromNode(node ast.Node, content []byte) string {
	var result strings.Builder

	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch textNode := n.(type) {
		case *ast.Text:
			segment := textNode.Segment
			result.Write(content[segment.Start:segment.Stop])
		case *ast.String:
			result.Write(textNode.Value)
		}

		return ast.WalkContinue, nil
	})

	return strings.TrimSpace(result.String())
}

// parseYAMLIntoTestCase parses YAML content into a test case
func parseYAMLIntoTestCase(yamlContent string, testCase *TestCase) error {
	var data map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &data); err != nil {
		return err
	}

	// Map YAML fields to test case fields
	if fixture, ok := data["fixture"]; ok {
		if fixtureMap, ok := fixture.(map[string]any); ok {
			testCase.Fixture = fixtureMap
		}
	}

	if parameters, ok := data["parameters"]; ok {
		if paramMap, ok := parameters.(map[string]any); ok {
			testCase.Parameters = paramMap
		}
	}

	if expected, ok := data["expected"]; ok {
		testCase.ExpectedResult = expected
	}

	// Also check for alternative field names
	if input, ok := data["input"]; ok {
		if inputMap, ok := input.(map[string]any); ok {
			for k, v := range inputMap {
				testCase.Parameters[k] = v
			}
		}
	}

	if output, ok := data["output"]; ok {
		testCase.ExpectedResult = output
	}

	return nil
}

// parseJSONIntoTestCase parses JSON content into a test case
func parseJSONIntoTestCase(jsonContent string, testCase *TestCase) error {
	// Use YAML parser for JSON (YAML is a superset of JSON)
	return parseYAMLIntoTestCase(jsonContent, testCase)
}

// parseListIntoTestCase parses list items into test case data
func parseListIntoTestCase(listNode *ast.List, content []byte, testCase *TestCase) error {
	ast.Walk(listNode, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if listItem, ok := n.(*ast.ListItem); ok {
			itemText := extractTextFromNode(listItem, content)

			// Parse key-value pairs from list items
			// Format: "key: value" or "Input: key = value"
			if err := parseKeyValueText(itemText, testCase); err != nil {
				// Ignore parsing errors for list items
				return ast.WalkContinue, nil
			}
		}

		return ast.WalkContinue, nil
	})

	return nil
}

// parseParagraphIntoTestCase parses paragraph text for simple key-value pairs
func parseParagraphIntoTestCase(paragraphText string, testCase *TestCase) error {
	lines := strings.Split(paragraphText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse as key-value pair
		if err := parseKeyValueText(line, testCase); err != nil {
			// Ignore parsing errors for paragraphs
			continue
		}
	}

	return nil
}

// parseKeyValueText parses text like "Input: user_id = 123" or "Expected: success"
func parseKeyValueText(text string, testCase *TestCase) error {
	// Handle formats like:
	// "Input: user_id = 123, include_email = true"
	// "Expected: Returns user data with email"
	// "Parameters: user_id = 123"

	if strings.Contains(text, ":") {
		parts := strings.SplitN(text, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key-value format")
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "input", "parameters", "params":
			// Parse parameter assignments like "user_id = 123, include_email = true"
			params := parseParameterAssignments(value)
			for k, v := range params {
				testCase.Parameters[k] = v
			}

		case "expected", "output", "result":
			testCase.ExpectedResult = value

		case "fixture", "setup":
			// Simple fixture parsing
			testCase.Fixture["description"] = value
		}
	}

	return nil
}

// parseParameterAssignments parses text like "user_id = 123, include_email = true"
func parseParameterAssignments(text string) map[string]any {
	params := make(map[string]any)

	// Split by comma
	assignments := strings.Split(text, ",")
	for _, assignment := range assignments {
		assignment = strings.TrimSpace(assignment)

		// Split by equals sign
		if strings.Contains(assignment, "=") {
			parts := strings.SplitN(assignment, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Try to parse value as different types
				params[key] = parseValue(value)
			}
		}
	}

	return params
}

// parseValue attempts to parse a string value into appropriate type
func parseValue(value string) any {
	value = strings.TrimSpace(value)

	// Remove quotes if present
	if (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) ||
		(strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) {
		return value[1 : len(value)-1]
	}

	// Try to parse as boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try to parse as number
	if strings.Contains(value, ".") {
		if f, err := parseFloat(value); err == nil {
			return f
		}
	} else {
		if i, err := parseInt(value); err == nil {
			return i
		}
	}

	// Return as string
	return value
}

// parseTableToMockData parses a markdown table into mock data
func parseTableToMockData(tableNode ast.Node, content []byte) (map[string]any, error) {
	mockData := make(map[string]any)
	var headers []string
	rowIndex := 0

	ast.Walk(tableNode, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *extast.TableHeader:
			// Extract headers
			ast.Walk(node, func(headerNode ast.Node, headerEntering bool) (ast.WalkStatus, error) {
				if !headerEntering {
					return ast.WalkContinue, nil
				}

				if cell, ok := headerNode.(*extast.TableCell); ok {
					cellText := extractTextFromNode(cell, content)
					headers = append(headers, strings.TrimSpace(cellText))
				}

				return ast.WalkContinue, nil
			})

		case *extast.TableRow:
			// Skip if this is part of the header
			if len(headers) == 0 {
				return ast.WalkContinue, nil
			}

			// Extract row data
			var cellValues []string
			ast.Walk(node, func(rowNode ast.Node, rowEntering bool) (ast.WalkStatus, error) {
				if !rowEntering {
					return ast.WalkContinue, nil
				}

				if cell, ok := rowNode.(*extast.TableCell); ok {
					cellText := extractTextFromNode(cell, content)
					cellValues = append(cellValues, strings.TrimSpace(cellText))
				}

				return ast.WalkContinue, nil
			})

			// Create row data
			if len(headers) > 0 && len(cellValues) > 0 {
				rowData := make(map[string]any)
				for i, header := range headers {
					if i < len(cellValues) {
						rowData[header] = parseValue(cellValues[i])
					}
				}
				mockData[fmt.Sprintf("row_%d", rowIndex)] = rowData
				rowIndex++
			}
		}

		return ast.WalkContinue, nil
	})

	return mockData, nil
}

// Helper functions for parsing numbers
func parseInt(s string) (int, error) {
	// Simple integer parsing
	result := 0
	negative := false

	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer")
		}
		result = result*10 + int(r-'0')
	}

	if negative {
		result = -result
	}

	return result, nil
}

func parseFloat(s string) (float64, error) {
	// Simple float parsing (basic implementation)
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid float")
	}

	intPart, err := parseInt(parts[0])
	if err != nil {
		return 0, err
	}

	fracPart, err := parseInt(parts[1])
	if err != nil {
		return 0, err
	}

	// Calculate decimal places
	decimalPlaces := len(parts[1])
	divisor := 1.0
	for i := 0; i < decimalPlaces; i++ {
		divisor *= 10
	}

	result := float64(intPart) + float64(fracPart)/divisor
	return result, nil
}

// parseFrontMatter extracts YAML front matter from markdown content
func parseFrontMatter(content string) (map[string]any, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		// No front matter, return empty metadata and original content
		return make(map[string]any), content, nil
	}

	// Find the end of front matter
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return nil, "", fmt.Errorf("%w: missing closing ---", ErrInvalidFrontMatter)
	}

	// Extract front matter YAML
	frontMatterLines := lines[1:endIndex]
	frontMatterYAML := strings.Join(frontMatterLines, "\n")

	var frontMatter map[string]any
	if err := yaml.Unmarshal([]byte(frontMatterYAML), &frontMatter); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidFrontMatter, err)
	}

	// Return content without front matter
	contentWithoutFrontMatter := strings.Join(lines[endIndex+1:], "\n")
	return frontMatter, contentWithoutFrontMatter, nil
}

// countFrontMatterLines counts the number of lines used by front matter
func countFrontMatterLines(content string) int {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return 0
	}

	// Find the end of front matter
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i + 1 // +1 to include the closing ---
		}
	}

	return 0
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

// generateFunctionNameFromTitle generates a function name from title
func generateFunctionNameFromTitle(title string) string {
	// Simple implementation: convert to camelCase
	words := strings.Fields(strings.ToLower(title))
	if len(words) == 0 {
		return "query"
	}

	result := words[0]
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			result += strings.ToUpper(string(words[i][0])) + words[i][1:]
		}
	}

	return result
}

// extractParameterBlock extracts parameter definitions from AST nodes
func extractParameterBlock(nodes []ast.Node, content []byte) string {
	var parameterContent strings.Builder
	
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FencedCodeBlock:
			// Extract code block content (YAML, JSON, etc.)
			codeContent := extractCodeBlockContent(n, content)
			info := getCodeBlockInfo(n, content)
			
			// Add info line if present
			if info != "" {
				parameterContent.WriteString("```" + info + "\n")
			} else {
				parameterContent.WriteString("```\n")
			}
			parameterContent.WriteString(codeContent)
			parameterContent.WriteString("\n```\n")
			
		case *ast.Paragraph:
			// Extract paragraph text
			paragraphText := extractTextFromNode(n, content)
			if paragraphText != "" {
				parameterContent.WriteString(paragraphText)
				parameterContent.WriteString("\n\n")
			}
			
		case *ast.List:
			// Extract list content
			listText := extractTextFromNode(n, content)
			if listText != "" {
				parameterContent.WriteString(listText)
				parameterContent.WriteString("\n\n")
			}
		}
	}
	
	return strings.TrimSpace(parameterContent.String())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
