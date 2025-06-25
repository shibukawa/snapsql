package markdownparser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestParseValidMarkdownFile(t *testing.T) {
	testDataPath := filepath.Join("..", "testdata", "markdown", "valid_query.md")

	file, err := os.Open(testDataPath)
	assert.NoError(t, err)
	defer file.Close()

	parser := NewParser()
	result, err := parser.Parse(file)

	assert.NoError(t, err)
	assert.NotEqual(t, nil, result)

	// Check front matter
	assert.Equal(t, "user search", result.FrontMatter.Name)
	assert.Equal(t, "postgres", result.FrontMatter.Dialect)

	// Check title
	assert.Equal(t, "User Search Query", result.Title)

	// Check required sections exist
	requiredSections := []string{"overview", "parameters", "sql", "test"}
	for _, section := range requiredSections {
		_, exists := result.Sections[section]
		assert.True(t, exists, "Required section %s should exist", section)
	}

	// Check optional sections exist
	optionalSections := []string{"mock", "performance", "security"}
	for _, section := range optionalSections {
		_, exists := result.Sections[section]
		assert.True(t, exists, "Optional section %s should exist", section)
	}

	// Test parameter parsing
	params, err := parser.ParseParameters(result.Sections["parameters"].Content)
	assert.NoError(t, err)
	assert.Equal(t, "int", params["user_id"])

	filters, exists := params["filters"]
	assert.True(t, exists)
	filtersMap, ok := filters.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "bool", filtersMap["active"])

	// Test test case parsing
	testCases, err := parser.ParseTestCases(result.Sections["test"].Content)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(testCases))

	// Check first test case
	assert.Equal(t, "Basic Search", testCases[0].Name)
	assert.NotEqual(t, nil, testCases[0].Fixture)
	assert.NotEqual(t, nil, testCases[0].Parameters)
	assert.NotEqual(t, nil, testCases[0].ExpectedResult)

	// Check fixture data
	users, exists := testCases[0].Fixture["users"]
	assert.True(t, exists)
	userList, ok := users.([]any)
	assert.True(t, ok)
	assert.Equal(t, 3, len(userList))

	// Check parameters data
	userID, exists := testCases[0].Parameters["user_id"]
	assert.True(t, exists)
	assert.Equal(t, 123, userID)

	// Check expected result
	expectedResult, ok := testCases[0].ExpectedResult.([]any)
	assert.True(t, ok)
	assert.Equal(t, 2, len(expectedResult))
}

func TestParseInvalidFrontMatterFile(t *testing.T) {
	testDataPath := filepath.Join("..", "testdata", "markdown", "invalid_frontmatter.md")

	file, err := os.Open(testDataPath)
	assert.NoError(t, err)
	defer file.Close()

	parser := NewParser()
	result, err := parser.Parse(file)

	assert.Error(t, err)
	assert.Equal(t, (*ParsedMarkdown)(nil), result)
	assert.Contains(t, err.Error(), "front matter")
}

func TestParseMissingSectionsFile(t *testing.T) {
	testDataPath := filepath.Join("..", "testdata", "markdown", "missing_sections.md")

	file, err := os.Open(testDataPath)
	assert.NoError(t, err)
	defer file.Close()

	parser := NewParser()
	result, err := parser.Parse(file)

	assert.Error(t, err)
	assert.Equal(t, (*ParsedMarkdown)(nil), result)
	assert.Contains(t, err.Error(), "missing required section")
}

