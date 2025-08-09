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
			name:             "Table with insert strategy",
			input:            "users[insert]",
			expectedTable:    "users",
			expectedStrategy: Insert,
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
			name:             "Complex table name",
			input:            "user_profiles[insert]",
			expectedTable:    "user_profiles",
			expectedStrategy: Insert,
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

func TestFixtureWithInsertStrategy(t *testing.T) {
	input := `---
function_name: "test_insert_strategies"
---

# Test Insert Strategies

## Description

Test different insert strategies for fixtures.

## SQL

` + "```sql" + `
SELECT * FROM users;
` + "```" + `

## Test Cases

### Test: Different Insert Strategies

**Parameters:**
` + "```yaml" + `
limit: 10
` + "```" + `

**Fixtures: users[insert]**
` + "```yaml" + `
- id: 1
  name: "John"
  email: "john@example.com"
- id: 2
  name: "Jane"
  email: "jane@example.com"
` + "```" + `

**Fixtures: profiles[upsert]**
` + "```yaml" + `
- user_id: 1
  bio: "Software Engineer"
- user_id: 2
  bio: "Designer"
` + "```" + `

**Fixtures: logs[clear-insert]**
` + "```yaml" + `
- id: 1
  message: "User logged in"
  user_id: 1
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: "John"
  email: "john@example.com"
- id: 2
  name: "Jane"
  email: "jane@example.com"
` + "```" + `
`

	doc, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Equal(t, 1, len(doc.TestCases))
	testCase := doc.TestCases[0]

	// Check that fixtures were parsed with correct strategies
	assert.Equal(t, 3, len(testCase.Fixtures))

	// Find each fixture by table name
	var usersFixture, profilesFixture, logsFixture *TableFixture

	for i := range testCase.Fixtures {
		switch testCase.Fixtures[i].TableName {
		case "users":
			usersFixture = &testCase.Fixtures[i]
		case "profiles":
			profilesFixture = &testCase.Fixtures[i]
		case "logs":
			logsFixture = &testCase.Fixtures[i]
		}
	}

	// Verify users fixture with insert strategy
	require.NotNil(t, usersFixture)
	assert.Equal(t, "users", usersFixture.TableName)
	assert.Equal(t, Insert, usersFixture.Strategy)
	assert.Equal(t, 2, len(usersFixture.Data))
	assert.Equal(t, "John", usersFixture.Data[0]["name"])
	assert.Equal(t, "Jane", usersFixture.Data[1]["name"])

	// Verify profiles fixture with upsert strategy
	require.NotNil(t, profilesFixture)
	assert.Equal(t, "profiles", profilesFixture.TableName)
	assert.Equal(t, Upsert, profilesFixture.Strategy)
	assert.Equal(t, 2, len(profilesFixture.Data))

	// Verify logs fixture with clear-insert strategy
	require.NotNil(t, logsFixture)
	assert.Equal(t, "logs", logsFixture.TableName)
	assert.Equal(t, ClearInsert, logsFixture.Strategy)
	assert.Equal(t, 1, len(logsFixture.Data))

	// Verify backward compatibility - old Fixture field should still work
	assert.Equal(t, 3, len(testCase.Fixture))
	assert.Equal(t, 2, len(testCase.Fixture["users"]))
	assert.Equal(t, 2, len(testCase.Fixture["profiles"]))
	assert.Equal(t, 1, len(testCase.Fixture["logs"]))
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

# Test Multiple Fixtures

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

**Fixtures: users[insert]**
` + "```yaml" + `
- id: 1
  name: "John"
` + "```" + `

**Fixtures: users[insert]**
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

	// Check that multiple fixtures for the same table are combined
	assert.Equal(t, 1, len(testCase.Fixtures))
	fixture := testCase.Fixtures[0]
	assert.Equal(t, "users", fixture.TableName)
	assert.Equal(t, Insert, fixture.Strategy)
	assert.Equal(t, 2, len(fixture.Data)) // Both records should be combined
	assert.Equal(t, "John", fixture.Data[0]["name"])
	assert.Equal(t, "Jane", fixture.Data[1]["name"])
}
