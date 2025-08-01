package markdownparser

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// TestCase represents a single test case
type TestCase struct {
	Name           string
	Fixture        map[string][]map[string]any // テーブル名をキーとしたマップ
	Parameters     map[string]any
	ExpectedResult []map[string]any
}

// TestSection represents a section within a test case
type TestSection struct {
	Type      string // "parameters", "expected", "fixtures"
	TableName string // Only used for CSV fixtures
}

// parseTestCasesFromAST parses test cases from AST nodes
func parseTestCasesFromAST(nodes []ast.Node, content []byte) ([]TestCase, error) {
	var testCases []TestCase
	var currentTestCase *TestCase
	var errors []error
	var currentSection TestSection

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.Heading:
			// Save previous test case if exists
			if currentTestCase != nil {
				if err := validateTestCase(currentTestCase); err != nil {
					errors = append(errors, err)
				}
				testCases = append(testCases, *currentTestCase)
			}

			// Start new test case
			testName := extractTextFromHeadingNode(n, content)
			currentTestCase = &TestCase{
				Name:           testName,
				Fixture:        make(map[string][]map[string]any),
				Parameters:     make(map[string]any),
				ExpectedResult: make([]map[string]any, 0),
			}
			currentSection = TestSection{}

		case *ast.Paragraph:
			// Check for section markers in emphasis nodes
			if currentTestCase != nil {
				emphasis := findFirstEmphasis(n)
				if emphasis != nil {
					text := extractTextFromNode(emphasis, content)
					text = strings.ToLower(strings.TrimSpace(text))

					if strings.HasPrefix(text, "parameters:") || text == "params:" {
						currentSection = TestSection{Type: "parameters"}
					} else if strings.HasPrefix(text, "expected:") || strings.HasPrefix(text, "expected results:") || text == "results:" {
						currentSection = TestSection{Type: "expected"}
					} else if strings.HasPrefix(text, "fixtures") {
						currentSection = TestSection{Type: "fixtures"}
						// Extract table name if present
						if i := strings.Index(text, ":"); i >= 0 {
							tableName := strings.TrimSpace(text[i+1:])
							if tableName != "" {
								currentSection.TableName = tableName
							}
						}
					}
				}
			}

		case *ast.FencedCodeBlock:
			if currentTestCase != nil && currentSection.Type != "" {
				// Get code block info
				info := strings.ToLower(strings.TrimSpace(string(n.Info.Text(content))))

				// Get code block content
				var codeContent strings.Builder
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					codeContent.Write(line.Value(content))
					if i < lines.Len()-1 {
						codeContent.WriteString("\n")
					}
				}

				// Process the section
				if err := processTestSection(currentTestCase, currentSection, info, []byte(codeContent.String())); err != nil {
					errors = append(errors, fmt.Errorf("in test case %q: %w", currentTestCase.Name, err))
				}

				// Reset section after processing
				currentSection = TestSection{}
			}
		}
	}

	// Handle last test case
	if currentTestCase != nil {
		if err := validateTestCase(currentTestCase); err != nil {
			errors = append(errors, err)
		}
		testCases = append(testCases, *currentTestCase)
	}

	// If there were any errors, return them
	if len(errors) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString("errors in test cases:\n")
		for _, err := range errors {
			errMsg.WriteString(fmt.Sprintf("- %v\n", err))
		}
		return nil, fmt.Errorf("%s", errMsg.String())
	}

	return testCases, nil
}

// findFirstEmphasis finds the first emphasis node in a paragraph
func findFirstEmphasis(paragraph *ast.Paragraph) *ast.Emphasis {
	var emphasis *ast.Emphasis
	ast.Walk(paragraph, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == ast.KindEmphasis {
			emphasis = n.(*ast.Emphasis)
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return emphasis
}

// extractTextFromNode extracts all text content from a node and its children
func extractTextFromNode(node ast.Node, content []byte) string {
	var text strings.Builder
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == ast.KindText {
			text.Write(n.Text(content))
		}
		return ast.WalkContinue, nil
	})
	return text.String()
}

// validateTestCase validates a test case for required sections and format
func validateTestCase(testCase *TestCase) error {
	// Parameters are required
	if len(testCase.Parameters) == 0 {
		return fmt.Errorf("test case %q is missing required parameters", testCase.Name)
	}

	// Expected Results are required
	if len(testCase.ExpectedResult) == 0 {
		return fmt.Errorf("test case %q is missing required expected results", testCase.Name)
	}

	return nil
}

// processTestSection processes a section of a test case
func processTestSection(testCase *TestCase, section TestSection, format string, content []byte) error {
	switch section.Type {
	case "parameters", "params":
		// パラメータが既に設定されている場合はエラー
		if len(testCase.Parameters) > 0 {
			return fmt.Errorf("%w in test case %q", ErrDuplicateParameters, testCase.Name)
		}
		params, err := parseParameters(content)
		if err != nil {
			return fmt.Errorf("failed to parse parameters in test case %q: %w", testCase.Name, err)
		}
		testCase.Parameters = params

	case "expected", "expected results", "results":
		// Expected Resultsが既に設定されている場合はエラー
		if len(testCase.ExpectedResult) > 0 {
			return fmt.Errorf("%w in test case %q", ErrDuplicateExpectedResults, testCase.Name)
		}
		results, err := parseExpectedResults(content)
		if err != nil {
			return fmt.Errorf("failed to parse expected results in test case %q: %w", testCase.Name, err)
		}
		testCase.ExpectedResult = results

	case "fixtures":
		// CSVの場合はテーブル名が必要
		if format == "csv" {
			if section.TableName == "" {
				return fmt.Errorf("table name is required for CSV fixtures in test case %q", testCase.Name)
			}
		} else if section.TableName != "" {
			// CSV以外でテーブル名が指定されている場合はエラー
			return fmt.Errorf("table name can only be specified for CSV fixtures in test case %q", testCase.Name)
		}

		// Parse fixtures data
		if format == "csv" {
			// CSVの場合は単一のテーブル
			rows, err := parseCSVData(content)
			if err != nil {
				return fmt.Errorf("failed to parse CSV fixtures in test case %q: %w", testCase.Name, err)
			}
			testCase.Fixture[section.TableName] = append(testCase.Fixture[section.TableName], rows...)
		} else {
			// YAML/XMLの場合はテーブル名が含まれている
			tableData, err := parseStructuredData(content, format)
			if err != nil {
				return fmt.Errorf("failed to parse fixtures in test case %q: %w", testCase.Name, err)
			}
			for tableName, rows := range tableData {
				testCase.Fixture[tableName] = append(testCase.Fixture[tableName], rows...)
			}
		}
	}

	return nil
}
