package markdownparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTableNameAndStrategy(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedTable    string
		expectedStrategy InsertStrategy
	}{
		{
			name:             "Table name only",
			input:            "users",
			expectedTable:    "users",
			expectedStrategy: ClearInsert,
		},
		{
			name:             "Table with clear-insert strategy",
			input:            "users[clear-insert]",
			expectedTable:    "users",
			expectedStrategy: ClearInsert,
		},
		{
			name:             "Table with upsert strategy",
			input:            "users[upsert]",
			expectedTable:    "users",
			expectedStrategy: Upsert,
		},
		{
			name:             "Table with delete strategy",
			input:            "users[delete]",
			expectedTable:    "users",
			expectedStrategy: Delete,
		},
		{
			name:             "Table with invalid strategy",
			input:            "users[invalid]",
			expectedTable:    "users",
			expectedStrategy: ClearInsert,
		},
		{
			name:             "Invalid format fallback",
			input:            "users@insert",
			expectedTable:    "users@insert",
			expectedStrategy: ClearInsert,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, strategy := parseTableNameAndStrategy(tt.input)
			assert.Equal(t, tt.expectedTable, table)
			assert.Equal(t, tt.expectedStrategy, strategy)
		})
	}
}

func TestFixtureWithoutStrategy(t *testing.T) {
	input := `---
function_name: "test_default_strategy"
---

# Test Default Strategy

## Description

Test default insert strategy for fixtures.

## SQL

` + "```sql" + `
SELECT * FROM users;
` + "```" + `

## Test Cases

### Test: Default Strategy

**Parameters:**
` + "```yaml" + `
limit: 10
` + "```" + `

**Fixtures: users**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `
`

	doc, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Equal(t, 1, len(doc.TestCases))
	testCase := doc.TestCases[0]

	// Check that fixture uses default strategy
	assert.Equal(t, 1, len(testCase.Fixtures))
	fixture := testCase.Fixtures[0]
	assert.Equal(t, "users", fixture.TableName)
	assert.Equal(t, ClearInsert, fixture.Strategy) // Default strategy
	assert.Equal(t, 1, len(fixture.Data))
}

func TestMultipleFixturesForSameTable(t *testing.T) {
	input := `---
function_name: "test_multiple_fixtures"
---

# Test Multiple Fixtures (separate blocks maintained)

## Description

Test multiple fixture blocks for the same table.

## SQL

` + "```sql" + `
SELECT * FROM users;
` + "```" + `

## Test Cases

### Test: Multiple Fixtures

**Parameters:**
` + "```yaml" + `
limit: 10
` + "```" + `

**Fixtures: users[upsert]**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `

**Fixtures: users[upsert]**
` + "```yaml" + `
- id: 2
  name: "Jane"
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John"
- id: 2
  name: "Jane"
` + "```" + `
`

	doc, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Equal(t, 1, len(doc.TestCases))
	testCase := doc.TestCases[0]

	// New spec: Each fixture block remains separate (even same strategy & table)
	assert.Equal(t, 2, len(testCase.Fixtures))
	first := testCase.Fixtures[0]
	second := testCase.Fixtures[1]

	assert.Equal(t, "users", first.TableName)
	assert.Equal(t, Upsert, first.Strategy)
	assert.Equal(t, 1, len(first.Data))
	assert.Equal(t, "John", first.Data[0]["name"])
	assert.Equal(t, "users", second.TableName)
	assert.Equal(t, Upsert, second.Strategy)
	assert.Equal(t, 1, len(second.Data))
	assert.Equal(t, "Jane", second.Data[0]["name"])
}
