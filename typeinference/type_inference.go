package typeinference

import (
	"errors"

	"github.com/shibukawa/snapsql"
)

// ColumnReference is a field type for column reference (仮実装)
const ColumnReference = 1

// ExpressionType is a field type for expressions (仮実装)
const ExpressionType = 2

// LiteralType is a field type for literal values (仮実装)
const LiteralType = 3

// CaseExprType is a field type for CASE expressions
const CaseExprType = 4

// FunctionType is a field type for function calls
const FunctionType = 5

// WhenClause represents a WHEN ... THEN ... clause in CASE
type WhenClause struct {
	Condition *SelectField
	Result    *SelectField
}

// CaseExpression represents a CASE WHEN ... THEN ... [ELSE ...] END
// ELSEは省略可能
// Whensは1つ以上必須
// ELSEがnilの場合はELSE NULLとみなす
type CaseExpression struct {
	Whens []*WhenClause
	Else  *SelectField // nilならELSE NULL
}

// FunctionCall represents a function call expression (e.g. LENGTH(name), COALESCE(...), CAST(...))
type FunctionCall struct {
	Name     string
	Args     []*SelectField
	CastType string // CAST(... AS type) 用
}

// SelectField mimics parser.SelectField forテスト用
// 本来はparser.SelectFieldを使うが、parserパッケージは変更しない
// 必要なフィールドのみ定義
// Typeはint型でColumnReferenceを使う
// Table, Column, Aliasはstring

type SelectField struct {
	Type       int
	Table      string
	Column     string
	Alias      string
	Expression *Expression
	CaseExpr   *CaseExpression // CASE式用
	FuncCall   *FunctionCall   // 関数呼び出し用
	Subquery   *SelectClause   // サブクエリ用
}

type Expression struct {
	Left     *SelectField
	Right    *SelectField
	Operator string
}

type SchemaStore struct {
	Tables  map[string]*TableInfo
	Dialect snapsql.Dialect // DB方言（postgres, mysql, sqlite など）
}

type TableInfo struct {
	Name    string
	Columns map[string]*ColumnInfo
}

type ColumnInfo struct {
	Name       string
	DataType   string
	IsNullable bool
	MaxLength  *int
	Precision  *int
	Scale      *int
}

type InferenceContext struct {
	Dialect      snapsql.Dialect
	TableAliases map[string]string
}

type TypeInfo struct {
	BaseType   string
	IsNullable bool
	MaxLength  *int
	Precision  *int
	Scale      *int
}

type FieldSource struct {
	Type   string
	Table  string
	Column string
}

type InferredField struct {
	Name   string
	Alias  string
	Type   *TypeInfo
	Source FieldSource
}

type TypeInferenceEngine struct {
	schema *SchemaStore
}

func NewTypeInferenceEngine(schema *SchemaStore) *TypeInferenceEngine {
	return &TypeInferenceEngine{schema: schema}
}

const StarType = 6 // SELECT * or t.* 対応

func (tie *TypeInferenceEngine) InferSelectTypes(selectClause *SelectClause, context *InferenceContext) ([]*InferredField, error) {
	var fields []*InferredField
	for _, field := range selectClause.Fields {
		if field.Type == StarType {
			// SELECT * or t.*
			tableNames := []string{}
			if field.Table != "" {
				tableNames = append(tableNames, field.Table)
			} else {
				// 全テーブル対象（FROM句のテーブル名を列挙する想定。ここではschemaの全テーブルを仮に使う）
				for name := range tie.schema.Tables {
					tableNames = append(tableNames, name)
				}
			}
			colNames, err := tie.GetAllColumnNames(tableNames, context)
			if err != nil {
				return nil, err
			}
			for _, col := range colNames {
				f := &SelectField{Type: ColumnReference, Table: field.Table, Column: col}
				inferred, err := tie.inferFieldType(f, context)
				if err != nil {
					return nil, err
				}
				fields = append(fields, inferred)
			}
			continue
		}
		inferredField, err := tie.inferFieldType(field, context)
		if err != nil {
			return nil, err
		}
		fields = append(fields, inferredField)
	}
	return fields, nil
}

type SelectClause struct {
	Fields []*SelectField
}

