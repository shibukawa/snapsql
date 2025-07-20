package query

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// TestCase represents a test case for query execution
type TestCase struct {
	Name           string                 // Test case name
	SQL            string                 // SQL template
	Params         map[string]interface{} // Parameters
	Options        QueryOptions           // Query options
	ExpectedSQL    string                 // Expected SQL after template processing
	ExpectedParams []interface{}          // Expected parameters
	MockColumns    []string               // Mock column names
	MockRows       [][]interface{}        // Mock rows to return
	ExpectedError  string                 // Expected error message (if any)
}

// TestIsDangerousQuery tests the dangerous query detection
func TestIsDangerousQuery(t *testing.T) {
	testCases := []struct {
		SQL      string
		Expected bool
	}{
		{
			SQL:      "SELECT * FROM users",
			Expected: false,
		},
		{
			SQL:      "SELECT * FROM users WHERE id = 1",
			Expected: false,
		},
		{
			SQL:      "DELETE FROM users",
			Expected: true,
		},
		{
			SQL:      "DELETE FROM users WHERE id = 1",
			Expected: false,
		},
		{
			SQL:      "UPDATE users SET active = false",
			Expected: true,
		},
		{
			SQL:      "UPDATE users SET active = false WHERE id = 1",
			Expected: false,
		},
		{
			SQL:      "  DELETE  FROM  users  ",
			Expected: true,
		},
		{
			SQL:      "delete from users",
			Expected: true,
		},
		{
			SQL:      "update users set name = 'test'",
			Expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.SQL, func(t *testing.T) {
			result := IsDangerousQuery(tc.SQL)
			if result != tc.Expected {
				t.Errorf("IsDangerousQuery(%q) = %v, expected %v", tc.SQL, result, tc.Expected)
			}
		})
	}
}

