package pygen

import (
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

func TestProcessParameters(t *testing.T) {
	tests := []struct {
		name       string
		parameters []intermediate.Parameter
		want       []parameterData
		wantErr    bool
	}{
		{
			name:       "empty parameters",
			parameters: []intermediate.Parameter{},
			want:       []parameterData{},
			wantErr:    false,
		},
		{
			name: "single required parameter",
			parameters: []intermediate.Parameter{
				{
					Name:        "userId",
					Type:        "int",
					Optional:    false,
					Description: "User ID",
				},
			},
			want: []parameterData{
				{
					Name:        "user_id",
					TypeHint:    "int",
					Description: "User ID",
					HasDefault:  false,
					Default:     "",
				},
			},
			wantErr: false,
		},
		{
			name: "optional parameter",
			parameters: []intermediate.Parameter{
				{
					Name:        "limit",
					Type:        "int",
					Optional:    true,
					Description: "Result limit",
				},
			},
			want: []parameterData{
				{
					Name:        "limit",
					TypeHint:    "int",
					Description: "Result limit",
					HasDefault:  true,
					Default:     "None",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple parameters with different types",
			parameters: []intermediate.Parameter{
				{
					Name:        "userId",
					Type:        "int",
					Optional:    false,
					Description: "User ID",
				},
				{
					Name:        "username",
					Type:        "string",
					Optional:    false,
					Description: "Username",
				},
				{
					Name:        "isActive",
					Type:        "bool",
					Optional:    true,
					Description: "Active status",
				},
			},
			want: []parameterData{
				{
					Name:        "user_id",
					TypeHint:    "int",
					Description: "User ID",
					HasDefault:  false,
				},
				{
					Name:        "username",
					TypeHint:    "str",
					Description: "Username",
					HasDefault:  false,
				},
				{
					Name:        "is_active",
					TypeHint:    "bool",
					Description: "Active status",
					HasDefault:  true,
					Default:     "None",
				},
			},
			wantErr: false,
		},
		{
			name: "array parameter",
			parameters: []intermediate.Parameter{
				{
					Name:        "userIds",
					Type:        "int[]",
					Optional:    false,
					Description: "List of user IDs",
				},
			},
			want: []parameterData{
				{
					Name:        "user_ids",
					TypeHint:    "List[int]",
					Description: "List of user IDs",
					HasDefault:  false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				Format: &intermediate.IntermediateFormat{
					Parameters: tt.parameters,
				},
			}

			got, err := g.processParameters()
			if (err != nil) != tt.wantErr {
				t.Errorf("processParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("processParameters() got %d parameters, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("parameter[%d].Name = %v, want %v", i, got[i].Name, tt.want[i].Name)
				}

				if got[i].TypeHint != tt.want[i].TypeHint {
					t.Errorf("parameter[%d].TypeHint = %v, want %v", i, got[i].TypeHint, tt.want[i].TypeHint)
				}

				if got[i].HasDefault != tt.want[i].HasDefault {
					t.Errorf("parameter[%d].HasDefault = %v, want %v", i, got[i].HasDefault, tt.want[i].HasDefault)
				}

				if got[i].HasDefault && got[i].Default != tt.want[i].Default {
					t.Errorf("parameter[%d].Default = %v, want %v", i, got[i].Default, tt.want[i].Default)
				}
			}
		})
	}
}

func TestProcessImplicitParameters(t *testing.T) {
	tests := []struct {
		name       string
		parameters []intermediate.ImplicitParameter
		want       []implicitParamData
		wantErr    bool
	}{
		{
			name:       "empty implicit parameters",
			parameters: []intermediate.ImplicitParameter{},
			want:       []implicitParamData{},
			wantErr:    false,
		},
		{
			name: "created_by without default",
			parameters: []intermediate.ImplicitParameter{
				{
					Name: "createdBy",
					Type: "string",
				},
			},
			want: []implicitParamData{
				{
					Name:            "created_by",
					TypeHint:        "str",
					Description:     "System column: created_by (from context if not provided)",
					HasDefaultValue: false,
				},
			},
			wantErr: false,
		},
		{
			name: "created_at with default",
			parameters: []intermediate.ImplicitParameter{
				{
					Name:    "createdAt",
					Type:    "timestamp",
					Default: "datetime.now()",
				},
			},
			want: []implicitParamData{
				{
					Name:            "created_at",
					TypeHint:        "datetime",
					Description:     "System column: created_at (from context if not provided)",
					HasDefaultValue: true,
					DefaultValue:    "datetime.now()",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple implicit parameters",
			parameters: []intermediate.ImplicitParameter{
				{
					Name: "createdBy",
					Type: "string",
				},
				{
					Name: "updatedBy",
					Type: "string",
				},
				{
					Name:    "createdAt",
					Type:    "timestamp",
					Default: "datetime.now()",
				},
			},
			want: []implicitParamData{
				{
					Name:            "created_by",
					TypeHint:        "str",
					Description:     "System column: created_by (from context if not provided)",
					HasDefaultValue: false,
				},
				{
					Name:            "updated_by",
					TypeHint:        "str",
					Description:     "System column: updated_by (from context if not provided)",
					HasDefaultValue: false,
				},
				{
					Name:            "created_at",
					TypeHint:        "datetime",
					Description:     "System column: created_at (from context if not provided)",
					HasDefaultValue: true,
					DefaultValue:    "datetime.now()",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				Format: &intermediate.IntermediateFormat{
					ImplicitParameters: tt.parameters,
				},
			}

			got, err := g.processImplicitParameters()
			if (err != nil) != tt.wantErr {
				t.Errorf("processImplicitParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("processImplicitParameters() got %d parameters, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("parameter[%d].Name = %v, want %v", i, got[i].Name, tt.want[i].Name)
				}

				if got[i].TypeHint != tt.want[i].TypeHint {
					t.Errorf("parameter[%d].TypeHint = %v, want %v", i, got[i].TypeHint, tt.want[i].TypeHint)
				}

				if got[i].HasDefaultValue != tt.want[i].HasDefaultValue {
					t.Errorf("parameter[%d].HasDefaultValue = %v, want %v", i, got[i].HasDefaultValue, tt.want[i].HasDefaultValue)
				}
			}
		})
	}
}

func TestProcessValidations(t *testing.T) {
	tests := []struct {
		name   string
		params []parameterData
		want   int // number of validations expected
	}{
		{
			name:   "no parameters",
			params: []parameterData{},
			want:   0,
		},
		{
			name: "all optional parameters",
			params: []parameterData{
				{Name: "limit", TypeHint: "int", HasDefault: true},
				{Name: "offset", TypeHint: "int", HasDefault: true},
			},
			want: 0,
		},
		{
			name: "all required parameters",
			params: []parameterData{
				{Name: "user_id", TypeHint: "int", HasDefault: false},
				{Name: "username", TypeHint: "str", HasDefault: false},
			},
			want: 2,
		},
		{
			name: "mixed required and optional",
			params: []parameterData{
				{Name: "user_id", TypeHint: "int", HasDefault: false},
				{Name: "limit", TypeHint: "int", HasDefault: true},
				{Name: "username", TypeHint: "str", HasDefault: false},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got := g.processValidations(tt.params)

			if len(got) != tt.want {
				t.Errorf("processValidations() got %d validations, want %d", len(got), tt.want)
			}

			// Check that each validation has proper structure
			for i, v := range got {
				if v.Condition == "" {
					t.Errorf("validation[%d].Condition is empty", i)
				}

				if v.Message == "" {
					t.Errorf("validation[%d].Message is empty", i)
				}
			}
		})
	}
}

func TestFormatDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		typeName string
		want     string
	}{
		{"nil value", nil, "string", "None"},
		{"string value", "default", "string", `"default"`},
		{"code expression with parentheses", "datetime.now()", "timestamp", "datetime.now()"},
		{"code expression with dots", "time.time()", "timestamp", "time.time()"},
		{"int value", 42, "int", "42"},
		{"int64 value", int64(100), "int64", "100"},
		{"float value", 3.14, "float", "3.14"},
		{"bool true", true, "bool", "True"},
		{"bool false", false, "bool", "False"},
		{"empty string", "", "string", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDefaultValue(tt.value, tt.typeName)
			if got != tt.want {
				t.Errorf("formatDefaultValue(%v, %q) = %q, want %q", tt.value, tt.typeName, got, tt.want)
			}
		})
	}
}
