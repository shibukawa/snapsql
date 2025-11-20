package pygen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

// generateQueryExecution generates query execution and result mapping code for Python
func generateQueryExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect snapsql.Dialect,
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
		panic("unsupported response affinity: " + format.ResponseAffinity)
	}
}

// generateNoneAffinityExecution generates code for queries that don't return results (INSERT/UPDATE/DELETE)
func generateNoneAffinityExecution(format *intermediate.IntermediateFormat, dialect snapsql.Dialect) (*queryExecutionData, error) {
	var code strings.Builder

	code.WriteString("# Execute query (no result expected)\n")

	switch dialect {
	case snapsql.DialectPostgres:
		code.WriteString(`result = await conn.execute(sql, *args)
# asyncpg returns status string like 'INSERT 0 1' or 'UPDATE 3'
# Extract the number of affected rows
if result.startswith('INSERT'):
    parts = result.split()
    affected_rows = int(parts[-1]) if len(parts) > 2 else 0
elif result.startswith('UPDATE') or result.startswith('DELETE'):
    parts = result.split()
    affected_rows = int(parts[-1]) if len(parts) > 1 else 0
else:
    affected_rows = 0
return affected_rows
`)

	case snapsql.DialectMySQL, snapsql.DialectSQLite:
		code.WriteString(`if isinstance(cursor, aiosqlite.Connection):
    cursor = await cursor.execute(sql, args)
else:
    await cursor.execute(sql, args)
return cursor.rowcount
`)

	default:
		panic(fmt.Sprintf("unsupported dialect: %s", dialect))
	}

	return &queryExecutionData{
		Code:              strings.TrimSuffix(code.String(), "\n"),
		UsesNotFoundError: true,
	}, nil
}

// generateOneAffinityExecution generates code for queries that return a single row
func generateOneAffinityExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect snapsql.Dialect,
) (*queryExecutionData, error) {
	if responseStruct == nil {
		panic(fmt.Sprintf("response struct missing for affinity %q", format.ResponseAffinity))
	}

	// Check if this is hierarchical structure
	isHierarchical := hasHierarchicalFields(format.Responses)

	if isHierarchical {
		return generateHierarchicalOneExecution(format, responseStruct, dialect)
	}

	var code strings.Builder

	code.WriteString("# Execute query and fetch single row\n")

	switch dialect {
	case snapsql.DialectPostgres:
		code.WriteString(fmt.Sprintf(`row = await conn.fetchrow(sql, *args)

if row is None:
    # Build parameter dict for error message
    param_dict = {}
    # Note: args is a list, parameter names would need to be tracked separately
    raise NotFoundError(
        message="Record not found",
        func_name="%s",
        query=sql,
        params=param_dict
    )

# Map row to dataclass
# asyncpg returns Record which supports dict() conversion
return %s(**dict(row))
`, format.FunctionName, responseStruct.ClassName))

	case snapsql.DialectMySQL, snapsql.DialectSQLite:
		comment := "# aiomysql with DictCursor returns dict"
		if dialect == snapsql.DialectSQLite {
			comment = "# aiosqlite with row_factory returns dict-like object"
		}

		if dialect == snapsql.DialectSQLite {
			code.WriteString(fmt.Sprintf(`if isinstance(cursor, aiosqlite.Connection):
    cursor = await cursor.execute(sql, args)
else:
    await cursor.execute(sql, args)

row = await cursor.fetchone()

if row is None:
    # Build parameter dict for error message
    param_dict = {}
    # Note: args is a list, parameter names would need to be tracked separately
    raise NotFoundError(
        message="Record not found",
        func_name="%s",
        query=sql,
        params=param_dict
    )

# Map row to dataclass
%s
return %s(**row)
`, format.FunctionName, comment, responseStruct.ClassName))
		} else {
			code.WriteString(fmt.Sprintf(`await cursor.execute(sql, args)
row = await cursor.fetchone()

if row is None:
    # Build parameter dict for error message
    param_dict = {}
    # Note: args is a list, parameter names would need to be tracked separately
    raise NotFoundError(
        message="Record not found",
        func_name="%s",
        query=sql,
        params=param_dict
    )

# Map row to dataclass
%s
return %s(**row)
`, format.FunctionName, comment, responseStruct.ClassName))
		}

	default:
		panic("unsupported dialect: " + dialect)
	}

	return &queryExecutionData{
		Code:              strings.TrimSuffix(code.String(), "\n"),
		UsesNotFoundError: true,
	}, nil
}

