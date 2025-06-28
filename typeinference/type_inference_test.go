package typeinference

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
)

func TestInferSimpleColumnSelection(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"id":         {Name: "id", DataType: "int", IsNullable: false},
					"name":       {Name: "name", DataType: "string", IsNullable: false, MaxLength: intPtr(255)},
					"email":      {Name: "email", DataType: "string", IsNullable: true, MaxLength: intPtr(255)},
					"created_at": {Name: "created_at", DataType: "time", IsNullable: true},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}
	tests := []struct {
		name       string
		field      *SelectField
		expectName string
		expectType string
		expectNull bool
		maxLength  *int
	}{
		{"id", &SelectField{Type: ColumnReference, Table: "users", Column: "id"}, "id", "int", false, nil},
		{"name", &SelectField{Type: ColumnReference, Table: "users", Column: "name"}, "name", "string", false, intPtr(255)},
		{"email", &SelectField{Type: ColumnReference, Table: "users", Column: "email"}, "email", "string", true, intPtr(255)},
		{"created_at", &SelectField{Type: ColumnReference, Table: "users", Column: "created_at"}, "created_at", "time", true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			fields, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(fields))
			assert.Equal(t, tt.expectName, fields[0].Name)
			assert.Equal(t, tt.expectType, fields[0].Type.BaseType)
			assert.Equal(t, tt.expectNull, fields[0].Type.IsNullable)
			if tt.maxLength != nil {
				assert.Equal(t, *tt.maxLength, *fields[0].Type.MaxLength)
			}
		})
	}
}

func TestInferJoinWithAliases(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"id":            {Name: "id", DataType: "int", IsNullable: false},
					"name":          {Name: "name", DataType: "string", IsNullable: false, MaxLength: intPtr(255)},
					"department_id": {Name: "department_id", DataType: "int", IsNullable: true},
					"salary":        {Name: "salary", DataType: "decimal", IsNullable: true, Precision: intPtr(10), Scale: intPtr(2)},
				},
			},
			"departments": {
				Name: "departments",
				Columns: map[string]*ColumnInfo{
					"id":   {Name: "id", DataType: "int", IsNullable: false},
					"name": {Name: "name", DataType: "string", IsNullable: false, MaxLength: intPtr(100)},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect: snapsql.DialectPostgres,
		TableAliases: map[string]string{
			"u": "users",
			"d": "departments",
		},
	}
	tests := []struct {
		name       string
		field      *SelectField
		expectName string
		expectType string
		expectNull bool
		alias      string
		maxLength  *int
	}{
		{"u.id", &SelectField{Type: ColumnReference, Table: "u", Column: "id"}, "id", "int", false, "", nil},
		{"u.name", &SelectField{Type: ColumnReference, Table: "u", Column: "name"}, "name", "string", false, "", intPtr(255)},
		{"d.name as department_name", &SelectField{Type: ColumnReference, Table: "d", Column: "name", Alias: "department_name"}, "name", "string", false, "department_name", intPtr(100)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			fields, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(fields))
			assert.Equal(t, tt.expectName, fields[0].Name)
			assert.Equal(t, tt.expectType, fields[0].Type.BaseType)
			assert.Equal(t, tt.expectNull, fields[0].Type.IsNullable)
			if tt.alias != "" {
				assert.Equal(t, tt.alias, fields[0].Alias)
			}
			if tt.maxLength != nil {
				assert.Equal(t, *tt.maxLength, *fields[0].Type.MaxLength)
			}
		})
	}
}

// 仮の式型（本来はparser.Expressionだが、ここで最小限定義）
// type Expression struct {
// 	Left     *SelectField
// 	Right    *SelectField
// 	Operator string
// }

// const ExpressionType = 2

// SelectFieldにExpression用フィールドを追加（テスト用）
// type SelectFieldWithExpr struct {
// 	SelectField
// 	Expression *Expression
// }

func TestInferArithmeticExpressions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"salary": {Name: "salary", DataType: "decimal", IsNullable: false, Precision: intPtr(10), Scale: intPtr(2)},
					"bonus":  {Name: "bonus", DataType: "float", IsNullable: true},
					"count":  {Name: "count", DataType: "int", IsNullable: false},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}
	tests := []struct {
		name  string
		field *SelectField
	}{
		{"salary * 1.1", &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "salary"}, Right: &SelectField{Type: LiteralType, Column: "1.1"}, Operator: "*"}}},
		{"count + 1", &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "count"}, Right: &SelectField{Type: LiteralType, Column: "1"}, Operator: "+"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			_, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
		})
	}
}

