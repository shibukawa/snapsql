package snapsqlgo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSystemValue(t *testing.T) {
	ctx := context.Background()

	// Test adding a single system value
	ctx = WithSystemValue(ctx, "created_by", 123)

	values := getSystemValuesFromContext(ctx)
	require.NotNil(t, values)
	assert.Equal(t, 123, values["created_by"])
}

func TestWithSystemValue_Multiple(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	// Test adding multiple system values
	ctx = WithSystemValue(ctx, "created_by", 123)
	ctx = WithSystemValue(ctx, "created_at", now)
	ctx = WithSystemValue(ctx, "version", 1)

	values := getSystemValuesFromContext(ctx)
	require.NotNil(t, values)
	assert.Equal(t, 123, values["created_by"])
	assert.Equal(t, now, values["created_at"])
	assert.Equal(t, 1, values["version"])
}

func TestWithSystemValue_Overwrite(t *testing.T) {
	ctx := context.Background()

	// Test overwriting a system value
	ctx = WithSystemValue(ctx, "created_by", 123)
	ctx = WithSystemValue(ctx, "created_by", 456)

	values := getSystemValuesFromContext(ctx)
	require.NotNil(t, values)
	assert.Equal(t, 456, values["created_by"])
}

func TestExtractImplicitParams_WithContext(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	// Set up context with system values
	ctx = WithSystemValue(ctx, "created_by", 123)
	ctx = WithSystemValue(ctx, "created_at", now)
	ctx = WithSystemValue(ctx, "updated_at", now)
	ctx = WithSystemValue(ctx, "version", 1)

	specs := []ImplicitParamSpec{
		{Name: "created_at", Type: "time.Time", Required: false},
		{Name: "updated_at", Type: "time.Time", Required: false},
		{Name: "created_by", Type: "int", Required: true},
		{Name: "version", Type: "int", Required: false},
	}

	result := ExtractImplicitParams(ctx, specs)

	assert.Equal(t, now, result["created_at"])
	assert.Equal(t, now, result["updated_at"])
	assert.Equal(t, 123, result["created_by"])
	assert.Equal(t, 1, result["version"])
}

func TestExtractImplicitParams_MissingRequired(t *testing.T) {
	ctx := context.Background()

	specs := []ImplicitParamSpec{
		{Name: "created_by", Type: "int", Required: true},
	}

	// Should panic when required parameter is missing
	assert.Panics(t, func() {
		ExtractImplicitParams(ctx, specs)
	})
}

func TestExtractImplicitParams_OptionalDefaults(t *testing.T) {
	ctx := context.Background()

	specs := []ImplicitParamSpec{
		{Name: "created_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
		{Name: "updated_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
		{Name: "version", Type: "int", Required: false, DefaultValue: 1},
		{Name: "other_field", Type: "string", Required: false},
	}

	result := ExtractImplicitParams(ctx, specs)

	// created_at and updated_at should have default values from spec
	assert.NotNil(t, result["created_at"])
	assert.NotNil(t, result["updated_at"])

	// version should have default value from spec
	assert.Equal(t, 1, result["version"])

	// other fields without default should be nil
	assert.Nil(t, result["other_field"])
}

func TestExtractImplicitParams_PartialContext(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	// Only set some values in context
	ctx = WithSystemValue(ctx, "created_by", 123)
	ctx = WithSystemValue(ctx, "created_at", now)

	specs := []ImplicitParamSpec{
		{Name: "created_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
		{Name: "updated_at", Type: "time.Time", Required: false, DefaultValue: time.Now()},
		{Name: "created_by", Type: "int", Required: true},
		{Name: "version", Type: "int", Required: false, DefaultValue: 1},
	}

	result := ExtractImplicitParams(ctx, specs)

	// Values from context
	assert.Equal(t, now, result["created_at"])
	assert.Equal(t, 123, result["created_by"])

	// Default values for missing optional parameters from spec
	assert.NotNil(t, result["updated_at"])
	assert.Equal(t, 1, result["version"])
}

func TestValidateImplicitParamTypeTemporalAliases(t *testing.T) {
	now := time.Now()
	aliases := []string{"timestamp", "datetime", "date", "time", "time.Time"}

	for _, alias := range aliases {
		if !validateImplicitParamType(now, alias) {
			t.Fatalf("expected alias %s to accept time.Time value", alias)
		}
	}
}

func TestGetSystemValuesFromContext_EmptyContext(t *testing.T) {
	ctx := context.Background()

	values := getSystemValuesFromContext(ctx)
	assert.Nil(t, values)
}

func TestWithSystemValue_ImmutableContext(t *testing.T) {
	ctx := context.Background()

	// Add value to first context
	ctx1 := WithSystemValue(ctx, "created_by", 123)

	// Add different value to second context
	ctx2 := WithSystemValue(ctx, "created_by", 456)

	// Original contexts should be unchanged
	values1 := getSystemValuesFromContext(ctx1)
	values2 := getSystemValuesFromContext(ctx2)

	assert.Equal(t, 123, values1["created_by"])
	assert.Equal(t, 456, values2["created_by"])

	// Adding to ctx1 should not affect ctx2
	ctx1Updated := WithSystemValue(ctx1, "version", 1)
	values1Updated := getSystemValuesFromContext(ctx1Updated)
	values2AfterUpdate := getSystemValuesFromContext(ctx2)

	assert.Equal(t, 1, values1Updated["version"])
	assert.NotContains(t, values2AfterUpdate, "version")
}
