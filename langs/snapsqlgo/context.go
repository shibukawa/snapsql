package snapsqlgo

import (
	"context"
	"maps"
)

// WithSystemValue adds a system value to the context
func WithSystemValue(ctx context.Context, key string, value any) context.Context {
	systemValues := getSystemValuesFromContext(ctx)
	if systemValues == nil {
		systemValues = make(map[string]any)
	}

	// Create a copy to avoid modifying the original map
	newSystemValues := make(map[string]any)
	maps.Copy(newSystemValues, systemValues)

	newSystemValues[key] = value

	return context.WithValue(ctx, systemColumnKey{}, newSystemValues)
}

// getSystemValuesFromContext retrieves system values from context
func getSystemValuesFromContext(ctx context.Context) map[string]any {
	if values := ctx.Value(systemColumnKey{}); values != nil {
		if systemValues, ok := values.(map[string]any); ok {
			return systemValues
		}
	}

	return nil
}
