package parser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
)

// Namespace manages namespace information and CEL functionality in an integrated manner
type Namespace struct {
	Constants    map[string]any `yaml:"env"`
	Schema       *InterfaceSchema
	celVariables map[string]any // CEL evaluation variables
	envCEL       *cel.Env       // CEL environment for environment variables
	paramCEL     *cel.Env       // CEL environment for parameters
	dummyData    map[string]any // dummy data environment
}

// NewNamespace creates a new namespace
// If schema is nil, creates an empty InterfaceSchema
func NewNamespace(schema *InterfaceSchema) *Namespace {
	// Create empty schema if nil
	if schema == nil {
		schema = &InterfaceSchema{
			Parameters: make(map[string]any),
		}
	}

	dummyData := generateDummyDataFromSchema(schema)

	ns := &Namespace{
		Constants:    make(map[string]any),
		celVariables: make(map[string]any),
		Schema:       schema,
		dummyData:    dummyData,
	}

	// Initialize CEL engines
	if err := ns.initializeCELEngines(); err != nil {
		panic(err)
	}

	return ns
}

// SetConstant sets a namespace constant
func (ns *Namespace) SetConstant(key string, value any) {
	ns.Constants[key] = value

	// Reinitialize CEL engines if they are already initialized
	if ns.envCEL != nil {
		ns.createEnvironmentCEL()
	}
}

// AddLoopVariable temporarily adds a loop variable
func (ns *Namespace) AddLoopVariable(variable string, valueType string) {
	if ns.Schema.Parameters == nil {
		ns.Schema.Parameters = make(map[string]any)
	}
	ns.Schema.Parameters[variable] = valueType

	// Reinitialize CEL engines
	if err := ns.initializeCELEngines(); err != nil {
		// Debug: output error
		fmt.Printf("DEBUG: CEL engine initialization failed in AddLoopVariable: %v\n", err)
	}
}

// RemoveLoopVariable removes a loop variable
func (ns *Namespace) RemoveLoopVariable(variable string) {
	if ns.Schema.Parameters != nil {
		delete(ns.Schema.Parameters, variable)
	}

	if err := ns.initializeCELEngines(); err != nil {
		panic(err)
	}
}

// Copy creates a copy of the namespace
func (ns *Namespace) Copy() *Namespace {
	// Copy schema
	schemaCopy := &InterfaceSchema{
		Name:        ns.Schema.Name,
		Description: ns.Schema.Description,
		Parameters:  make(map[string]any),
	}

	// Copy parameters
	for k, v := range ns.Schema.Parameters {
		schemaCopy.Parameters[k] = v
	}

	// Copy constants
	constantsCopy := make(map[string]any)
	for k, v := range ns.Constants {
		constantsCopy[k] = v
	}

	// Copy dummy data
	dummyDataCopy := make(map[string]any)
	for k, v := range ns.dummyData {
		dummyDataCopy[k] = v
	}

	// Create new namespace
	newNs := &Namespace{
		Constants:    constantsCopy,
		Schema:       schemaCopy,
		celVariables: make(map[string]any),
		dummyData:    dummyDataCopy,
	}

	// CELエンジンを初期化
	if err := newNs.initializeCELEngines(); err != nil {
		panic(err)
	}

	return newNs
}

// AddLoopVariableToNew creates a new namespace with added loop variable
func (ns *Namespace) AddLoopVariableToNew(variable string, valueType string) *Namespace {
	newNs := ns.Copy()
	newNs.AddLoopVariable(variable, valueType)
	return newNs
}

// GetConstant retrieves namespace constants
func (ns *Namespace) GetConstant(key string) (any, bool) {
	value, exists := ns.Constants[key]
	return value, exists
}

// SetSchema sets the schema and initializes CEL engines
func (ns *Namespace) SetSchema(schema *InterfaceSchema) error {
	ns.Schema = schema

	// Add parameters as CEL variables
	if schema != nil {
		for key, value := range schema.Parameters {
			ns.celVariables[key] = value
		}
	}

	// CELエンジンを初期化
	return ns.initializeCELEngines()
}

