package formatter

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestSQLFormatter_Format(t *testing.T) {
	formatter := NewSQLFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Basic SELECT statement",
			input: `select id,name,email from users where active=true`,
			expected: `SELECT
    id,
    name,
    email
FROM users
WHERE active = true`,
		},
		{
			name: "SELECT with JOIN",
			input: `select u.id,u.name,p.title from users u join posts p on u.id=p.user_id`,
			expected: `SELECT
    u.id,
    u.name,
    p.title
FROM users u
JOIN posts p
    ON u.id = p.user_id`,
		},
		{
			name: "Complex query with WHERE conditions",
			input: `select * from users where age>18 and status='active' or premium=true`,
			expected: `SELECT *
FROM users
WHERE age > 18 AND status = 'active' OR premium = true`,
		},
		{
			name: "SnapSQL with inline expressions",
			input: `select id,name from users where id=/*= user_id */123`,
			expected: `SELECT
    id,
    name
FROM users
WHERE id = /*= user_id */123`,
		},
		{
			name: "SnapSQL with if directive",
			input: `select id,name /*# if include_email */ ,email /*# end */ from users`,
			expected: `SELECT
    id,
    name
    /*# if include_email */
        ,email
    /*# end */
FROM users`,
		},
		{
			name: "SnapSQL with for loop",
			input: `select id /*# for field in fields */ ,/*= field */ /*# end */ from users`,
			expected: `SELECT
    id
    /*# for field in fields */
        ,/*= field */
    /*# end */
FROM users`,
		},
		{
			name: "SnapSQL with function definition",
			input: `/*#
function_name: get_users
parameters:
  user_id: int
  include_email: bool
*/
select id,name from users where id=/*= user_id */`,
			expected: `/*#
function_name: get_users
parameters:
  user_id: int
  include_email: bool
*/
SELECT
    id,
    name
FROM users
WHERE id = /*= user_id */`,
		},
		{
			name: "INSERT statement",
			input: `insert into users(name,email,created_at) values(/*= name */,'test@example.com',now())`,
			expected: `INSERT INTO users(
    name,
    email,
    created_at
) VALUES(
    /*= name */,
    'test@example.com',
    now()
)`,
		},
		{
			name: "UPDATE statement",
			input: `update users set name=/*= name */,email=/*= email */ where id=/*= user_id */`,
			expected: `UPDATE users
SET
    name = /*= name */,
    email = /*= email */
WHERE id = /*= user_id */`,
		},
		{
			name: "Complex query with GROUP BY and HAVING",
			input: `select department,count(*) as cnt from users where active=true group by department having count(*)>5 order by cnt desc`,
			expected: `SELECT
    department,
    count(*) AS cnt
FROM users
WHERE active = true
GROUP BY department
HAVING count(*) > 5
ORDER BY cnt DESC`,
		},
		{
			name: "Nested if/else conditions",
			input: `select id,name /*# if include_details */ /*# if include_email */ ,email /*# end */ /*# if include_phone */ ,phone /*# end */ /*# end */ from users`,
			expected: `SELECT
    id,
    name
    /*# if include_details */
        /*# if include_email */
            ,email
        /*# end */
        /*# if include_phone */
            ,phone
        /*# end */
    /*# end */
FROM users`,
		},
		{
			name: "Query with comments",
			input: `-- Get active users
select id, -- user identifier
name, -- user name
email -- user email
from users -- main users table
where active = true -- only active users`,
			expected: `-- Get active users
SELECT
    id, -- user identifier
    name, -- user name
    email -- user email
FROM users -- main users table
WHERE active = true -- only active users`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Format(tt.input)
			assert.NoError(t, err)
			
			// Normalize whitespace for comparison
			expected := normalizeWhitespace(tt.expected)
			actual := normalizeWhitespace(result)
			
			if expected != actual {
				t.Errorf("Format() mismatch:\nExpected:\n%s\n\nActual:\n%s", tt.expected, result)
			}
		})
	}
}

func TestSQLFormatter_TokenizeSnapDirectives(t *testing.T) {
	formatter := NewSQLFormatter()

	tests := []struct {
		name     string
		input    string
		expected []string // Expected token values for SnapSQL directives
	}{
		{
			name:  "Function definition directive",
			input: `/*# function_name: test */`,
			expected: []string{
				"/*# function_name: test */",
			},
		},
		{
			name:  "If directive",
			input: `/*# if condition */`,
			expected: []string{
				"/*# if condition */",
			},
		},
		{
			name:  "Inline expression",
			input: `/*= user_id */`,
			expected: []string{
				"/*= user_id */",
			},
		},
		{
			name:  "Multiple directives",
			input: `/*# if test */ /*= value */ /*# end */`,
			expected: []string{
				"/*# if test */",
				"/*= value */",
				"/*# end */",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := formatter.tokenize(tt.input)
			assert.NoError(t, err)

			var snapTokens []string
			for _, token := range tokens {
				if token.Type == TokenSnapDirective {
					snapTokens = append(snapTokens, token.Value)
				}
			}

			assert.Equal(t, tt.expected, snapTokens)
		})
	}
}

func TestSQLFormatter_KeywordCasing(t *testing.T) {
	formatter := NewSQLFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Lowercase keywords",
			input:    `select id from users where active = true`,
			expected: `SELECT`,
		},
		{
			name:     "Mixed case keywords",
			input:    `SeLeCt id FrOm users WhErE active = true`,
			expected: `SELECT`,
		},
		{
			name:     "Uppercase keywords",
			input:    `SELECT id FROM users WHERE active = true`,
			expected: `SELECT`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Format(tt.input)
			assert.NoError(t, err)
			assert.True(t, strings.Contains(result, tt.expected))
		})
	}
}

func TestSQLFormatter_IndentationLevels(t *testing.T) {
	formatter := NewSQLFormatter()

	input := `/*# if condition1 */ /*# if condition2 */ select id /*# end */ /*# end */`
	
	result, err := formatter.Format(input)
	assert.NoError(t, err)

	lines := strings.Split(result, "\n")
	
	// Check indentation levels
	for _, line := range lines {
		if strings.Contains(line, "/*# if condition1 */") {
			assert.Equal(t, 0, countLeadingSpaces(line))
		}
		if strings.Contains(line, "/*# if condition2 */") {
			assert.Equal(t, 4, countLeadingSpaces(line))
		}
		if strings.Contains(line, "select id") {
			assert.Equal(t, 8, countLeadingSpaces(line))
		}
	}
}

// Helper functions

func normalizeWhitespace(s string) string {
	// Remove trailing whitespace from each line
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func countLeadingSpaces(s string) int {
	count := 0
	for _, char := range s {
		if char == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

func BenchmarkSQLFormatter_Format(t *testing.B) {
	formatter := NewSQLFormatter()
	
	complexSQL := `/*#
function_name: complex_query
parameters:
  user_id: int
  include_profile: bool
  start_date: string
  end_date: string
*/
select u.id,u.name,u.email /*# if include_profile */ ,p.bio,p.avatar_url /*# end */ from users u /*# if include_profile */ left join profiles p on u.id=p.user_id /*# end */ where u.id=/*= user_id */ /*# if start_date != "" && end_date != "" */ and u.created_at between /*= start_date */ and /*= end_date */ /*# end */ order by u.created_at desc limit 100`

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_, err := formatter.Format(complexSQL)
		if err != nil {
			t.Fatal(err)
		}
	}
}