func TestInferStringExpressions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"first_name": {Name: "first_name", DataType: "string", IsNullable: false, MaxLength: intPtr(50)},
					"last_name":  {Name: "last_name", DataType: "string", IsNullable: false, MaxLength: intPtr(50)},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}

	// first_name || ' ' || last_name (PostgreSQL)
	fieldFirst := &SelectField{Type: ColumnReference, Table: "users", Column: "first_name"}
	fieldSpace := &SelectField{Type: LiteralType, Column: " "}
	fieldLast := &SelectField{Type: ColumnReference, Table: "users", Column: "last_name"}

	expr1 := &Expression{Left: fieldFirst, Right: fieldSpace, Operator: "||"}
	fieldExpr1 := &SelectField{Type: ExpressionType, Expression: expr1}

	expr2 := &Expression{Left: fieldExpr1, Right: fieldLast, Operator: "||"}
	fieldExpr2 := &SelectField{Type: ExpressionType, Expression: expr2}

	selectClause := &SelectClause{
		Fields: []*SelectField{fieldExpr2},
	}

	fields, err := engine.InferSelectTypes(selectClause, context)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "string", fields[0].Type.BaseType)
}

func TestInferComparisonExpressions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"salary": {Name: "salary", DataType: "decimal", IsNullable: false},
					"name":   {Name: "name", DataType: "string", IsNullable: false},
					"count":  {Name: "count", DataType: "int", IsNullable: false},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}
	tests := []struct {
		name  string
		field *SelectField
	}{
		{"salary > 1000", &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "salary"}, Right: &SelectField{Type: LiteralType, Column: "1000"}, Operator: ">"}}},
		{"name = 'foo'", &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "name"}, Right: &SelectField{Type: LiteralType, Column: "foo"}, Operator: "="}}},
		{"count <> 0", &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "count"}, Right: &SelectField{Type: LiteralType, Column: "0"}, Operator: "<>"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			fields, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(fields))
			assert.Equal(t, "bool", fields[0].Type.BaseType)
		})
	}
}

func TestInferLogicalExpressions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"active": {Name: "active", DataType: "bool", IsNullable: false},
					"salary": {Name: "salary", DataType: "decimal", IsNullable: false},
					"count":  {Name: "count", DataType: "int", IsNullable: false},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}
	tests := []struct {
		name  string
		field *SelectField
	}{
		{"active = true AND salary > 1000", &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "active"}, Right: &SelectField{Type: LiteralType, Column: "true"}, Operator: "="}}, Right: &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "salary"}, Right: &SelectField{Type: LiteralType, Column: "1000"}, Operator: ">"}}, Operator: "AND"}}},
		{"NOT (count = 0)", &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "count"}, Right: &SelectField{Type: LiteralType, Column: "0"}, Operator: "="}}, Operator: "NOT"}}},
		{"a > 1 OR b < 2", &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "salary"}, Right: &SelectField{Type: LiteralType, Column: "1"}, Operator: ">"}}, Right: &SelectField{Type: ExpressionType, Expression: &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "count"}, Right: &SelectField{Type: LiteralType, Column: "2"}, Operator: "<"}}, Operator: "OR"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			fields, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(fields))
			assert.Equal(t, "bool", fields[0].Type.BaseType)
		})
	}
}

