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
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	snapsql "github.com/shibukawa/snapsql"
	"github.com/shopspring/decimal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name     string                    // CEL上で使用する名前
	GoName   string                    // Go構造体のフィールド名
	CelType  *types.Type               // CEL型情報
	Accessor func(interface{}) ref.Val // 静的アクセサー関数
}

// StructInfo represents information about a struct type
type StructInfo struct {
	Name      string                    // 構造体名
	CelType   *types.Type               // CEL型情報
	Fields    map[string]FieldInfo      // フィールド情報
	Converter func(interface{}) ref.Val // 静的変換関数
}

// LocalTypeRegistry manages custom types for a specific function
type LocalTypeRegistry struct {
	structs map[string]StructInfo
}

// NewLocalTypeRegistry creates a new local type registry
func NewLocalTypeRegistry() *LocalTypeRegistry {
	return &LocalTypeRegistry{
		structs: make(map[string]StructInfo),
	}
}

// RegisterStructWithFields registers a struct type with explicit field definitions
func (r *LocalTypeRegistry) RegisterStructWithFields(name string, fields map[string]FieldInfo) {
	celType := cel.ObjectType(name)
	r.structs[name] = StructInfo{
		Name:    name,
		CelType: celType,
		Fields:  fields,
	}
}

// GetStructInfo returns struct information by name
func (r *LocalTypeRegistry) GetStructInfo(name string) (StructInfo, bool) {
	info, exists := r.structs[name]
	return info, exists
}

// CustomValue wraps a Go struct to implement CEL ref.Val interface
type CustomValue struct {
	structInfo StructInfo
	value      interface{}
	registry   *LocalTypeRegistry
}

// NewCustomValue creates a new custom value
func NewCustomValue(structInfo StructInfo, value interface{}, registry *LocalTypeRegistry) *CustomValue {
	return &CustomValue{
		structInfo: structInfo,
		value:      value,
		registry:   registry,
	}
}

// Get implements traits.Indexer for field access
func (v *CustomValue) Get(index ref.Val) ref.Val {
	if fieldName, ok := index.Value().(string); ok {
		if fieldInfo, exists := v.structInfo.Fields[fieldName]; exists {
			// 静的アクセサー関数を使用（リフレクション不要）
			return fieldInfo.Accessor(v.value)
		}

		return types.NewErr("unknown field: %s", fieldName)
	}

	return types.NewErr("index must be a string, got: %s", index.Type())
}

// ConvertToNative converts to Go native type
func (v *CustomValue) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	if typeDesc == reflect.TypeOf(v.value) {
		return v.value, nil
	}

	return nil, fmt.Errorf("%w: %v", snapsql.ErrUnsupportedConversion, typeDesc)
}

// ConvertToType converts to CEL type
func (v *CustomValue) ConvertToType(typeVal ref.Type) ref.Val {
	if typeVal == types.TypeType {
		return cel.ObjectType(v.structInfo.Name)
	}

	return types.NewErr("type conversion not supported")
}

// Equal checks equality
func (v *CustomValue) Equal(other ref.Val) ref.Val {
	if otherCustom, ok := other.(*CustomValue); ok {
		if v.structInfo.Name == otherCustom.structInfo.Name {
			return types.Bool(reflect.DeepEqual(v.value, otherCustom.value))
		}
	}

	return types.False
}

// Type returns CEL type
func (v *CustomValue) Type() ref.Type {
	return cel.ObjectType(v.structInfo.Name)
}

// Value returns Go value
func (v *CustomValue) Value() interface{} {
	return v.value
}

// Ensure CustomValue implements required interfaces
var _ ref.Val = (*CustomValue)(nil)
var _ traits.Indexer = (*CustomValue)(nil)

// LocalTypeAdapter adapts Go values to CEL values for a specific registry
type LocalTypeAdapter struct {
	registry *LocalTypeRegistry
}

// NewLocalTypeAdapter creates a new type adapter
func NewLocalTypeAdapter(registry *LocalTypeRegistry) *LocalTypeAdapter {
	return &LocalTypeAdapter{registry: registry}
}

// NativeToValue converts Go native values to CEL values
func (a *LocalTypeAdapter) NativeToValue(value interface{}) ref.Val {
	if value == nil {
		return types.NullValue
	}

	// Get the type name
	valueType := reflect.TypeOf(value)
	if valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
	}

	typeName := valueType.Name()

	// Check if this type is registered
	if structInfo, exists := a.registry.GetStructInfo(typeName); exists {
		// 静的変換関数を使用
		return structInfo.Converter(value)
	}

	// Fall back to default adapter
	return types.DefaultTypeAdapter.NativeToValue(value)
}

var _ types.Adapter = (*LocalTypeAdapter)(nil)

// LocalTypeProvider provides type information to CEL for a specific registry
type LocalTypeProvider struct {
	registry *LocalTypeRegistry
}

// NewLocalTypeProvider creates a new type provider
func NewLocalTypeProvider(registry *LocalTypeRegistry) *LocalTypeProvider {
	return &LocalTypeProvider{registry: registry}
}

// EnumValue returns enum values (not supported)
func (p *LocalTypeProvider) EnumValue(enumName string) ref.Val {
	return types.NewErr("EnumValue is not supported for %s", enumName)
}

