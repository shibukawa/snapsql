package pygen

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

// generateQueryExecution generates query execution and result mapping code for Python
func generateQueryExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect string,
) (*queryExecutionData, error) {
	// Determine response affinity
	affinity := strings.ToLower(format.ResponseAffinity)
	if affinity == "" {
		affinity = "none"
	}

	switch affinity {
	case "none":
		return generateNoneAffinityExecution(format, dialect)
	case "one":
		return generateOneAffinityExecution(format, responseStruct, dialect)
	case "many":
		return generateManyAffinityExecution(format, responseStruct, dialect)
	default:
		return nil, fmt.Errorf("unsupported response affinity: %s", format.ResponseAffinity)
	}
}

// generateNoneAffinityExecution generates code for queries that don't return results (INSERT/UPDATE/DELETE)
func generateNoneAffinityExecution(format *intermediate.IntermediateFormat, dialect string) (*queryExecutionData, error) {
	var code []string

	code = append(code, "# Execute query (no result expected)")

	switch dialect {
	case "postgres":
		// PostgreSQL with asyncpg
		code = append(code, "result = await conn.execute(sql, *args)")
		code = append(code, "# asyncpg returns status string like 'INSERT 0 1' or 'UPDATE 3'")
		code = append(code, "# Extract the number of affected rows")
		code = append(code, "if result.startswith('INSERT'):")
		code = append(code, "    parts = result.split()")
		code = append(code, "    affected_rows = int(parts[-1]) if len(parts) > 2 else 0")
		code = append(code, "elif result.startswith('UPDATE') or result.startswith('DELETE'):")
		code = append(code, "    parts = result.split()")
		code = append(code, "    affected_rows = int(parts[-1]) if len(parts) > 1 else 0")
		code = append(code, "else:")
		code = append(code, "    affected_rows = 0")
		code = append(code, "return affected_rows")

	case "mysql", "sqlite":
		// MySQL with aiomysql or SQLite with aiosqlite
		code = append(code, "await cursor.execute(sql, args)")
		code = append(code, "return cursor.rowcount")

	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	return &queryExecutionData{
		Code: strings.Join(code, "\n"),
	}, nil
}

// generateOneAffinityExecution generates code for queries that return a single row
func generateOneAffinityExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect string,
) (*queryExecutionData, error) {
	if responseStruct == nil {
		return nil, errors.New("response struct required for 'one' affinity")
	}

	// Check if this is hierarchical structure
	isHierarchical := hasHierarchicalFields(format.Responses)

	if isHierarchical {
		return generateHierarchicalOneExecution(format, responseStruct, dialect)
	}

	var code []string

	code = append(code, "# Execute query and fetch single row")

	switch dialect {
	case "postgres":
		// PostgreSQL with asyncpg
		code = append(code, "row = await conn.fetchrow(sql, *args)")
		code = append(code, "")
		code = append(code, "if row is None:")
		code = append(code, "    # Build parameter dict for error message")
		code = append(code, "    param_dict = {}")
		code = append(code, "    # Note: args is a list, parameter names would need to be tracked separately")
		code = append(code, "    raise NotFoundError(")
		code = append(code, "        message=\"Record not found\",")
		code = append(code, fmt.Sprintf("        func_name=%q,", format.FunctionName))
		code = append(code, "        query=sql,")
		code = append(code, "        params=param_dict")
		code = append(code, "    )")
		code = append(code, "")
		code = append(code, "# Map row to dataclass")
		code = append(code, "# asyncpg returns Record which supports dict() conversion")
		code = append(code, fmt.Sprintf("return %s(**dict(row))", responseStruct.ClassName))

	case "mysql", "sqlite":
		// MySQL with aiomysql or SQLite with aiosqlite
		code = append(code, "await cursor.execute(sql, args)")
		code = append(code, "row = await cursor.fetchone()")
		code = append(code, "")
		code = append(code, "if row is None:")
		code = append(code, "    # Build parameter dict for error message")
		code = append(code, "    param_dict = {}")
		code = append(code, "    # Note: args is a list, parameter names would need to be tracked separately")
		code = append(code, "    raise NotFoundError(")
		code = append(code, "        message=\"Record not found\",")
		code = append(code, fmt.Sprintf("        func_name=%q,", format.FunctionName))
		code = append(code, "        query=sql,")
		code = append(code, "        params=param_dict")
		code = append(code, "    )")
		code = append(code, "")

		code = append(code, "# Map row to dataclass")
		if dialect == "mysql" {
			code = append(code, "# aiomysql with DictCursor returns dict")
		} else {
			code = append(code, "# aiosqlite with row_factory returns dict-like object")
		}

		code = append(code, fmt.Sprintf("return %s(**row)", responseStruct.ClassName))

	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	return &queryExecutionData{
		Code: strings.Join(code, "\n"),
	}, nil
}

// generateHierarchicalOneExecution generates code for hierarchical data aggregation with one affinity
func generateHierarchicalOneExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect string,
) (*queryExecutionData, error) {
	// Detect hierarchical structure
	nodes, rootFields, err := detectHierarchicalStructure(format.Responses)
	if err != nil {
		return nil, fmt.Errorf("failed to detect hierarchical structure: %w", err)
	}

	// Find parent key fields
	parentKeyFields := findParentKeyFields(rootFields)
	if len(parentKeyFields) == 0 {
		return nil, errors.New("no parent key field found for hierarchical aggregation")
	}

	var code []string

	code = append(code, "# Execute query for hierarchical aggregation (one affinity)")
	code = append(code, "# This aggregates multiple rows into a single parent object with child lists")
	code = append(code, "")

	// Execute query based on dialect
	switch dialect {
	case "postgres":
		code = append(code, "rows = await conn.fetch(sql, *args)")
		code = append(code, "")
		code = append(code, "if not rows:")
		code = append(code, "    raise NotFoundError(")
		code = append(code, "        message=\"Record not found\",")
		code = append(code, fmt.Sprintf("        func_name=%q,", format.FunctionName))
		code = append(code, "        query=sql")
		code = append(code, "    )")
		code = append(code, "")
		code = append(code, "# Process rows for hierarchical aggregation")
		code = append(code, "result = None")
		code = append(code, "")
		code = append(code, "for row in rows:")
		code = append(code, "    row_dict = dict(row)")

	case "mysql", "sqlite":
		code = append(code, "await cursor.execute(sql, args)")
		code = append(code, "")
		code = append(code, "# Collect all rows first")
		code = append(code, "rows = []")
		code = append(code, "async for row in cursor:")
		code = append(code, "    rows.append(row if isinstance(row, dict) else dict(row))")
		code = append(code, "")
		code = append(code, "if not rows:")
		code = append(code, "    raise NotFoundError(")
		code = append(code, "        message=\"Record not found\",")
		code = append(code, fmt.Sprintf("        func_name=%q,", format.FunctionName))
		code = append(code, "        query=sql")
		code = append(code, "    )")
		code = append(code, "")
		code = append(code, "# Process rows for hierarchical aggregation")
		code = append(code, "result = None")
		code = append(code, "")
		code = append(code, "for row_dict in rows:")

	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	// Generate aggregation logic
	code = append(code, "    ")
	code = append(code, "    # Create parent object on first row")
	code = append(code, "    if result is None:")

	// Generate parent object creation with root fields
	parentFields := make([]string, 0)
	for _, rf := range rootFields {
		parentFields = append(parentFields, fmt.Sprintf("            %s=row_dict.get('%s')", rf.Name, rf.JSONTag))
	}

	// Add child list fields initialized to empty lists
	for key := range nodes {
		n := nodes[key]
		if len(n.PathSegments) == 1 {
			childFieldName := toSnakeCase(n.PathSegments[0])
			parentFields = append(parentFields, fmt.Sprintf("            %s=[]", childFieldName))
		}
	}

	code = append(code, fmt.Sprintf("        result = %s(", responseStruct.ClassName))
	code = append(code, strings.Join(parentFields, ",\n"))
	code = append(code, "        )")
	code = append(code, "    ")

	// Generate child object creation and appending
	code = append(code, "    # Add child objects if present")

	// Process each top-level child group
	for key := range nodes {
		n := nodes[key]
		if len(n.PathSegments) != 1 {
			continue
		}

		childFieldName := toSnakeCase(n.PathSegments[0])
		childClassName := generateChildClassName(responseStruct.ClassName, n.PathSegments)

		// Check if child data exists
		childCheckFields := make([]string, 0)
		for _, f := range n.Fields {
			childCheckFields = append(childCheckFields, fmt.Sprintf("row_dict.get('%s__%s')", n.PathSegments[0], f.JSONTag))
		}

		code = append(code, fmt.Sprintf("    if any([%s]):", strings.Join(childCheckFields, ", ")))

		// Create child object
		childFields := make([]string, 0)
		for _, f := range n.Fields {
			childFields = append(childFields, fmt.Sprintf("            %s=row_dict.get('%s__%s')", f.Name, n.PathSegments[0], f.JSONTag))
		}

		// Add nested children if any
		for childKey := range n.Children {
			nestedFieldName := toSnakeCase(childKey)
			childFields = append(childFields, fmt.Sprintf("            %s=[]", nestedFieldName))
		}

		code = append(code, fmt.Sprintf("        child_obj = %s(", childClassName))
		code = append(code, strings.Join(childFields, ",\n"))
		code = append(code, "        )")
		code = append(code, fmt.Sprintf("        result.%s.append(child_obj)", childFieldName))
	}

	code = append(code, "")
	code = append(code, "return result")

	return &queryExecutionData{
		Code: strings.Join(code, "\n"),
	}, nil
}

// generateManyAffinityExecution generates code for queries that return multiple rows
func generateManyAffinityExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect string,
) (*queryExecutionData, error) {
	if responseStruct == nil {
		return nil, errors.New("response struct required for 'many' affinity")
	}

	// Check if this is hierarchical structure
	isHierarchical := hasHierarchicalFields(format.Responses)

	if isHierarchical {
		return generateHierarchicalManyExecution(format, responseStruct, dialect)
	}

	var code []string

	code = append(code, "# Execute query and yield rows as async generator")

	switch dialect {
	case "postgres":
		// PostgreSQL with asyncpg - use async for with fetch
		code = append(code, "rows = await conn.fetch(sql, *args)")
		code = append(code, "")
		code = append(code, "# Yield each row as dataclass instance")
		code = append(code, "for row in rows:")
		code = append(code, fmt.Sprintf("    yield %s(**dict(row))", responseStruct.ClassName))

	case "mysql", "sqlite":
		// MySQL with aiomysql or SQLite with aiosqlite
		code = append(code, "await cursor.execute(sql, args)")
		code = append(code, "")
		code = append(code, "# Fetch and yield rows")
		code = append(code, "async for row in cursor:")
		code = append(code, fmt.Sprintf("    yield %s(**row)", responseStruct.ClassName))

	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	return &queryExecutionData{
		Code: strings.Join(code, "\n"),
	}, nil
}

// hasHierarchicalFields checks if responses contain hierarchical fields (with __)
func hasHierarchicalFields(responses []intermediate.Response) bool {
	for _, r := range responses {
		if strings.Contains(r.Name, "__") {
			return true
		}
	}

	return false
}

// generateHierarchicalManyExecution generates code for hierarchical data aggregation with async generator
func generateHierarchicalManyExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect string,
) (*queryExecutionData, error) {
	// Detect hierarchical structure
	nodes, rootFields, err := detectHierarchicalStructure(format.Responses)
	if err != nil {
		return nil, fmt.Errorf("failed to detect hierarchical structure: %w", err)
	}

	// Find parent key fields (fields ending with _id or named id)
	parentKeyFields := findParentKeyFields(rootFields)
	if len(parentKeyFields) == 0 {
		return nil, errors.New("no parent key field found for hierarchical aggregation")
	}

	var code []string

	code = append(code, "# Execute query for hierarchical aggregation")
	code = append(code, "# This aggregates child records into parent objects")
	code = append(code, "")

	// Execute query based on dialect
	switch dialect {
	case "postgres":
		code = append(code, "rows = await conn.fetch(sql, *args)")
		code = append(code, "")
		code = append(code, "# Process rows for hierarchical aggregation")
		code = append(code, "current_parent = None")
		code = append(code, "current_parent_key = None")
		code = append(code, "")
		code = append(code, "for row in rows:")
		code = append(code, "    row_dict = dict(row)")

	case "mysql", "sqlite":
		code = append(code, "await cursor.execute(sql, args)")
		code = append(code, "")
		code = append(code, "# Process rows for hierarchical aggregation")
		code = append(code, "current_parent = None")
		code = append(code, "current_parent_key = None")
		code = append(code, "")
		code = append(code, "async for row in cursor:")
		code = append(code, "    row_dict = row if isinstance(row, dict) else dict(row)")

	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	// Generate parent key extraction
	code = append(code, "    ")
	code = append(code, "    # Extract parent key")

	if len(parentKeyFields) == 1 {
		keyField := parentKeyFields[0]
		code = append(code, fmt.Sprintf("    parent_key = row_dict.get('%s')", keyField.JSONTag))
	} else {
		// Composite key
		keyParts := make([]string, len(parentKeyFields))
		for i, kf := range parentKeyFields {
			keyParts[i] = fmt.Sprintf("row_dict.get('%s')", kf.JSONTag)
		}

		code = append(code, fmt.Sprintf("    parent_key = (%s)", strings.Join(keyParts, ", ")))
	}

	code = append(code, "    ")
	code = append(code, "    # Check if we have a new parent")
	code = append(code, "    if parent_key != current_parent_key:")
	code = append(code, "        # Yield previous parent if exists")
	code = append(code, "        if current_parent is not None:")
	code = append(code, "            yield current_parent")
	code = append(code, "        ")
	code = append(code, "        # Create new parent object")

	// Generate parent object creation with root fields
	parentFields := make([]string, 0)
	for _, rf := range rootFields {
		parentFields = append(parentFields, fmt.Sprintf("            %s=row_dict.get('%s')", rf.Name, rf.JSONTag))
	}

	// Add child list fields initialized to empty lists
	for key := range nodes {
		n := nodes[key]
		if len(n.PathSegments) == 1 {
			// Top-level child
			childFieldName := toSnakeCase(n.PathSegments[0])
			parentFields = append(parentFields, fmt.Sprintf("            %s=[]", childFieldName))
		}
	}

	code = append(code, fmt.Sprintf("        current_parent = %s(", responseStruct.ClassName))
	code = append(code, strings.Join(parentFields, ",\n"))
	code = append(code, "        )")
	code = append(code, "        current_parent_key = parent_key")
	code = append(code, "    ")

	// Generate child object creation and appending
	code = append(code, "    # Add child objects if present")

	// Process each top-level child group
	for key := range nodes {
		n := nodes[key]
		if len(n.PathSegments) != 1 {
			continue // Only process top-level children here
		}

		childFieldName := toSnakeCase(n.PathSegments[0])
		childClassName := generateChildClassName(responseStruct.ClassName, n.PathSegments)

		// Check if child data exists (at least one non-null field)
		childCheckFields := make([]string, 0)
		for _, f := range n.Fields {
			childCheckFields = append(childCheckFields, fmt.Sprintf("row_dict.get('%s__%s')", n.PathSegments[0], f.JSONTag))
		}

		code = append(code, fmt.Sprintf("    if any([%s]):", strings.Join(childCheckFields, ", ")))

		// Create child object
		childFields := make([]string, 0)
		for _, f := range n.Fields {
			childFields = append(childFields, fmt.Sprintf("            %s=row_dict.get('%s__%s')", f.Name, n.PathSegments[0], f.JSONTag))
		}

		// Add nested children if any
		for childKey := range n.Children {
			nestedFieldName := toSnakeCase(childKey)
			childFields = append(childFields, fmt.Sprintf("            %s=[]", nestedFieldName))
		}

		code = append(code, fmt.Sprintf("        child_obj = %s(", childClassName))
		code = append(code, strings.Join(childFields, ",\n"))
		code = append(code, "        )")
		code = append(code, fmt.Sprintf("        current_parent.%s.append(child_obj)", childFieldName))
	}

	code = append(code, "")
	code = append(code, "# Yield last parent if exists")
	code = append(code, "if current_parent is not None:")
	code = append(code, "    yield current_parent")

	return &queryExecutionData{
		Code: strings.Join(code, "\n"),
	}, nil
}

// findParentKeyFields finds fields that can serve as parent keys (id or *_id fields)
func findParentKeyFields(fields []hierarchicalField) []hierarchicalField {
	var keyFields []hierarchicalField

	// First, look for fields with HierarchyKeyLevel == 1
	for _, f := range fields {
		// Note: hierarchicalField doesn't have HierarchyKeyLevel, so we use heuristic
		lower := strings.ToLower(f.Name)
		if lower == "id" || strings.HasSuffix(lower, "_id") {
			keyFields = append(keyFields, f)
			break // Use first matching field as primary key
		}
	}

	return keyFields
}

// generateChildClassName generates the class name for a child structure
func generateChildClassName(parentClassName string, pathSegments []string) string {
	var sb strings.Builder
	sb.WriteString(parentClassName)

	for _, seg := range pathSegments {
		sb.WriteString(toPascalCase(seg))
	}

	return sb.String()
}
