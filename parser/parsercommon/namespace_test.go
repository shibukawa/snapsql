package parsercommon

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNamespace_Eval(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	type args struct {
		src string
		exp string
	}

	tests := []struct {
		name      string
		args      args
		wantValue any
		wantType  string
		wantErr   bool
	}{
		{
			name: "simple expression",
			args: args{
				src: `
name: simple
function_name: simpleFunc
parameters:
  a: int
  b: string
  c: bool
`,
				exp: "a",
			},
			wantValue: 1,
			wantType:  "int",
			wantErr:   false,
		},
		{
			name: "comples expression",
			args: args{
				src: `
name: simple
function_name: simpleFunc
parameters:
  persons:
    - id: int
      name: string
      hobbies: string[]
`,
				exp: "persons[0].hobbies[0]",
			},
			wantValue: "dummy",
			wantType:  "string",
			wantErr:   false,
		},
		{
			name: "non-existent parameter",
			args: args{
				src: `name: simple
function_name: simpleFunc
parameters:
  a: int
  b: string`,
				exp: "c",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var def FunctionDefinition

			err := yaml.Unmarshal([]byte(tt.args.src), &def)
			assert.NoError(t, err)

			dir, _ := os.Getwd()
			err = def.Finalize(dir, dir)
			assert.NoError(t, err)
			ns, err := NewNamespaceFromDefinition(&def)
			assert.NoError(t, err)

			v, tp, err := ns.Eval(tt.args.exp)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValue, v)
				assert.Equal(t, tt.wantType, tp)
			}
		})
	}
}

func TestNamespace_Loop(t *testing.T) {
	yamlSrc := `name: loop_test
function_name: loopTestFunc
parameters:
  global: bool
  items:
    - id: int
      name: string
      hobbies: string[]`

	var def FunctionDefinition

	err := yaml.Unmarshal([]byte(yamlSrc), &def)
	assert.NoError(t, err)

	dir, _ := os.Getwd()
	err = def.Finalize(dir, dir)
	assert.NoError(t, err)
	ns, err := NewNamespaceFromDefinition(&def)
	assert.NoError(t, err)

	// enter Loop
	target, _, err := ns.Eval("items")
	assert.NoError(t, err)
	err = ns.EnterLoop("item", target)
	assert.NoError(t, err)

	// it access via loop variable
	v, tp, err := ns.Eval("item.id")
	assert.NoError(t, err)
	assert.Equal(t, "int", tp)
	assert.Equal(t, 1, v)

	// it access global variable
	v, tp, err = ns.Eval("global")
	assert.NoError(t, err)
	assert.Equal(t, "bool", tp)
	assert.Equal(t, true, v)

	// it enter nested loop
	target2, _, err := ns.Eval("item.hobbies")
	assert.NoError(t, err)
	err = ns.EnterLoop("hobby", target2)
	assert.NoError(t, err)

	// it access via nested loop variable
	v, tp, err = ns.Eval("hobby")
	assert.NoError(t, err)
	assert.Equal(t, "string", tp)
	assert.Equal(t, "dummy", v)

	// exit nested loop
	err = ns.ExitLoop()
	assert.NoError(t, err)

	// can't access exited loop variable
	_, _, err = ns.Eval("hobby")
	assert.Error(t, err)

	// exit outer loop
	err = ns.ExitLoop()
	assert.NoError(t, err)

	// can't access exited loop variable
	_, _, err = ns.Eval("item")
	assert.Error(t, err)
}

func TestNamespace_ConstantMode(t *testing.T) {
	uuidVal := uuid.New()
	constants := map[string]any{
		"constant_int":       42,
		"constant_string":    "hello",
		"constant_float":     3.14,
		"constant_bool":      true,
		"constant_list":      []any{"a", "b", "c"},
		"constant_map":       map[string]any{"key1": "value1", "key2": 2},
		"constant_date":      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		"constant_datetime":  time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		"constant_timestamp": time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
		"constant_uuid":      uuidVal,
		"constant_decimal":   decimal.NewFromInt(100),
	}

	ns, err := NewNamespaceFromConstants(constants)
	assert.NoError(t, err)

	val, tp, err := ns.Eval("constant_int")
	assert.NoError(t, err)
	assert.Equal(t, 42, val)
	assert.Equal(t, "int", tp)

	val, tp, err = ns.Eval("constant_string")
	assert.NoError(t, err)
	assert.Equal(t, "hello", val)
	assert.Equal(t, "string", tp)

	val, tp, err = ns.Eval("constant_float")
	assert.NoError(t, err)
	assert.Equal(t, 3.14, val)
	assert.Equal(t, "float", tp)

	val, tp, err = ns.Eval("constant_bool")
	assert.NoError(t, err)
	assert.Equal(t, true, val)
	assert.Equal(t, "bool", tp)

	val, tp, err = ns.Eval("constant_list")
	assert.NoError(t, err)
	assert.Equal(t, any([]any{"a", "b", "c"}), val)
	assert.Equal(t, "string[]", tp)

	val, tp, err = ns.Eval("constant_map")
	assert.NoError(t, err)
	assert.Equal(t, any(map[string]any{"key1": "value1", "key2": 2}), val)
	assert.Equal(t, "map", tp)

	val, tp, err = ns.Eval("constant_date")
	assert.NoError(t, err)
	assert.Equal(t, any(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)), val)
	assert.Equal(t, "timestamp", tp)

	val, tp, err = ns.Eval("constant_uuid")
	assert.NoError(t, err)
	assert.Equal(t, any(uuidVal), val)
	assert.Equal(t, "uuid", tp)

	val, tp, err = ns.Eval("constant_decimal")
	assert.NoError(t, err)
	assert.Equal(t, any(decimal.NewFromInt(100)), val)
	assert.Equal(t, "decimal", tp)
}

func TestNamespace_WithCustomRequestObject(t *testing.T) {

}
