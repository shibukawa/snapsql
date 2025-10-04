package markdownparser

import (
	"bytes"
	"testing"
)

func TestParseExpectedErrorFromMarkdown(t *testing.T) {
	tests := []struct {
		name          string
		markdown      string
		expectError   bool
		expectedType  string
		expectedCount int
	}{
		{
			name: "parse expected error with space notation",
			markdown: `## Description

Test error handling

## SQL

` + "```sql" + `
INSERT INTO users (email) VALUES (/*= email */'test')
` + "```" + `

## Test Cases

### Test Unique Violation

**Fixtures:**
` + "```yaml" + `
users:
  - id: 1
    email: test@example.com
` + "```" + `

**Parameters:**
` + "```yaml" + `
email: test@example.com
` + "```" + `

**Expected Error:** unique violation
`,
			expectError:   false,
			expectedType:  "unique violation",
			expectedCount: 1,
		},
		{
			name: "parse expected error with underscore notation",
			markdown: `## Description

Test foreign key error

## SQL

` + "```sql" + `
INSERT INTO orders (user_id) VALUES (/*= user_id */1)
` + "```" + `

## Test Cases

### Test Foreign Key

**Parameters:**
` + "```yaml" + `
user_id: 999
` + "```" + `

**Expected Error:** foreign_key_violation
`,
			expectError:   false,
			expectedType:  "foreign key violation",
			expectedCount: 1,
		},
		{
			name: "parse expected error with hyphen notation",
			markdown: `## Description

Test not null error

## SQL

` + "```sql" + `
INSERT INTO users (name) VALUES (/*= name */'test')
` + "```" + `

## Test Cases

### Test Not Null

**Parameters:**
` + "```yaml" + `
name: Alice
` + "```" + `

**Expected Error:** not-null-violation
`,
			expectError:   false,
			expectedType:  "not null violation",
			expectedCount: 1,
		},
		{
			name: "parse expected error with uppercase",
			markdown: `## Description

Test check constraint

## SQL

` + "```sql" + `
INSERT INTO users (age) VALUES (/*= age */20)
` + "```" + `

## Test Cases

### Test Check Constraint

**Parameters:**
` + "```yaml" + `
age: -5
` + "```" + `

**Expected Error:** CHECK VIOLATION
`,
			expectError:   false,
			expectedType:  "check violation",
			expectedCount: 1,
		},
		{
			name: "invalid error type should fail",
			markdown: `## Description

Test invalid error type

## SQL

` + "```sql" + `
SELECT * FROM users WHERE id = /*= id */1
` + "```" + `

## Test Cases

### Test Invalid Error

**Parameters:**
` + "```yaml" + `
id: 1
` + "```" + `

**Expected Error:** invalid_error_type
`,
			expectError:   true,
			expectedCount: 0,
		},
		{
			name: "multiple test cases with mixed notations",
			markdown: `## Description

Test multiple error cases

## SQL

` + "```sql" + `
INSERT INTO users (email) VALUES (/*= email */'test')
` + "```" + `

## Test Cases

### Test Case 1

**Expected Error:** unique violation

### Test Case 2

**Expected Error:** foreign_key_violation

### Test Case 3

**Expected Error:** Not-Null-Violation
`,
			expectError:   false,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse(bytes.NewReader([]byte(tt.markdown)))

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			testCases := doc.TestCases
			if len(testCases) != tt.expectedCount {
				t.Fatalf("expected %d test cases, got %d", tt.expectedCount, len(testCases))
			}

			if tt.expectedCount > 0 && tt.expectedType != "" {
				tc := testCases[0]
				if tc.ExpectedError == nil {
					t.Errorf("ExpectedError is nil")
					return
				}

				if *tc.ExpectedError != tt.expectedType {
					t.Errorf("expected error type %q, got %q", tt.expectedType, *tc.ExpectedError)
				}
			}

			// Verify all test cases have expected errors when multiple
			if tt.expectedCount > 1 {
				for i, tc := range testCases {
					if tc.ExpectedError == nil {
						t.Errorf("test case %d: ExpectedError is nil", i)
					}
				}
			}
		})
	}
}

func TestExpectedErrorAndExpectedResultsMutuallyExclusive(t *testing.T) {
	tests := []struct {
		name        string
		markdown    string
		expectError bool
	}{
		{
			name: "only expected error is valid",
			markdown: `## Description

Test with only error

## SQL

` + "```sql" + `
INSERT INTO users (email) VALUES (/*= email */'test')
` + "```" + `

## Test Cases

### Test Error Only

**Parameters:**
` + "```yaml" + `
id: 1
` + "```" + `

**Expected Error:** unique violation
`,
			expectError: false,
		},
		{
			name: "only expected results is valid",
			markdown: `## Description

Test with only results

## SQL

` + "```sql" + `
SELECT * FROM users WHERE id = /*= id */1
` + "```" + `

## Test Cases

### Test Results Only

**Parameters:**
` + "```yaml" + `
id: 1
` + "```" + `

**Expected Results:**
` + "```yaml" + `
- id: 1
  name: Alice
` + "```" + `
`,
			expectError: false,
		},
		{
			name: "neither expected error nor results should fail",
			markdown: `## Description

Test with neither

## SQL

` + "```sql" + `
SELECT * FROM users
` + "```" + `

## Test Cases

### Test Missing Both

**Parameters:**
` + "```yaml" + `
id: 1
` + "```" + `
`,
			expectError: true,
		},
		{
			name: "both expected error and results should fail",
			markdown: `## Description

Test with both error and results

## SQL

` + "```sql" + `
INSERT INTO users (email) VALUES (/*= email */'test')
` + "```" + `

## Test Cases

### Test Both Specified

**Parameters:**
` + "```yaml" + `
email: test@example.com
` + "```" + `

**Expected Error:** unique violation

**Expected Results:**
` + "```yaml" + `
- id: 1
  email: test@example.com
` + "```" + `
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(bytes.NewReader([]byte(tt.markdown)))

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
