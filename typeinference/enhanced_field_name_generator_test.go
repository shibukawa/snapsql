package typeinference2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// TestEnhancedFieldNameGenerator tests the enhanced field name generation functionality
func TestEnhancedFieldNameGenerator(t *testing.T) {
	t.Run("Basic Field Name Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// Simple field
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.SingleField,
			OriginalField: "name",
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "name", name)
	})

	t.Run("Function Field Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// COUNT function
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.FunctionField,
			OriginalField: "COUNT(id)",
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "count_id", name)

		// SUM function
		field = &parsercommon.SelectField{
			FieldKind:     parsercommon.FunctionField,
			OriginalField: "SUM(amount)",
		}
		name = generator.GenerateComplexFieldName(field)
		assert.Equal(t, "sum_amount", name)
	})

	t.Run("Arithmetic Expression Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// Addition
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "price + tax",
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "price_plus_tax", name)

		// Multiplication
		field = &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "quantity * rate",
		}
		name = generator.GenerateComplexFieldName(field)
		assert.Equal(t, "quantity_times_rate", name)
	})

	t.Run("JSON Expression Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// JSON field access
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "data -> 'name'",
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "data_name_field", name)

		// JSON text access
		field = &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "metadata ->> 'title'",
		}
		name = generator.GenerateComplexFieldName(field)
		assert.Equal(t, "metadata_title_text", name)
	})

	t.Run("CASE Expression Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// Price-based CASE
		field := &parsercommon.SelectField{
			FieldKind: parsercommon.ComplexField,
			OriginalField: `CASE 
				WHEN price > 100 THEN 'HIGH'
				WHEN price > 50 THEN 'MEDIUM' 
				ELSE 'LOW' 
			END`,
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "price_range", name)

		// Status-based CASE
		field = &parsercommon.SelectField{
			FieldKind: parsercommon.ComplexField,
			OriginalField: `CASE 
				WHEN status = 1 THEN 'ACTIVE'
				ELSE 'INACTIVE' 
			END`,
		}
		name = generator.GenerateComplexFieldName(field)
		assert.Equal(t, "status_category", name)
	})

	t.Run("CAST Expression Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// CAST function
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "CAST(amount AS DECIMAL)",
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "amount_as_decimal", name)

		// PostgreSQL cast
		field = &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "price::NUMERIC",
		}
		name = generator.GenerateComplexFieldName(field)
		assert.Equal(t, "price_as_numeric", name)
	})

	t.Run("Comparison Expression Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// Equality comparison
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "status = 'active'",
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "status_equals", name)

		// Greater than comparison
		field = &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "age > 18",
		}
		name = generator.GenerateComplexFieldName(field)
		assert.Equal(t, "age_greater", name)
	})

	t.Run("String Concatenation Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// String concatenation
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "first_name || last_name",
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "first_name_last_name_concat", name)
	})

	t.Run("Literal Field Generation", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// String literal
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.LiteralField,
			OriginalField: "'constant_value'",
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "string_literal", name)

		// NULL literal
		field = &parsercommon.SelectField{
			FieldKind:     parsercommon.LiteralField,
			OriginalField: "NULL",
		}
		name = generator.GenerateComplexFieldName(field)
		assert.Equal(t, "null_value", name)
	})

	t.Run("Duplicate Name Handling", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// First usage
		field1 := &parsercommon.SelectField{
			FieldKind:     parsercommon.SingleField,
			OriginalField: "name",
		}
		name1 := generator.GenerateComplexFieldName(field1)
		assert.Equal(t, "name", name1)

		// Second usage - should get suffix
		field2 := &parsercommon.SelectField{
			FieldKind:     parsercommon.SingleField,
			OriginalField: "name",
		}
		name2 := generator.GenerateComplexFieldName(field2)
		assert.Equal(t, "name2", name2)
	})

	t.Run("Explicit Name and Type", func(t *testing.T) {
		generator := NewEnhancedFieldNameGenerator()

		// Explicit field name
		field := &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "price + tax",
			FieldName:     "total_cost",
			ExplicitName:  true,
		}
		name := generator.GenerateComplexFieldName(field)
		assert.Equal(t, "total_cost", name)

		// TypeName with CAST
		field = &parsercommon.SelectField{
			FieldKind:     parsercommon.ComplexField,
			OriginalField: "amount",
			TypeName:      "VARCHAR",
		}
		name = generator.GenerateComplexFieldName(field)
		assert.Equal(t, "amount_as_varchar", name)
	})
}
