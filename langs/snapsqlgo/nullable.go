package snapsqlgo

import (
	"reflect"
	"time"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

// NormalizeNullableTimestamp converts timestamp-like values to nil when they represent CEL nulls
// or zero-value time instants. It transparently handles CEL ref.Val, time.Time, *time.Time, and
// structpb.NullValue. For any other type the original value is returned unchanged.
func NormalizeNullableTimestamp(value any) any {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		if v.IsZero() {
			return nil
		}

		return v
	case *time.Time:
		if v == nil {
			return nil
		}

		if v.IsZero() {
			return nil
		}

		return *v
	case structpb.NullValue:
		return nil
	case ref.Val:
		if v == types.NullValue || types.IsUnknown(v) {
			return nil
		}

		if v.Type() == types.TimestampType {
			native, err := v.ConvertToNative(reflect.TypeOf(time.Time{}))
			if err == nil {
				if ts, ok := native.(time.Time); ok {
					if ts.IsZero() {
						return nil
					}

					return ts
				}
			}
		}

		native, err := v.ConvertToNative(reflect.TypeOf((*any)(nil)).Elem())
		if err == nil {
			return NormalizeNullableTimestamp(native)
		}

		return v.Value()
	default:
		return value
	}
}
