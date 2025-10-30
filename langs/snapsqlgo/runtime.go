// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snapsqlgo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	snapsql "github.com/shibukawa/snapsql"
)

// DBExecutor interface supports sql.DB, sql.Conn, and sql.Tx
type DBExecutor interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// MockResult implements sql.Result for mock operations
type MockResult struct {
	rowsAffected int64
	lastInsertId int64
}

func (m *MockResult) LastInsertId() (int64, error) {
	return m.lastInsertId, nil
}

func (m *MockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}

// NewMockResult creates a new MockResult with specified values
func NewMockResult(rowsAffected, lastInsertId int64) *MockResult {
	return &MockResult{
		rowsAffected: rowsAffected,
		lastInsertId: lastInsertId,
	}
}

// LoadMockDataFromFile loads mock data from a JSON file
func LoadMockDataFromFile(mockPath, testCaseName string) (any, error) {
	filePath := filepath.Join("testdata/snapsql_mock", mockPath, testCaseName+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock file %s: %w", filePath, err)
	}

	var result any

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mock JSON %s: %w", filePath, err)
	}

	return result, nil
}

// GetMockDataFromFiles loads and combines mock data from multiple test cases
func GetMockDataFromFiles(mockPath string, dataNames []string) (any, error) {
	if len(dataNames) == 0 {
		return nil, snapsql.ErrNoMockDataNames
	}

	if len(dataNames) == 1 {
		return LoadMockDataFromFile(mockPath, dataNames[0])
	}

	// Combine multiple test cases
	var combinedData []any

	for _, dataName := range dataNames {
		data, err := LoadMockDataFromFile(mockPath, dataName)
		if err != nil {
			return nil, fmt.Errorf("failed to load mock data '%s': %w", dataName, err)
		}

		if slice, ok := data.([]any); ok {
			combinedData = append(combinedData, slice...)
		} else {
			combinedData = append(combinedData, data)
		}
	}

	return combinedData, nil
}

// Context keys
type systemColumnKey struct{}
type funcConfigKey struct{}

// ImplicitParamSpec defines required implicit parameters for a function
type ImplicitParamSpec struct {
	Name         string
	Type         string
	Required     bool
	DefaultValue any
}

// FuncConfig holds configuration for a specific function
type FuncConfig struct {
	MockDataNames        []string
	PerformanceThreshold time.Duration
	EnableExplain        bool
	ExplainCallback      func(ExplainResult)
	EnableLogging        bool
	LogFormat            LogFormat
	LogOutput            any
	RuntimeLimit         *int
	RuntimeOffset        *int
}

// LogFormat defines the output format for logs
type LogFormat int

const (
	LogFormatText LogFormat = iota
	LogFormatColor
	LogFormatJSON
)

// ExplainResult provides detailed query execution information
type ExplainResult struct {
	QueryPlan string  `json:"query_plan"`
	Cost      float64 `json:"cost"`
	Rows      int64   `json:"rows"`
}

// WithSystemColumnValues adds system column values to context
func WithSystemColumnValues(ctx context.Context, values map[string]any) context.Context {
	return context.WithValue(ctx, systemColumnKey{}, values)
}

// WithConfig sets function-specific configuration options with flexible matching
func WithConfig(ctx context.Context, funcPattern string, opts ...FuncOpt) context.Context {
	configData := ctx.Value(funcConfigKey{})
	if configData == nil {
		configData = make(map[string]*FuncConfig)
	}

	configMap, ok := configData.(map[string]*FuncConfig)
	if !ok {
		configMap = make(map[string]*FuncConfig)
	}

	config := &FuncConfig{}

	// Apply function options
	for _, opt := range opts {
		opt(config)
	}

	configMap[funcPattern] = config

	return context.WithValue(ctx, funcConfigKey{}, configMap)
}

// FuncOpt is a function option for configuring individual functions
type FuncOpt func(*FuncConfig)

// WithMockData configures mock data for a function
func WithMockData(dataNames ...string) FuncOpt {
	return func(config *FuncConfig) {
		config.MockDataNames = dataNames
	}
}

// WithPerformanceThreshold sets performance monitoring threshold
func WithPerformanceThreshold(threshold time.Duration) FuncOpt {
	return func(config *FuncConfig) {
		config.PerformanceThreshold = threshold
	}
}

// ExtractImplicitParams extracts and validates implicit parameters from context
func ExtractImplicitParams(ctx context.Context, specs []ImplicitParamSpec) map[string]any {
	systemValues := ctx.Value(systemColumnKey{})
	if systemValues == nil {
		// No system values in context - check for required parameters and set defaults
		result := make(map[string]any)

		for _, spec := range specs {
			if spec.Required {
				panic(fmt.Sprintf("implementation error: required implicit parameter '%s' not found in context - WithSystemValue() not called", spec.Name))
			}

			// Use default value from spec (which comes from config file)
			result[spec.Name] = spec.DefaultValue
		}

		return result
	}

	systemMap, ok := systemValues.(map[string]any)
	if !ok {
		panic("implementation error: invalid system column values type in context")
	}

	result := make(map[string]any)

	for _, spec := range specs {
		value, exists := systemMap[spec.Name]

		if !exists {
			if spec.Required {
				panic(fmt.Sprintf("implementation error: required implicit parameter '%s' (%s) not found in context", spec.Name, spec.Type))
			}

			// Use default value from spec (which comes from config file)
			result[spec.Name] = spec.DefaultValue

			continue
		}

		if !validateImplicitParamType(value, spec.Type) {
			panic(fmt.Sprintf("implementation error: implicit parameter '%s' has invalid type: expected %s, got %T", spec.Name, spec.Type, value))
		}

		result[spec.Name] = value
	}

	return result
}

