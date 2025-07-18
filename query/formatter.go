package query

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/shibukawa/formatdata-go"
)

// Formatter formats query results
type Formatter struct {
	Format OutputFormat
}

// NewFormatter creates a new result formatter
func NewFormatter(format OutputFormat) *Formatter {
	return &Formatter{
		Format: format,
	}
}

// Format formats query results according to the specified format
func (f *Formatter) Format(result *QueryResult, output io.Writer) error {
	switch f.Format {
	case FormatTable:
		return f.formatAsTable(result, output)
	case FormatJSON:
		return f.formatAsJSON(result, output)
	case FormatCSV:
		return f.formatAsCSV(result, output)
	case FormatYAML:
		return f.formatAsYAML(result, output)
	case FormatMarkdown:
		return f.formatAsMarkdown(result, output)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidOutputFormat, f.Format)
	}
}

// FormatExplain formats an EXPLAIN result
func (f *Formatter) FormatExplain(result *QueryResult, output io.Writer) error {
	// For EXPLAIN queries, just output the plan
	_, err := fmt.Fprintln(output, result.ExplainPlan)
	return err
}

// formatAsTable formats results as a text table using formatdata-go
func (f *Formatter) formatAsTable(result *QueryResult, output io.Writer) error {
	if len(result.Rows) == 0 {
		fmt.Fprintln(output, "No results")
		return nil
	}

	// Convert rows to maps for formatdata
	data := rowsToMaps(result.Columns, result.Rows)

	// Create table formatter
	table := formatdata.NewTableFormatter()
	table.SetHeader(result.Columns)
	
	// Add footer with count and duration
	footer := []string{
		fmt.Sprintf("%d rows", result.Count),
		fmt.Sprintf("Time: %v", result.Duration),
	}
	// Pad footer to match column count
	for len(footer) < len(result.Columns) {
		footer = append(footer, "")
	}
	table.SetFooter(footer[:len(result.Columns)])

	// Format and output
	formatted := table.Format(data)
	_, err := fmt.Fprintln(output, formatted)
	return err
}

// formatAsMarkdown formats results as a Markdown table using formatdata-go
func (f *Formatter) formatAsMarkdown(result *QueryResult, output io.Writer) error {
	if len(result.Rows) == 0 {
		fmt.Fprintln(output, "No results")
		return nil
	}

	// Convert rows to maps for formatdata
	data := rowsToMaps(result.Columns, result.Rows)

	// Create markdown formatter
	md := formatdata.NewMarkdownFormatter()
	md.SetHeader(result.Columns)

	// Format and output
	formatted := md.Format(data)
	
	// Add footer as a comment
	footer := fmt.Sprintf("\n<!-- %d rows, Time: %v -->", result.Count, result.Duration)
	
	_, err := fmt.Fprintln(output, formatted+footer)
	return err
}

// formatAsJSON formats results as JSON
func (f *Formatter) formatAsJSON(result *QueryResult, output io.Writer) error {
	// Convert rows to maps
	maps := rowsToMaps(result.Columns, result.Rows)

	// Create result object
	jsonResult := map[string]interface{}{
		"data":     maps,
		"count":    result.Count,
		"duration": result.Duration.String(),
	}

	// Encode as JSON
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonResult)
}

// formatAsCSV formats results as CSV
func (f *Formatter) formatAsCSV(result *QueryResult, output io.Writer) error {
	writer := csv.NewWriter(output)
	defer writer.Flush()

	// Write header
	if err := writer.Write(result.Columns); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, row := range result.Rows {
		// Convert row values to strings
		strValues := make([]string, len(row))
		for i, val := range row {
			strValues[i] = formatValue(val)
		}
		if err := writer.Write(strValues); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// formatAsYAML formats results as YAML
func (f *Formatter) formatAsYAML(result *QueryResult, output io.Writer) error {
	// Convert rows to maps
	maps := rowsToMaps(result.Columns, result.Rows)

	// Create result object
	yamlResult := map[string]interface{}{
		"data":     maps,
		"count":    result.Count,
		"duration": result.Duration.String(),
	}

	// Encode as YAML
	data, err := yaml.Marshal(yamlResult)
	if err != nil {
		return fmt.Errorf("failed to marshal results to YAML: %w", err)
	}
	_, err = output.Write(data)
	return err
}

// rowsToMaps converts rows to maps
func rowsToMaps(columns []string, rows [][]interface{}) []map[string]interface{} {
	var result []map[string]interface{}
	for _, row := range rows {
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			if i < len(row) {
				rowMap[col] = row[i]
			}
		}
		result = append(result, rowMap)
	}
	return result
}

// formatValue formats a value as a string
func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		return fmt.Sprintf("%t", v)
	case map[string]interface{}:
		// Format JSON objects
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	case []interface{}:
		// Format JSON arrays
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// IsValidOutputFormat checks if the output format is valid
func IsValidOutputFormat(format string) bool {
	f := OutputFormat(strings.ToLower(format))
	return f == FormatTable || f == FormatJSON || f == FormatCSV || f == FormatYAML || f == FormatMarkdown
}