func (tie *TypeInferenceEngine) inferFieldType(field *SelectField, context *InferenceContext) (*InferredField, error) {
	if field.Type == ColumnReference {
		return tie.inferColumnType(field, context)
	}
	if field.Type == ExpressionType && field.Expression != nil {
		return tie.inferExpressionType(field, context)
	}
	if field.Type == LiteralType {
		// NULLリテラル（空文字）は型なしとして扱う
		if field.Column == "" {
			return &InferredField{Type: &TypeInfo{BaseType: ""}}, nil
		}
		// 簡易: 数値リテラルかどうかで型を判定
		if isFloatLiteral(field.Column) {
			return &InferredField{Type: &TypeInfo{BaseType: "float"}}, nil
		}
		if isIntLiteral(field.Column) {
			return &InferredField{Type: &TypeInfo{BaseType: "int"}}, nil
		}
		return &InferredField{Type: &TypeInfo{BaseType: "string"}}, nil
	}
	if field.Type == CaseExprType && field.CaseExpr != nil {
		return tie.inferCaseType(field, context)
	}
	if field.Type == FunctionType && field.FuncCall != nil {
		return tie.inferFunctionType(field, context)
	}
	if field.Subquery != nil {
		// サブクエリ: 最初のフィールドの型を返す（スカラサブクエリ想定）
		fields, err := tie.InferSelectTypes(field.Subquery, context)
		if err != nil {
			return nil, err
		}
		if len(fields) == 0 {
			return nil, errors.New("subquery returns no columns")
		}
		return fields[0], nil
	}
	return nil, nil
}

func (tie *TypeInferenceEngine) inferColumnType(field *SelectField, context *InferenceContext) (*InferredField, error) {
	tableName := field.Table
	columnName := field.Column

	// テーブルエイリアス解決
	if real, ok := context.TableAliases[tableName]; ok {
		tableName = real
	}
	table, exists := tie.schema.Tables[tableName]
	if !exists {
		return nil, ErrTableNotFound
	}
	column, exists := table.Columns[columnName]
	if !exists {
		return nil, ErrColumnNotFound
	}

	// pull仕様: DataTypeはすでに正規化済み型
	typeInfo := &TypeInfo{
		BaseType:   column.DataType,
		IsNullable: column.IsNullable,
		MaxLength:  column.MaxLength,
		Precision:  column.Precision,
		Scale:      column.Scale,
	}

	return &InferredField{
		Name:  columnName,
		Alias: field.Alias,
		Type:  typeInfo,
		Source: FieldSource{
			Type:   "ColumnSource",
			Table:  tableName,
			Column: columnName,
		},
	}, nil
}

