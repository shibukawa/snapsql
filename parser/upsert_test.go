package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestInsertOnConflictAndDuplicateKey(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "valid INSERT ON CONFLICT",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'Alice') ON CONFLICT DO NOTHING;",
			expectError: false,
		},
		{
			name:        "valid INSERT ON DUPLICATE KEY UPDATE",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'Alice') ON DUPLICATE KEY UPDATE name = 'Bob';",
			expectError: false,
		},
		{
			name:        "invalid INSERT ON (missing CONFLICT/DUPLICATE)",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'Alice') ON SOMETHING ELSE;",
			expectError: true,
		},
		{
			name:        "invalid INSERT ON DUPLICATE (missing KEY)",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'Alice') ON DUPLICATE UPDATE name = 'Bob';",
			expectError: true,
		},
		{
			name:        "invalid INSERT ON DUPLICATE KEY (missing UPDATE)",
			sql:         "INSERT INTO users (id, name) VALUES (1, 'Alice') ON DUPLICATE KEY name = 'Bob';",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			parser := NewSqlParser(tokens, nil, nil)
			_, err = parser.Parse()

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
