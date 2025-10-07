package snapsqlgo

import (
	"math"
	"reflect"
	"time"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/shopspring/decimal"
)

const truthyEpsilon = 1e-4

var epsilonDecimal = decimal.NewFromFloat(truthyEpsilon)

// Truthy converts arbitrary values to boolean semantics used by SnapSQL templates.
func Truthy(value any) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case *bool:
		return v != nil && *v
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case float32:
		return math.Abs(float64(v)) >= truthyEpsilon
	case float64:
		return math.Abs(v) >= truthyEpsilon
	case string:
		return v != ""
	case *string:
		return v != nil && *v != ""
	case time.Time:
		return !v.IsZero()
	case *time.Time:
		return v != nil && !v.IsZero()
	case decimal.Decimal:
		return decimalTruthy(v)
	case *decimal.Decimal:
		return v != nil && decimalTruthy(*v)
	case Decimal:
		return decimalTruthy(v.Decimal)
	case *Decimal:
		return v != nil && decimalTruthy(v.Decimal)
	case types.Bool:
		return bool(v)
	case types.Int:
		return v != 0
	case types.Uint:
		return v != 0
	case types.Double:
		return math.Abs(float64(v)) >= truthyEpsilon
	case types.String:
		return string(v) != ""
	case types.Null, types.Unknown:
		return false
	case ref.Val:
		if v == nil {
			return false
		}

		if types.IsUnknown(v) {
			return false
		}

		if v.Type() == types.NullType {
			return false
		}

		native, err := v.ConvertToNative(reflect.TypeOf((*any)(nil)).Elem())
		if err == nil {
			if native == nil {
				return false
			}

			return Truthy(native)
		}
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return false
		}

		return Truthy(rv.Elem().Interface())
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return rv.Len() > 0
	case reflect.String:
		return rv.Len() > 0
	case reflect.Struct:
		if v, ok := rv.Interface().(time.Time); ok {
			return !v.IsZero()
		}
	}

	return !rv.IsZero()
}

func decimalTruthy(d decimal.Decimal) bool {
	return d.Abs().Cmp(epsilonDecimal) >= 0
}
