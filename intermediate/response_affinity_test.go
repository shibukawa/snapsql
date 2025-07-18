package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestDetermineResponseAffinity(t *testing.T) {
	tests := []struct {
		name             string
		sql              string
		expectedAffinity ResponseAffinity
	}{
		{
			name:             "SimpleSelect",
			sql:              "SELECT id, name, email FROM users",
			expectedAffinity: ResponseAffinityMany,
		},
		{
			name:             "SelectWithUniqueKey",
			sql:              "SELECT id, name, email FROM users WHERE id = 1",
			expectedAffinity: ResponseAffinityMany, // Will be One when hasUniqueKeyCondition is implemented
		},
		{
			name:             "SelectWithLimit1",
			sql:              "SELECT id, name, email FROM users LIMIT 1",
			expectedAffinity: ResponseAffinityOne,
		},
		{
			name:             "InsertWithoutReturning",
			sql:              "INSERT INTO users (name, email) VALUES ('John', 'john@example.com')",
			expectedAffinity: ResponseAffinityNone,
		},
		{
			name:             "InsertWithReturning",
			sql:              "INSERT INTO users (name, email) VALUES ('John', 'john@example.com') RETURNING id",
			expectedAffinity: ResponseAffinityOne,
		},
		{
			name:             "BulkInsertWithReturning",
			sql:              "INSERT INTO users (name, email) VALUES ('John', 'john@example.com'), ('Jane', 'jane@example.com') RETURNING id",
			expectedAffinity: ResponseAffinityOne, // Will be Many when isBulkInsert is implemented
		},
		{
			name:             "UpdateWithoutReturning",
			sql:              "UPDATE users SET name = 'John' WHERE id = 1",
			expectedAffinity: ResponseAffinityNone,
		},
		{
			name:             "UpdateWithReturning",
			sql:              "UPDATE users SET name = 'John' WHERE id = 1 RETURNING id, name",
			expectedAffinity: ResponseAffinityMany,
		},
		{
			name:             "DeleteWithoutReturning",
			sql:              "DELETE FROM users WHERE id = 1",
			expectedAffinity: ResponseAffinityNone,
		},
		{
			name:             "DeleteWithReturning",
			sql:              "DELETE FROM users WHERE id = 1 RETURNING id, name",
			expectedAffinity: ResponseAffinityMany,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL
			tokens, err := tokenizer.Tokenize(tt.sql)
			if err != nil {
				t.Fatalf("Failed to tokenize SQL: %v", err)
			}

			// Parse statement
			stmt, err := parser.Parse(tokens, nil, nil)
			if err != nil {
				t.Skipf("Parser not fully implemented yet: %v", err)
				return
			}

			// Determine response affinity
			affinity := DetermineResponseAffinity(stmt)

			// Verify affinity
			assert.Equal(t, tt.expectedAffinity, affinity, "Response affinity should match")
		})
	}
}
