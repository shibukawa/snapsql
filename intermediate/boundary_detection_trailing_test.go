package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

// TestDetectBoundaryDelimiter_TrailingPatterns tests the detection of trailing boundary patterns
func TestDetectBoundaryDelimiter_TrailingPatterns(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
		reason   string
	}{
		{
			name:     "Trailing comma after parenthesis",
			value:    "),",
			expected: true,
			reason:   "Should detect trailing comma in VALUES clause",
		},
		{
			name:     "Trailing comma after value",
			value:    "'value',",
			expected: true,
			reason:   "Should detect trailing comma after string literal",
		},
		{
			name:     "Trailing comma after number",
			value:    "123,",
			expected: true,
			reason:   "Should detect trailing comma after number",
		},
		{
			name:     "Just comma (leading pattern)",
			value:    ",",
			expected: true,
			reason:   "Single comma matches leading comma pattern",
		},
		{
			name:     "Trailing AND keyword",
			value:    "condition AND",
			expected: true,
			reason:   "Should detect trailing AND in WHERE clause",
		},
		{
			name:     "Trailing OR keyword",
			value:    "condition OR",
			expected: true,
			reason:   "Should detect trailing OR in WHERE clause",
		},
		{
			name:     "Leading comma (not trailing)",
			value:    ", field",
			expected: true,
			reason:   "Should detect leading comma (different pattern)",
		},
		{
			name:     "Leading AND as separate token",
			value:    "AND",
			expected: true,
			reason:   "Should detect leading AND as a separate token",
		},
		{
			name:     "No boundary delimiter",
			value:    "field_name",
			expected: false,
			reason:   "Should not detect boundary in plain identifier",
		},
		{
			name:     "Parenthesis without comma",
			value:    ")",
			expected: false,
			reason:   "Should not detect boundary in plain parenthesis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectBoundaryDelimiter(tt.value)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

// TestTrailingBoundaryPatterns tests the trailing boundary patterns directly
func TestTrailingBoundaryPatterns(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		pattern int // index in trailingBoundaryPatterns
		matches bool
	}{
		{
			name:    "Trailing comma pattern matches ),",
			value:   "),",
			pattern: 0, // comma pattern
			matches: true,
		},
		{
			name:    "Trailing comma pattern matches value,",
			value:   "value,",
			pattern: 0,
			matches: true,
		},
		{
			name:    "Trailing comma pattern does not match single comma",
			value:   ",",
			pattern: 0,
			matches: false,
		},
		{
			name:    "Trailing AND pattern matches condition AND",
			value:   "condition AND",
			pattern: 1, // AND pattern
			matches: true,
		},
		{
			name:    "Trailing OR pattern matches condition OR",
			value:   "condition OR",
			pattern: 2, // OR pattern
			matches: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := trailingBoundaryPatterns[tt.pattern]
			result := pattern.Pattern.MatchString(tt.value)
			assert.Equal(t, tt.matches, result, "Pattern: %s", pattern.Description)
		})
	}
}

// TestLeadingBoundaryPatterns tests the leading boundary patterns
func TestLeadingBoundaryPatterns(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		pattern int // index in boundaryPatterns
		matches bool
	}{
		{
			name:    "Leading comma pattern matches , field",
			value:   ", field",
			pattern: 0, // comma pattern
			matches: true,
		},
		{
			name:    "Leading comma pattern matches ,field",
			value:   ",field",
			pattern: 0,
			matches: true,
		},
		{
			name:    "Leading AND pattern matches AND token",
			value:   "AND",
			pattern: 1, // AND pattern
			matches: true,
		},
		{
			name:    "Leading OR pattern matches OR token",
			value:   "OR",
			pattern: 2, // OR pattern
			matches: true,
		},
		{
			name:    "Leading AND pattern does not match AND with text",
			value:   "AND condition",
			pattern: 1, // AND pattern
			matches: false,
		},
		{
			name:    "Leading comma does not match field,",
			value:   "field,",
			pattern: 0,
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := boundaryPatterns[tt.pattern]
			result := pattern.Pattern.MatchString(tt.value)
			assert.Equal(t, tt.matches, result, "Pattern: %s", pattern.Description)
		})
	}
}