func TestParseParametersFromRealFile(t *testing.T) {
	testDataPath := filepath.Join("..", "testdata", "markdown", "valid_query.md")

	file, err := os.Open(testDataPath)
	assert.NoError(t, err)
	defer file.Close()

	parser := NewParser()
	result, err := parser.Parse(file)
	assert.NoError(t, err)

	params, err := parser.ParseParameters(result.Sections["parameters"].Content)
	assert.NoError(t, err)

	// Check complex parameter structure
	assert.Equal(t, "int", params["user_id"])
	assert.Equal(t, "str", params["sort_by"])
	assert.Equal(t, "bool", params["include_email"])
	assert.Equal(t, "str", params["table_suffix"])

	// Check nested structures
	filters, exists := params["filters"]
	assert.True(t, exists)
	filtersMap, ok := filters.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "bool", filtersMap["active"])

	departments, exists := filtersMap["departments"]
	assert.True(t, exists)
	departmentList, ok := departments.([]any)
	assert.True(t, ok)
	assert.Equal(t, []any{"str"}, departmentList)

	assert.Equal(t, "str", filtersMap["name_pattern"])

	pagination, exists := params["pagination"]
	assert.True(t, exists)
	paginationMap, ok := pagination.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "int", paginationMap["limit"])
	assert.Equal(t, "int", paginationMap["offset"])
}

func TestParseTestCasesFromRealFile(t *testing.T) {
	testDataPath := filepath.Join("..", "testdata", "markdown", "valid_query.md")

	file, err := os.Open(testDataPath)
	assert.NoError(t, err)
	defer file.Close()

	parser := NewParser()
	result, err := parser.Parse(file)
	assert.NoError(t, err)

	testCases, err := parser.ParseTestCases(result.Sections["test"].Content)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(testCases))

	// Test Case 1: Basic Search
	case1 := testCases[0]
	assert.Equal(t, "Basic Search", case1.Name)

	// Check fixture
	users, exists := case1.Fixture["users"]
	assert.True(t, exists)
	userList, ok := users.([]any)
	assert.True(t, ok)
	assert.Equal(t, 3, len(userList))

	// Check first user in fixture
	firstUser, ok := userList[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, 1, firstUser["id"])
	assert.Equal(t, "John Doe", firstUser["name"])
	assert.Equal(t, "john@example.com", firstUser["email"])
	assert.Equal(t, "engineering", firstUser["department"])
	assert.Equal(t, true, firstUser["active"])

	// Check parameters
	assert.Equal(t, 123, case1.Parameters["user_id"])
	assert.Equal(t, "name", case1.Parameters["sort_by"])
	assert.Equal(t, false, case1.Parameters["include_email"])
	assert.Equal(t, "test", case1.Parameters["table_suffix"])

	// Check filters in parameters
	filters, exists := case1.Parameters["filters"]
	assert.True(t, exists)
	filtersMap, ok := filters.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, true, filtersMap["active"])
	assert.Equal(t, nil, filtersMap["name_pattern"])

	departments, exists := filtersMap["departments"]
	assert.True(t, exists)
	departmentList, ok := departments.([]any)
	assert.True(t, ok)
	assert.Equal(t, 2, len(departmentList))
	assert.Equal(t, "engineering", departmentList[0])
	assert.Equal(t, "design", departmentList[1])

	// Check expected result
	expectedResult, ok := case1.ExpectedResult.([]any)
	assert.True(t, ok)
	assert.Equal(t, 2, len(expectedResult))

	// Test Case 2: Full Options with Email
	case2 := testCases[1]
	assert.Equal(t, "Full Options with Email", case2.Name)

	// Check that case2 has different fixture data
	users2, exists := case2.Fixture["users"]
	assert.True(t, exists)
	userList2, ok := users2.([]any)
	assert.True(t, ok)
	assert.Equal(t, 2, len(userList2))

	// Check parameters for case2
	assert.Equal(t, 456, case2.Parameters["user_id"])
	assert.Equal(t, true, case2.Parameters["include_email"])
	assert.Equal(t, "created_at DESC", case2.Parameters["sort_by"])
}

func TestErrorHandlingWithRealFiles(t *testing.T) {
	// Test with non-existent file
	_, err := os.Open("non_existent_file.md")
	assert.Error(t, err)

	// Test with directory instead of file
	_, err = os.Open("..")
	assert.NoError(t, err) // Opening directory succeeds, but parsing should fail
}