// TestExecutor tests the query executor with various test cases
func TestExecutor(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "Simple SELECT query",
			SQL: `SELECT id, name 
                  FROM users 
                  WHERE id = /*= user_id */123`,
			Params: map[string]interface{}{
				"user_id": 456,
			},
			ExpectedSQL:    "SELECT id, name FROM users WHERE id = $1",
			ExpectedParams: []interface{}{456},
			MockColumns:    []string{"id", "name"},
			MockRows: [][]interface{}{
				{456, "John Doe"},
			},
		},
		{
			Name: "Query with LIMIT option",
			SQL: `SELECT id, name 
                  FROM users`,
			Params: map[string]interface{}{},
			Options: QueryOptions{
				Limit: 10,
			},
			ExpectedSQL:    "SELECT id, name FROM users LIMIT 10",
			ExpectedParams: []interface{}{},
			MockColumns:    []string{"id", "name"},
			MockRows: [][]interface{}{
				{1, "User 1"},
				{2, "User 2"},
			},
		},
		{
			Name: "Query with EXPLAIN option",
			SQL: `SELECT id, name 
                  FROM users 
                  WHERE id = /*= user_id */123`,
			Params: map[string]interface{}{
				"user_id": 789,
			},
			Options: QueryOptions{
				Explain: true,
			},
			ExpectedSQL:    "EXPLAIN SELECT id, name FROM users WHERE id = $1",
			ExpectedParams: []interface{}{789},
			MockColumns:    []string{"QUERY PLAN"},
			MockRows: [][]interface{}{
				{"Seq Scan on users (cost=0.00..1.01 rows=1 width=36)"},
				{"  Filter: (id = 789)"},
			},
		},
		{
			Name: "Query with conditional block",
			SQL: `SELECT id, name 
                  FROM users 
                  /*# if include_filter */
                  WHERE active = /*= active */true
                  /*# end */`,
			Params: map[string]interface{}{
				"include_filter": true,
				"active":         false,
			},
			ExpectedSQL:    "SELECT id, name FROM users WHERE active = $1",
			ExpectedParams: []interface{}{false},
			MockColumns:    []string{"id", "name"},
			MockRows: [][]interface{}{
				{1, "Inactive User"},
			},
		},
		{
			Name: "Query with conditional block (not included)",
			SQL: `SELECT id, name 
                  FROM users 
                  /*# if include_filter */
                  WHERE active = /*= active */true
                  /*# end */`,
			Params: map[string]interface{}{
				"include_filter": false,
				"active":         false,
			},
			ExpectedSQL:    "SELECT id, name FROM users",
			ExpectedParams: []interface{}{},
			MockColumns:    []string{"id", "name"},
			MockRows: [][]interface{}{
				{1, "User 1"},
				{2, "User 2"},
			},
		},
		{
			Name: "Query with EXPLAIN ANALYZE option",
			SQL: `SELECT id, name 
                  FROM users 
                  WHERE id = /*= user_id */123`,
			Params: map[string]interface{}{
				"user_id": 789,
			},
			Options: QueryOptions{
				Explain:        true,
				ExplainAnalyze: true,
			},
			ExpectedSQL:    "EXPLAIN ANALYZE SELECT id, name FROM users WHERE id = $1",
			ExpectedParams: []interface{}{789},
			MockColumns:    []string{"QUERY PLAN"},
			MockRows: [][]interface{}{
				{"Seq Scan on users (cost=0.00..1.01 rows=1 width=36) (actual time=0.010..0.011 rows=1 loops=1)"},
				{"  Filter: (id = 789)"},
				{"Planning Time: 0.066 ms"},
				{"Execution Time: 0.025 ms"},
			},
		},
		{
			Name: "Query with LIMIT and OFFSET options",
			SQL: `SELECT id, name 
                  FROM users 
                  ORDER BY id`,
			Params: map[string]interface{}{},
			Options: QueryOptions{
				Limit:  5,
				Offset: 10,
			},
			ExpectedSQL:    "SELECT id, name FROM users ORDER BY id LIMIT 5 OFFSET 10",
			ExpectedParams: []interface{}{},
			MockColumns:    []string{"id", "name"},
			MockRows: [][]interface{}{
				{11, "User 11"},
				{12, "User 12"},
			},
		},
		{
			Name:           "Dangerous query without flag",
			SQL:            `DELETE FROM users`,
			Params:         map[string]interface{}{},
			ExpectedSQL:    "DELETE FROM users",
			ExpectedParams: []interface{}{},
			ExpectedError:  "dangerous query detected",
		},
		{
			Name:   "Dangerous query with flag",
			SQL:    `DELETE FROM users`,
			Params: map[string]interface{}{},
			Options: QueryOptions{
				ExecuteDangerousQuery: true,
			},
			ExpectedSQL:    "DELETE FROM users",
			ExpectedParams: []interface{}{},
			MockColumns:    []string{"affected_rows"},
			MockRows: [][]interface{}{
				{10}, // 10 rows affected
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Create mock database
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock: %v", err)
			}
			defer db.Close()

			// Create intermediate format from SQL template
			format, err := createIntermediateFormat(tc.SQL)
			if err != nil {
				t.Fatalf("Failed to create intermediate format: %v", err)
			}

			// Apply options to system directives
			if tc.Options.Explain {
				err = format.EnableSystemDirective("explain", true)
				if err != nil {
					t.Fatalf("Failed to enable explain: %v", err)
				}
				if tc.Options.ExplainAnalyze {
					err = format.SetSystemDirectiveProperty("explain", "analyze", true)
					if err != nil {
						t.Fatalf("Failed to set explain analyze: %v", err)
					}
				}
			}

			if tc.Options.Limit > 0 {
				err = format.EnableSystemDirective("limit", true)
				if err != nil {
					t.Fatalf("Failed to enable limit: %v", err)
				}
				err = format.SetSystemDirectiveProperty("limit", "value", tc.Options.Limit)
				if err != nil {
					t.Fatalf("Failed to set limit value: %v", err)
				}
			}

			if tc.Options.Offset > 0 {
				err = format.EnableSystemDirective("offset", true)
				if err != nil {
					t.Fatalf("Failed to enable offset: %v", err)
				}
				err = format.SetSystemDirectiveProperty("offset", "value", tc.Options.Offset)
				if err != nil {
					t.Fatalf("Failed to set offset value: %v", err)
				}
			}

			// Set up mock expectations
			if tc.ExpectedError == "" || tc.ExpectedError != "dangerous query detected" || tc.Options.ExecuteDangerousQuery {
				mockRows := sqlmock.NewRows(tc.MockColumns)
				for _, row := range tc.MockRows {
					mockRows.AddRow(convertToDriverValues(row)...)
				}

				// Set up mock expectation with SQL and parameters
				expectation := mock.ExpectQuery(regexp.QuoteMeta(tc.ExpectedSQL))
				if len(tc.ExpectedParams) > 0 {
					expectation = expectation.WithArgs(convertToDriverValues(tc.ExpectedParams)...)
				}
				expectation.WillReturnRows(mockRows)
			}

			// Create executor
			executor := NewExecutor(db)

			// Execute query
			result, err := executor.Execute(context.Background(), format, tc.Params, tc.Options)

			// Check for expected error
			if tc.ExpectedError != "" {
				if err == nil {
					t.Fatalf("Expected error containing %q, got nil", tc.ExpectedError)
				}
				if !strings.Contains(err.Error(), tc.ExpectedError) {
					t.Fatalf("Expected error containing %q, got %q", tc.ExpectedError, err.Error())
				}
				return
			}

			// Verify no error
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("Result is nil")
			}

			// Verify SQL and parameters
			if result.SQL != tc.ExpectedSQL {
				t.Errorf("Expected SQL %q, got %q", tc.ExpectedSQL, result.SQL)
			}
			if len(result.Parameters) != len(tc.ExpectedParams) {
				t.Errorf("Expected %d parameters, got %d", len(tc.ExpectedParams), len(result.Parameters))
			}

			// Verify columns
			if !equalStringSlices(tc.MockColumns, result.Columns) {
				t.Errorf("Expected columns %v, got %v", tc.MockColumns, result.Columns)
			}

			// Verify row count
			if result.Count != len(tc.MockRows) {
				t.Errorf("Expected %d rows, got %d", len(tc.MockRows), result.Count)
			}

			// Verify rows
			if len(result.Rows) != len(tc.MockRows) {
				t.Errorf("Expected %d rows, got %d", len(tc.MockRows), len(result.Rows))
			} else {
				for i, expectedRow := range tc.MockRows {
					if len(result.Rows[i]) != len(expectedRow) {
						t.Errorf("Row %d: Expected %d columns, got %d", i, len(expectedRow), len(result.Rows[i]))
					} else {
						for j, expectedValue := range expectedRow {
							if !equalValues(expectedValue, result.Rows[i][j]) {
								t.Errorf("Row %d, Col %d: Expected %v, got %v", i, j, expectedValue, result.Rows[i][j])
							}
						}
					}
				}
			}

			// Verify all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

