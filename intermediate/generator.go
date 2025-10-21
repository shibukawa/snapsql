package intermediate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// MockDataGenerator generates mock data JSON files from markdown test cases
type MockDataGenerator struct {
	ProjectRoot string
	OutputDir   string
}

// NewMockDataGenerator creates a new mock data generator
func NewMockDataGenerator(projectRoot string) *MockDataGenerator {
	return &MockDataGenerator{
		ProjectRoot: projectRoot,
		OutputDir:   filepath.Join(projectRoot, "testdata", "snapsql_mock"),
	}
}

// TestCase represents a test case from markdown
type TestCase struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Parameters  map[string]interface{}   `json:"parameters"`
	MockData    []map[string]interface{} `json:"mock_data"`
}

// GenerateMockFiles generates mock data JSON files from markdown content
func (g *MockDataGenerator) GenerateMockFiles(markdownFile string, markdownContent string) error {
	// Parse test cases from markdown
	testCases := g.parseTestCasesFromMarkdown(markdownContent)

	if len(testCases) == 0 {
		// No test cases found, skip mock data generation
		return nil
	}

	// Extract function name from markdown content
	functionName := g.extractFunctionName(markdownContent)

	// Generate mock data files (one per test case)
	if err := g.writeMockDataFile(markdownFile, testCases, functionName); err != nil {
		return fmt.Errorf("failed to write mock data files: %w", err)
	}

	fmt.Printf("Generated %d mock data files for function: %s\n", len(testCases), functionName)

	return nil
}

// parseTestCasesFromMarkdown parses test cases from markdown content
func (g *MockDataGenerator) parseTestCasesFromMarkdown(content string) []TestCase {
	testCases := make([]TestCase, 0)

	// Regular expressions to match test case sections
	testCaseRegex := regexp.MustCompile(`(?m)^### Test Case \d+: (.+)$`)
	parametersRegex := regexp.MustCompile(`(?s)\*\*Input Parameters:\*\*\s*` + "```json\\s*(\\{.*?\\})\\s*```")
	mockDataRegex := regexp.MustCompile(`(?s)` + "```yaml\\s*#[^\\n]*\\n(.*?)\\s*```")

	// Find all test case headers
	testCaseMatches := testCaseRegex.FindAllStringSubmatch(content, -1)

	for i, match := range testCaseMatches {
		if len(match) < 2 {
			continue
		}

		testCase := TestCase{
			Name: strings.TrimSpace(match[1]),
		}

		// Find the section content for this test case
		startPos := strings.Index(content, match[0])

		var endPos int
		if i+1 < len(testCaseMatches) {
			endPos = strings.Index(content, testCaseMatches[i+1][0])
		} else {
			endPos = len(content)
		}

		if startPos == -1 || endPos <= startPos {
			continue
		}

		sectionContent := content[startPos:endPos]

		// Extract parameters
		paramMatches := parametersRegex.FindStringSubmatch(sectionContent)
		if len(paramMatches) >= 2 {
			var params map[string]interface{}

			err := json.Unmarshal([]byte(paramMatches[1]), &params)
			if err == nil {
				testCase.Parameters = params
			}
		}

		// Extract mock data from YAML blocks
		mockMatches := mockDataRegex.FindAllStringSubmatch(sectionContent, -1)
		if len(mockMatches) > 0 {
			// For now, create simple mock data structure
			// In a real implementation, this would parse YAML and convert to appropriate format
			mockData := make([]map[string]interface{}, 0)

			// Create sample mock data based on common patterns
			sampleRecord := map[string]interface{}{
				"id":         1,
				"name":       "Sample User",
				"email":      "user@example.com",
				"active":     true,
				"created_at": "2024-01-15T10:30:00Z",
			}
			mockData = append(mockData, sampleRecord)

			testCase.MockData = mockData
		}

		testCases = append(testCases, testCase)
	}

	return testCases
}