// valueToLiteral は値をSQLリテラルに変換する
func (ns *Namespace) valueToLiteral(value any) string {
	switch v := value.(type) {
	case string:
		// 文字列はシングルクォートで囲む
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []any:
		// 配列をカンマ区切りのリテラルに変換
		var literals []string
		for _, item := range v {
			literals = append(literals, ns.valueToLiteral(item))
		}
		return strings.Join(literals, ", ")
	case []string:
		// Special handling for string arrays
		var literals []string
		for _, item := range v {
			literals = append(literals, fmt.Sprintf("'%s'", strings.ReplaceAll(item, "'", "''")))
		}
		return strings.Join(literals, ", ")
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// initializeCELEngines initializes both environment and parameter CEL engines
func (ns *Namespace) initializeCELEngines() error {
	// Create environment CEL engine
	if err := ns.createEnvironmentCEL(); err != nil {
		return fmt.Errorf("failed to create environment CEL: %w", err)
	}

	// Create parameter CEL engine
	if err := ns.createParameterCEL(); err != nil {
		return fmt.Errorf("failed to create parameter CEL: %w", err)
	}

	return nil
}

// createEnvironmentCEL creates CEL environment for environment constants (/*@ */)
func (ns *Namespace) createEnvironmentCEL() error {
	envOptions := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}

	// Register environment constants directly as CEL variables (no prefix)
	for key := range ns.Constants {
		envOptions = append(envOptions, cel.Variable(key, cel.DynType))
	}

	env, err := cel.NewEnv(envOptions...)
	if err != nil {
		return fmt.Errorf("failed to create environment CEL: %w", err)
	}

	ns.envCEL = env
	return nil
}

// createParameterCEL creates CEL environment for parameters (/*= */ and /*# */)
func (ns *Namespace) createParameterCEL() error {
	envOptions := []cel.EnvOption{
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
	}

	// Register using dummy data as CEL variables
	if ns.dummyData != nil {
		ns.addDummyDataVariables("", ns.dummyData, &envOptions)
	}

	env, err := cel.NewEnv(envOptions...)
	if err != nil {
		return fmt.Errorf("failed to create parameter CEL: %w", err)
	}

	ns.paramCEL = env
	return nil
}

// ValidateEnvironmentExpression validates environment constant expressions (/*@ */)
func (ns *Namespace) ValidateEnvironmentExpression(expression string) error {
	if ns.envCEL == nil {
		return ErrEnvironmentCELNotInit
	}

	// CEL式をコンパイル
	ast, issues := ns.envCEL.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	// 型チェック（出力型が存在することを確認）
	if ast.OutputType() == nil {
		return ErrNoOutputType
	}

	return nil
}

// ValidateParameterExpression validates parameter expressions (/*= */ and /*# */)
func (ns *Namespace) ValidateParameterExpression(expression string) error {
	if ns.paramCEL == nil {
		return ErrParameterCELNotInit
	}

	// CEL式をコンパイル
	ast, issues := ns.paramCEL.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	// 型チェック（出力型が存在することを確認）
	if ast.OutputType() == nil {
		return ErrNoOutputType
	}

	return nil
}

// ValidateExpression validates the validity of CEL expressions (general purpose)
func (ns *Namespace) ValidateExpression(expr string) error {
	// First try validation as environment constant expression
	if ns.envCEL != nil {
		if err := ns.ValidateEnvironmentExpression(expr); err == nil {
			return nil
		}
	}

	// If failed as environment constant expression, try validation as parameter expression
	if ns.paramCEL != nil {
		if err := ns.ValidateParameterExpression(expr); err == nil {
			return nil
		}
	}

	// 両方とも失敗した場合はエラー
	return ErrExpressionValidationFailed
}

// EvaluateEnvironmentExpression evaluates environment constant expressions (/*@ */)
func (ns *Namespace) EvaluateEnvironmentExpression(expression string) (any, error) {
	if ns.envCEL == nil {
		return nil, ErrEnvironmentCELNotInit
	}

	// CEL式をコンパイル
	ast, issues := ns.envCEL.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	// Create program
	program, err := ns.envCEL.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Prepare environment constant values
	envConstants := make(map[string]any)
	for key, value := range ns.Constants {
		envConstants[key] = value
	}

	// 式を評価
	result, _, err := program.Eval(envConstants)
	if err != nil {
		return nil, fmt.Errorf("CEL evaluation error: %w", err)
	}

	return result.Value(), nil
}

// ParameterDefinition represents parameter definition
type ParameterDefinition struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// addDummyDataVariables adds dummy data as CEL variables
func (ns *Namespace) addDummyDataVariables(prefix string, data map[string]any, envOptions *[]cel.EnvOption) {
	for key, value := range data {
		varName := key
		if prefix != "" {
			varName = prefix + "." + key
		}

		// Declare CEL variables based on value type
		celType := ns.getCELTypeFromValue(value)
		if celType != nil {
			*envOptions = append(*envOptions, cel.Variable(varName, celType))
		}
	}
}

// getCELTypeFromValue は値からCEL型を取得する
func (ns *Namespace) getCELTypeFromValue(value any) *cel.Type {
	if value == nil {
		return cel.AnyType
	}

	switch v := value.(type) {
	case string:
		return cel.StringType
	case int, int32, int64:
		return cel.IntType
	case float32, float64:
		return cel.DoubleType
	case bool:
		return cel.BoolType
	case []string:
		return cel.ListType(cel.StringType)
	case []int:
		return cel.ListType(cel.IntType)
	case []int64:
		return cel.ListType(cel.IntType)
	case []float64:
		return cel.ListType(cel.DoubleType)
	case []bool:
		return cel.ListType(cel.BoolType)
	case []any:
		// 要素の型を推定
		if len(v) > 0 {
			elementType := ns.getCELTypeFromValue(v[0])
			if elementType != nil {
				return cel.ListType(elementType)
			}
		}
		return cel.ListType(cel.AnyType)
	case map[string]any:
		// ネストしたオブジェクトの場合、各フィールドの型を定義
		fieldTypes := make(map[string]*cel.Type)
		for key, fieldValue := range v {
			fieldTypes[key] = ns.getCELTypeFromValue(fieldValue)
		}
		// CELでは複雑なオブジェクト型の定義が困難なため、MapTypeを使用
		return cel.MapType(cel.StringType, cel.AnyType)
	default:
		// リフレクションを使用してより詳細な型判定
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			return cel.ListType(cel.AnyType)
		case reflect.Map:
			return cel.MapType(cel.StringType, cel.AnyType)
		default:
			return cel.AnyType
		}
	}
}