// TestFormatterOutput tests the formatter output
func TestFormatterOutput(t *testing.T) {
	// Create a sample result
	result := &QueryResult{
		SQL:        "SELECT id, name FROM users",
		Parameters: []interface{}{1},
		Duration:   100 * time.Millisecond,
		Columns:    []string{"id", "name"},
		Rows: [][]interface{}{
			{1, "John Doe"},
			{2, "Jane Smith"},
		},
		Count: 2,
	}

	// Test JSON format
	t.Run("JSON Format", func(t *testing.T) {
		formatter := NewFormatter(FormatJSON)
		var buf strings.Builder
		err := formatter.Format(result, &buf)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Parse the JSON output
		var output map[string]interface{}
		err = json.Unmarshal([]byte(buf.String()), &output)
		if err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Verify the output
		if output["count"] != float64(2) {
			t.Errorf("Expected count 2, got %v", output["count"])
		}
		if output["duration"] != "100ms" {
			t.Errorf("Expected duration 100ms, got %v", output["duration"])
		}

		data, ok := output["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", output["data"])
		}
		if len(data) != 2 {
			t.Errorf("Expected 2 rows, got %d", len(data))
		}

		row1, ok := data[0].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected row to be a map, got %T", data[0])
		}
		if row1["id"] != float64(1) {
			t.Errorf("Expected id 1, got %v", row1["id"])
		}
		if row1["name"] != "John Doe" {
			t.Errorf("Expected name John Doe, got %v", row1["name"])
		}
	})

	// Test CSV format
	t.Run("CSV Format", func(t *testing.T) {
		formatter := NewFormatter(FormatCSV)
		var buf strings.Builder
		err := formatter.Format(result, &buf)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify the CSV output
		expected := "id,name\n1,John Doe\n2,Jane Smith\n"
		if buf.String() != expected {
			t.Errorf("Expected CSV:\n%s\nGot:\n%s", expected, buf.String())
		}
	})

	// Test Markdown format
	t.Run("Markdown Format", func(t *testing.T) {
		formatter := NewFormatter(FormatMarkdown)
		var buf strings.Builder
		err := formatter.Format(result, &buf)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify the Markdown output contains header and rows
		output := buf.String()
		if !strings.Contains(output, "| id | name |") {
			t.Errorf("Expected Markdown header, got:\n%s", output)
		}
		if !strings.Contains(output, "| 1 | John Doe |") {
			t.Errorf("Expected row with John Doe, got:\n%s", output)
		}
		if !strings.Contains(output, "| 2 | Jane Smith |") {
			t.Errorf("Expected row with Jane Smith, got:\n%s", output)
		}
	})

	// Test EXPLAIN format
	t.Run("EXPLAIN Format", func(t *testing.T) {
		explainResult := &QueryResult{
			ExplainPlan: "Seq Scan on users\n  Filter: (id = 1)\n",
		}

		formatter := NewFormatter(FormatTable)
		var buf strings.Builder
		err := formatter.FormatExplain(explainResult, &buf)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify the EXPLAIN output
		expected := "Seq Scan on users\n  Filter: (id = 1)\n\n"
		if buf.String() != expected {
			t.Errorf("Expected EXPLAIN:\n%s\nGot:\n%s", expected, buf.String())
		}
	})
}