func (tie *TypeInferenceEngine) inferExpressionType(field *SelectField, context *InferenceContext) (*InferredField, error) {
	expr := field.Expression
	if expr == nil {
		return nil, errors.New("expression is nil")
	}
	// 論理演算子対応
	switch expr.Operator {
	case "AND", "OR", "NOT":
		left, err := tie.inferFieldType(expr.Left, context)
		if err != nil {
			return nil, err
		}
		var isNullable bool = left.Type.IsNullable
		if expr.Operator != "NOT" && expr.Right != nil {
			right, err := tie.inferFieldType(expr.Right, context)
			if err != nil {
				return nil, err
			}
			isNullable = isNullable || right.Type.IsNullable
		}
		return &InferredField{
			Type:   &TypeInfo{BaseType: "bool", IsNullable: isNullable},
			Source: FieldSource{Type: "ExpressionSource"},
		}, nil
	}
	// 比較演算子対応
	switch expr.Operator {
	case "=", "<", ">", "<=", ">=", "<>", "!=":
		left, err := tie.inferFieldType(expr.Left, context)
		if err != nil {
			return nil, err
		}
		right, err := tie.inferFieldType(expr.Right, context)
		if err != nil {
			return nil, err
		}
		isNullable := left.Type.IsNullable || right.Type.IsNullable
		return &InferredField{
			Type:   &TypeInfo{BaseType: "bool", IsNullable: isNullable},
			Source: FieldSource{Type: "ExpressionSource"},
		}, nil
	}
	// 文字列連結の方言対応
	if expr.Operator == "||" || expr.Operator == "CONCAT" {
		dialect := tie.schema.Dialect
		if expr.Operator == "||" && !snapsql.Capabilities[dialect][snapsql.FeatureConcatOperator] {
			return nil, errors.New("'||' operator not supported in this dialect")
		}
		if expr.Operator == "CONCAT" && !snapsql.Capabilities[dialect][snapsql.FeatureConcatFunction] {
			return nil, errors.New("CONCAT() not supported in this dialect")
		}
		left, err := tie.inferFieldType(expr.Left, context)
		if err != nil {
			return nil, err
		}
		right, err := tie.inferFieldType(expr.Right, context)
		if err != nil {
			return nil, err
		}
		isNullable := left.Type.IsNullable || right.Type.IsNullable
		return &InferredField{
			Type:   &TypeInfo{BaseType: "string", IsNullable: isNullable},
			Source: FieldSource{Type: "ExpressionSource"},
		}, nil
	}
	// 左右の型を推論
	left, err := tie.inferFieldType(expr.Left, context)
	if err != nil {
		return nil, err
	}
	right, err := tie.inferFieldType(expr.Right, context)
	if err != nil {
		return nil, err
	}
	// NULLリテラルの型昇格: どちらかがNULLリテラル（BaseType=="string"かつColumn==""）なら、もう一方の型を返す
	isLeftNull := expr.Left != nil && expr.Left.Type == LiteralType && expr.Left.Column == ""
	isRightNull := expr.Right != nil && expr.Right.Type == LiteralType && expr.Right.Column == ""
	if isLeftNull && isRightNull {
		// 両方NULLリテラルならany型(nullable)
		return &InferredField{
			Type:   &TypeInfo{BaseType: "any", IsNullable: true},
			Source: FieldSource{Type: "ExpressionSource"},
		}, nil
	}
	if isLeftNull {
		return &InferredField{
			Type:   &TypeInfo{BaseType: right.Type.BaseType, IsNullable: true},
			Source: FieldSource{Type: "ExpressionSource"},
		}, nil
	}
	if isRightNull {
		return &InferredField{
			Type:   &TypeInfo{BaseType: left.Type.BaseType, IsNullable: true},
			Source: FieldSource{Type: "ExpressionSource"},
		}, nil
	}
	// 型昇格（int/decimal/floatのみ対応、stringはstring優先）
	baseType := promoteTypes(left.Type.BaseType, right.Type.BaseType)
	isNullable := left.Type.IsNullable || right.Type.IsNullable

	return &InferredField{
		Name:  "",
		Alias: field.Alias,
		Type: &TypeInfo{
			BaseType:   baseType,
			IsNullable: isNullable,
		},
		Source: FieldSource{Type: "ExpressionSource"},
	}, nil
}

func (tie *TypeInferenceEngine) inferCaseType(field *SelectField, context *InferenceContext) (*InferredField, error) {
	caseExpr := field.CaseExpr
	if caseExpr == nil || len(caseExpr.Whens) == 0 {
		return nil, errors.New("invalid CASE expression")
	}
	var (
		baseType   string
		isNullable bool
	)
	// すべてのWHEN/ELSEの型を推論し昇格
	for i, when := range caseExpr.Whens {
		res, err := tie.inferFieldType(when.Result, context)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			baseType = res.Type.BaseType
			isNullable = res.Type.IsNullable
		} else {
			if baseType != res.Type.BaseType {
				isNullable = true // 型が異なる
			}
			baseType = promoteTypes(baseType, res.Type.BaseType)
			isNullable = isNullable || res.Type.IsNullable
		}
	}
	// ELSE句
	if caseExpr.Else != nil {
		elseRes, err := tie.inferFieldType(caseExpr.Else, context)
		if err != nil {
			return nil, err
		}
		if baseType != elseRes.Type.BaseType {
			isNullable = true // 型が異なる場合はnullable
		}
		baseType = promoteTypes(baseType, elseRes.Type.BaseType)
		isNullable = isNullable || elseRes.Type.IsNullable
	} else {
		// ELSE省略時はNULL型（nullable）と昇格
		isNullable = true
	}
	return &InferredField{
		Name:  "",
		Alias: field.Alias,
		Type: &TypeInfo{
			BaseType:   baseType,
			IsNullable: isNullable,
		},
		Source: FieldSource{Type: "CaseExpressionSource"},
	}, nil
}

