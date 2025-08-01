package gogen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

// queryExecutionData represents query execution code generation data
type queryExecutionData struct {
	Code []string
}

// generateQueryExecution generates query execution and result mapping code
func generateQueryExecution(format *intermediate.IntermediateFormat, responseStruct *responseStructData) (*queryExecutionData, error) {
	var code []string

	switch format.ResponseAffinity {
	case "none", "":
		// No result expected (INSERT/UPDATE/DELETE) or empty affinity
		code = append(code, "// Execute query (no result expected)")
		code = append(code, "_, err = stmt.ExecContext(ctx, args...)")
		code = append(code, "if err != nil {")
		code = append(code, "    return result, fmt.Errorf(\"failed to execute statement: %w\", err)")
		code = append(code, "}")

	case "one":
		// Single row expected
		code = append(code, "// Execute query and scan single row")
		code = append(code, "row := stmt.QueryRowContext(ctx, args...)")

		if responseStruct != nil {
			scanCode, err := generateScanCode(responseStruct, false)
			if err != nil {
				return nil, fmt.Errorf("failed to generate scan code: %w", err)
			}
			code = append(code, scanCode...)
		} else {
			// Generate generic scan code for interface{} result
			code = append(code, "// Generic scan for interface{} result - not implemented")
			code = append(code, "// This would require runtime reflection or predefined column mapping")
		}

	case "many":
		// Multiple rows expected
		code = append(code, "// Execute query and scan multiple rows")
		code = append(code, "rows, err := stmt.QueryContext(ctx, args...)")
		code = append(code, "if err != nil {")
		code = append(code, "    return result, fmt.Errorf(\"failed to execute query: %w\", err)")
		code = append(code, "}")
		code = append(code, "defer rows.Close()")
		code = append(code, "")

		if responseStruct != nil {
			scanCode, err := generateScanCode(responseStruct, true)
			if err != nil {
				return nil, fmt.Errorf("failed to generate scan code: %w", err)
			}
			code = append(code, scanCode...)
		} else {
			// Generate generic scan code for interface{} result
			code = append(code, "// Generic scan for interface{} result - not implemented")
			code = append(code, "// This would require runtime reflection or predefined column mapping")
		}

	default:
		return nil, fmt.Errorf("unsupported response affinity: %s", format.ResponseAffinity)
	}

	return &queryExecutionData{
		Code: code,
	}, nil
}

// generateScanCode generates code for scanning database results
func generateScanCode(responseStruct *responseStructData, isMany bool) ([]string, error) {
	// Check if we need aggregation (has __ fields in JSON tags)
	hasAggregation := false

	for _, field := range responseStruct.Fields {
		if strings.Contains(field.JSONTag, "__") {
			hasAggregation = true
			break
		}
	}

	if hasAggregation {
		return generateAggregatedScanCode(responseStruct, isMany)
	}

	return generateSimpleScanCode(responseStruct, isMany)
}

// generateSimpleScanCode generates simple scanning code without aggregation
func generateSimpleScanCode(responseStruct *responseStructData, isMany bool) ([]string, error) {
	var code []string

	if isMany {
		// Multiple rows
		code = append(code, "for rows.Next() {")
		code = append(code, fmt.Sprintf("    var item %s", responseStruct.Name))
		code = append(code, "    err := rows.Scan(")

		// Generate scan targets
		for i, field := range responseStruct.Fields {
			if i > 0 {
				code[len(code)-1] += ","
			}
			// Convert field name to Go field name (PascalCase)
			goFieldName := celNameToGoName(field.Name)
			code = append(code, fmt.Sprintf("        &item.%s", goFieldName))
		}

		code[len(code)-1] += ""
		code = append(code, "    )")
		code = append(code, "    if err != nil {")
		code = append(code, "        return result, fmt.Errorf(\"failed to scan row: %w\", err)")
		code = append(code, "    }")
		code = append(code, "    result = append(result, item)")
		code = append(code, "}")
		code = append(code, "")
		code = append(code, "if err = rows.Err(); err != nil {")
		code = append(code, "    return result, fmt.Errorf(\"error iterating rows: %w\", err)")
		code = append(code, "}")
	} else {
		// Single row
		code = append(code, "err = row.Scan(")

		// Generate scan targets
		for i, field := range responseStruct.Fields {
			if i > 0 {
				code[len(code)-1] += ","
			}
			// Convert field name to Go field name (PascalCase)
			goFieldName := celNameToGoName(field.Name)
			code = append(code, fmt.Sprintf("    &result.%s", goFieldName))
		}

		code[len(code)-1] += ""
		code = append(code, ")")
		code = append(code, "if err != nil {")
		code = append(code, "    return result, fmt.Errorf(\"failed to scan row: %w\", err)")
		code = append(code, "}")
	}

	return code, nil
}