// generateHierarchicalOneExecution generates code for hierarchical data aggregation with one affinity
func generateHierarchicalOneExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect snapsql.Dialect,
) (*queryExecutionData, error) {
	// Detect hierarchical structure
	nodes, rootFields, err := detectHierarchicalStructure(format.Responses)
	if err != nil {
		return nil, fmt.Errorf("failed to detect hierarchical structure: %w", err)
	}

	// Find parent key fields
	parentKeyFields := findParentKeyFields(rootFields)
	if len(parentKeyFields) == 0 {
		return nil, snapsql.ErrHierarchicalNoParentPrimaryKey
	}

	var code strings.Builder

	code.WriteString("# Execute query for hierarchical aggregation (one affinity)\n")
	code.WriteString("# This aggregates multiple rows into a single parent object with child lists\n\n")

	// Execute query based on dialect
	switch dialect {
	case snapsql.DialectPostgres:
		code.WriteString(fmt.Sprintf(`rows = await conn.fetch(sql, *args)

if not rows:
    raise NotFoundError(
        message="Record not found",
        func_name="%s",
        query=sql
    )

# Process rows for hierarchical aggregation
result = None

for row in rows:
    row_dict = dict(row)
`, format.FunctionName))

	case snapsql.DialectMySQL, snapsql.DialectSQLite:
		if dialect == snapsql.DialectSQLite {
			code.WriteString(fmt.Sprintf(`if isinstance(cursor, aiosqlite.Connection):
    cursor = await cursor.execute(sql, args)
else:
    await cursor.execute(sql, args)

# Collect all rows first
rows = []
async for row in cursor:
    rows.append(row if isinstance(row, dict) else dict(row))

if not rows:
    raise NotFoundError(
        message="Record not found",
        func_name="%s",
        query=sql
    )

# Process rows for hierarchical aggregation
result = None

for row_dict in rows:
`, format.FunctionName))
		} else {
			code.WriteString(fmt.Sprintf(`await cursor.execute(sql, args)

# Collect all rows first
rows = []
async for row in cursor:
    rows.append(row if isinstance(row, dict) else dict(row))

if not rows:
    raise NotFoundError(
        message="Record not found",
        func_name="%s",
        query=sql
    )

# Process rows for hierarchical aggregation
result = None

for row_dict in rows:
`, format.FunctionName))
		}

	default:
		panic("unsupported dialect: " + dialect)
	}

	// Generate aggregation logic
	code.WriteString("    \n")
	code.WriteString("    # Create parent object on first row\n")
	code.WriteString("    if result is None:\n")

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

	code.WriteString(fmt.Sprintf("        result = %s(\n", responseStruct.ClassName))
	code.WriteString(strings.Join(parentFields, ",\n"))
	code.WriteString("\n        )\n")
	code.WriteString("    \n")
	code.WriteString("    # Add child objects if present\n")
	buildChildObjectBlocks(&code, nodes, responseStruct, "result")
	code.WriteByte('\n')
	code.WriteString("return result\n")

	return &queryExecutionData{
		Code:              strings.TrimSuffix(code.String(), "\n"),
		UsesNotFoundError: false,
	}, nil
}