func TestInferCaseExpressions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"count": {Name: "count", DataType: "int", IsNullable: false},
					"name":  {Name: "name", DataType: "string", IsNullable: false},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}

	// CASE WHEN count > 0 THEN 1 ELSE 0 END → int
	cond1 := &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "count"}, Right: &SelectField{Type: LiteralType, Column: "0"}, Operator: ">"}
	caseExpr1 := &CaseExpression{
		Whens: []*WhenClause{{
			Condition: &SelectField{Type: ExpressionType, Expression: cond1},
			Result:    &SelectField{Type: LiteralType, Column: "1"},
		}},
		Else: &SelectField{Type: LiteralType, Column: "0"},
	}
	fieldCase1 := &SelectField{Type: CaseExprType, CaseExpr: caseExpr1}

	// CASE WHEN name = 'foo' THEN 'yes' ELSE 'no' END → string
	cond2 := &Expression{Left: &SelectField{Type: ColumnReference, Table: "users", Column: "name"}, Right: &SelectField{Type: LiteralType, Column: "foo"}, Operator: "="}
	caseExpr2 := &CaseExpression{
		Whens: []*WhenClause{{
			Condition: &SelectField{Type: ExpressionType, Expression: cond2},
			Result:    &SelectField{Type: LiteralType, Column: "yes"},
		}},
		Else: &SelectField{Type: LiteralType, Column: "no"},
	}
	fieldCase2 := &SelectField{Type: CaseExprType, CaseExpr: caseExpr2}

	// CASE WHEN count > 0 THEN 1 ELSE 'none' END → string（int/string混合はstring昇格）
	caseExpr3 := &CaseExpression{
		Whens: []*WhenClause{{
			Condition: &SelectField{Type: ExpressionType, Expression: cond1},
			Result:    &SelectField{Type: LiteralType, Column: "1"},
		}},
		Else: &SelectField{Type: LiteralType, Column: "none"},
	}
	fieldCase3 := &SelectField{Type: CaseExprType, CaseExpr: caseExpr3}

	// CASE WHEN count > 0 THEN 1 END → int（ELSE省略時はnull型との昇格）
	caseExpr4 := &CaseExpression{
		Whens: []*WhenClause{{
			Condition: &SelectField{Type: ExpressionType, Expression: cond1},
			Result:    &SelectField{Type: LiteralType, Column: "1"},
		}},
		Else: nil,
	}
	fieldCase4 := &SelectField{Type: CaseExprType, CaseExpr: caseExpr4}

	selectClause := &SelectClause{
		Fields: []*SelectField{fieldCase1, fieldCase2, fieldCase3, fieldCase4},
	}

	fields, err := engine.InferSelectTypes(selectClause, context)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(fields))
	assert.Equal(t, "int", fields[0].Type.BaseType)
	assert.Equal(t, false, fields[0].Type.IsNullable)
	assert.Equal(t, "string", fields[1].Type.BaseType)
	assert.Equal(t, false, fields[1].Type.IsNullable)
	assert.Equal(t, "string", fields[2].Type.BaseType) // int/string混合はstring昇格
	assert.Equal(t, true, fields[2].Type.IsNullable)   // どちらかがnullableならnullable
	assert.Equal(t, "int", fields[3].Type.BaseType)
	assert.Equal(t, true, fields[3].Type.IsNullable) // ELSE省略時はnullable
}

func TestInferFunctionExpressions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"name":   {Name: "name", DataType: "string", IsNullable: false, MaxLength: intPtr(255)},
					"email":  {Name: "email", DataType: "string", IsNullable: true, MaxLength: intPtr(255)},
					"salary": {Name: "salary", DataType: "decimal", IsNullable: true, Precision: intPtr(10), Scale: intPtr(2)},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}

	fieldName := &SelectField{Type: ColumnReference, Table: "users", Column: "name"}
	fieldEmail := &SelectField{Type: ColumnReference, Table: "users", Column: "email"}
	fieldSalary := &SelectField{Type: ColumnReference, Table: "users", Column: "salary"}
	fieldUnknown := &SelectField{Type: LiteralType, Column: "unknown"}

	tests := []struct {
		name       string
		field      *SelectField
		expectType string
		expectNull bool
	}{
		{
			name:       "LENGTH(name)",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "LENGTH", Args: []*SelectField{fieldName}}},
			expectType: "int",
			expectNull: false,
		},
		{
			name:       "COALESCE(email, 'unknown')",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "COALESCE", Args: []*SelectField{fieldEmail, fieldUnknown}}},
			expectType: "string",
			expectNull: false,
		},
		{
			name:       "CAST(salary AS int)",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "CAST", Args: []*SelectField{fieldSalary}, CastType: "int"}},
			expectType: "int",
			expectNull: true, // salaryはnullable
		},
		{
			name:       "UPPER(name)",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "UPPER", Args: []*SelectField{fieldName}}},
			expectType: "string",
			expectNull: false,
		},
		{
			name:       "NOW()",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "NOW", Args: []*SelectField{}}},
			expectType: "time",
			expectNull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			fields, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(fields))
			assert.Equal(t, tt.expectType, fields[0].Type.BaseType)
			assert.Equal(t, tt.expectNull, fields[0].Type.IsNullable)
		})
	}
}

