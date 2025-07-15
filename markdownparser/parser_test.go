package markdownparser

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestParseBasic(t *testing.T) {
	input := `---
name: "user_query"
function_name: "getUserData"
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
SELECT id, name, email FROM users WHERE id = /*= user_id */
` + "```"

	reader := strings.NewReader(input)
	doc, err := Parse(reader)

	assert.NoError(t, err)
	assert.NotEqual(t, nil, doc)

	// Check metadata
	assert.Equal(t, "user_query", doc.Metadata["name"])
	assert.Equal(t, "getUserData", doc.Metadata["function_name"])
	assert.Equal(t, "Get user data", doc.Metadata["description"])

	// Check parameter block
	assert.Equal(t, "user_id: int\ninclude_email: bool", doc.ParameterBlock)

	// Check SQL
	assert.Equal(t, "SELECT id, name, email FROM users WHERE id = /*= user_id */", doc.SQL)
}

func TestParseWithGeneratedFunctionName(t *testing.T) {
	input := `---
name: "user_query"
---

# Get User Data

## Description

This query retrieves user data.

## SQL

` + "```sql" + `
SELECT * FROM users;
` + "```"

	reader := strings.NewReader(input)
	doc, err := Parse(reader)

	assert.NoError(t, err)
	assert.NotEqual(t, nil, doc)

	// Check that function_name was generated from title
	assert.Equal(t, "getUserData", doc.Metadata["function_name"])
}

func TestParseWithTestCases(t *testing.T) {
	input := `---
name: "user_query"
---

# User Query

## Description

Test query with test cases.

## SQL

` + "```sql" + `
SELECT * FROM users;
` + "```" + `

## Test

### Case 1: Basic test

**Parameters:**
` + "```yaml" + `
user_id: 1
` + "```" + `

**Expected Result:**
` + "```yaml" + `
count: 1
` + "```"

	reader := strings.NewReader(input)
	doc, err := Parse(reader)

	assert.NoError(t, err)
	assert.NotEqual(t, nil, doc)

	// Check test cases
	assert.Equal(t, 1, len(doc.TestCases))
	assert.Equal(t, "Basic test", doc.TestCases[0].Name)
	assert.Equal(t, 1, doc.TestCases[0].Parameters["user_id"])
	assert.Equal(t, 1, doc.TestCases[0].ExpectedResult.(map[string]any)["count"])
}

func TestParseMissingRequiredSection(t *testing.T) {
	input := `---
name: "user_query"
---

# User Query

## Description

This query is missing SQL section.`

	reader := strings.NewReader(input)
	_, err := Parse(reader)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sql")
}

func TestParseEmptyFrontMatter(t *testing.T) {
	input := `# User Query

## Description

This query has no front matter.

## SQL

` + "```sql" + `
SELECT * FROM users;
` + "```"

	reader := strings.NewReader(input)
	doc, err := Parse(reader)

	assert.NoError(t, err)
	assert.NotEqual(t, nil, doc)

	// Check that function_name was generated from title
	assert.Equal(t, "userQuery", doc.Metadata["function_name"])
}

func TestGenerateFunctionNameFromTitle(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Get User Data", "getUserData"},
		{"Search Users By Email", "searchUsersByEmail"},
		{"user-profile_query", "userprofilequery"},
		{"", "query"},
		{"Single", "single"},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			result := generateFunctionNameFromTitle(test.title)
			assert.Equal(t, test.expected, result)
		})
	}
}