// generateManyAffinityExecution generates code for queries that return multiple rows
func generateManyAffinityExecution(
	format *intermediate.IntermediateFormat,
	responseStruct *responseStructData,
	dialect snapsql.Dialect,
) (*queryExecutionData, error) {
	if responseStruct == nil {
		panic(fmt.Sprintf("response struct missing for affinity %q", format.ResponseAffinity))
	}

	// Check if this is hierarchical structure
	isHierarchical := hasHierarchicalFields(format.Responses)

	if isHierarchical {
		return generateHierarchicalManyExecution(format, responseStruct, dialect)
	}

	var code strings.Builder

	code.WriteString("# Execute query and yield rows as async generator\n")

	switch dialect {
	case snapsql.DialectPostgres:
		code.WriteString(fmt.Sprintf(`rows = await conn.fetch(sql, *args)

# Yield each row as dataclass instance
for row in rows:
    yield %s(**dict(row))
`, responseStruct.ClassName))

	case snapsql.DialectMySQL, snapsql.DialectSQLite:
		code.WriteString(fmt.Sprintf(`await cursor.execute(sql, args)

# Fetch and yield rows
async for row in cursor:
    yield %s(**row)
`, responseStruct.ClassName))

	default:
		panic(fmt.Sprintf("unsupported dialect: %s", dialect))
	}

	return &queryExecutionData{
		Code:              strings.TrimSuffix(code.String(), "\n"),
		UsesNotFoundError: false,
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
	dialect snapsql.Dialect,
) (*queryExecutionData, error) {
	// Detect hierarchical structure
	nodes, rootFields, err := detectHierarchicalStructure(format.Responses)
	if err != nil {
		return nil, fmt.Errorf("failed to detect hierarchical structure: %w", err)
	}

	// Find parent key fields (fields ending with _id or named id)
	parentKeyFields := findParentKeyFields(rootFields)
	if len(parentKeyFields) == 0 {
		return nil, snapsql.ErrHierarchicalNoParentPrimaryKey
	}

	var code strings.Builder

	code.WriteString("# Execute query for hierarchical aggregation\n")
	code.WriteString("# This aggregates child records into parent objects\n\n")

	// Execute query based on dialect
	switch dialect {
	case snapsql.DialectPostgres:
		code.WriteString(`rows = await conn.fetch(sql, *args)

# Process rows for hierarchical aggregation
current_parent = None
current_parent_key = None

for row in rows:
    row_dict = dict(row)
`)

	case snapsql.DialectMySQL, snapsql.DialectSQLite:
		if dialect == snapsql.DialectSQLite {
			code.WriteString(`if isinstance(cursor, aiosqlite.Connection):
    cursor = await cursor.execute(sql, args)
else:
    await cursor.execute(sql, args)

# Process rows for hierarchical aggregation
current_parent = None
current_parent_key = None

async for row in cursor:
    row_dict = row if isinstance(row, dict) else dict(row)
`)
		} else {
			code.WriteString(`await cursor.execute(sql, args)

# Process rows for hierarchical aggregation
current_parent = None
current_parent_key = None

async for row in cursor:
    row_dict = row if isinstance(row, dict) else dict(row)
`)
		}

	default:
		panic(fmt.Sprintf("unsupported dialect: %s", dialect))
	}

	// Generate parent key extraction
	code.WriteString("    \n")
	code.WriteString("    # Extract parent key\n")

	if len(parentKeyFields) == 1 {
		keyField := parentKeyFields[0]
		code.WriteString(fmt.Sprintf("    parent_key = row_dict.get('%s')\n", keyField.JSONTag))
	} else {
		// Composite key
		keyParts := make([]string, len(parentKeyFields))
		for i, kf := range parentKeyFields {
			keyParts[i] = fmt.Sprintf("row_dict.get('%s')", kf.JSONTag)
		}

		code.WriteString(fmt.Sprintf("    parent_key = (%s)\n", strings.Join(keyParts, ", ")))
	}

	code.WriteString("    \n")
	code.WriteString("    # Check if we have a new parent\n")
	code.WriteString("    if parent_key != current_parent_key:\n")
	code.WriteString("        # Yield previous parent if exists\n")
	code.WriteString("        if current_parent is not None:\n")
	code.WriteString("            yield current_parent\n")
	code.WriteString("        \n")
	code.WriteString("        # Create new parent object\n")

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

	code.WriteString(fmt.Sprintf("        current_parent = %s(\n", responseStruct.ClassName))
	code.WriteString(strings.Join(parentFields, ",\n"))
	code.WriteString("\n        )\n")
	code.WriteString("        current_parent_key = parent_key\n")
	code.WriteString("    \n")
	code.WriteString("    # Add child objects if present\n")
	buildChildObjectBlocks(&code, nodes, responseStruct, "current_parent")
	code.WriteByte('\n')
	code.WriteString("# Yield last parent if exists\n")
	code.WriteString("if current_parent is not None:\n")
	code.WriteString("    yield current_parent\n")

	return &queryExecutionData{
		Code:              strings.TrimSuffix(code.String(), "\n"),
		UsesNotFoundError: true,
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

func buildChildObjectBlocks(sb *strings.Builder, nodes map[string]*node, responseStruct *responseStructData, parentVar string) {
	keys := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if len(n.PathSegments) == 1 {
			keys = append(keys, pathKey(n.PathSegments))
		}
	}

	sort.Strings(keys)

	for _, key := range keys {
		n := nodes[key]
		childFieldName := toSnakeCase(n.PathSegments[0])
		childClassName := generateChildClassName(responseStruct.ClassName, n.PathSegments)

		childCheckFields := make([]string, 0, len(n.Fields))
		for _, f := range n.Fields {
			childCheckFields = append(childCheckFields, fmt.Sprintf("row_dict.get('%s__%s')", n.PathSegments[0], f.JSONTag))
		}

		fmt.Fprintf(sb, "    if any([%s]):\n", strings.Join(childCheckFields, ", "))

		childFields := make([]string, 0, len(n.Fields)+len(n.Children))
		for _, f := range n.Fields {
			childFields = append(childFields, fmt.Sprintf("            %s=row_dict.get('%s__%s')", f.Name, n.PathSegments[0], f.JSONTag))
		}

		childKeys := make([]string, 0, len(n.Children))
		for childKey := range n.Children {
			childKeys = append(childKeys, childKey)
		}

		sort.Strings(childKeys)

		for _, childKey := range childKeys {
			nestedFieldName := toSnakeCase(childKey)
			childFields = append(childFields, fmt.Sprintf("            %s=[]", nestedFieldName))
		}

		fmt.Fprintf(sb, "        child_obj = %s(\n", childClassName)
		sb.WriteString(strings.Join(childFields, ",\n"))
		sb.WriteString("\n        )\n")
		fmt.Fprintf(sb, "        %s.%s.append(child_obj)\n", parentVar, childFieldName)
	}
}
