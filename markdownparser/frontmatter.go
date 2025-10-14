package markdownparser

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

var (
	errPerformanceMapType       = errors.New("performance must be a map with string keys")
	errPerformanceThresholdType = errors.New("performance.slow_query_threshold must be a string duration")
)

// parseFrontMatter extracts YAML front matter from markdown content
func parseFrontMatter(content string) (map[string]any, string, error) {
	// Check if content starts with front matter delimiter
	if !strings.HasPrefix(content, "---\n") {
		return make(map[string]any), content, nil
	}

	// Find the closing delimiter
	endIndex := strings.Index(content[4:], "\n---")
	if endIndex == -1 {
		return nil, "", ErrInvalidFrontMatter
	}

	endIndex += 4 // Adjust for the initial slice

	// Extract front matter content
	frontMatterContent := content[4:endIndex]
	remainingContent := content[endIndex+4:]

	// Parse YAML front matter
	var frontMatter map[string]any

	err := yaml.Unmarshal([]byte(frontMatterContent), &frontMatter)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %w", ErrInvalidFrontMatter, err)
	}

	return frontMatter, remainingContent, nil
}

func parsePerformanceSettings(frontMatter map[string]any) (PerformanceSettings, error) {
	var settings PerformanceSettings

	if frontMatter == nil {
		return settings, nil
	}

	raw, ok := frontMatter["performance"]
	if !ok || raw == nil {
		return settings, nil
	}

	perfMap, ok := normalizeStringMap(raw)
	if !ok {
		return settings, errPerformanceMapType
	}

	if rawThreshold, exists := perfMap["slow_query_threshold"]; exists && rawThreshold != nil {
		thresholdStr, ok := rawThreshold.(string)
		if !ok {
			return settings, errPerformanceThresholdType
		}

		thresholdStr = strings.TrimSpace(thresholdStr)
		if thresholdStr != "" {
			dur, err := time.ParseDuration(thresholdStr)
			if err != nil {
				return settings, fmt.Errorf("invalid performance.slow_query_threshold: %w", err)
			}

			settings.SlowQueryThreshold = dur
		}
	}

	return settings, nil
}

func normalizeStringMap(value any) (map[string]any, bool) {
	switch m := value.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, v := range m {
			key, ok := k.(string)
			if !ok {
				return nil, false
			}

			out[key] = v
		}

		return out, true
	default:
		return nil, false
	}
}

// generateFunctionNameFromTitle generates a snake_case function name from title
// Note: function name is resolved from front matter or file name, not from title.
