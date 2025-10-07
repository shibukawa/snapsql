package snapsqlgo

import (
	"math"
	"testing"
	"time"

	"github.com/google/cel-go/common/types"
	"github.com/shopspring/decimal"
)

func TestTruthyPrimitives(t *testing.T) {
	testCases := []struct {
		name  string
		value any
		want  bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"int zero", 0, false},
		{"int nonzero", 42, true},
		{"float epsilon below", 5e-5, false},
		{"float epsilon above", 2e-4, true},
		{"string empty", "", false},
		{"string nonempty", "snap", true},
		{"time zero", time.Time{}, false},
		{"time nonzero", time.Now(), true},
		{"decimal zero", decimal.NewFromInt(0), false},
		{"decimal nonzero", decimal.NewFromFloat(1.25), true},
		{"cel null", types.NullValue, false},
		{"cel bool true", types.True, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			if got := Truthy(tc.value); got != tc.want {
				t.Fatalf("Truthy(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestTruthyCollections(t *testing.T) {
	now := time.Now()
	ptrTime := &now

	var nilTime *time.Time

	testCases := []struct {
		name  string
		value any
		want  bool
	}{
		{"nil", nil, false},
		{"slice empty", []int{}, false},
		{"slice nonempty", []int{1}, true},
		{"map empty", map[string]string{}, false},
		{"map nonempty", map[string]string{"k": "v"}, true},
		{"pointer nil", nilTime, false},
		{"pointer time", ptrTime, true},
		{"struct zero", struct{ A int }{}, false},
		{"struct nonzero", struct{ A int }{A: 1}, true},
	}

	for _, tc := range testCases {
		if got := Truthy(tc.value); got != tc.want {
			t.Fatalf("%s: Truthy(%v) = %v, want %v", tc.name, tc.value, got, tc.want)
		}
	}
}

func TestTruthyDecimalPointer(t *testing.T) {
	var nilDecimal *decimal.Decimal

	nonzero := decimal.NewFromFloat(0.5)
	ptr := &nonzero

	if Truthy(nilDecimal) {
		t.Fatalf("expected nil decimal pointer to be false")
	}

	if !Truthy(ptr) {
		t.Fatalf("expected non-zero decimal pointer to be true")
	}
}

func TestTruthyFloatThreshold(t *testing.T) {
	if Truthy(0.0000999) {
		t.Fatalf("values below threshold should be false")
	}

	if !Truthy(0.0001001) {
		t.Fatalf("values above threshold should be true")
	}

	if !Truthy(-0.0001001) {
		t.Fatalf("negative values above threshold magnitude should be true")
	}

	if Truthy(-0.00005) {
		t.Fatalf("negative values below threshold magnitude should be false")
	}

	if !Truthy(math.Inf(1)) {
		t.Fatalf("infinity should be true")
	}
}
