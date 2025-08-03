package markdownparser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// InsertStrategy represents the strategy for inserting fixture data
type InsertStrategy string

const (
	// ClearInsert truncates the table then inserts data (default)
	ClearInsert InsertStrategy = "clear-insert"
	// Insert just inserts data into the table
	Insert InsertStrategy = "insert"
	// Upsert inserts data into the table, or updates if the row already exists
	Upsert InsertStrategy = "upsert"
	// Delete deletes rows in the table that match the dataset's primary keys
	Delete InsertStrategy = "delete"
)

// TableFixture represents fixture data for a single table with its insert strategy
type TableFixture struct {
	TableName string
	Strategy  InsertStrategy
	Data      []map[string]any
}

// TestCase represents a single test case
type TestCase struct {
	Name           string
	Fixtures       []TableFixture                // テーブルごとのfixture情報
	Fixture        map[string][]map[string]any   // 後方互換性のため残す
	Parameters     map[string]any
	VerifyQuery    string                        // 検証用SELECTクエリ
	ExpectedResult []map[string]any
}

// TestSection represents a section within a test case
type TestSection struct {
	Type      string // "parameters", "expected", "fixtures"
	TableName string // Only used for CSV fixtures
	Strategy  InsertStrategy // Insert strategy for fixtures
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
				Fixtures:       make([]TableFixture, 0),
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

					if strings.HasPrefix(text, "parameters:") || text == "params:" || strings.HasPrefix(text, "input parameters:") {
						currentSection = TestSection{Type: "parameters"}
					} else if strings.HasPrefix(text, "expected:") || strings.HasPrefix(text, "expected results:") || strings.HasPrefix(text, "expected result:") || text == "results:" {
						currentSection = TestSection{Type: "expected"}
					} else if strings.HasPrefix(text, "verify query:") || strings.HasPrefix(text, "verification query:") {
						currentSection = TestSection{Type: "verify_query"}
					} else if strings.HasPrefix(text, "fixtures") {
						currentSection = TestSection{Type: "fixtures", Strategy: ClearInsert} // デフォルト戦略
						// Extract table name and strategy if present
						if i := strings.Index(text, ":"); i >= 0 {
							tableSpec := strings.TrimSpace(text[i+1:])
							if tableSpec != "" {
								tableName, strategy := parseTableNameAndStrategy(tableSpec)
								currentSection.TableName = tableName
								currentSection.Strategy = strategy
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

// findFirstEmphasis finds the first emphasis node (italic or bold) in a paragraph
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

	case "verify_query":
		// Verify Queryが既に設定されている場合はエラー
		if testCase.VerifyQuery != "" {
			return fmt.Errorf("duplicate verify query in test case %q", testCase.Name)
		}

		// SQLコードブロックの内容をそのまま設定
		testCase.VerifyQuery = strings.TrimSpace(string(content))

	case "fixtures":
		// CSVの場合はテーブル名が必要
		if format == "csv" {
			if section.TableName == "" {
				return fmt.Errorf("table name is required for CSV fixtures in test case %q", testCase.Name)
			}
		}

		// Parse fixtures data
		if format == "csv" {
			// CSVの場合は単一のテーブル
			rows, err := parseCSVData(content)
			if err != nil {
				return fmt.Errorf("failed to parse CSV fixtures in test case %q: %w", testCase.Name, err)
			}
			// 後方互換性のため既存のFixtureフィールドにも追加
			testCase.Fixture[section.TableName] = append(testCase.Fixture[section.TableName], rows...)
			
			// 新しいFixturesフィールドに追加
			addOrUpdateTableFixture(testCase, section.TableName, section.Strategy, rows)
		} else {
			// YAML/XMLの場合
			if section.TableName != "" {
				// テーブル名が指定されている場合（例: "Fixtures: users[insert]"）
				// データを直接そのテーブルに割り当て
				var rows []map[string]any
				if err := parseYAMLData(content, &rows); err != nil {
					return fmt.Errorf("failed to parse fixtures in test case %q: %w", testCase.Name, err)
				}
				
				// 後方互換性のため既存のFixtureフィールドにも追加
				testCase.Fixture[section.TableName] = append(testCase.Fixture[section.TableName], rows...)
				
				// 新しいFixturesフィールドに追加
				addOrUpdateTableFixture(testCase, section.TableName, section.Strategy, rows)
			} else {
				// テーブル名が含まれている構造化データ
				tableData, err := parseStructuredData(content, format)
				if err != nil {
					return fmt.Errorf("failed to parse fixtures in test case %q: %w", testCase.Name, err)
				}
				for tableName, rows := range tableData {
					// 後方互換性のため既存のFixtureフィールドにも追加
					testCase.Fixture[tableName] = append(testCase.Fixture[tableName], rows...)
					
					// 新しいFixturesフィールドに追加（デフォルト戦略を使用）
					addOrUpdateTableFixture(testCase, tableName, ClearInsert, rows)
				}
			}
		}
	}

	return nil
}
// parseTableNameAndStrategy parses table name and insert strategy from fixture specification
// Format: "table_name" or "table_name[strategy]"
func parseTableNameAndStrategy(spec string) (string, InsertStrategy) {
	// 正規表現でテーブル名と戦略を解析
	// 形式: "table_name" または "table_name[strategy]"
	re := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)(?:\[([^\]]+)\])?$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(spec))
	
	if len(matches) == 0 {
		// 無効な形式の場合はそのままテーブル名として扱い、デフォルト戦略を使用
		return spec, ClearInsert
	}
	
	tableName := matches[1]
	strategy := ClearInsert // デフォルト
	
	if len(matches) > 2 && matches[2] != "" {
		switch InsertStrategy(matches[2]) {
		case Insert:
			strategy = Insert
		case Upsert:
			strategy = Upsert
		case Delete:
			strategy = Delete
		case ClearInsert:
			strategy = ClearInsert
		default:
			// 無効な戦略の場合はデフォルトを使用し、テーブル名はそのまま
			strategy = ClearInsert
		}
	}
	
	return tableName, strategy
}

// addOrUpdateTableFixture adds or updates fixture data for a table
func addOrUpdateTableFixture(testCase *TestCase, tableName string, strategy InsertStrategy, rows []map[string]any) {
	// 既存のTableFixtureを探す
	for i := range testCase.Fixtures {
		if testCase.Fixtures[i].TableName == tableName {
			// 既存のテーブルが見つかった場合、データを追加
			testCase.Fixtures[i].Data = append(testCase.Fixtures[i].Data, rows...)
			return
		}
	}
	
	// 新しいTableFixtureを作成
	testCase.Fixtures = append(testCase.Fixtures, TableFixture{
		TableName: tableName,
		Strategy:  strategy,
		Data:      rows,
	})
}