// AddLoopVariableWithEvaluation creates a new namespace with loop variable added through CEL evaluation
func (ns *Namespace) AddLoopVariableWithEvaluation(variable string, listExpr string) (*Namespace, error) {
	// Evaluate CEL expression in dummy data environment
	result, err := ns.EvaluateParameterExpression(listExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate list expression '%s': %w", listExpr, err)
	}

	// 評価結果から要素の型と値を取得
	elementValue, elementType, err := ns.extractElementFromList(result)
	if err != nil {
		return nil, fmt.Errorf("failed to extract element from list result: %w", err)
	}

	// Create new namespace
	newNs := ns.Copy()

	// Add loop variable to schema
	if newNs.Schema.Parameters == nil {
		newNs.Schema.Parameters = make(map[string]any)
	}
	newNs.Schema.Parameters[variable] = elementType

	// Add loop variable to dummy data
	newNs.dummyData[variable] = elementValue

	// CELエンジンを再初期化
	if err := newNs.initializeCELEngines(); err != nil {
		return nil, fmt.Errorf("failed to reinitialize CEL engines: %w", err)
	}

	return newNs, nil
}

// EvaluateParameterExpression はパラメータ式をダミーデータ環境で評価する
func (ns *Namespace) EvaluateParameterExpression(expression string) (any, error) {
	if ns.paramCEL == nil {
		return nil, ErrParameterCELNotInit
	}

	// CEL式をコンパイル
	ast, issues := ns.paramCEL.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	// Create program
	program, err := ns.paramCEL.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// ダミーデータを使用して評価
	result, _, err := program.Eval(ns.dummyData)
	if err != nil {
		return nil, fmt.Errorf("CEL evaluation error: %w", err)
	}

	return result.Value(), nil
}