// validateImplicitParamType validates the type of an implicit parameter
func validateImplicitParamType(value any, expectedType string) bool {
	if value == nil {
		return true
	}

	switch normalizeTemporalExpectedType(expectedType) {
	case "int":
		_, ok := value.(int)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "bool":
		_, ok := value.(bool)
		return ok
	case "timestamp":
		_, ok := value.(time.Time)
		return ok
	default:
		return value != nil
	}
}

func normalizeTemporalExpectedType(expectedType string) string {
	lower := strings.ToLower(expectedType)
	switch lower {
	case "datetime", "timestamp", "date", "time", "time.time":
		return "timestamp"
	default:
		return lower
	}
}

// GetFunctionConfig extracts function-specific configuration from context
func GetFunctionConfig(ctx context.Context, funcName string, queryType string) *FuncConfig {
	configData := ctx.Value(funcConfigKey{})
	if configData == nil {
		return nil
	}

	configMap, ok := configData.(map[string]*FuncConfig)
	if !ok {
		return nil
	}

	return matchFunctionConfig(configMap, funcName, queryType)
}

// matchFunctionConfig finds the best matching configuration for a function
func matchFunctionConfig(configMap map[string]*FuncConfig, funcName string, queryType string) *FuncConfig {
	// 1. Exact match
	if config, exists := configMap[funcName]; exists {
		return config
	}

	// 2. Query type with function name glob
	queryTypePattern := queryType + ":"
	for pattern, config := range configMap {
		if after, ok := strings.CutPrefix(pattern, queryTypePattern); ok {
			globPattern := after
			if matchGlob(globPattern, funcName) {
				return config
			}
		}
	}

	// 3. Function name glob
	for pattern, config := range configMap {
		if !strings.Contains(pattern, ":") && matchGlob(pattern, funcName) {
			return config
		}
	}

	// 4. Query type wildcard
	queryTypeWildcard := queryType + ":*"
	if config, exists := configMap[queryTypeWildcard]; exists {
		return config
	}

	// 5. Global wildcard
	if config, exists := configMap["*"]; exists {
		return config
	}

	return nil
}

func matchGlob(pattern, name string) bool {
	matched, _ := filepath.Match(pattern, name)
	return matched
}

// IsTrue evaluates a pre-compiled CEL program and returns boolean result
func IsTrue(program cel.Program, namespace map[string]any) bool {
	result, _, err := program.Eval(namespace)
	if err != nil {
		return false
	}

	if boolVal, ok := result.Value().(bool); ok {
		return boolVal
	}

	return false
}

// EvalToSlice evaluates a pre-compiled CEL program and returns slice result
func EvalToSlice(program cel.Program, namespace map[string]any) []any {
	result, _, err := program.Eval(namespace)
	if err != nil {
		return nil
	}

	if listVal, ok := result.Value().([]any); ok {
		return listVal
	}

	return nil
}

// EvalToString evaluates a pre-compiled CEL program and returns string result
func EvalToString(program cel.Program, namespace map[string]any) string {
	result, _, err := program.Eval(namespace)
	if err != nil {
		return ""
	}

	if strVal, ok := result.Value().(string); ok {
		return strVal
	}

	return ""
}

// EvalToAny evaluates a pre-compiled CEL program and returns any result
func EvalToAny(program cel.Program, namespace map[string]any) any {
	result, _, err := program.Eval(namespace)
	if err != nil {
		return nil
	}

	return result.Value()
}

// MapMockDataToStruct maps mock data to the appropriate struct type
func MapMockDataToStruct[T any](mockData any) (T, error) {
	var result T

	jsonData, err := json.Marshal(mockData)
	if err != nil {
		return result, fmt.Errorf("failed to marshal mock data: %w", err)
	}

	if err := json.Unmarshal(jsonData, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal mock data to struct: %w", err)
	}

	return result, nil
}

// MapMockDataToSlice maps mock data array to slice of structs
func MapMockDataToSlice[T any](mockData any) ([]T, error) {
	var result []T

	slice, ok := mockData.([]any)
	if !ok {
		item, err := MapMockDataToStruct[T](mockData)
		if err != nil {
			return nil, fmt.Errorf("failed to map single mock data item: %w", err)
		}

		return []T{item}, nil
	}

	jsonData, err := json.Marshal(slice)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mock data slice: %w", err)
	}

	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mock data to struct slice: %w", err)
	}

	return result, nil
}
