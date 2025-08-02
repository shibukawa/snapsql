package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/markdownparser"
)

// Error definitions for template loading
var (
	ErrUnsupportedFileFormat = fmt.Errorf("unsupported template file format")
	ErrFileNotFound          = fmt.Errorf("template file not found")
	ErrFileRead              = fmt.Errorf("failed to read template file")
	ErrTemplateGeneration    = fmt.Errorf("failed to generate intermediate format")
	ErrParameterParsing      = fmt.Errorf("failed to parse parameters")
)

// LoadIntermediateFormat loads an intermediate format from various file types
func LoadIntermediateFormat(templateFile string) (*intermediate.IntermediateFormat, error) {
	// Check if file exists
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrFileNotFound, templateFile)
	}

	ext := strings.ToLower(filepath.Ext(templateFile))
	
	switch ext {
	case ".json":
		return loadFromJSON(templateFile)
	case ".sql":
		return generateFromSQL(templateFile)
	case ".md":
		return generateFromMarkdown(templateFile)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFileFormat, ext)
	}
}

// loadFromJSON loads intermediate format from a JSON file
func loadFromJSON(filename string) (*intermediate.IntermediateFormat, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFileRead, err)
	}

	format, err := intermediate.FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplateGeneration, err)
	}

	return format, nil
}

// generateFromSQL generates intermediate format from a .snap.sql file
func generateFromSQL(filename string) (*intermediate.IntermediateFormat, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFileRead, err)
	}
	defer file.Close()

	// Generate intermediate format using the existing pipeline
	format, err := intermediate.GenerateFromSQL(file, nil, filename, ".", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplateGeneration, err)
	}

	return format, nil
}

// generateFromMarkdown generates intermediate format from a .snap.md file
func generateFromMarkdown(filename string) (*intermediate.IntermediateFormat, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFileRead, err)
	}
	defer file.Close()

	// Parse markdown to extract SQL and parameters
	doc, err := markdownparser.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse markdown: %v", ErrTemplateGeneration, err)
	}

	// Generate intermediate format from the markdown document directly
	format, err := intermediate.GenerateFromMarkdown(doc, filename, ".", nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplateGeneration, err)
	}

	// Add metadata from markdown if available
	if doc.Metadata != nil {
		// Set function name if available
		if functionName, ok := doc.Metadata["function_name"].(string); ok {
			format.FunctionName = functionName
		}
		
		// Set description if available
		if description, ok := doc.Metadata["description"].(string); ok {
			format.Description = description
		}
	}

	return format, nil
}