// extractElementFromList はリスト結果から要素の値と型を抽出する
func (ns *Namespace) extractElementFromList(listResult any) (elementValue any, elementType string, err error) {
	switch list := listResult.(type) {
	case []any:
		if len(list) > 0 {
			element := list[0]
			return element, ns.inferTypeFromValue(element), nil
		}
		return "", "str", nil // 空リストの場合はデフォルト
	case []string:
		if len(list) > 0 {
			return list[0], "str", nil
		}
		return "", "str", nil
	case []int:
		if len(list) > 0 {
			return list[0], "int", nil
		}
		return 0, "int", nil
	case []int64:
		if len(list) > 0 {
			return list[0], "int", nil
		}
		return 0, "int", nil
	case []float64:
		if len(list) > 0 {
			return list[0], "float", nil
		}
		return 0.0, "float", nil
	case []bool:
		if len(list) > 0 {
			return list[0], "bool", nil
		}
		return false, "bool", nil
	default:
		// リストでない場合はエラー
		return nil, "", ErrExpressionNotList
	}
}

// inferTypeFromValue は値から型を推論する
func (ns *Namespace) inferTypeFromValue(value any) string {
	switch value.(type) {
	case string:
		return "str"
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "float"
	case bool:
		return "bool"
	case map[string]any:
		// ネストしたオブジェクトの場合、オブジェクト型として返す
		return "object"
	case []any:
		return "list"
	default:
		return "any"
	}
}

// generateDummyDataFromSchema はスキーマからダミーデータ環境を生成する（関数スタイル）
func generateDummyDataFromSchema(schema *InterfaceSchema) map[string]any {
	if schema == nil || schema.Parameters == nil {
		return make(map[string]any)
	}

	result := make(map[string]any)
	visited := make(map[string]bool) // パスベースで循環参照を防ぐ
	for key, typeInfo := range schema.Parameters {
		result[key] = generateDummyValueWithPath(typeInfo, visited, key)
	}
	return result
}

// generateDummyValueWithPath はパスベースで循環参照を検出しながらダミー値を生成する
func generateDummyValueWithPath(typeInfo any, visited map[string]bool, path string) any {
	// パスベースで循環参照をチェック
	if visited[path] {
		return "" // 循環参照の場合はデフォルト値を返す
	}
	visited[path] = true
	defer delete(visited, path) // 処理完了後にvisitedから削除

	switch t := typeInfo.(type) {
	case string:
		return generateDummyValueFromString(t)
	case []any:
		// 配列型の場合、最初の要素をテンプレートとして使用
		if len(t) > 0 {
			elementTemplate := t[0]
			elementValue := generateDummyValueWithPath(elementTemplate, visited, path+"[0]")
			// 要素の型に応じて適切な配列型を返す
			switch elementTemplate {
			case "str", "string":
				return []string{"dummy"}
			case "int", "integer":
				return []int{0}
			case "float", "double":
				return []float64{0.0}
			case "bool", "boolean":
				return []bool{false}
			default:
				return []any{elementValue}
			}
		}
		return []any{}
	case map[string]any:
		// For object types, process recursively
		result := make(map[string]any)
		for key, value := range t {
			childPath := path + "." + key
			result[key] = generateDummyValueWithPath(value, visited, childPath)
		}
		return result
	default:
		return ""
	}
}

// generateDummyValueFromString は文字列型定義からダミー値を生成する
func generateDummyValueFromString(typeStr string) any {
	switch typeStr {
	case "str", "string":
		return ""
	case "int", "integer":
		return 0
	case "float", "double":
		return 0.0
	case "bool", "boolean":
		return false
	case "list[str]", "[]string":
		return []string{"dummy"}
	case "list[int]", "[]int":
		return []int{0}
	case "list[float]", "[]float":
		return []float64{0.0}
	case "list[bool]", "[]bool":
		return []bool{false}
	case "map[str]", "map[string]any":
		return map[string]any{"": ""}
	default:
		// list[T] パターンの解析
		if len(typeStr) > 5 && typeStr[:5] == "list[" && typeStr[len(typeStr)-1] == ']' {
			elementType := typeStr[5 : len(typeStr)-1]
			elementValue := generateDummyValueFromString(elementType)
			return []any{elementValue}
		}
		// デフォルトは空文字列
		return ""
	}
}