func TestInferMoreFunctionExpressions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"created_at": {Name: "created_at", DataType: "time", IsNullable: false},
					"name":       {Name: "name", DataType: "string", IsNullable: false, MaxLength: intPtr(255)},
					"meta":       {Name: "meta", DataType: "any", IsNullable: true},
					"tags":       {Name: "tags", DataType: "array", IsNullable: true},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}

	fieldCreatedAt := &SelectField{Type: ColumnReference, Table: "users", Column: "created_at"}
	fieldName := &SelectField{Type: ColumnReference, Table: "users", Column: "name"}
	fieldMeta := &SelectField{Type: ColumnReference, Table: "users", Column: "meta"}
	fieldTags := &SelectField{Type: ColumnReference, Table: "users", Column: "tags"}
	fieldStr := &SelectField{Type: LiteralType, Column: "abc"}

	tests := []struct {
		name       string
		field      *SelectField
		expectType string
		expectNull bool
	}{
		{
			name:       "DATE_ADD(created_at, INTERVAL '1 day')",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "DATE_ADD", Args: []*SelectField{fieldCreatedAt, fieldStr}}},
			expectType: "time",
			expectNull: false,
		},
		{
			name:       "SUBSTRING(name, 1, 2)",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "SUBSTRING", Args: []*SelectField{fieldName, {Type: LiteralType, Column: "1"}, {Type: LiteralType, Column: "2"}}}},
			expectType: "string",
			expectNull: false,
		},

		{
			name:       "TRIM(name)",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "TRIM", Args: []*SelectField{fieldName}}},
			expectType: "string",
			expectNull: false,
		},
		{
			name:       "IFNULL(meta, '{}')",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "IFNULL", Args: []*SelectField{fieldMeta, {Type: LiteralType, Column: "{}"}}}},
			expectType: "any",
			expectNull: false,
		},
		{
			name:       "JSONB_BUILD_OBJECT('k', name)",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "JSONB_BUILD_OBJECT", Args: []*SelectField{{Type: LiteralType, Column: "k"}, fieldName}}},
			expectType: "any",
			expectNull: false,
		},
		{
			name:       "ARRAY[name, name]",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "ARRAY", Args: []*SelectField{fieldName, fieldName}}},
			expectType: "array",
			expectNull: false,
		},
		{
			name:       "UNNEST(tags)",
			field:      &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "UNNEST", Args: []*SelectField{fieldTags}}},
			expectType: "any",
			expectNull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			fields, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(fields))
			assert.Equal(t, tt.expectType, fields[0].Type.BaseType)
			assert.Equal(t, tt.expectNull, fields[0].Type.IsNullable)
		})
	}
}

func TestNullPropagationExpressions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"email": {Name: "email", DataType: "string", IsNullable: true},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}
	nullLit := &SelectField{Type: LiteralType, Column: ""}
	oneLit := &SelectField{Type: LiteralType, Column: "1"}
	emailCol := &SelectField{Type: ColumnReference, Table: "users", Column: "email"}
	tests := []struct {
		name       string
		field      *SelectField
		expectType string
		expectNull bool
	}{
		{"NULL + 1", &SelectField{Type: ExpressionType, Expression: &Expression{Left: nullLit, Right: oneLit, Operator: "+"}}, "int", true},
		{"UPPER(NULL)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "UPPER", Args: []*SelectField{nullLit}}}, "string", true},
		{"email = NULL", &SelectField{Type: ExpressionType, Expression: &Expression{Left: emailCol, Right: nullLit, Operator: "="}}, "bool", true},
		{"COALESCE(NULL, 1)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "COALESCE", Args: []*SelectField{nullLit, oneLit}}}, "int", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			fields, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(fields))
			assert.Equal(t, tt.expectType, fields[0].Type.BaseType)
			assert.Equal(t, tt.expectNull, fields[0].Type.IsNullable)
		})
	}
}

func TestInferSubqueryField(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"id":   {Name: "id", DataType: "int", IsNullable: false},
					"name": {Name: "name", DataType: "string", IsNullable: false},
				},
			},
			"posts": {
				Name: "posts",
				Columns: map[string]*ColumnInfo{
					"user_id": {Name: "user_id", DataType: "int", IsNullable: false},
					"title":   {Name: "title", DataType: "string", IsNullable: false},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}

	// サブクエリ: (SELECT name FROM users WHERE id = posts.user_id)
	subquery := &SelectClause{
		Fields: []*SelectField{
			{Type: ColumnReference, Table: "users", Column: "name"},
		},
	}
	fieldSubquery := &SelectField{Subquery: subquery}

	selectClause := &SelectClause{
		Fields: []*SelectField{fieldSubquery},
	}

	fields, err := engine.InferSelectTypes(selectClause, context)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fields))
	assert.Equal(t, "string", fields[0].Type.BaseType)
	assert.Equal(t, false, fields[0].Type.IsNullable)
}

