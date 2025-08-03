package formatter

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestMarkdownFormatter_Format(t *testing.T) {
	formatter := NewMarkdownFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Basic SQL code block",
			input: `# My Query

Here's a SQL query:

` + "```sql" + `
select id,name from users where active=true
` + "```" + `

That's it!`,
			expected: `# My Query

Here's a SQL query:

` + "```sql" + `
SELECT
    id,
    name
FROM users
WHERE active = true
` + "```" + `

That's it!`,
		},
		{
			name: "Multiple SQL code blocks",
			input: `# Database Queries

## Users Query

` + "```sql" + `
select * from users
` + "```" + `

## Posts Query

` + "```sql" + `
select p.id,p.title,u.name from posts p join users u on p.user_id=u.id
` + "```",
			expected: `# Database Queries

## Users Query

` + "```sql" + `
SELECT
    *
FROM users
` + "```" + `

## Posts Query

` + "```sql" + `
SELECT
    p.id,
    p.title,
    u.name
FROM posts p
JOIN users u
    ON p.user_id = u.id
` + "```",
		},
		{
			name: "Indented SQL code block",
			input: `# Nested Example

1. First item:
   
   ` + "```sql" + `
   select id from users
   ` + "```" + `

2. Second item`,
			expected: `# Nested Example

1. First item:
   
   ` + "```sql" + `
   SELECT
       id
   FROM users
   ` + "```" + `

2. Second item`,
		},
		{
			name: "Non-SQL code blocks should be unchanged",
			input: `# Code Examples

` + "```javascript" + `
console.log("Hello World");
` + "```" + `

` + "```sql" + `
select 1
` + "```",
			expected: `# Code Examples

` + "```javascript" + `
console.log("Hello World");
` + "```" + `

` + "```sql" + `
SELECT
    1
` + "```",
		},
		{
			name: "Empty SQL block",
			input: `# Empty

` + "```sql" + `
` + "```",
			expected: `# Empty

` + "```sql" + `
` + "```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Format(tt.input)
			assert.NoError(t, err)
			
			if result != tt.expected {
				t.Errorf("Format() mismatch:\nExpected:\n%s\n\nActual:\n%s", tt.expected, result)
			}
		})
	}
}

func TestMarkdownFormatter_FormatSnapSQLMarkdown(t *testing.T) {
	formatter := NewMarkdownFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "SnapSQL Markdown with SQL section",
			input: `# Get User Posts

## Description

This query retrieves user posts with optional filtering.

## Parameters

` + "```yaml" + `
user_id: int
include_drafts: bool
` + "```" + `

## SQL

` + "```sql" + `
select u.id,u.name,p.title from users u join posts p on u.id=p.user_id where u.id=/*= user_id */
` + "```",
			expected: `# Get User Posts

## Description

This query retrieves user posts with optional filtering.

## Parameters

` + "```yaml" + `
user_id: int
include_drafts: bool
` + "```" + `

## SQL

` + "```sql" + `
SELECT
    u.id,
    u.name,
    p.title
FROM users u
JOIN posts p
    ON u.id = p.user_id
WHERE u.id = /*= user_id */
` + "```",
		},
		{
			name: "SnapSQL with if directives",
			input: `# Conditional Query

## SQL

` + "```sql" + `
select id,name /*# if include_email */ ,email /*# end */ from users
` + "```",
			expected: `# Conditional Query

## SQL

` + "```sql" + `
SELECT
    id,
    name/*# if include_email */,
        email/*# end */

FROM users
` + "```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.FormatSnapSQLMarkdown(tt.input)
			assert.NoError(t, err)
			
			if result != tt.expected {
				t.Errorf("FormatSnapSQLMarkdown() mismatch:\nExpected:\n%s\n\nActual:\n%s", tt.expected, result)
			}
		})
	}
}

func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"query.snap.md", true},
		{"README.md", true},
		{"query.snap.sql", false},
		{"query.sql", false},
		{"config.yaml", false},
		{"test.MD", true}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsMarkdownFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkMarkdownFormatter_Format(t *testing.B) {
	formatter := NewMarkdownFormatter()
	
	complexMarkdown := `# Complex Query Documentation

## Overview
This is a complex query with multiple SQL blocks.

## User Query
` + "```sql" + `
select u.id,u.name,u.email,u.created_at from users u where u.active=true and u.role in ('admin','user') order by u.created_at desc limit 100
` + "```" + `

## Post Query
` + "```sql" + `
select p.id,p.title,p.content,u.name as author from posts p join users u on p.user_id=u.id where p.published=true order by p.created_at desc
` + "```" + `

## Analytics Query
` + "```sql" + `
select date_trunc('day',created_at) as date,count(*) as post_count,count(distinct user_id) as unique_authors from posts where created_at >= current_date - interval '30 days' group by date_trunc('day',created_at) order by date
` + "```"

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_, err := formatter.Format(complexMarkdown)
		if err != nil {
			t.Fatal(err)
		}
	}
}
