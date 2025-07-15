package typeinference2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestExpressionCastAnalyzer(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []tokenizer.Token
		expected string
	}{
		{
			name: "Simple CAST expression",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "CAST"},
				{Type: tokenizer.OPENED_PARENS, Value: "("},
				{Type: tokenizer.IDENTIFIER, Value: "col"},
				{Type: tokenizer.AS, Value: "AS"},
				{Type: tokenizer.IDENTIFIER, Value: "INTEGER"},
				{Type: tokenizer.CLOSED_PARENS, Value: ")"},
			},
			expected: "int",
		},
		{
			name: "PostgreSQL cast expression",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "col"},
				{Type: tokenizer.DOUBLE_COLON, Value: "::"},
				{Type: tokenizer.IDENTIFIER, Value: "TEXT"},
			},
			expected: "string",
		},
		{
			name: "Function call without CAST",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "COUNT"},
				{Type: tokenizer.OPENED_PARENS, Value: "("},
				{Type: tokenizer.MULTIPLY, Value: "*"},
				{Type: tokenizer.CLOSED_PARENS, Value: ")"},
			},
			expected: "int",
		},
		{
			name: "Binary operation",
			tokens: []tokenizer.Token{
				{Type: tokenizer.NUMBER, Value: "5"},
				{Type: tokenizer.PLUS, Value: "+"},
				{Type: tokenizer.NUMBER, Value: "3"},
			},
			expected: "int",
		},
		{
			name: "JSON operator",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "data"},
				{Type: tokenizer.JSON_OPERATOR, Value: "->"},
				{Type: tokenizer.STRING, Value: "'name'"},
			},
			expected: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewExpressionCastAnalyzer(tt.tokens, nil)
			result, err := analyzer.InferExpressionType()

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.BaseType)
		})
	}
}

func TestCaseExpressionAnalyzer(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []tokenizer.Token
		expected string
	}{
		{
			name: "Simple CASE expression",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "CASE"},
				{Type: tokenizer.IDENTIFIER, Value: "WHEN"},
				{Type: tokenizer.IDENTIFIER, Value: "age"},
				{Type: tokenizer.GREATER_THAN, Value: ">"},
				{Type: tokenizer.NUMBER, Value: "18"},
				{Type: tokenizer.IDENTIFIER, Value: "THEN"},
				{Type: tokenizer.STRING, Value: "'adult'"},
				{Type: tokenizer.IDENTIFIER, Value: "ELSE"},
				{Type: tokenizer.STRING, Value: "'minor'"},
				{Type: tokenizer.IDENTIFIER, Value: "END"},
			},
			expected: "string",
		},
		{
			name: "Numeric CASE expression",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "CASE"},
				{Type: tokenizer.IDENTIFIER, Value: "WHEN"},
				{Type: tokenizer.IDENTIFIER, Value: "status"},
				{Type: tokenizer.EQUAL, Value: "="},
				{Type: tokenizer.STRING, Value: "'active'"},
				{Type: tokenizer.IDENTIFIER, Value: "THEN"},
				{Type: tokenizer.NUMBER, Value: "1"},
				{Type: tokenizer.IDENTIFIER, Value: "ELSE"},
				{Type: tokenizer.NUMBER, Value: "0"},
				{Type: tokenizer.IDENTIFIER, Value: "END"},
			},
			expected: "int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewCaseExpressionAnalyzer(tt.tokens, nil)
			result, err := analyzer.AnalyzeCaseExpression()

			assert.NoError(t, err)
			if result != nil {
				assert.Equal(t, tt.expected, result.InferredType.BaseType)
			}
		})
	}
}

func TestAdvancedFunctionInference(t *testing.T) {
	// Create a mock engine for testing
	engine := &TypeInferenceEngine2{}

	tests := []struct {
		name     string
		funcName string
		argTypes []*TypeInfo
		expected string
	}{
		{
			name:     "SUM with integer argument",
			funcName: "SUM",
			argTypes: []*TypeInfo{{BaseType: "int", IsNullable: false}},
			expected: "decimal",
		},
		{
			name:     "COUNT function",
			funcName: "COUNT",
			argTypes: []*TypeInfo{},
			expected: "int",
		},
		{
			name:     "COALESCE with string arguments",
			funcName: "COALESCE",
			argTypes: []*TypeInfo{
				{BaseType: "string", IsNullable: true},
				{BaseType: "string", IsNullable: false},
			},
			expected: "string",
		},
		{
			name:     "MIN with decimal argument",
			funcName: "MIN",
			argTypes: []*TypeInfo{{BaseType: "decimal", IsNullable: false}},
			expected: "decimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.applyAdvancedFunctionRule(tt.funcName, tt.argTypes)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.BaseType)
		})
	}
}

