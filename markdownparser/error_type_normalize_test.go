package markdownparser

import (
	"testing"
)

func TestNormalizeErrorType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 標準形式（スペース区切り）
		{
			name:     "standard space",
			input:    "unique violation",
			expected: "unique violation",
		},
		{
			name:     "standard multi-word",
			input:    "foreign key violation",
			expected: "foreign key violation",
		},

		// アンダースコア区切りからスペースへ
		{
			name:     "underscore to space",
			input:    "unique_violation",
			expected: "unique violation",
		},
		{
			name:     "multi underscore to space",
			input:    "foreign_key_violation",
			expected: "foreign key violation",
		},

		// 大文字小文字のバリエーション
		{
			name:     "uppercase",
			input:    "UNIQUE_VIOLATION",
			expected: "unique violation",
		},
		{
			name:     "mixed case underscore",
			input:    "Unique_Violation",
			expected: "unique violation",
		},
		{
			name:     "title case space",
			input:    "Foreign Key Violation",
			expected: "foreign key violation",
		},
		{
			name:     "uppercase space",
			input:    "UNIQUE VIOLATION",
			expected: "unique violation",
		},

		// ハイフン区切り
		{
			name:     "hyphen to space",
			input:    "unique-violation",
			expected: "unique violation",
		},
		{
			name:     "multi hyphen to space",
			input:    "foreign-key-violation",
			expected: "foreign key violation",
		},
		{
			name:     "hyphen title case",
			input:    "Unique-Violation",
			expected: "unique violation",
		},
		{
			name:     "hyphen uppercase",
			input:    "FOREIGN-KEY-VIOLATION",
			expected: "foreign key violation",
		},

		// 混合形式
		{
			name:     "mixed space and hyphen",
			input:    "foreign key-violation",
			expected: "foreign key violation",
		},
		{
			name:     "mixed hyphen and underscore",
			input:    "foreign-key_violation",
			expected: "foreign key violation",
		},
		{
			name:     "mixed all three",
			input:    "foreign_key-violation",
			expected: "foreign key violation",
		},

		// 前後の空白
		{
			name:     "leading space",
			input:    "  unique violation",
			expected: "unique violation",
		},
		{
			name:     "trailing space",
			input:    "unique violation  ",
			expected: "unique violation",
		},
		{
			name:     "both spaces",
			input:    "  foreign key violation  ",
			expected: "foreign key violation",
		},

		// 連続する区切り文字
		{
			name:     "double underscore",
			input:    "unique__violation",
			expected: "unique violation",
		},
		{
			name:     "double space",
			input:    "foreign  key  violation",
			expected: "foreign key violation",
		},
		{
			name:     "double hyphen",
			input:    "not--null--violation",
			expected: "not null violation",
		},
		{
			name:     "triple space",
			input:    "data   too   long",
			expected: "data too long",
		},

		// 実際のエラータイプ全種類
		{
			name:     "not null violation space",
			input:    "not null violation",
			expected: "not null violation",
		},
		{
			name:     "not null violation underscore",
			input:    "not_null_violation",
			expected: "not null violation",
		},
		{
			name:     "check violation",
			input:    "check violation",
			expected: "check violation",
		},
		{
			name:     "check_violation",
			input:    "check_violation",
			expected: "check violation",
		},
		{
			name:     "not found",
			input:    "not found",
			expected: "not found",
		},
		{
			name:     "not_found",
			input:    "not_found",
			expected: "not found",
		},
		{
			name:     "data too long space",
			input:    "data too long",
			expected: "data too long",
		},
		{
			name:     "data_too_long",
			input:    "data_too_long",
			expected: "data too long",
		},
		{
			name:     "numeric overflow",
			input:    "numeric overflow",
			expected: "numeric overflow",
		},
		{
			name:     "numeric_overflow",
			input:    "numeric_overflow",
			expected: "numeric overflow",
		},
		{
			name:     "invalid text representation space",
			input:    "invalid text representation",
			expected: "invalid text representation",
		},
		{
			name:     "invalid_text_representation",
			input:    "invalid_text_representation",
			expected: "invalid text representation",
		},
		{
			name:     "Invalid-Text-Representation",
			input:    "Invalid-Text-Representation",
			expected: "invalid text representation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeErrorType(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeErrorType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseExpectedError(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name:      "valid space",
			input:     "unique violation",
			expected:  "unique violation",
			expectErr: false,
		},
		{
			name:      "valid underscore",
			input:     "unique_violation",
			expected:  "unique violation",
			expectErr: false,
		},
		{
			name:      "valid multi-word space",
			input:     "foreign key violation",
			expected:  "foreign key violation",
			expectErr: false,
		},
		{
			name:      "valid multi-word underscore",
			input:     "foreign_key_violation",
			expected:  "foreign key violation",
			expectErr: false,
		},
		{
			name:      "valid hyphen",
			input:     "not-null-violation",
			expected:  "not null violation",
			expectErr: false,
		},
		{
			name:      "valid uppercase",
			input:     "CHECK VIOLATION",
			expected:  "check violation",
			expectErr: false,
		},
		{
			name:      "invalid error type",
			input:     "invalid_error_type",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "empty string",
			input:     "",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "with leading/trailing spaces",
			input:     "  not found  ",
			expected:  "not found",
			expectErr: false,
		},
		{
			name:      "all error types with underscore",
			input:     "data_too_long",
			expected:  "data too long",
			expectErr: false,
		},
		{
			name:      "all error types with hyphen",
			input:     "numeric-overflow",
			expected:  "numeric overflow",
			expectErr: false,
		},
		{
			name:      "complex multi-word",
			input:     "invalid_text_representation",
			expected:  "invalid text representation",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseExpectedError(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("ParseExpectedError(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseExpectedError(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result == nil {
				t.Errorf("ParseExpectedError(%q) returned nil result", tt.input)
				return
			}

			if *result != tt.expected {
				t.Errorf("ParseExpectedError(%q) = %q, want %q", tt.input, *result, tt.expected)
			}
		})
	}
}
