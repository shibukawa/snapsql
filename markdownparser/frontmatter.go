package markdownparser

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
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

// generateFunctionNameFromTitle generates a snake_case function name from title
func generateFunctionNameFromTitle(title string) string {
	words := strings.Fields(strings.ToLower(title))
	if len(words) == 0 {
		return "query"
	}

	return strings.Join(words, "_")
}