func (tie *TypeInferenceEngine) inferFunctionType(field *SelectField, context *InferenceContext) (*InferredField, error) {
	funcCall := field.FuncCall
	if funcCall == nil {
		return nil, errors.New("function call is nil")
	}
	dialect := tie.schema.Dialect
	name := funcCall.Name
	// 引数の型を推論
	var argTypes []*TypeInfo
	allArgsNull := true
	for _, arg := range funcCall.Args {
		argType, err := tie.inferFieldType(arg, context)
		if err != nil {
			return nil, err
		}
		argTypes = append(argTypes, argType.Type)
		if !(arg.Type == LiteralType && arg.Column == "") {
			allArgsNull = false
		}
	}
	sigs, ok := snapsql.FunctionSignatures[dialect]
	if !ok {
		sigs = snapsql.FunctionSignatures[snapsql.DialectPostgres] // fallback
	}
	sig, ok := sigs[name]
	if !ok {
		// 未知の関数は最初の引数の型・nullableを返す（暫定）
		if len(argTypes) > 0 {
			return &InferredField{
				Type:   &TypeInfo{BaseType: argTypes[0].BaseType, IsNullable: argTypes[0].IsNullable},
				Source: FieldSource{Type: "FunctionCallSource"},
			}, nil
		}
		return &InferredField{
			Type:   &TypeInfo{BaseType: "any", IsNullable: true},
			Source: FieldSource{Type: "FunctionCallSource"},
		}, nil
	}
	// ウインドウ関数系は返却型がstringなら常にstringで返す
	if sig.ReturnType == "string" && (name == "SUM" || name == "AVG" || name == "MIN" || name == "MAX" || name == "FIRST_VALUE" || name == "LAST_VALUE" || name == "LEAD" || name == "LAG") {
		return &InferredField{
			Type:   &TypeInfo{BaseType: "string", IsNullable: sig.Nullable},
			Source: FieldSource{Type: "FunctionCallSource"},
		}, nil
	}
	// ReturnTypeByArg系のみ: すべての引数がNULLリテラルならany型(nullable)
	if sig.ReturnTypeByArg && allArgsNull {
		return &InferredField{
			Type:   &TypeInfo{BaseType: "any", IsNullable: true},
			Source: FieldSource{Type: "FunctionCallSource"},
		}, nil
	}
	// 型・nullable判定
	var baseType string
	var isNullable bool
	if sig.CastType {
		baseType = funcCall.CastType
	} else if sig.ReturnTypeByArg && len(argTypes) > 0 {
		baseType = argTypes[0].BaseType
	} else {
		baseType = sig.ReturnType
	}
	if sig.NullableByArg && len(argTypes) > 0 {
		isNullable = false
		for _, t := range argTypes {
			if t.IsNullable {
				isNullable = true
				break
			}
		}
	} else {
		isNullable = sig.Nullable
	}
	// COALESCE/IFNULL: 型はany型があればany、なければ優先型。nullableは全引数がnullableならtrue、1つでも非nullableがあればfalse
	if (name == "COALESCE" || name == "IFNULL") && len(argTypes) > 0 {
		baseType = "any"
		containsAny := false
		for _, t := range argTypes {
			if t.BaseType == "any" {
				containsAny = true
				break
			}
		}
		if !containsAny {
			typesPriority := []string{"string", "float", "decimal", "int", "any"}
			for _, typ := range typesPriority {
				for _, t := range argTypes {
					if t.BaseType == typ {
						baseType = typ
						break
					}
				}
				if baseType != "any" {
					break
				}
			}
		}
		isNullable = true
		for _, t := range argTypes {
			if !t.IsNullable {
				isNullable = false
				break
			}
		}
	}
	// ARRAY, UNNEST, JSONB_BUILD_OBJECTなどのNullableByArg関数
	if (name == "ARRAY" || name == "UNNEST" || name == "JSONB_BUILD_OBJECT") && len(argTypes) > 0 {
		isNullable = false
		for _, t := range argTypes {
			if t.IsNullable {
				isNullable = true
				break
			}
		}
		baseType = sig.ReturnType
	}
	// 固定型関数で引数が全てNULLリテラルの場合: 型は固定型、nullableのみtrue
	if !sig.ReturnTypeByArg && !sig.CastType && allArgsNull {
		if len(funcCall.Args) > 0 {
			isNullable = true
		}
	} else if !sig.ReturnTypeByArg && !sig.CastType {
		// 固定型関数はnullableを伝搬しない
		isNullable = sig.Nullable
	}
	return &InferredField{
		Type:   &TypeInfo{BaseType: baseType, IsNullable: isNullable},
		Source: FieldSource{Type: "FunctionCallSource"},
	}, nil
}

