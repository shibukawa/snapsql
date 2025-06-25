package markdownparser

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotEqual(t, nil, parser)
	assert.NotEqual(t, nil, parser.markdown)
}

func TestParseFrontMatter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *FrontMatter
		expectError bool
	}{
		{
			name: "Valid front matter",
			input: `---
name: "user search"
dialect: "postgres"
---
# Title

## Overview
Content

## Parameters
` + "```yaml" + `
user_id: int
` + "```" + `

## SQL
` + "```sql" + `
SELECT * FROM users;
` + "```" + `

## Test Cases
### Case 1: Test
**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + ``,
			expected: &FrontMatter{
				Name:    "user search",
				Dialect: "postgres",
			},
			expectError: false,
		},
		{
			name: "Missing name field",
			input: `---
dialect: "postgres"
---
# Title

## Overview
Content`,
			expected:    nil,
			expectError: true,
		},
		{
			name: "Missing dialect field",
			input: `---
name: "user search"
---
# Title

## Overview
Content`,
			expected:    nil,
			expectError: true,
		},
		{
			name: "No front matter",
			input: `# Title

## Overview
Content`,
			expected:    nil,
			expectError: true,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, (*ParsedMarkdown)(nil), result)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, nil, result)
				assert.Equal(t, tt.expected.Name, result.FrontMatter.Name)
				assert.Equal(t, tt.expected.Dialect, result.FrontMatter.Dialect)
			}
		})
	}
}

func TestParseTitle(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name: "Simple title",
			input: `---
name: "test"
dialect: "postgres"
---

# User Search Query

## Overview
Content

## Parameters
` + "```yaml" + `
user_id: int
` + "```" + `

## SQL
` + "```sql" + `
SELECT * FROM users;
` + "```" + `

## Test Cases
### Case 1: Test
**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + ``,
			expected:    "User Search Query",
			expectError: false,
		},
		{
			name: "Title with additional text",
			input: `---
name: "test"
dialect: "postgres"
---

# User Search Query (User Search Query)

## Overview
Content

## Parameters
` + "```yaml" + `
user_id: int
` + "```" + `

## SQL
` + "```sql" + `
SELECT * FROM users;
` + "```" + `

## Test Cases
### Case 1: Test
**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + ``,
			expected:    "User Search Query (User Search Query)",
			expectError: false,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parser.Parse(reader)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result.Title)
			}
		})
	}
}

func TestParseSections(t *testing.T) {
	input := `---
name: "user search"
dialect: "postgres"
---

# User Search Query

## Overview
This is the overview section.
It can have multiple lines.

## Parameters
` + "```yaml" + `
user_id: int
name: str
` + "```" + `

## SQL
` + "```sql" + `
SELECT * FROM users WHERE id = /*= user_id */1;
` + "```" + `

## Test Cases
### Case 1: Basic test
**Fixture:**
` + "```yaml" + `
users:
  - id: 1
    name: "John"
` + "```" + `

## Custom Section
This section should be ignored during processing.

## Mock Data
` + "```yaml" + `
users:
  - id: 1
    name: "Test User"
` + "```" + `
`

	parser := NewParser()
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotEqual(t, nil, result)

	// Check that all expected sections are present
	expectedSections := []string{"overview", "parameters", "sql", "test", "custom", "mock"}
	for _, section := range expectedSections {
		_, exists := result.Sections[section]
		assert.True(t, exists, "Section %s should exist", section)
	}

	// Check section content
	assert.True(t, strings.Contains(result.Sections["overview"].Content, "This is the overview section"))
	assert.True(t, strings.Contains(result.Sections["parameters"].Content, "user_id: int"))
	assert.True(t, strings.Contains(result.Sections["sql"].Content, "SELECT * FROM users"))
	assert.True(t, strings.Contains(result.Sections["test"].Content, "Case 1: Basic test"))
}

func TestValidateRequiredSections(t *testing.T) {
	tests := []struct {
		name        string
		sections    map[string]Section
		expectError bool
	}{
		{
			name: "All required sections present",
			sections: map[string]Section{
				"overview":   {Heading: "overview", Content: "content"},
				"parameters": {Heading: "parameters", Content: "content"},
				"sql":        {Heading: "sql", Content: "content"},
				"test":       {Heading: "test", Content: "content"},
			},
			expectError: false,
		},
		{
			name: "Missing overview section",
			sections: map[string]Section{
				"parameters": {Heading: "parameters", Content: "content"},
				"sql":        {Heading: "sql", Content: "content"},
				"test":       {Heading: "test", Content: "content"},
			},
			expectError: true,
		},
		{
			name: "Missing multiple sections",
			sections: map[string]Section{
				"overview": {Heading: "overview", Content: "content"},
			},
			expectError: true,
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateRequiredSections(tt.sections)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractKeywordFromHeading(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input    string
		expected string
	}{
		{"Overview", "Overview"},
		{"Overview (Summary)", "Overview"},
		{"Parameters something else", "Parameters"},
		{"SQL Template", "SQL"},
		{"Test Cases", "Test"},
		{"", ""},
		{"123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.extractKeywordFromHeading(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSectionsWithDifferentHeadingLevels(t *testing.T) {
	input := `---
name: "user search"
dialect: "postgres"
---

# User Search Query

## Overview
This is the overview section with H2.

## Parameters
` + "```yaml" + `
user_id: int
name: str
` + "```" + `

## SQL
` + "```sql" + `
SELECT * FROM users WHERE id = /*= user_id */1;
` + "```" + `

## Test Cases
### Case 1: Basic test
**Fixture:**
` + "```yaml" + `
users:
  - id: 1
    name: "John"
` + "```" + `

#### Custom Section
This section should be captured as well.

## Mock Data
` + "```yaml" + `
users:
  - id: 1
    name: "Test User"
` + "```" + `
`

	parser := NewParser()
	reader := strings.NewReader(input)
	result, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotEqual(t, nil, result)

	// Check that all expected sections are present (only H2 level sections)
	expectedSections := []string{"overview", "parameters", "sql", "test", "mock"}
	for _, section := range expectedSections {
		_, exists := result.Sections[section]
		assert.True(t, exists, "Section %s should exist", section)
	}

	// Check section content
	assert.True(t, strings.Contains(result.Sections["overview"].Content, "This is the overview section with H2"))
	assert.True(t, strings.Contains(result.Sections["parameters"].Content, "user_id: int"))
	assert.True(t, strings.Contains(result.Sections["test"].Content, "### Case 1: Basic test"))
	assert.True(t, strings.Contains(result.Sections["test"].Content, "#### Custom Section"))
}

func TestIsKnownSection(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input    string
		expected bool
	}{
		{"Overview", true},
		{"Parameters", true},
		{"SQL", true},
		{"Test", true},
		{"Mock", true},
		{"Performance", true},
		{"Security", true},
		{"Change", true},
		{"Custom", false},
		{"Unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.isKnownSection(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFirstWord(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input    string
		expected string
	}{
		{"Custom Section", "Custom"},
		{"Implementation Notes", "Implementation"},
		{"Database-Schema Requirements", "DatabaseSchema"},
		{"123 Numbers", "123"},
		{"", ""},
		{"   Whitespace   ", "Whitespace"},
		{"Special!@#$%Characters", "SpecialCharacters"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.extractFirstWord(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompleteMarkdownParsing(t *testing.T) {
	completeMarkdown := `---
name: "user search"
dialect: "postgres"
---

# User Search Query

## Overview
Searches for active users based on various criteria with pagination support.

## Parameters
` + "```yaml" + `
user_id: int
filters:
  active: bool
  departments: [str]
pagination:
  limit: int
  offset: int
` + "```" + `

## SQL
` + "```sql" + `
SELECT id, name, email
FROM users
WHERE active = /*= filters.active */true
LIMIT /*= pagination.limit */10;
` + "```" + `

## Test Cases

### Case 1: Basic Search

**Fixture:**
` + "```yaml" + `
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    active: true
` + "```" + `

**Parameters:**
` + "```yaml" + `
user_id: 1
filters:
  active: true
  departments: ["engineering"]
pagination:
  limit: 10
  offset: 0
` + "```" + `

**Expected Result:**
` + "```yaml" + `
- id: 1
  name: "John Doe"
  email: "john@example.com"
` + "```" + `

## Mock Data
` + "```yaml" + `
users:
  - id: 1
    name: "Test User"
    email: "test@example.com"
    active: true
` + "```" + `
`

	parser := NewParser()
	reader := strings.NewReader(completeMarkdown)
	result, err := parser.Parse(reader)

	assert.NoError(t, err)
	assert.NotEqual(t, nil, result)

	// Check front matter
	assert.Equal(t, "user search", result.FrontMatter.Name)
	assert.Equal(t, "postgres", result.FrontMatter.Dialect)

	// Check title
	assert.Equal(t, "User Search Query", result.Title)

	// Check sections
	assert.True(t, len(result.Sections) >= 5)
	assert.True(t, strings.Contains(result.Sections["overview"].Content, "Searches for active users"))
	assert.True(t, strings.Contains(result.Sections["sql"].Content, "SELECT id, name, email"))

	// Test parameter parsing
	params, err := parser.ParseParameters(result.Sections["parameters"].Content)
	assert.NoError(t, err)
	assert.Equal(t, "int", params["user_id"])

	// Test test case parsing
	testCases, err := parser.ParseTestCases(result.Sections["test"].Content)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(testCases))
	if len(testCases) > 0 {
		assert.Equal(t, "Basic Search", testCases[0].Name)
	}
}
