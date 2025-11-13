package pygen

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
)

func TestConvertToPythonType_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		snapType string
		nullable bool
		want     string
		wantErr  bool
	}{
		// Integer types
		{name: "int", snapType: "int", nullable: false, want: "int", wantErr: false},
		{name: "int32", snapType: "int32", nullable: false, want: "int", wantErr: false},
		{name: "int64", snapType: "int64", nullable: false, want: "int", wantErr: false},
		{name: "nullable int", snapType: "int", nullable: true, want: "Optional[int]", wantErr: false},

		// String type
		{name: "string", snapType: "string", nullable: false, want: "str", wantErr: false},
		{name: "nullable string", snapType: "string", nullable: true, want: "Optional[str]", wantErr: false},

		// Boolean type
		{name: "bool", snapType: "bool", nullable: false, want: "bool", wantErr: false},
		{name: "nullable bool", snapType: "bool", nullable: true, want: "Optional[bool]", wantErr: false},

		// Float types
		{name: "float", snapType: "float", nullable: false, want: "float", wantErr: false},
		{name: "float32", snapType: "float32", nullable: false, want: "float", wantErr: false},
		{name: "float64", snapType: "float64", nullable: false, want: "float", wantErr: false},
		{name: "double", snapType: "double", nullable: false, want: "float", wantErr: false},
		{name: "nullable float", snapType: "float", nullable: true, want: "Optional[float]", wantErr: false},

		// Decimal type
		{name: "decimal", snapType: "decimal", nullable: false, want: "Decimal", wantErr: false},
		{name: "nullable decimal", snapType: "decimal", nullable: true, want: "Optional[Decimal]", wantErr: false},

		// Temporal types
		{name: "timestamp", snapType: "timestamp", nullable: false, want: "datetime", wantErr: false},
		{name: "date", snapType: "date", nullable: false, want: "datetime", wantErr: false},
		{name: "time", snapType: "time", nullable: false, want: "datetime", wantErr: false},
		{name: "datetime", snapType: "datetime", nullable: false, want: "datetime", wantErr: false},
		{name: "nullable timestamp", snapType: "timestamp", nullable: true, want: "Optional[datetime]", wantErr: false},

		// Bytes type
		{name: "bytes", snapType: "bytes", nullable: false, want: "bytes", wantErr: false},
		{name: "nullable bytes", snapType: "bytes", nullable: true, want: "Optional[bytes]", wantErr: false},

		// Any type
		{name: "any", snapType: "any", nullable: false, want: "Any", wantErr: false},
		{name: "nullable any", snapType: "any", nullable: true, want: "Optional[Any]", wantErr: false},

		// Case insensitivity
		{name: "STRING uppercase", snapType: "STRING", nullable: false, want: "str", wantErr: false},
		{name: "INT uppercase", snapType: "INT", nullable: false, want: "int", wantErr: false},
		{name: "BOOL uppercase", snapType: "BOOL", nullable: false, want: "bool", wantErr: false},

		// Unsupported types
		{name: "unsupported type", snapType: "unknown", nullable: false, want: "", wantErr: true},
		{name: "empty type", snapType: "", nullable: false, want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToPythonType(tt.snapType, tt.nullable)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertToPythonType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ConvertToPythonType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToPythonType_ArrayTypes(t *testing.T) {
	tests := []struct {
		name     string
		snapType string
		nullable bool
		want     string
		wantErr  bool
	}{
		// Basic array types
		{name: "int array", snapType: "int[]", nullable: false, want: "List[int]", wantErr: false},
		{name: "string array", snapType: "string[]", nullable: false, want: "List[str]", wantErr: false},
		{name: "bool array", snapType: "bool[]", nullable: false, want: "List[bool]", wantErr: false},
		{name: "float array", snapType: "float[]", nullable: false, want: "List[float]", wantErr: false},
		{name: "decimal array", snapType: "decimal[]", nullable: false, want: "List[Decimal]", wantErr: false},
		{name: "timestamp array", snapType: "timestamp[]", nullable: false, want: "List[datetime]", wantErr: false},

		// Nullable array types
		{name: "nullable int array", snapType: "int[]", nullable: true, want: "Optional[List[int]]", wantErr: false},
		{name: "nullable string array", snapType: "string[]", nullable: true, want: "Optional[List[str]]", wantErr: false},

		// Temporal alias arrays
		{name: "date array", snapType: "date[]", nullable: false, want: "List[datetime]", wantErr: false},
		{name: "time array", snapType: "time[]", nullable: false, want: "List[datetime]", wantErr: false},
		{name: "datetime array", snapType: "datetime[]", nullable: false, want: "List[datetime]", wantErr: false},

		// Unsupported array types
		{name: "unsupported array", snapType: "unknown[]", nullable: false, want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToPythonType(tt.snapType, tt.nullable)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertToPythonType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ConvertToPythonType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPlaceholder(t *testing.T) {
	t.Run("supported dialects", func(t *testing.T) {
		tests := []struct {
			name    string
			dialect snapsql.Dialect
			index   int
			want    string
		}{
			{name: "postgres index 1", dialect: snapsql.DialectPostgres, index: 1, want: "$1"},
			{name: "postgres index 10", dialect: snapsql.DialectPostgres, index: 10, want: "$10"},
			{name: "mysql index 2", dialect: snapsql.DialectMySQL, index: 2, want: "%s"},
			{name: "sqlite index 3", dialect: snapsql.DialectSQLite, index: 3, want: "?"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := GetPlaceholder(tt.dialect, tt.index); got != tt.want {
					t.Errorf("GetPlaceholder() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("unsupported dialect panics", func(t *testing.T) {
		tests := []struct {
			name    string
			dialect snapsql.Dialect
		}{
			{name: "oracle", dialect: snapsql.Dialect("oracle")},
			{name: "empty", dialect: snapsql.Dialect("")},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Fatalf("expected panic for dialect %q", tt.dialect)
					}
				}()

				_ = GetPlaceholder(tt.dialect, 1)
			})
		}
	})
}

func TestGetPlaceholderList(t *testing.T) {
	tests := []struct {
		name    string
		dialect snapsql.Dialect
		count   int
		want    []string
	}{
		{
			name:    "postgres 3 placeholders",
			dialect: snapsql.DialectPostgres,
			count:   3,
			want:    []string{"$1", "$2", "$3"},
		},
		{
			name:    "mysql 2 placeholders",
			dialect: snapsql.DialectMySQL,
			count:   2,
			want:    []string{"%s", "%s"},
		},
		{
			name:    "sqlite 0 placeholders",
			dialect: snapsql.DialectSQLite,
			count:   0,
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPlaceholderList(tt.dialect, tt.count)
			if len(got) != len(tt.want) {
				t.Fatalf("GetPlaceholderList() length = %v, want %v", len(got), len(tt.want))
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("GetPlaceholderList()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}

	t.Run("unsupported dialect panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for unsupported dialect")
			}
		}()

		_ = GetPlaceholderList(snapsql.Dialect("oracle"), 1)
	})
}

func TestGetRequiredImports(t *testing.T) {
	tests := []struct {
		name  string
		types []string
		want  []string
	}{
		{
			name:  "decimal import",
			types: []string{"Decimal", "int"},
			want:  []string{"from decimal import Decimal"},
		},
		{
			name:  "datetime import",
			types: []string{"datetime", "str"},
			want:  []string{"from datetime import datetime"},
		},
		{
			name:  "typing imports",
			types: []string{"Optional[int]", "List[str]"},
			want:  []string{"from typing import Optional, List, Any, Dict, AsyncGenerator"},
		},
		{
			name:  "multiple imports",
			types: []string{"Optional[Decimal]", "List[datetime]"},
			want:  []string{"from decimal import Decimal", "from datetime import datetime", "from typing import Optional, List, Any, Dict, AsyncGenerator"},
		},
		{
			name:  "no special imports",
			types: []string{"int", "str", "bool"},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRequiredImports(tt.types)
			if len(got) != len(tt.want) {
				t.Errorf("GetRequiredImports() length = %v, want %v", len(got), len(tt.want))
				return
			}
			// Check that all expected imports are present (order doesn't matter)
			gotMap := make(map[string]bool)
			for _, imp := range got {
				gotMap[imp] = true
			}

			for _, wantImp := range tt.want {
				if !gotMap[wantImp] {
					t.Errorf("GetRequiredImports() missing import: %v", wantImp)
				}
			}
		})
	}
}

func TestNormalizeTemporalAlias(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     string
	}{
		{name: "date", typeName: "date", want: "timestamp"},
		{name: "time", typeName: "time", want: "timestamp"},
		{name: "datetime", typeName: "datetime", want: "timestamp"},
		{name: "timestamp", typeName: "timestamp", want: "timestamp"},
		{name: "DATE uppercase", typeName: "DATE", want: "timestamp"},
		{name: "Time mixed case", typeName: "Time", want: "timestamp"},
		{name: "int unchanged", typeName: "int", want: "int"},
		{name: "string unchanged", typeName: "string", want: "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeTemporalAlias(tt.typeName)
			if got != tt.want {
				t.Errorf("normalizeTemporalAlias() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnsupportedTypeError(t *testing.T) {
	tests := []struct {
		name        string
		typeName    string
		context     string
		wantMessage string
		wantHints   bool
	}{
		{
			name:        "parameter context",
			typeName:    "unknown",
			context:     "parameter",
			wantMessage: "unsupported parameter type 'unknown'",
			wantHints:   true,
		},
		{
			name:        "response context",
			typeName:    "invalid",
			context:     "response",
			wantMessage: "unsupported response type 'invalid'",
			wantHints:   true,
		},
		{
			name:        "type conversion context",
			typeName:    "badtype",
			context:     "type conversion",
			wantMessage: "unsupported type conversion type 'badtype'",
			wantHints:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewUnsupportedTypeError(tt.typeName, tt.context)
			if err == nil {
				t.Fatal("NewUnsupportedTypeError() returned nil")
			}

			errMsg := err.Error()
			if !strings.Contains(errMsg, tt.wantMessage) {
				t.Errorf("Error message = %v, want to contain %v", errMsg, tt.wantMessage)
			}

			if tt.wantHints && len(err.Hints) == 0 {
				t.Errorf("Expected hints but got none")
			}

			if err.Type != tt.typeName {
				t.Errorf("Error.Type = %v, want %v", err.Type, tt.typeName)
			}

			if err.Context != tt.context {
				t.Errorf("Error.Context = %v, want %v", err.Context, tt.context)
			}
		})
	}
}
