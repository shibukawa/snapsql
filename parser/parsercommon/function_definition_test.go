package parsercommon

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/goccy/go-yaml"
)

func TestFunctionDefinition(t *testing.T) {
	type testCase struct {
		name           string
		yamlSrc        string
		expectedOrder  []string
		expectedParams map[string]any
		expectedDummy  map[string]any
		wantErr        bool
	}
	tests := []testCase{
		{
			name: "simple order",
			yamlSrc: `
name: simple
function_name: simpleFunc
parameters:
  a: int
  b: string
  c: bool
`,
			expectedOrder: []string{"a", "b", "c"},
			expectedParams: map[string]any{
				"a": "int",
				"b": "string",
				"c": "bool",
			},
			expectedDummy: map[string]any{
				"a": int64(1),
				"b": "dummy",
				"c": true,
			},
			wantErr: false,
		},
		{
			name: "single parameter",
			yamlSrc: `
name: single
function_name: singleFunc
parameters:
  only: float
`,
			expectedOrder: []string{"only"},
			expectedParams: map[string]any{
				"only": "float",
			},
			expectedDummy: map[string]any{
				"only": 1.1,
			},
			wantErr: false,
		},
		{
			name: "nested parameters",
			yamlSrc: `
name: nested
function_name: nestedFunc
parameters:
  user:
    id: int
    name: string
  active: bool
  score: float
`,
			expectedOrder: []string{"user", "active", "score"},
			expectedParams: map[string]any{
				"user": map[string]any{
					"id":   "int",
					"name": "string",
				},
				"active": "bool",
				"score":  "float",
			},
			expectedDummy: map[string]any{
				"user": map[string]any{
					"id":   int64(1),
					"name": "dummy",
				},
				"active": true,
				"score":  1.1,
			},
			wantErr: false,
		},
		{
			name: "empty parameters",
			yamlSrc: `
name: empty
function_name: emptyFunc
parameters: {}
`,
			expectedOrder:  []string{},
			expectedParams: map[string]any{},
			expectedDummy:  map[string]any{},
			wantErr:        false,
		},
		{
			name: "empty parameters2",
			yamlSrc: `
name: empty
function_name: emptyFunc
`,
			expectedOrder:  []string{},
			expectedParams: map[string]any{},
			expectedDummy:  map[string]any{},
			wantErr:        false,
		},
		{
			name: "array of primitive",
			yamlSrc: `
name: array_primitive
function_name: arrayPrimitiveFunc
parameters:
  ids: [int]
  names: [string]
  flags: [bool]
`,
			expectedOrder: []string{"ids", "names", "flags"},
			expectedParams: map[string]any{
				"ids":   "int[]",
				"names": "string[]",
				"flags": "bool[]",
			},
			expectedDummy: map[string]any{
				"ids":   []any{int64(1)},
				"names": []any{"dummy"},
				"flags": []any{true},
			},
			wantErr: false,
		},
		{
			name: "array of object",
			yamlSrc: `
name: array_object
function_name: arrayObjectFunc
parameters:
  users:
    - id: int
      name: string
  scores: [float]
`,
			expectedOrder: []string{"users", "scores"},
			expectedParams: map[string]any{
				"users": []any{
					map[string]any{
						"id":   "int",
						"name": "string",
					},
				},
				"scores": "float[]",
			},
			expectedDummy: map[string]any{
				"users": []any{
					map[string]any{
						"id":   int64(1),
						"name": "dummy",
					},
				},
				"scores": []any{1.1},
			},
			wantErr: false,
		},
		{
			name: "type normalization and int32/float32",
			yamlSrc: `
name: normalization
function_name: normalizationFunc
parameters:
  i1: integer
  i2: long
  i3: int32
  i4: int16
  i5: int8
  f1: double
  f2: decimal
  f3: numeric
  f4: float32
`,
			expectedOrder: []string{"i1", "i2", "i3", "i4", "i5", "f1", "f2", "f3", "f4"},
			expectedParams: map[string]any{
				"i1": "int",
				"i2": "int",
				"i3": "int32",
				"i4": "int16",
				"i5": "int8",
				"f1": "float",
				"f2": "decimal",
				"f3": "decimal",
				"f4": "float32",
			},
			expectedDummy: map[string]any{
				"i1": int64(1),
				"i2": int64(1),
				"i3": int32(2),
				"i4": int16(3),
				"i5": int8(4),
				"f1": 1.1,
				"f2": "1.0",
				"f3": "1.0",
				"f4": float32(2.2),
			},
			wantErr: false,
		},
		{
			name: "invalid parameter name",
			yamlSrc: `
name: invalid
function_name: invalidFunc
parameters:
  "1abc": int
  valid_name: string
`,
			expectedOrder: []string{"valid_name"},
			expectedParams: map[string]any{
				"valid_name": "string",
			},
			expectedDummy: map[string]any{
				"valid_name": "dummy",
			},
			wantErr: true,
		},
		{
			name: "array of int32 and float32",
			yamlSrc: `
name: array_special
function_name: arraySpecialFunc
parameters:
  ids: [int32]
  scores: [float32]
`,
			expectedOrder: []string{"ids", "scores"},
			expectedParams: map[string]any{
				"ids":    "int32[]",
				"scores": "float32[]",
			},
			expectedDummy: map[string]any{
				"ids":    []any{int32(2)},
				"scores": []any{float32(2.2)},
			},
			wantErr: false,
		},
		{
			name: "array of object (complex)",
			yamlSrc: `
name: array_object_complex
function_name: arrayObjectComplexFunc
parameters:
  items:
    - id: int64
      value: float32
      flag: bool
      meta:
        tag: string
        score: float
`,
			expectedOrder: []string{"items"},
			expectedParams: map[string]any{
				"items": []any{
					map[string]any{
						"id":    "int",
						"value": "float32",
						"flag":  "bool",
						"meta": map[string]any{
							"tag":   "string",
							"score": "float",
						},
					},
				},
			},
			expectedDummy: map[string]any{
				"items": []any{
					map[string]any{
						"id":    int64(1),
						"value": float32(2.2),
						"flag":  true,
						"meta": map[string]any{
							"tag":   "dummy",
							"score": 1.1,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var def FunctionDefinition
			err := yaml.Unmarshal([]byte(tc.yamlSrc), &def)
			assert.NoError(t, err)
			err = def.Finalize()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedOrder, def.ParameterOrder)
				assert.Equal(t, tc.expectedParams, def.Parameters)
				assert.Equal(t, tc.expectedDummy, def.DummyData().(map[string]any))
			}
		})
	}
}
