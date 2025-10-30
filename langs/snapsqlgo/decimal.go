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

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	snapsql "github.com/shibukawa/snapsql"
	"github.com/shopspring/decimal"
)

// Decimal is a wrapper around decimal.Decimal to implement CEL's ref.Val interface
type Decimal struct {
	decimal.Decimal
}

func NewDecimal(d decimal.Decimal) *Decimal {
	return &Decimal{d}
}

func NewDecimalFromFloat64(f float64) *Decimal {
	return &Decimal{decimal.NewFromFloat(f)}
}

func NewDecimalFromString(s string) (*Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", snapsql.ErrInvalidDecimalString, s)
	}

	return &Decimal{d}, nil
}

func NewDecimalFromInt(i int) *Decimal {
	return &Decimal{decimal.NewFromInt(int64(i))}
}

// Ensure Decimal implements ref.Val
var _ ref.Val = (*Decimal)(nil)

// Type returns the CEL type of the value (e.g., decimal type)
func (d *Decimal) Type() ref.Type {
	return DecimalType
}

// Value returns the raw Go value
func (d *Decimal) Value() any {
	return d.Decimal
}

// ConvertToNative converts the CelDecimal to a Go native type
func (d *Decimal) ConvertToNative(typeDesc reflect.Type) (any, error) {
	// If the target type is decimal.Decimal, return the underlying value
	if typeDesc == reflect.TypeFor[decimal.Decimal]() {
		return d.Decimal, nil
	}
	// If the target type is *decimal.Decimal, return a pointer
	if typeDesc == reflect.TypeFor[*decimal.Decimal]() {
		return &d.Decimal, nil
	}
	// Handle conversion to other common types like float64, string, etc.
	if typeDesc == reflect.TypeFor[float64]() {
		f, _ := d.Float64()
		return f, nil
	}

	if typeDesc == reflect.TypeFor[string]() {
		return d.String(), nil
	}

	return nil, fmt.Errorf("%w: to %v for Decimal", snapsql.ErrUnsupportedConversion, typeDesc)
}

// ConvertToType converts Decimal to another CEL type (e.g., DOUBLE, STRING)
func (d *Decimal) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case types.DoubleType:
		f, _ := d.Float64()
		return types.Double(f)
	case types.StringType:
		return types.String(d.String())
	case d.Type(): // Self conversion
		return d
	}

	return types.NewErr("type conversion error from Decimal to %s", typeVal)
}

// Equal returns true if the two Decimal values are equal
func (d *Decimal) Equal(other ref.Val) ref.Val {
	o, ok := other.(*Decimal)
	if !ok {
		// Try to convert other to Decimal if possible
		converted, err := other.ConvertToNative(reflect.TypeFor[decimal.Decimal]())
		if err == nil {
			o, ok = converted.(*Decimal)
		}

		if !ok {
			return types.NewErr("type conversion error during comparison")
		}
	}

	return types.Bool(d.Decimal.Equal(o.Decimal))
}

// DecimalTypeName is the fully qualified CEL type name for Decimal
const DecimalTypeName = "snapsqlgo.Decimal"

// DecimalType is the CEL type representation for Decimal
var DecimalType = types.NewObjectType(DecimalTypeName)

// type provider
// CustomTypeAdapter: Go値→CEL値変換
type customDecimalTypeAdapter struct{}

func (customDecimalTypeAdapter) NativeToValue(value any) ref.Val {
	switch v := value.(type) {
	case *Decimal:
		return v
	case decimal.Decimal:
		return &Decimal{v}
	default:
		return types.DefaultTypeAdapter.NativeToValue(value)
	}
}

var _ types.Adapter = (*customDecimalTypeAdapter)(nil)

// CustomTypeProvider: 型名→型情報解決
type customDecimalTypeProvider struct {
}

// EnumValue implements types.Provider.
func (p *customDecimalTypeProvider) EnumValue(enumName string) ref.Val {
	return types.NewErr("not found enum: %s", enumName)
}

// FindIdent implements types.Provider.
func (p *customDecimalTypeProvider) FindIdent(identName string) (ref.Val, bool) {
	return nil, false
}

// FindStructFieldNames implements types.Provider.
func (p *customDecimalTypeProvider) FindStructFieldNames(structType string) ([]string, bool) {
	return nil, false
}

// FindStructFieldType implements types.Provider.
func (p *customDecimalTypeProvider) FindStructFieldType(structType string, fieldName string) (*types.FieldType, bool) {
	return nil, false
}

// FindStructType implements types.Provider.
func (p *customDecimalTypeProvider) FindStructType(structType string) (*types.Type, bool) {
	return nil, false
}

// NewValue implements types.Provider.
func (p *customDecimalTypeProvider) NewValue(structType string, fields map[string]ref.Val) ref.Val {
	return types.NewErr("not value: %s", structType)
}

func (p *customDecimalTypeProvider) FindType(typeName string) (*cel.Type, bool) {
	if typeName == DecimalTypeName {
		return DecimalType, true
	}

	return nil, false
}

var _ types.Provider = (*customDecimalTypeProvider)(nil)

type decimalLibrary struct{}

func (l *decimalLibrary) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.CustomTypeAdapter(customDecimalTypeAdapter{}),
		cel.CustomTypeProvider(&customDecimalTypeProvider{}),
	}
}

func (l *decimalLibrary) ProgramOptions() []cel.ProgramOption {
	return nil
}

var _ cel.Library = (*decimalLibrary)(nil)

var DecimalLibrary = cel.Lib(&decimalLibrary{})