func TestOperatorTypeInference(t *testing.T) {
	analyzer := &ExpressionCastAnalyzer{}

	tests := []struct {
		name      string
		operator  string
		leftType  *TypeInfo
		rightType *TypeInfo
		expected  string
	}{
		{
			name:      "Integer addition",
			operator:  "+",
			leftType:  &TypeInfo{BaseType: "int", IsNullable: false},
			rightType: &TypeInfo{BaseType: "int", IsNullable: false},
			expected:  "int",
		},
		{
			name:      "Integer division",
			operator:  "/",
			leftType:  &TypeInfo{BaseType: "int", IsNullable: false},
			rightType: &TypeInfo{BaseType: "int", IsNullable: false},
			expected:  "float",
		},
		{
			name:      "String concatenation",
			operator:  "||",
			leftType:  &TypeInfo{BaseType: "string", IsNullable: false},
			rightType: &TypeInfo{BaseType: "string", IsNullable: false},
			expected:  "string",
		},
		{
			name:      "Comparison operation",
			operator:  ">",
			leftType:  &TypeInfo{BaseType: "int", IsNullable: false},
			rightType: &TypeInfo{BaseType: "int", IsNullable: false},
			expected:  "bool",
		},
		{
			name:      "JSON field access",
			operator:  "->",
			leftType:  &TypeInfo{BaseType: "json", IsNullable: false},
			rightType: &TypeInfo{BaseType: "string", IsNullable: false},
			expected:  "json",
		},
		{
			name:      "JSON text access",
			operator:  "->>",
			leftType:  &TypeInfo{BaseType: "json", IsNullable: false},
			rightType: &TypeInfo{BaseType: "string", IsNullable: false},
			expected:  "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.applyOperatorTypeRule(tt.operator, tt.leftType, tt.rightType)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.BaseType)
		})
	}
}

func TestCastDetection(t *testing.T) {
	tests := []struct {
		name          string
		tokens        []tokenizer.Token
		expectedCasts int
		expectedType  string
	}{
		{
			name: "Single CAST expression",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "CAST"},
				{Type: tokenizer.OPENED_PARENS, Value: "("},
				{Type: tokenizer.IDENTIFIER, Value: "age"},
				{Type: tokenizer.AS, Value: "AS"},
				{Type: tokenizer.IDENTIFIER, Value: "VARCHAR"},
				{Type: tokenizer.CLOSED_PARENS, Value: ")"},
			},
			expectedCasts: 1,
			expectedType:  "string",
		},
		{
			name: "PostgreSQL cast",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "id"},
				{Type: tokenizer.DOUBLE_COLON, Value: "::"},
				{Type: tokenizer.IDENTIFIER, Value: "TEXT"},
			},
			expectedCasts: 1,
			expectedType:  "string",
		},
		{
			name: "Nested CAST in function",
			tokens: []tokenizer.Token{
				{Type: tokenizer.IDENTIFIER, Value: "SUM"},
				{Type: tokenizer.OPENED_PARENS, Value: "("},
				{Type: tokenizer.IDENTIFIER, Value: "CAST"},
				{Type: tokenizer.OPENED_PARENS, Value: "("},
				{Type: tokenizer.IDENTIFIER, Value: "price"},
				{Type: tokenizer.AS, Value: "AS"},
				{Type: tokenizer.IDENTIFIER, Value: "DECIMAL"},
				{Type: tokenizer.CLOSED_PARENS, Value: ")"},
				{Type: tokenizer.CLOSED_PARENS, Value: ")"},
			},
			expectedCasts: 1,
			expectedType:  "decimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewExpressionCastAnalyzer(tt.tokens, nil)
			casts := analyzer.DetectCasts()

			assert.Equal(t, tt.expectedCasts, len(casts))
			if len(casts) > 0 {
				assert.Equal(t, tt.expectedType, casts[0].TargetType.BaseType)
			}
		})
	}
}

func TestTypePromotion(t *testing.T) {
	analyzer := &CaseExpressionAnalyzer{}

	tests := []struct {
		name     string
		types    []*TypeInfo
		expected string
	}{
		{
			name: "Integer promotion",
			types: []*TypeInfo{
				{BaseType: "int", IsNullable: false},
				{BaseType: "int", IsNullable: false},
			},
			expected: "int",
		},
		{
			name: "Numeric promotion to float",
			types: []*TypeInfo{
				{BaseType: "int", IsNullable: false},
				{BaseType: "float", IsNullable: false},
			},
			expected: "float",
		},
		{
			name: "Numeric promotion to decimal",
			types: []*TypeInfo{
				{BaseType: "int", IsNullable: false},
				{BaseType: "decimal", IsNullable: false},
			},
			expected: "decimal",
		},
		{
			name: "String types",
			types: []*TypeInfo{
				{BaseType: "string", IsNullable: false},
				{BaseType: "string", IsNullable: true},
			},
			expected: "string",
		},
		{
			name: "Mixed types default to string",
			types: []*TypeInfo{
				{BaseType: "int", IsNullable: false},
				{BaseType: "string", IsNullable: false},
			},
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.promoteTypes(tt.types)

			assert.Equal(t, tt.expected, result.BaseType)
		})
	}
}
