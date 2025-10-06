package gogen

import "testing"

func TestEnsureKeywordSpacing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AND followed by identifier",
			input:    "ANDi.read_at IS NULL",
			expected: "AND i.read_at IS NULL",
		},
		{
			name:     "AND followed by parenthesis",
			input:    "AND(n.expires_at IS NULL OR n.expires_at > NOW())",
			expected: "AND (n.expires_at IS NULL OR n.expires_at > NOW())",
		},
		{
			name:     "WHERE without space",
			input:    "WHEREi.user_id = $1",
			expected: "WHERE i.user_id = $1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ensureKeywordSpacing(tt.input); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
