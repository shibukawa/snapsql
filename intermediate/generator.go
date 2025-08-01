package intermediate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/shibukawa/snapsql"
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
	testCases, err := g.parseTestCasesFromMarkdown(markdownContent)
	if err != nil {
		return fmt.Errorf("failed to parse test cases: %w", err)
	}

	if len(testCases) == 0 {
		// No test cases found, skip mock data generation
		return nil
	}

	// Extract function name from markdown content
	functionName := g.extractFunctionName(markdownContent)

	// Generate mock data files (one per test case)
	err = g.writeMockDataFile(markdownFile, testCases, functionName)
	if err != nil {
		return fmt.Errorf("failed to write mock data files: %w", err)
	}

	fmt.Printf("Generated %d mock data files for function: %s\n", len(testCases), functionName)
	return nil
}

// parseTestCasesFromMarkdown parses test cases from markdown content
func (g *MockDataGenerator) parseTestCasesFromMarkdown(content string) ([]TestCase, error) {
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
			if err := json.Unmarshal([]byte(paramMatches[1]), &params); err == nil {
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

	return testCases, nil
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

// getMockFilePath generates the mock data file path
func (g *MockDataGenerator) getMockFilePath(sourceFile string) string {
	// Convert source file path to mock file path
	// e.g., queries/users/find.snap.md -> testdata/snapsql_mock/users/find.json

	baseName := filepath.Base(sourceFile)
	// Remove .snap.md extension and add .json
	if filepath.Ext(baseName) == ".md" {
		baseName = baseName[:len(baseName)-3] // Remove .md
		if filepath.Ext(baseName) == ".snap" {
			baseName = baseName[:len(baseName)-5] // Remove .snap
		}
	}
	baseName += ".json"

	// Get directory structure
	dir := filepath.Dir(sourceFile)

	return filepath.Join(g.OutputDir, dir, baseName)
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
func GenerateFromSQL(reader io.Reader, constants map[string]any, basePath string, projectRootPath string, tableInfo map[string]*TableInfo, config *Config) (*IntermediateFormat, error) {
	// Parse the SQL
	stmt, funcDef, err := parser.ParseSQLFile(reader, constants, basePath, projectRootPath)
	if err != nil {
		return nil, err
	}

	// Generate intermediate format
	return generateIntermediateFormat(stmt, funcDef, basePath, tableInfo, config)
}

// GenerateFromMarkdown generates the intermediate format for a Markdown file containing SQL
func GenerateFromMarkdown(doc *markdownparser.SnapSQLDocument, basePath string, projectRootPath string, constants map[string]any, tableInfo map[string]*TableInfo, config *Config) (*IntermediateFormat, error) {
	// Parse the Markdown
	stmt, funcDef, err := parser.ParseMarkdownFile(doc, basePath, projectRootPath, constants)
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
	return generateIntermediateFormat(stmt, funcDef, basePath, tableInfo, config)
}

// generateIntermediateFormat is the common implementation using the new pipeline approach
func generateIntermediateFormat(stmt parsercommon.StatementNode, funcDef *parsercommon.FunctionDefinition, filePath string, tableInfo map[string]*TableInfo, config *Config) (*IntermediateFormat, error) {
	// Create and execute the token processing pipeline
	pipeline := CreateDefaultPipeline(stmt, funcDef, config, tableInfo)

	result, err := pipeline.Execute()
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	return result, nil
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

	// Process tokens from each clause
	for _, clause := range stmt.Clauses() {
		tokens = append(tokens, clause.RawTokens()...)
	}

	// Process CTE tokens if available
	if cte := stmt.CTE(); cte != nil {
		tokens = append(tokens, cte.RawTokens()...)
	}

	return tokens
}

// extractSystemFieldsInfo extracts system fields information from config
func extractSystemFieldsInfo(config *Config, stmt parsercommon.StatementNode) []SystemFieldInfo {
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

// addClauseIfConditions adds IF instructions for clause-level conditions
func addClauseIfConditions(stmt parsercommon.StatementNode, instructions []Instruction) []Instruction {
	// Process each clause
	for _, clause := range stmt.Clauses() {
		// Check if the clause has an IF condition
		if condition := clause.IfCondition(); condition != "" {
			// Find the position to insert the IF instruction
			// This is a simplified approach - in a real implementation, we would need to find the exact position
			// based on the clause's position in the SQL

			// For now, we'll just add the IF instruction at the beginning of the instructions
			// and the END instruction at the end

			// Create IF instruction
			ifInstruction := Instruction{
				Op:        OpIf,
				Pos:       "0:0", // Placeholder position
				Condition: condition,
			}

			// Create END instruction
			endInstruction := Instruction{
				Op:  OpEnd,
				Pos: "0:0", // Placeholder position
			}

			// Insert IF instruction at the beginning
			instructions = append([]Instruction{ifInstruction}, instructions...)

			// Append END instruction at the end
			instructions = append(instructions, endInstruction)
		}
	}

	return instructions
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
	for i := 0; i < len(envs); i++ {
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
