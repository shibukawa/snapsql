package markdownparser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/goccy/go-yaml"
	snapsql "github.com/shibukawa/snapsql"
)

// parseParameters parses parameter data in various formats into a single map
func parseParameters(content []byte) (map[string]any, error) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return nil, snapsql.ErrEmptyContent
	}

	// Try YAML/JSON first
	var params map[string]any

	err := yaml.Unmarshal(content, &params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML parameters: %w", err)
	}

	if len(params) == 0 {
		return nil, snapsql.ErrEmptyParameters
	}

	// Convert numeric types to ensure consistency
	for k, v := range params {
		params[k] = normalizeValue(v)
	}

	return params, nil
}

// parseYAMLData parses YAML data into a slice of maps
func parseYAMLData(content []byte, result *[]map[string]any) error {
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return snapsql.ErrEmptyContent
	}

	err := yaml.Unmarshal(content, result)
	if err != nil {
		return fmt.Errorf("failed to parse YAML data: %w", err)
	}

	// Normalize values
	for i, row := range *result {
		for k, v := range row {
			(*result)[i][k] = normalizeValue(v)
		}
	}

	return nil
}

// parseStructuredData parses data in various formats into a map of table names to rows
func parseStructuredData(content []byte, format string) (map[string][]map[string]any, error) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return nil, snapsql.ErrEmptyContent
	}

	result := make(map[string][]map[string]any)

	switch format {
	case "yaml", "json":
		// Try YAML/JSON array format first
		var data map[string]any

		err := yaml.Unmarshal(content, &data)
		if err == nil {
			// データがテーブル名をキーとしたマップの場合
			for tableName, tableContent := range data {
				if rows, ok := tableContent.([]any); ok {
					tableRows := make([]map[string]any, 0, len(rows))

					for _, row := range rows {
						if mapRow, ok := row.(map[string]any); ok {
							// Normalize values
							normalizedRow := make(map[string]any)
							for k, v := range mapRow {
								normalizedRow[k] = normalizeValue(v)
							}

							tableRows = append(tableRows, normalizedRow)
						}
					}

					result[tableName] = tableRows
				}
			}

			if len(result) > 0 {
				return result, nil
			}
		}

	case "xml":
		// Try DBUnit XML
		if dataset, err := parseDBUnitXML(string(content)); err == nil {
			for _, table := range dataset.Tables {
				rows := make([]map[string]any, len(table.Rows))
				for i, row := range table.Rows {
					rows[i] = row.Data
				}

				result[table.Name] = rows
			}

			if len(result) > 0 {
				return result, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: %q", snapsql.ErrFailedToParse, string(content))
}

// parseExpectedResults parses expected results data
func parseExpectedResults(content []byte) ([]map[string]any, error) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return nil, snapsql.ErrEmptyContent
	}

	// Try YAML/JSON array format
	var result []map[string]any

	err := yaml.Unmarshal(content, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected results: %w", err)
	}

	// Normalize values
	for i, item := range result {
		for k, v := range item {
			item[k] = normalizeValue(v)
		}

		result[i] = item
	}

	if len(result) == 0 {
		return nil, snapsql.ErrEmptyExpectedResults
	}

	return result, nil
}

// parseValue converts string value to appropriate type
func parseValue(value string) any {
	value = strings.TrimSpace(value)

	// Try boolean
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}

	// Try integer
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// Try array
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		items := strings.Split(strings.Trim(value, "[]"), ",")

		result := make([]any, 0, len(items))
		for _, item := range items {
			result = append(result, parseValue(strings.TrimSpace(item)))
		}

		return result
	}

	// Remove quotes if present
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		return strings.Trim(value, "\"'")
	}

	// Default to string
	return value
}

// normalizeValue ensures consistent types for values
func normalizeValue(v any) any {
	switch val := v.(type) {
	case float64:
		// Convert to int if it's a whole number
		if float64(int64(val)) == val {
			return int64(val)
		}

		return val
	case float32:
		// Convert to int if it's a whole number
		if float32(int32(val)) == val {
			return int32(val)
		}

		return val
	case []any:
		// Recursively normalize array values
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = normalizeValue(item)
		}

		return result
	case map[any]any:
		// Convert map keys to strings and normalize values
		result := make(map[string]any)

		for k, v := range val {
			if strKey, ok := k.(string); ok {
				result[strKey] = normalizeValue(v)
			}
		}

		return result
	case map[string]any:
		// Recursively normalize map values
		result := make(map[string]any)
		for k, v := range val {
			result[k] = normalizeValue(v)
		}

		return result
	default:
		return v
	}
}

// DBUnitXML represents the structure of a DBUnit XML dataset
type DBUnitXML struct {
	Tables []struct {
		Name string
		Rows []struct {
			Data map[string]any
		}
	}
}

// parseDBUnitXML parses DBUnit XML format
func parseDBUnitXML(content string) (*DBUnitXML, error) {
	doc := &DBUnitXML{}

	// Parse XML
	root := etree.NewDocument()

	err := root.ReadFromString(content)
	if err != nil {
		return nil, err
	}

	// Process dataset
	dataset := root.SelectElement("dataset")
	if dataset == nil {
		return nil, snapsql.ErrNoDatasetElement
	}

	// Group elements by tag name (table name)
	tableMap := make(map[string][]map[string]any)

	for _, elem := range dataset.ChildElements() {
		tableName := elem.Tag
		row := make(map[string]any)

		// Convert attributes to map and check for table field
		hasTableField := false

		for _, attr := range elem.Attr {
			row[attr.Key] = parseValue(attr.Value)

			if attr.Key == "table" {
				hasTableField = true
			}
		}

		// Add table name to the row if there's no conflict
		if !hasTableField {
			row["_table"] = tableName // Use _table to avoid conflicts
		}

		tableMap[tableName] = append(tableMap[tableName], row)
	}

	// Convert to DBUnitXML structure
	for tableName, rows := range tableMap {
		tableData := struct {
			Name string
			Rows []struct{ Data map[string]any }
		}{
			Name: tableName,
			Rows: make([]struct{ Data map[string]any }, len(rows)),
		}

		for i, row := range rows {
			tableData.Rows[i].Data = row
		}

		doc.Tables = append(doc.Tables, tableData)
	}

	return doc, nil
}

// parseCSVData parses CSV content into a slice of maps
func parseCSVData(content []byte) ([]map[string]any, error) {
	records, err := parseCSV(string(content))
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, snapsql.ErrInvalidCSVFormat
	}

	headers := records[0]
	result := make([]map[string]any, 0, len(records)-1)

	for _, record := range records[1:] {
		if len(record) > 0 && !isEmptyRow(record) {
			row := make(map[string]any)

			for j, value := range record {
				if j < len(headers) {
					header := strings.TrimSpace(headers[j])
					if header != "" {
						row[header] = parseValue(strings.TrimSpace(value))
					}
				}
			}

			if len(row) > 0 {
				result = append(result, row)
			}
		}
	}

	return result, nil
}

// parseCSV parses CSV content into records
func parseCSV(content string) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(content))
	r.TrimLeadingSpace = true
	r.Comment = '#'

	return r.ReadAll()
}

// isEmptyRow checks if a CSV row is empty or contains only whitespace
func isEmptyRow(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}

	return true
}
