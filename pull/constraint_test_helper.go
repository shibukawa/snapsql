package pull

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

// ConstraintParser interface for testing constraint parsing
type ConstraintParser interface {
	ParseConstraintType(constraintType string) string
}

// testConstraintParsing is a common test helper for constraint parsing
func testConstraintParsing(t *testing.T, parser ConstraintParser, unknownValue string) {
	t.Helper()

	t.Run("ParsePrimaryKeyConstraint", func(t *testing.T) {
		result := parser.ParseConstraintType("PRIMARY KEY")
		assert.Equal(t, "PRIMARY_KEY", result)
	})

	t.Run("ParseForeignKeyConstraint", func(t *testing.T) {
		result := parser.ParseConstraintType("FOREIGN KEY")
		assert.Equal(t, "FOREIGN_KEY", result)
	})

	t.Run("ParseUniqueConstraint", func(t *testing.T) {
		result := parser.ParseConstraintType("UNIQUE")
		assert.Equal(t, "UNIQUE", result)
	})

	t.Run("ParseCheckConstraint", func(t *testing.T) {
		result := parser.ParseConstraintType("CHECK")
		assert.Equal(t, "CHECK", result)
	})

	t.Run("ParseUnknownConstraint", func(t *testing.T) {
		result := parser.ParseConstraintType(unknownValue)
		assert.Equal(t, unknownValue, result)
	})
}