func TestInferWindowFunctions(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"id":    {Name: "id", DataType: "int", IsNullable: false},
					"score": {Name: "score", DataType: "float", IsNullable: true},
					"group": {Name: "group", DataType: "string", IsNullable: false},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}

	fieldScore := &SelectField{Type: ColumnReference, Table: "users", Column: "score"}

	tests := []struct {
		name       string
		field      *SelectField
		expectType string
		expectNull bool
	}{
		{"ROW_NUMBER() OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "ROW_NUMBER", Args: []*SelectField{}}}, "int", false},
		{"RANK() OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "RANK", Args: []*SelectField{}}}, "int", false},
		{"DENSE_RANK() OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "DENSE_RANK", Args: []*SelectField{}}}, "int", false},
		{"SUM(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "SUM", Args: []*SelectField{fieldScore}}}, "float", true},
		{"AVG(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "AVG", Args: []*SelectField{fieldScore}}}, "float", true},
		{"COUNT(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "COUNT", Args: []*SelectField{fieldScore}}}, "int", false},
		{"MIN(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "MIN", Args: []*SelectField{fieldScore}}}, "float", true},
		{"MAX(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "MAX", Args: []*SelectField{fieldScore}}}, "float", true},
		{"FIRST_VALUE(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "FIRST_VALUE", Args: []*SelectField{fieldScore}}}, "float", true},
		{"LAST_VALUE(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "LAST_VALUE", Args: []*SelectField{fieldScore}}}, "float", true},
		{"LEAD(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "LEAD", Args: []*SelectField{fieldScore}}}, "float", true},
		{"LAG(score) OVER (...)", &SelectField{Type: FunctionType, FuncCall: &FunctionCall{Name: "LAG", Args: []*SelectField{fieldScore}}}, "float", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectClause := &SelectClause{Fields: []*SelectField{tt.field}}
			fields, err := engine.InferSelectTypes(selectClause, context)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(fields))
			assert.Equal(t, tt.expectType, fields[0].Type.BaseType)
			assert.Equal(t, tt.expectNull, fields[0].Type.IsNullable)
		})
	}
}

func TestInferInsertUpdateReturning(t *testing.T) {
	schema := &SchemaStore{
		Tables: map[string]*TableInfo{
			"users": {
				Name: "users",
				Columns: map[string]*ColumnInfo{
					"id":   {Name: "id", DataType: "int", IsNullable: false},
					"name": {Name: "name", DataType: "string", IsNullable: false, MaxLength: intPtr(255)},
					"age":  {Name: "age", DataType: "int", IsNullable: true},
				},
			},
		},
		Dialect: snapsql.DialectPostgres,
	}
	engine := NewTypeInferenceEngine(schema)
	context := &InferenceContext{
		Dialect:      snapsql.DialectPostgres,
		TableAliases: map[string]string{},
	}
	insert := &InsertStatement{
		Table:   "users",
		Columns: []string{"name", "age"},
		Values: []*SelectField{
			{Type: LiteralType, Column: "'Alice'"},
			{Type: LiteralType, Column: "30"},
		},
		Returning: &ReturningClause{
			Fields: []*SelectField{
				{Type: ColumnReference, Table: "users", Column: "id"},
				{Type: ColumnReference, Table: "users", Column: "name"},
			},
		},
	}
	update := &UpdateStatement{
		Table: "users",
		Set: map[string]*SelectField{
			"age": {Type: LiteralType, Column: "31"},
		},
		Where: &SelectField{Type: ColumnReference, Table: "users", Column: "name"},
		Returning: &ReturningClause{
			Fields: []*SelectField{
				{Type: ColumnReference, Table: "users", Column: "id"},
				{Type: ColumnReference, Table: "users", Column: "age"},
			},
		},
	}
	tests := []struct {
		name       string
		returning  *ReturningClause
		expectType []string
	}{
		{"insert returning", insert.Returning, []string{"int", "string"}},
		{"update returning", update.Returning, []string{"int", "int"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := engine.InferReturningTypes(tt.returning, context)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.expectType), len(fields))
			for i, typ := range tt.expectType {
				assert.Equal(t, typ, fields[i].Type.BaseType)
			}
		})
	}
}

func intPtr(i int) *int { return &i }
