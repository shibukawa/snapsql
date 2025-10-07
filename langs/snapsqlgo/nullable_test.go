package snapsqlgo

import (
	"testing"
	"time"

	"github.com/google/cel-go/common/types"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func TestNormalizeNullableTimestamp_TimeValues(t *testing.T) {
	now := time.Now().UTC()

	if got := NormalizeNullableTimestamp(time.Time{}); got != nil {
		t.Fatalf("zero time should become nil, got %v", got)
	}

	if got := NormalizeNullableTimestamp(&time.Time{}); got != nil {
		t.Fatalf("pointer zero time should become nil, got %v", got)
	}

	if got := NormalizeNullableTimestamp(&now); got != now {
		t.Fatalf("expected dereferenced time, got %v", got)
	}
}

func TestNormalizeNullableTimestamp_CELValues(t *testing.T) {
	now := time.Now().UTC()

	var val = types.DefaultTypeAdapter.NativeToValue(now)
	if got, ok := NormalizeNullableTimestamp(val).(time.Time); !ok || !got.Equal(now) {
		t.Fatalf("expected time.Time from CEL timestamp, got %T %v", got, got)
	}

	if got := NormalizeNullableTimestamp(types.NullValue); got != nil {
		t.Fatalf("types.NullValue should become nil, got %v", got)
	}

	if got := NormalizeNullableTimestamp(structpb.NullValue_NULL_VALUE); got != nil {
		t.Fatalf("structpb.NullValue should become nil, got %v", got)
	}
}