// Helper functions

// createIntermediateFormat creates an intermediate format from SQL template
func createIntermediateFormat(sqlTemplate string) (*intermediate.IntermediateFormat, error) {
	// Tokenize SQL
	tokens, err := tokenizer.Tokenize(sqlTemplate)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Extract function definition
	functionDef, err := parser.NewFunctionDefinitionFromSQL(tokens)
	if err != nil {
		// Create default schema if extraction fails
		functionDef = &parser.FunctionDefinition{
			Name:         "query",
			FunctionName: "executeQuery",
			Parameters:   make(map[string]any),
		}
	}

	// Parse SQL
	ast, err := parser.RawParse(tokens, functionDef, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	// Create a minimal intermediate format
	format := &intermediate.IntermediateFormat{
		Source: intermediate.SourceInfo{
			File:    "test.snap.sql",
			Content: sqlTemplate,
			Hash:    calculateHash(sqlTemplate),
		},
		Instructions: []intermediate.Instruction{
			{
				Op:    intermediate.OpEmitLiteral,
				Pos:   []int{1, 1, 0},
				Value: sqlTemplate,
			},
		},
		Dependencies: intermediate.VariableDependencies{
			AllVariables:        extractVariableNames(functionDef),
			StructuralVariables: []string{},
			ParameterVariables:  extractVariableNames(functionDef),
			CacheKeyTemplate:    "static",
		},
	}

	// Add system directives
	format.AddSystemDirectives(intermediate.NewSystemDirectives())

	return format, nil
}

// calculateHash generates SHA-256 hash of content
func calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// extractVariableNames extracts variable names from function definition
func extractVariableNames(schema *parser.FunctionDefinition) []string {
	if schema == nil || len(schema.Parameters) == 0 {
		return []string{}
	}

	names := make([]string, 0, len(schema.Parameters))
	for name := range schema.Parameters {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// convertToDriverValues converts interface values to driver.Value
func convertToDriverValues(values []interface{}) []driver.Value {
	result := make([]driver.Value, len(values))
	for i, v := range values {
		result[i] = v
	}
	return result
}

// equalStringSlices compares two string slices
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// equalValues compares two values
func equalValues(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
