package markdownparser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	snapsql "github.com/shibukawa/snapsql"
	"github.com/yuin/goldmark/ast"
)

// InsertStrategy represents the strategy for inserting fixture data
type InsertStrategy string

const (
	// ClearInsert truncates the table then inserts data (default)
	ClearInsert InsertStrategy = "clear-insert"
	// Upsert inserts data into the table, or updates if the row already exists
	Upsert InsertStrategy = "upsert"
	// Delete deletes rows in the table that match the dataset's primary keys
	Delete InsertStrategy = "delete"
)

// TableFixture represents fixture data for a single table with its insert strategy
type TableFixture struct {
	TableName    string
	Strategy     InsertStrategy
	Data         []map[string]any
	ExternalFile string // when fixture rows are provided via external YAML/JSON link
	Line         int    // Source line of the fixture block
}

// ExpectedResultSpec represents expected result for a table with strategy and data
type ExpectedResultSpec struct {
	TableName    string
	Strategy     string           // "all", "pk-match", "pk-exists", "pk-not-exists"
	Data         []map[string]any // 値比較特殊指定（[null],[notnull],[any],[regexp,...]）含む
	ExternalFile string           // 外部ファイル参照時のパス
}

// TestCase represents a single test case
type TestCase struct {
	Name               string
	SQL                string
	Fixtures           []TableFixture              // テーブルごとのfixture情報
	Fixture            map[string][]map[string]any // 後方互換性のため残す
	Parameters         map[string]any
	HasParameters      bool
	VerifyQuery        string               // 検証用SELECTクエリ
	ExpectedResult     []map[string]any     // 従来型（無名配列）
	ExpectedResults    []ExpectedResultSpec // 新型（テーブル名・戦略付き）
	ExpectedError      *string              // 期待されるエラータイプ（normalized form）
	SourceFile         string               // 元となるMarkdownファイルのパス
	Line               int                  // 見出し行番号（1-origin）
	PreparedSQL        string               // 方言・条件適用後に評価されたSQL
	SQLArgs            []any                // PreparedSQLに対応するパラメータ
	ResultOrdered      bool
	SlowQueryThreshold time.Duration
}

// TestSection represents a section within a test case
type TestSection struct {
	Type      string         // "parameters", "expected", "fixtures"
	TableName string         // Only used for CSV fixtures
	Strategy  InsertStrategy // Insert strategy for fixtures
}

// parseTestCasesFromAST parses test cases from AST nodes
func parseTestCasesFromAST(nodes []ast.Node, content []byte, mapper *indexToLine) ([]TestCase, error) {
	var (
		testCases       []TestCase
		currentTestCase *TestCase
		errors          []error
		currentSection  TestSection
	)

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.Heading:
			// Save previous test case if exists
			if currentTestCase != nil {
				err := validateTestCase(currentTestCase)
				if err != nil {
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
			if lines := n.Lines(); lines != nil && lines.Len() > 0 {
				currentTestCase.Line = mapper.lineFor(lines.At(0).Start)
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
					} else if strings.HasPrefix(text, "expected error:") {
						// Extract error type from the same paragraph
						fullText := extractTextFromNode(n, content)
						if idx := strings.Index(strings.ToLower(fullText), "expected error:"); idx >= 0 {
							errorText := strings.TrimSpace(fullText[idx+len("expected error:"):])
							if errorText != "" {
								parsedError, err := ParseExpectedError(errorText)
								if err != nil {
									errors = append(errors, fmt.Errorf("in test case %q: %w", currentTestCase.Name, err))
								} else {
									currentTestCase.ExpectedError = parsedError
								}
							}
						}

						currentSection = TestSection{Type: "expected_error"}
					} else if strings.HasPrefix(text, "expected:") || strings.HasPrefix(text, "expected results:") || strings.HasPrefix(text, "expected result:") || text == "results:" {
						currentSection = TestSection{Type: "expected"}
						// Allow table-qualified expected results like: "Expected Results: users[pk-match]"
						if i := strings.Index(text, ":"); i >= 0 {
							if spec := strings.TrimSpace(text[i+1:]); spec != "" {
								currentSection.TableName = spec
							}
						}
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
				var info string
				if n.Info != nil {
					info = strings.ToLower(strings.TrimSpace(string(n.Info.Value(content))))
				}

				var codeContent strings.Builder

				lines := n.Lines()
				for i := range lines.Len() {
					line := lines.At(i)
					codeContent.Write(line.Value(content))

					if i < lines.Len()-1 {
						codeContent.WriteString("\n")
					}
				}

				sectionLine := -1
				if lines != nil && lines.Len() > 0 {
					sectionLine = mapper.lineFor(lines.At(0).Start)
				}

				err := processTestSection(currentTestCase, currentSection, info, []byte(codeContent.String()), sectionLine)
				if err != nil {
					errors = append(errors, fmt.Errorf("in test case %q: %w", currentTestCase.Name, err))
				}

				// Reset section after processing
				currentSection = TestSection{}
			}
		}
	}

	// Handle last test case
	if currentTestCase != nil {
		err := validateTestCase(currentTestCase)
		if err != nil {
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

		return nil, fmt.Errorf("%w: %s", snapsql.ErrFailedToParse, errMsg.String())
	}

	return testCases, nil
}

// findFirstEmphasis finds the first emphasis node (italic or bold) in a paragraph
func findFirstEmphasis(paragraph *ast.Paragraph) *ast.Emphasis {
	var emphasis *ast.Emphasis

	ast.Walk(paragraph, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == ast.KindEmphasis {
			if emp, ok := n.(*ast.Emphasis); ok {
				emphasis = emp
			}

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
			if textNode, ok := n.(*ast.Text); ok {
				text.Write(textNode.Value(content))
			}
		}

		return ast.WalkContinue, nil
	})

	return text.String()
}

