package pygen

import (
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple camelCase",
			input: "getUserById",
			want:  "get_user_by_id",
		},
		{
			name:  "PascalCase",
			input: "GetUserById",
			want:  "get_user_by_id",
		},
		{
			name:  "single word lowercase",
			input: "user",
			want:  "user",
		},
		{
			name:  "single word uppercase",
			input: "User",
			want:  "user",
		},
		{
			name:  "acronym at start",
			input: "HTTPServer",
			want:  "http_server",
		},
		{
			name:  "acronym in middle",
			input: "getHTTPResponse",
			want:  "get_http_response",
		},
		{
			name:  "multiple words",
			input: "GetUserByEmailAddress",
			want:  "get_user_by_email_address",
		},
		{
			name:  "already snake_case",
			input: "get_user_by_id",
			want:  "get_user_by_id",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single character",
			input: "A",
			want:  "a",
		},
		{
			name:  "numbers",
			input: "GetUser123",
			want:  "get_user123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIndentString(t *testing.T) {
	tests := []struct {
		name   string
		spaces int
		input  string
		want   string
	}{
		{
			name:   "single line with 4 spaces",
			spaces: 4,
			input:  "hello",
			want:   "    hello",
		},
		{
			name:   "multiple lines with 4 spaces",
			spaces: 4,
			input:  "line1\nline2\nline3",
			want:   "    line1\n    line2\n    line3",
		},
		{
			name:   "empty string",
			spaces: 4,
			input:  "",
			want:   "",
		},
		{
			name:   "single line with 2 spaces",
			spaces: 2,
			input:  "hello",
			want:   "  hello",
		},
		{
			name:   "empty lines preserved",
			spaces: 4,
			input:  "line1\n\nline3",
			want:   "    line1\n\n    line3",
		},
		{
			name:   "zero spaces",
			spaces: 0,
			input:  "hello\nworld",
			want:   "hello\nworld",
		},
		{
			name:   "trailing newline",
			spaces: 4,
			input:  "hello\n",
			want:   "    hello\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentString(tt.spaces, tt.input)
			if got != tt.want {
				t.Errorf("indentString(%d, %q) = %q, want %q", tt.spaces, tt.input, got, tt.want)
			}
		})
	}
}

func TestGetTemplateFuncs(t *testing.T) {
	funcs := getTemplateFuncs()

	// Check that expected functions are present
	expectedFuncs := []string{"snakeCase", "indent"}

	for _, name := range expectedFuncs {
		if _, ok := funcs[name]; !ok {
			t.Errorf("getTemplateFuncs() missing function: %q", name)
		}
	}
}