// FindIdent finds identifiers (not used for struct types)
func (p *LocalTypeProvider) FindIdent(identName string) (ref.Val, bool) {
	return nil, false
}

// FindStructFieldNames returns field names for a struct type
func (p *LocalTypeProvider) FindStructFieldNames(structType string) ([]string, bool) {
	if structInfo, exists := p.registry.GetStructInfo(structType); exists {
		fieldNames := make([]string, 0, len(structInfo.Fields))
		for fieldName := range structInfo.Fields {
			fieldNames = append(fieldNames, fieldName)
		}

		return fieldNames, true
	}

	return nil, false
}

// FindStructFieldType returns field type information
func (p *LocalTypeProvider) FindStructFieldType(structType string, fieldName string) (*types.FieldType, bool) {
	if structInfo, exists := p.registry.GetStructInfo(structType); exists {
		if fieldInfo, fieldExists := structInfo.Fields[fieldName]; fieldExists {
			return &types.FieldType{
				Type: fieldInfo.CelType,
			}, true
		}
	}

	return nil, false
}

// FindStructType returns struct type information
func (p *LocalTypeProvider) FindStructType(structType string) (*types.Type, bool) {
	if structInfo, exists := p.registry.GetStructInfo(structType); exists {
		return structInfo.CelType, true
	}

	return nil, false
}

// NewValue creates a new value of the specified type (not implemented for now)
func (p *LocalTypeProvider) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	return types.NewErr("NewValue not implemented for type: %s", typeName)
}

var _ types.Provider = (*LocalTypeProvider)(nil)

// ConvertGoValueToCEL converts Go values to CEL values (exported for generated code)
func ConvertGoValueToCEL(value interface{}) ref.Val {
	if value == nil {
		return types.NullValue
	}

	switch v := value.(type) {
	case int:
		return types.Int(int64(v))
	case int32:
		return types.Int(int64(v))
	case int64:
		return types.Int(v)
	case string:
		return types.String(v)
	case bool:
		return types.Bool(v)
	case float32:
		return types.Double(float64(v))
	case float64:
		return types.Double(v)
	case time.Time:
		return types.DefaultTypeAdapter.NativeToValue(v)
	case *time.Time:
		if v == nil {
			return types.NullValue
		}

		return types.DefaultTypeAdapter.NativeToValue(*v)
	case decimal.Decimal:
		// Use existing CEL wrapper to preserve precision
		return NewDecimal(v)
	case *decimal.Decimal:
		if v == nil {
			return types.NullValue
		}

		return NewDecimal(*v)
	default:
		// Handle slices and arrays
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			return convertSliceToCEL(rv)
		}

		// Handle pointers to structs
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return types.NullValue
			}

			rv = rv.Elem()
		}

		// Handle structs - check if it's a registered type
		if rv.Kind() == reflect.Struct {
			typeName := rv.Type().Name()
			if registry := GetGlobalRegistry(); registry != nil {
				if structInfo, exists := registry.GetStructInfo(typeName); exists {
					return NewCustomValue(structInfo, value, registry)
				}
			}
		}

		// For complex types, use default adapter
		return types.DefaultTypeAdapter.NativeToValue(value)
	}
}

// convertSliceToCEL converts Go slices to CEL lists
func convertSliceToCEL(rv reflect.Value) ref.Val {
	length := rv.Len()
	celValues := make([]ref.Val, length)

	for i := range length {
		element := rv.Index(i).Interface()
		celValues[i] = ConvertGoValueToCEL(element)
	}

	return types.DefaultTypeAdapter.NativeToValue(celValues)
}

// Helper functions for creating CEL options with local registry

// CreateCELOptionsWithTypes creates CEL options with a local type registry
func CreateCELOptionsWithTypes(typeDefinitions map[string]map[string]FieldInfo) []cel.EnvOption {
	registry := NewLocalTypeRegistry()

	// Register all types
	for typeName, fields := range typeDefinitions {
		registry.RegisterStructWithFields(typeName, fields)
	}

	return []cel.EnvOption{
		cel.CustomTypeAdapter(NewLocalTypeAdapter(registry)),
		cel.CustomTypeProvider(NewLocalTypeProvider(registry)),
	}
}

// Helper function to create FieldInfo with static accessor
func CreateFieldInfo(celName string, celType *types.Type, accessor func(interface{}) ref.Val) FieldInfo {
	// Convert CEL name to Go name automatically
	goName := celNameToGoName(celName)

	return FieldInfo{
		Name:     celName,
		GoName:   goName,
		CelType:  celType,
		Accessor: accessor,
	}
}

// Global registry for type lookup
var globalRegistry *LocalTypeRegistry

// SetGlobalRegistry sets the global type registry
func SetGlobalRegistry(registry *LocalTypeRegistry) {
	globalRegistry = registry
}

// GetGlobalRegistry returns the global type registry
func GetGlobalRegistry() *LocalTypeRegistry {
	return globalRegistry
}
func celNameToGoName(celName string) string {
	parts := strings.Split(celName, "_")
	caser := cases.Title(language.English)

	for i, part := range parts {
		if part == "id" {
			parts[i] = "ID"
		} else {
			parts[i] = caser.String(part)
		}
	}

	return strings.Join(parts, "")
}
