package query

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

// Formatter formats query results
type Formatter struct {
	FormatType OutputFormat
}

// NewFormatter creates a new result formatter
func NewFormatter(format OutputFormat) *Formatter {
	return &Formatter{
		FormatType: format,
	}
}

// Format formats query results according to the specified format
func (f *Formatter) Format(result *QueryResult, output io.Writer) error {
	switch f.FormatType {
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
		return fmt.Errorf("%w: %s", ErrInvalidOutputFormat, f.FormatType)
	}
}

// FormatExplain formats an EXPLAIN result
func (f *Formatter) FormatExplain(result *QueryResult, output io.Writer) error {
	// For EXPLAIN queries, just output the plan
	_, err := fmt.Fprintln(output, result.ExplainPlan)
	return err
}

// formatAsTable formats results as a text table
func (f *Formatter) formatAsTable(result *QueryResult, output io.Writer) error {
	if len(result.Rows) == 0 {
		fmt.Fprintln(output, "No results")
		return nil
	}

	// Convert rows to maps for formatting
	data := rowsToMaps(result.Columns, result.Rows)

	// Create a simple ASCII table
	var sb strings.Builder

	// Calculate column widths
	colWidths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		colWidths[i] = len(col)
	}

	for _, row := range data {
		for i, col := range result.Columns {
			strVal := formatValue(row[col])
			if len(strVal) > colWidths[i] {
				colWidths[i] = len(strVal)
			}
		}
	}

	// Print header
	sb.WriteString("+")

	for _, width := range colWidths {
		sb.WriteString(strings.Repeat("-", width+2))
		sb.WriteString("+")
	}

	sb.WriteString("\n|")

	for i, col := range result.Columns {
		sb.WriteString(fmt.Sprintf(" %-*s |", colWidths[i], col))
	}

	sb.WriteString("\n+")

	for _, width := range colWidths {
		sb.WriteString(strings.Repeat("-", width+2))
		sb.WriteString("+")
	}

	sb.WriteString("\n")

	// Print rows
	for _, row := range data {
		sb.WriteString("|")

		for i, col := range result.Columns {
			strVal := formatValue(row[col])
			sb.WriteString(fmt.Sprintf(" %-*s |", colWidths[i], strVal))
		}

		sb.WriteString("\n")
	}

	// Print footer
	sb.WriteString("+")

	for _, width := range colWidths {
		sb.WriteString(strings.Repeat("-", width+2))
		sb.WriteString("+")
	}

	sb.WriteString("\n")

	// Add count and duration
	sb.WriteString(fmt.Sprintf("%d rows in set (%.3f sec)\n", result.Count, result.Duration.Seconds()))

	_, err := fmt.Fprintln(output, sb.String())

	return err
}

// formatAsMarkdown formats results as a Markdown table
func (f *Formatter) formatAsMarkdown(result *QueryResult, output io.Writer) error {
	if len(result.Rows) == 0 {
		fmt.Fprintln(output, "No results")
		return nil
	}

	// Convert rows to maps for formatting
	data := rowsToMaps(result.Columns, result.Rows)

	// Create a Markdown table
	var sb strings.Builder

	// Print header
	sb.WriteString("| ")

	for _, col := range result.Columns {
		sb.WriteString(col)
		sb.WriteString(" | ")
	}

	sb.WriteString("\n|")

	for range result.Columns {
		sb.WriteString(" --- |")
	}

	sb.WriteString("\n")

	// Print rows
	for _, row := range data {
		sb.WriteString("| ")

		for _, col := range result.Columns {
			strVal := formatValue(row[col])
			sb.WriteString(strVal)
			sb.WriteString(" | ")
		}

		sb.WriteString("\n")
	}

	// Add footer as a comment
	sb.WriteString(fmt.Sprintf("\n<!-- %d rows, Time: %v -->", result.Count, result.Duration))

	_, err := fmt.Fprintln(output, sb.String())

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
	err := writer.Write(result.Columns)
	if err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, row := range result.Rows {
		// Convert row values to strings
		strValues := make([]string, len(row))
		for i, val := range row {
			strValues[i] = formatValue(val)
		}

		err := writer.Write(strValues)
		if err != nil {
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
		return strconv.FormatBool(v)
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