// ReturningClause for RETURNING句
// Fields: []*SelectField
type ReturningClause struct {
	Fields []*SelectField
}

// InsertStatement for INSERT ... RETURNING
// Table: string, Columns: []string, Values: []*SelectField, Returning: *ReturningClause
type InsertStatement struct {
	Table     string
	Columns   []string
	Values    []*SelectField
	Returning *ReturningClause
}

// UpdateStatement for UPDATE ... RETURNING
// Table: string, Set: map[string]*SelectField, Where: *SelectField, Returning: *ReturningClause
type UpdateStatement struct {
	Table     string
	Set       map[string]*SelectField
	Where     *SelectField
	Returning *ReturningClause
}

// InferReturningTypes infers types for RETURNING clause in Insert/Update statements
func (tie *TypeInferenceEngine) InferReturningTypes(returning *ReturningClause, context *InferenceContext) ([]*InferredField, error) {
	if returning == nil {
		return nil, nil
	}
	return tie.InferSelectTypes(&SelectClause{Fields: returning.Fields}, context)
}

// GetAllColumnNames returns all column names for the given table names or aliases.
func (tie *TypeInferenceEngine) GetAllColumnNames(tableNamesOrAliases []string, context *InferenceContext) ([]string, error) {
	var columns []string
	seen := map[string]struct{}{}
	for _, name := range tableNamesOrAliases {
		tableName := name
		if real, ok := context.TableAliases[name]; ok {
			tableName = real
		}
		table, exists := tie.schema.Tables[tableName]
		if !exists {
			return nil, ErrTableNotFound
		}
		for col := range table.Columns {
			if _, exists := seen[col]; !exists {
				columns = append(columns, col)
				seen[col] = struct{}{}
			}
		}
	}
	return columns, nil
}

// GetAllColumnsWithTypeInfo returns all columns (with type info) for the given table names or aliases.
func (tie *TypeInferenceEngine) GetAllColumnsWithTypeInfo(tableNamesOrAliases []string, context *InferenceContext) ([]*InferredField, error) {
	var fields []*InferredField
	seen := map[string]struct{}{}
	for _, name := range tableNamesOrAliases {
		tableName := name
		if real, ok := context.TableAliases[name]; ok {
			tableName = real
		}
		table, exists := tie.schema.Tables[tableName]
		if !exists {
			return nil, ErrTableNotFound
		}
		for colName, colInfo := range table.Columns {
			if _, exists := seen[colName]; !exists {
				field := &InferredField{
					Name:  colName,
					Alias: "",
					Type: &TypeInfo{
						BaseType:   colInfo.DataType,
						IsNullable: colInfo.IsNullable,
						MaxLength:  colInfo.MaxLength,
						Precision:  colInfo.Precision,
						Scale:      colInfo.Scale,
					},
					Source: FieldSource{
						Type:   "ColumnSource",
						Table:  tableName,
						Column: colName,
					},
				}
				fields = append(fields, field)
				seen[colName] = struct{}{}
			}
		}
	}
	return fields, nil
}

func promoteTypes(left, right string) string {
	if left == "" && right == "" {
		return "any"
	}
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	// any型は最下位: もう一方がanyでなければそちらを優先
	if left == "any" && right != "any" {
		return right
	}
	if right == "any" && left != "any" {
		return left
	}
	if left == "string" || right == "string" {
		return "string"
	}
	if left == "float" || right == "float" {
		return "float"
	}
	if left == "decimal" || right == "decimal" {
		return "decimal"
	}
	return "int"
}

func isFloatLiteral(s string) bool {
	// 簡易実装: 小数点を含む場合float
	for _, c := range s {
		if c == '.' {
			return true
		}
	}
	return false
}

func isIntLiteral(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

var (
	ErrTableNotFound  = errors.New("table not found")
	ErrColumnNotFound = errors.New("column not found")
)