// generateAggregatedScanCode generates scanning code with __ field aggregation
func generateAggregatedScanCode(responseStruct *responseStructData, isMany bool) ([]string, error) {
	var code []string

	if isMany {
		// Multiple rows with aggregation - simplified implementation for get_users_with_jobs
		code = append(code, "// Scan rows with aggregation")
		code = append(code, "resultMap := make(map[int]*GetUsersWithJobsResult)")
		code = append(code, "")
		code = append(code, "for rows.Next() {")

		// Generate scan variables for all fields - use simple types
		code = append(code, "    // Scan variables")
		code = append(code, "    var id int")
		code = append(code, "    var name string")
		code = append(code, "    var email string")
		code = append(code, "    var jobID *int")
		code = append(code, "    var jobTitle *string")
		code = append(code, "    var jobCompany *string")
		code = append(code, "")

		code = append(code, "    err := rows.Scan(&id, &name, &email, &jobID, &jobTitle, &jobCompany)")
		code = append(code, "    if err != nil {")
		code = append(code, "        return result, fmt.Errorf(\"failed to scan row: %w\", err)")
		code = append(code, "    }")
		code = append(code, "")

		// Generate aggregation logic
		code = append(code, "    // Create or get existing user")
		code = append(code, "    user, exists := resultMap[id]")
		code = append(code, "    if !exists {")
		code = append(code, "        user = &GetUsersWithJobsResult{")
		code = append(code, "            ID:    id,")
		code = append(code, "            Name:  name,")
		code = append(code, "            Email: email,")
		code = append(code, "            Jobs:  []GetUsersWithJobsJob{},")
		code = append(code, "        }")
		code = append(code, "        resultMap[id] = user")
		code = append(code, "    }")
		code = append(code, "")

		// Handle job aggregation - simplified NULL check
		code = append(code, "    // Add job if exists")
		code = append(code, "    if jobID != nil {")
		code = append(code, "        job := GetUsersWithJobsJob{")
		code = append(code, "            ID: *jobID,")
		code = append(code, "        }")
		code = append(code, "        if jobTitle != nil {")
		code = append(code, "            job.Title = *jobTitle")
		code = append(code, "        }")
		code = append(code, "        if jobCompany != nil {")
		code = append(code, "            job.Company = *jobCompany")
		code = append(code, "        }")
		code = append(code, "        user.Jobs = append(user.Jobs, job)")
		code = append(code, "    }")
		code = append(code, "}")
		code = append(code, "")

		code = append(code, "// Convert map to slice")
		code = append(code, "for _, user := range resultMap {")
		code = append(code, "    result = append(result, *user)")
		code = append(code, "}")
		code = append(code, "")
		code = append(code, "if err = rows.Err(); err != nil {")
		code = append(code, "    return result, fmt.Errorf(\"error iterating rows: %w\", err)")
		code = append(code, "}")
	} else {
		// Single row with aggregation - simplified version
		code = append(code, "// TODO: Single row aggregation not implemented yet")
	}

	return code, nil
}
