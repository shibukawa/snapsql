package markdownparser

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func TestParseBasic(t *testing.T) {
	t.Log("Running TestParseBasic with debug output enabled")

	input := `---
function_name: "get_user_data"
description: "Get user data"
---

# User Data Query

## Description

This query retrieves user data from the database.

## Parameters

` + "```yaml" + `
user_id: int
include_email: bool
` + "```" + `

## SQL

` + "```sql" + `
SELECT id, name, email
FROM users
WHERE id = /*= user_id */1
` + "```" + `

## Test Cases

### Test: Basic user data

**Parameters:**
` + "```yaml" + `
user_id: 1
include_email: true
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John"
  email: "john@example.com"
` + "```" + `
`

	doc, err := Parse(strings.NewReader(input))
	assert.NoError(t, err)
	assert.NotZero(t, doc)

	// Check metadata
	assert.Equal(t, "get_user_data", doc.Metadata["function_name"])
	assert.Equal(t, "Get user data", doc.Metadata["description"])

	// Check SQL
	assert.True(t, strings.Contains(doc.SQL, "SELECT id, name, email"))
	assert.True(t, strings.Contains(doc.SQL, "WHERE id = /*= user_id */1"))

	// Check test cases
	assert.Equal(t, 1, len(doc.TestCases))
	testCase := doc.TestCases[0]
	assert.Equal(t, "Test: Basic user data", testCase.Name)
	assert.Equal(t, 1, testCase.Parameters["user_id"])
	assert.Equal(t, true, testCase.Parameters["include_email"])
	assert.Equal(t, 1, len(testCase.ExpectedResult))
	assert.Equal(t, "John", testCase.ExpectedResult[0]["name"])
}

func TestParseAllowsEmptyExpectedResults(t *testing.T) {
	input := `# Empty Expected Results

## Description

Allow empty expected results blocks.

## SQL

` + "```sql" + `
SELECT 1;
` + "```" + `

## Test Cases

### No rows expected

**Expected Results:**
` + "```yaml" + `
[]
` + "```" + `
`

	doc, err := Parse(strings.NewReader(input))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(doc.TestCases))

	testCase := doc.TestCases[0]
	assert.Equal(t, "No rows expected", testCase.Name)
	assert.Equal(t, 0, len(testCase.ExpectedResult))
	assert.Equal(t, 1, len(testCase.ExpectedResults))
	assert.Equal(t, 0, len(testCase.ExpectedResults[0].Data))
}

func TestDuplicateSections(t *testing.T) {
	input := `---
function_name: "test_duplicates"
---

# Test Duplicates

## Description

Test handling of duplicate sections.

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

### Test: Duplicate Parameters

**Parameters:**
` + "```yaml" + `
user_id: 1
include_email: true
` + "```" + `

**Parameters:**
` + "```yaml" + `
user_id: 2
include_email: false
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `

### Test: Duplicate Expected Results

**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 2
  name: "Jane"
` + "```" + `

### Test: Multiple Fixtures

**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `

**Fixtures:**
` + "```yaml" + `
users:
  - id: 1
    name: "John"
` + "```" + `

**Fixtures:**
` + "```csv" + `
departments
1,Engineering
` + "```" + `
`

	_, err := Parse(strings.NewReader(input))
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "duplicate parameters"))
	assert.True(t, strings.Contains(err.Error(), "duplicate expected results"))
}

func TestMultipleFixtureFormats(t *testing.T) {
	input := `---
function_name: "test_multiple_formats"
---

# Test Multiple Formats

## Description

Test combining fixtures in different formats.

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

### Test: Combined Fixtures

**Parameters:**
` + "```yaml" + `
user_id: 1
include_roles: true
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John Doe"
  department: "Engineering"
  role: "Admin"
` + "```" + `

**Fixtures**
` + "```yaml" + `
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
` + "```" + `

**Fixtures: departments**
` + "```csv" + `
id,name
1,Engineering
2,Design
` + "```" + `

**Fixtures**
` + "```xml" + `
<dataset>
  <roles id="1" name="Admin" />
  <roles id="2" name="User" />
</dataset>
` + "```" + `
`

	doc, err := Parse(strings.NewReader(input))
	assert.NoError(t, err)
	assert.NotZero(t, doc)

	// Should have 1 test case
	assert.Equal(t, 1, len(doc.TestCases))

	testCase := doc.TestCases[0]
	assert.True(t, strings.Contains(testCase.Name, "Combined Fixtures"))

	// Check fixtures were combined from all formats
	assert.Equal(t, 3, len(testCase.Fixture), "Should have fixtures from three tables (users, departments, roles)")

	// Verify fixtures content
	var foundDepartments, foundUsers, foundRoles bool

	for tableName, rows := range testCase.Fixture {
		switch tableName {
		case "departments":
			foundDepartments = true

			assert.Equal(t, 2, len(rows), "departments should have 2 rows")

			for _, row := range rows {
				name, ok := row["name"].(string)
				assert.True(t, ok, "name should be string")
				assert.True(t, name == "Engineering" || name == "Design", "name should be Engineering or Design")
			}
		case "users":
			foundUsers = true

			assert.Equal(t, 1, len(rows), "users should have 1 row")
			assert.Equal(t, "John Doe", rows[0]["name"])
			assert.Equal(t, "john@example.com", rows[0]["email"])
		case "roles":
			foundRoles = true

			assert.Equal(t, 2, len(rows), "roles should have 2 rows")

			for _, row := range rows {
				name, ok := row["name"].(string)
				assert.True(t, ok, "name should be string")
				assert.True(t, name == "Admin" || name == "User", "name should be Admin or User")
			}
		default:
			t.Errorf("unexpected table name: %s", tableName)
		}
	}

	assert.True(t, foundDepartments, "Should have departments fixtures")
	assert.True(t, foundUsers, "Should have users fixtures")
	assert.True(t, foundRoles, "Should have roles fixtures")

	// Verify parameters (single definition)
	assert.Equal(t, 2, len(testCase.Parameters))
	assert.Equal(t, 1, testCase.Parameters["user_id"])
	assert.Equal(t, true, testCase.Parameters["include_roles"])

	// Verify expected results (single definition)
	assert.Equal(t, 1, len(testCase.ExpectedResult))
	result := testCase.ExpectedResult[0]
	assert.Equal(t, 1, result["id"])
	assert.Equal(t, "John Doe", result["name"])
	assert.Equal(t, "Engineering", result["department"])
	assert.Equal(t, "Admin", result["role"])
}