// extractFunctionName extracts the function name from SQL comments in markdown
func (g *MockDataGenerator) extractFunctionName(content string) string {
	// Look for @name: directive in SQL code blocks
	nameRegex := regexp.MustCompile(`--\s*@name:\s*([^\s\n]+)`)

	matches := nameRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// writeMockDataFile writes mock data to individual JSON files for each test case
func (g *MockDataGenerator) writeMockDataFile(sourceFile string, testCases []TestCase, functionName string) error {
	if len(testCases) == 0 {
		return nil
	}

	// Create base directory for the function
	baseName := filepath.Base(sourceFile)
	// Remove .snap.md extension
	if filepath.Ext(baseName) == ".md" {
		baseName = baseName[:len(baseName)-3] // Remove .md
		if filepath.Ext(baseName) == ".snap" {
			baseName = baseName[:len(baseName)-5] // Remove .snap
		}
	}

	// Use function name if available, otherwise use file name
	dirName := functionName
	if dirName == "" {
		dirName = baseName
	}

	baseDir := filepath.Join(g.OutputDir, dirName)

	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", baseDir, err)
	}

	// Write each test case to a separate file
	for _, testCase := range testCases {
		// Create safe filename from test case name
		safeFileName := strings.ReplaceAll(strings.ToLower(testCase.Name), " ", "_")
		safeFileName = strings.ReplaceAll(safeFileName, ":", "")
		safeFileName = strings.ReplaceAll(safeFileName, "-", "_")
		filePath := filepath.Join(baseDir, safeFileName+".json")

		// Write JSON file with pretty formatting - only mock_data
		jsonData, err := json.MarshalIndent(testCase.MockData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for test case %s: %w", testCase.Name, err)
		}

		err = os.WriteFile(filePath, jsonData, 0644)
		if err != nil {
			return fmt.Errorf("failed to write mock data file %s: %w", filePath, err)
		}
	}

	return nil
}

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

	// Generate mock data from test cases if enabled
	if config != nil && config.Generation.GenerateMockData {
		// Read the original markdown content from file
		markdownContent, err := os.ReadFile(basePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read markdown file for mock generation: %w", err)
		}

		mockGenerator := NewMockDataGenerator(projectRootPath)

		err = mockGenerator.GenerateMockFiles(basePath, string(markdownContent))
		if err != nil {
			return nil, fmt.Errorf("failed to generate mock data: %w", err)
		}
	}

	// Generate intermediate format
	return generateIntermediateFormat(stmt, typeInfoMap, funcDef, basePath, tableInfo, config)
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

// extractSystemFieldsInfo extracts system fields information from config
func extractSystemFieldsInfo(config *snapsql.Config, stmt parsercommon.StatementNode) []SystemFieldInfo {
	_ = stmt // Statement not currently used for system field extraction

	var systemFields []SystemFieldInfo

	// Get all system fields from config
	for _, field := range config.System.Fields {
		systemFieldInfo := SystemFieldInfo{
			Name:              field.Name,
			ExcludeFromSelect: field.ExcludeFromSelect,
		}

		// Convert OnInsert configuration
		if field.OnInsert.Default != nil || field.OnInsert.Parameter != "" {
			systemFieldInfo.OnInsert = &SystemFieldOperationInfo{
				Default:   field.OnInsert.Default,
				Parameter: string(field.OnInsert.Parameter),
			}
		}

		// Convert OnUpdate configuration
		if field.OnUpdate.Default != nil || field.OnUpdate.Parameter != "" {
			systemFieldInfo.OnUpdate = &SystemFieldOperationInfo{
				Default:   field.OnUpdate.Default,
				Parameter: string(field.OnUpdate.Parameter),
			}
		}

		systemFields = append(systemFields, systemFieldInfo)
	}

	return systemFields
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

// setEnvIndexInInstructions sets env_index in loop instructions based on envs data
func setEnvIndexInInstructions(envs [][]EnvVar, instructions []Instruction) {
	// Stack to track environment indices for nested loops
	var loopStack []int

	for i := range instructions {
		instruction := &instructions[i]

		switch instruction.Op {
		case OpLoopStart:
			if instruction.Variable != "" {
				// Find the environment index where this variable is introduced
				envIndex := findVariableEnvironmentIndex(envs, instruction.Variable)
				if envIndex > 0 {
					instruction.EnvIndex = &envIndex
					loopStack = append(loopStack, envIndex)
				}
			}

		case OpLoopEnd:
			if len(loopStack) > 0 {
				// Pop the current loop environment
				loopStack = loopStack[:len(loopStack)-1]

				// Set env_index to the environment we return to after this loop ends
				var envIndex int
				if len(loopStack) > 0 {
					// Still inside nested loops, use the parent loop's environment
					envIndex = loopStack[len(loopStack)-1]
				} else {
					// Exiting the outermost loop, return to base environment (index 0)
					envIndex = 0
				}

				instruction.EnvIndex = &envIndex
			}
		}
	}
}

// findVariableEnvironmentIndex finds the environment index where the variable is first introduced
func findVariableEnvironmentIndex(envs [][]EnvVar, variable string) int {
	for i := range envs {
		// Check if this variable is in this environment level
		for _, envVar := range envs[i] {
			if envVar.Name == variable {
				// Check if it was also in the previous environment level
				if i > 0 {
					found := false

					for _, prevEnvVar := range envs[i-1] {
						if prevEnvVar.Name == variable {
							found = true
							break
						}
					}

					if !found {
						// This is where the variable was introduced
						return i + 1 // Convert to 1-based index
					}
				} else {
					// First environment level, this is where it was introduced
					return i + 1 // Convert to 1-based index
				}
			}
		}
	}

	return 0 // Default to base environment
}