// validateTestCase validates a test case for required sections and format
func validateTestCase(testCase *TestCase) error {
	// ExpectedError and ExpectedResults are mutually exclusive
	hasResults := len(testCase.ExpectedResult) > 0 || len(testCase.ExpectedResults) > 0
	hasError := testCase.ExpectedError != nil

	if hasResults && hasError {
		return fmt.Errorf("%w: test case %q", ErrConflictingExpectations, testCase.Name)
	}

	// Either Expected Results or Expected Error must be specified
	if !hasResults && !hasError {
		return fmt.Errorf("%w: %q must specify either Expected Results or Expected Error", snapsql.ErrTestCaseMissingData, testCase.Name)
	}

	return nil
}

// processTestSection processes a section of a test case
func processTestSection(testCase *TestCase, section TestSection, format string, content []byte, line int) error {
	switch section.Type {
	case "parameters", "params":
		if testCase.HasParameters {
			return fmt.Errorf("%w in test case %q", ErrDuplicateParameters, testCase.Name)
		}

		params, err := parseParameters(content)
		if err != nil {
			return fmt.Errorf("failed to parse parameters in test case %q: %w", testCase.Name, err)
		}

		testCase.Parameters = params
		testCase.HasParameters = true

	case "expected", "expected results", "results":
		// Check for conflict with ExpectedError
		if testCase.ExpectedError != nil {
			return fmt.Errorf("%w: test case %q", ErrConflictingExpectations, testCase.Name)
		}

		if len(testCase.ExpectedResult) > 0 || len(testCase.ExpectedResults) > 0 {
			return fmt.Errorf("%w in test case %q", ErrDuplicateExpectedResults, testCase.Name)
		}

		// Extract tableName and table-level strategy from section.TableName if provided
		tableName := ""
		strategy := "all"

		if section.TableName != "" {
			re := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)(?:\[([^\]]+)\])?$`)

			matches := re.FindStringSubmatch(section.TableName)
			if len(matches) > 0 {
				tableName = matches[1]

				if len(matches) > 2 && matches[2] != "" {
					strategy = matches[2]
				}
			}
		}

		contentStr := strings.TrimSpace(string(content))

		var results []map[string]any

		externalFile := ""

		if strings.HasPrefix(contentStr, "[") && strings.Contains(contentStr, "](") {
			re := regexp.MustCompile(`\[.*?\]\((.*?)\)`)

			matches := re.FindStringSubmatch(contentStr)
			if len(matches) == 2 {
				externalFile = matches[1]
			} else {
				return fmt.Errorf("%w in test case %q", ErrInvalidExpectedResultsExternalLinkFormat, testCase.Name)
			}
		} else {
			var err error

			results, err = parseExpectedResults(content)
			if err != nil {
				return fmt.Errorf("failed to parse expected results in test case %q: %w", testCase.Name, err)
			}
		}

		testCase.ExpectedResults = append(testCase.ExpectedResults, ExpectedResultSpec{
			TableName:    tableName,
			Strategy:     strategy,
			Data:         results,
			ExternalFile: externalFile,
		})

		if tableName == "" {
			testCase.ExpectedResult = results
		}

	case "verify_query":
		if testCase.VerifyQuery != "" {
			return fmt.Errorf("%w: %q", snapsql.ErrDuplicateVerifyQuery, testCase.Name)
		}

		testCase.VerifyQuery = strings.TrimSpace(string(content))

	case "fixtures":
		if format == "csv" {
			if section.TableName == "" {
				return fmt.Errorf("%w: %q", snapsql.ErrTableNameRequired, testCase.Name)
			}
		}

		if format == "csv" {
			rows, err := parseCSVData(content)
			if err != nil {
				return fmt.Errorf("failed to parse CSV fixtures in test case %q: %w", testCase.Name, err)
			}

			testCase.Fixture[section.TableName] = append(testCase.Fixture[section.TableName], rows...)
			addOrUpdateTableFixture(testCase, section.TableName, section.Strategy, rows, line)
		} else {
			// YAML/XML
			if section.TableName != "" {
				contentStr := strings.TrimSpace(string(content))
				if strings.HasPrefix(contentStr, "[") && strings.Contains(contentStr, "](") {
					re := regexp.MustCompile(`\[.*?\]\((.*?)\)`)

					matches := re.FindStringSubmatch(contentStr)
					if len(matches) == 2 {
						addOrUpdateTableFixture(testCase, section.TableName, section.Strategy, nil, line)
						last := len(testCase.Fixtures) - 1
						testCase.Fixtures[last].ExternalFile = matches[1]
					} else {
						return fmt.Errorf("%w in test case %q", ErrInvalidFixturesExternalLinkFormat, testCase.Name)
					}
				} else {
					var rows []map[string]any
					if err := parseYAMLData(content, &rows); err != nil {
						return fmt.Errorf("failed to parse fixtures in test case %q: %w", testCase.Name, err)
					}

					testCase.Fixture[section.TableName] = append(testCase.Fixture[section.TableName], rows...)
					addOrUpdateTableFixture(testCase, section.TableName, section.Strategy, rows, line)
				}
			} else {
				entries, err := parseStructuredData(content, format)
				if err != nil {
					return fmt.Errorf("failed to parse fixtures in test case %q: %w", testCase.Name, err)
				}

				strategy := section.Strategy
				if strategy == "" {
					strategy = ClearInsert
				}

				for _, entry := range entries {
					testCase.Fixture[entry.Name] = append(testCase.Fixture[entry.Name], entry.Rows...)

					switch strategy {
					case Upsert:
						addOrUpdateTableFixture(testCase, entry.Name, Upsert, entry.Rows, line)
					case Delete:
						addOrUpdateTableFixture(testCase, entry.Name, Delete, entry.Rows, line)
					default:
						addOrUpdateTableFixture(testCase, entry.Name, ClearInsert, entry.Rows, line)
					}
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
		case Upsert:
			strategy = Upsert
		case Delete:
			strategy = Delete
		case ClearInsert:
			strategy = ClearInsert
		default:
			strategy = ClearInsert
		}
	}

	return tableName, strategy
}

// addOrUpdateTableFixture adds or updates fixture data for a table
func addOrUpdateTableFixture(testCase *TestCase, tableName string, strategy InsertStrategy, rows []map[string]any, line int) {
	// さらに仕様変更: clear-insert など同一戦略が複数回出現してもブロック境界を保持したい。
	// => すべて常に新規追加し順序を厳密維持。マージは行わない。
	testCase.Fixtures = append(testCase.Fixtures, TableFixture{TableName: tableName, Strategy: strategy, Data: rows, Line: line})
}