func TestInvalidFixturesFormat(t *testing.T) {
	// Test that YAML fixtures with table names are now supported
	input := `---
function_name: "test_valid_fixtures"
---

# Test Valid Fixtures Format

## Description

Test valid fixtures format with table name.

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

### Test: Valid Fixtures Format with Table Name

**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `

**Fixtures: users**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `
`

	result, err := Parse(strings.NewReader(input))
	assert.NoError(t, err)
	assert.True(t, result != nil)
	assert.True(t, len(result.TestCases) == 1)

	testCase := result.TestCases[0]
	assert.Equal(t, "Test: Valid Fixtures Format with Table Name", testCase.Name)
	_, exists := testCase.Fixture["users"]
	assert.True(t, exists)
	assert.True(t, len(testCase.Fixture["users"]) == 1)
	assert.Equal(t, "John", testCase.Fixture["users"][0]["name"])
}

func TestInvalidCombinations(t *testing.T) {
	// Test various invalid combinations in separate subtests
	testCases := []struct {
		name     string
		input    string
		errorMsg string
	}{
		{
			name: "Empty Parameters",
			input: `# Test Invalid Cases

## Description

Test invalid cases.

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

### Test: Empty Parameters
**Parameters:**
` + "```yaml" + `
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
` + "```" + `
`,
			errorMsg: "failed to parse test cases",
		},
		{
			name: "Invalid YAML in Parameters",
			input: `# Test Invalid Cases

## Description

Test invalid cases.

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

### Test: Invalid YAML
**Parameters:**
` + "```yaml" + `
user_id: [invalid
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
` + "```" + `
`,
			errorMsg: "failed to parse test cases",
		},
		{
			name: "Invalid CSV in Fixtures",
			input: `# Test Invalid Cases

## Description

Test invalid cases.

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

### Test: Invalid CSV
**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
` + "```" + `

**Fixtures:**
` + "```csv" + `
id,name
1,"unclosed quote
` + "```" + `
`,
			errorMsg: "failed to parse test cases",
		},
		{
			name: "Invalid XML in Expected Results",
			input: `# Test Invalid Cases

## Description

Test invalid cases.

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

### Test: Invalid XML
**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + `

**Expected Results:**
` + "```xml" + `
<dataset>
  <unclosed>
` + "```" + `
`,
			errorMsg: "failed to parse test cases",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fullInput := `---
function_name: "test_invalid"
---

# Test Invalid Cases

## Description

Test invalid cases.

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

` + tc.input

			_, err := Parse(strings.NewReader(fullInput))
			assert.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), tc.errorMsg),
				"Expected error containing %q, got %q", tc.errorMsg, err.Error())
		})
	}
}

func TestASTStructure(t *testing.T) {
	input := `### Test: Basic Test

**Parameters:**
` + "```yaml" + `
param1: value1
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- result: value1
` + "```" + `
`

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
	doc := md.Parser().Parse(text.NewReader([]byte(input)))

	// Walk through AST and print structure
	t.Log("AST Structure:")
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			t.Logf("Node type: %T, Kind: %v", n, n.Kind())

			if n.Kind() == ast.KindText {
				if textNode, ok := n.(*ast.Text); ok {
					t.Logf("  Text content: %q", string(textNode.Value([]byte(input))))
				}
			}

			if n.Kind() == ast.KindFencedCodeBlock {
				if cb, ok := n.(*ast.FencedCodeBlock); ok {
					var info string
					if cb.Info != nil {
						info = string(cb.Info.Value([]byte(input)))
					}

					t.Logf("  Code block info: %q", info)

					lines := cb.Lines()
					for i := range lines.Len() {
						line := lines.At(i)
						t.Logf("  Line %d: %q", i, string(line.Value([]byte(input))))
					}
				}
			}
		}

		return ast.WalkContinue, nil
	})
}
