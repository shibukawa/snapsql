package gogen

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

var (
	ErrIteratorRequiresStruct = errors.New("iterator generation requires a response struct")
)

// queryExecutionData represents query execution code generation data
type queryExecutionData struct {
	Code []string
	// NeedsSnapsqlImport indicates whether generated code references snapsql errors
	NeedsSnapsqlImport bool
	// Iterator generation for many affinity
	IsIterator        bool
	IteratorBody      []string
	IteratorYieldType string
	ReturnsSQLResult  bool
	ResponseAffinity  string
}

// generateQueryExecution generates query execution and result mapping code
func generateQueryExecution(format *intermediate.IntermediateFormat, responseStruct *responseStructData, metas []*hierarchicalNodeMeta, responseType, functionName, errorZeroValue string, withLogger bool) (*queryExecutionData, error) {
	var code []string

	needsSnapsql := false
	returnsSQLResult := strings.EqualFold(responseType, "sql.Result")
	errorPrefix := functionName + ": "

	switch format.ResponseAffinity {
	case "none", "":
		// Legacy path: no result mapping
		code = append(code, "// Execute query (no result expected)")
		if returnsSQLResult {
			code = append(code, "execResult, err := stmt.ExecContext(ctx, args...)")
		} else {
			code = append(code, "_, err = stmt.ExecContext(ctx, args...)")
		}

		code = append(code, "if err != nil {")
		code = append(code, fmt.Sprintf("    return %s, fmt.Errorf(\"%sfailed to execute statement: %%w\", err)", errorZeroValue, errorPrefix))

		code = append(code, "}")
		if returnsSQLResult {
			code = append(code, "result = execResult")
		}
	case "one":
		// Decide whether this is a simple row scan or hierarchical aggregation that requires rows loop
		needsAggregation := false

		if responseStruct != nil {
			// Check raw responses (original column list) for hierarchical fields
			if len(metas) > 0 {
				needsAggregation = true
			} else {
				for _, r := range responseStruct.RawResponses {
					if strings.Contains(r.Name, "__") {
						needsAggregation = true
						break
					}
				}
			}
		}

		if !needsAggregation {
			if responseStruct != nil {
				code = append(code, "// Execute query and scan single row")
				code = append(code, "row := stmt.QueryRowContext(ctx, args...)")
			} else {
				// Safety fallback: execute without scan when no response struct is available
				code = append(code, "// Execute statement (no response struct available)")
				if returnsSQLResult {
					code = append(code, "execResult, err := stmt.ExecContext(ctx, args...)")
				} else {
					code = append(code, "_, err = stmt.ExecContext(ctx, args...)")
				}

				code = append(code, "if err != nil {")
				code = append(code, fmt.Sprintf("    return %s, fmt.Errorf(\"%sfailed to execute statement: %%w\", err)", errorZeroValue, errorPrefix))

				code = append(code, "}")
				if returnsSQLResult {
					code = append(code, "result = execResult")
				}
			}
		} else {
			code = append(code, "// Execute query for hierarchical aggregation (one affinity)")
			needsSnapsql = true // aggregation(one) uses snapsql error constants
		}

		if responseStruct != nil {
			scanCode, err := generateScanCode(responseStruct, false, metas)
			if err != nil {
				return nil, fmt.Errorf("failed to generate scan code: %w", err)
			}

			code = append(code, scanCode...)
		}
	case "many":
		needsAggregation := false

		if responseStruct != nil {
			if len(metas) > 0 {
				needsAggregation = true
			} else {
				for _, r := range responseStruct.RawResponses {
					if strings.Contains(r.Name, "__") {
						needsAggregation = true
						break
					}
				}
			}
		}

		if responseStruct != nil && !needsAggregation {
			iteratorBody, err := generateIteratorBody(responseStruct, functionName)
			if err != nil {
				return nil, fmt.Errorf("failed to generate iterator body: %w", err)
			}

			return &queryExecutionData{
				NeedsSnapsqlImport: needsSnapsql,
				IsIterator:         true,
				IteratorBody:       iteratorBody,
				IteratorYieldType:  "*" + responseStruct.Name,
				ReturnsSQLResult:   returnsSQLResult,
				ResponseAffinity:   format.ResponseAffinity,
			}, nil
		}

		code = append(code, "// Execute query and scan multiple rows (many affinity)")
		code = append(code, "rows, err := stmt.QueryContext(ctx, args...)")
		code = append(code, "if err != nil {")
		code = append(code, fmt.Sprintf("    return %s, fmt.Errorf(\"%sfailed to execute query: %%w\", err)", errorZeroValue, errorPrefix))
		code = append(code, "}")
		code = append(code, "defer rows.Close()")
		code = append(code, "")

		if responseStruct != nil {
			scanCode, err := generateScanCode(responseStruct, true, metas)
			if err != nil {
				return nil, fmt.Errorf("failed to generate scan code: %w", err)
			}

			code = append(code, scanCode...)
		} else {
			code = append(code, "// Generic scan for any result - not implemented")
			code = append(code, "// This would require runtime reflection or predefined column mapping")
		}
	default:
		return nil, fmt.Errorf("%w: %s", snapsql.ErrUnsupportedResponseAffinity, format.ResponseAffinity)
	}

	// Post-process to wrap error returns when logging is enabled
	if withLogger {
		processed := make([]string, 0, len(code))
		for _, line := range code {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "return ") {
				if idx := strings.Index(line, ", fmt.Errorf"); idx != -1 {
					indent := line[:strings.Index(line, "return")]
					left := strings.TrimSpace(line[strings.Index(line, "return")+len("return ") : idx])
					fmtExpr := strings.TrimSpace(line[idx+2:])
					processed = append(processed,
						indent+"err = "+fmtExpr,
						indent+"logger.SetErr(err)",
						indent+"return "+left+", err",
					)

					continue
				}

				if idx := strings.Index(line, ", snapsql."); idx != -1 {
					indent := line[:strings.Index(line, "return")]
					left := strings.TrimSpace(line[strings.Index(line, "return")+len("return ") : idx])
					errExpr := strings.TrimSpace(line[idx+2:])
					processed = append(processed,
						indent+"err = "+errExpr,
						indent+"logger.SetErr(err)",
						indent+"return "+left+", err",
					)

					continue
				}

				if strings.HasSuffix(trimmed, ", err") {
					indent := line[:strings.Index(line, "return")]
					processed = append(processed,
						indent+"logger.SetErr(err)",
						indent+trimmed,
					)

					continue
				}
			}

			processed = append(processed, line)
		}

		code = processed
	}

	return &queryExecutionData{
		Code:               code,
		NeedsSnapsqlImport: needsSnapsql,
		ReturnsSQLResult:   returnsSQLResult,
		ResponseAffinity:   format.ResponseAffinity,
	}, nil
}

// generateScanCode generates code for scanning database results
func generateScanCode(responseStruct *responseStructData, isMany bool, metas []*hierarchicalNodeMeta) ([]string, error) {
	// Check if we need aggregation (has __ fields in JSON tags)
	hasAggregation := false
	if len(metas) > 0 {
		hasAggregation = true
	} else if responseStruct != nil {
		for _, r := range responseStruct.RawResponses {
			if strings.Contains(r.Name, "__") {
				hasAggregation = true
				break
			}
		}
	}

	if hasAggregation {
		// Prefer meta-driven aggregation if metas supplied
		if len(metas) > 0 {
			return generateMetaDrivenAggregatedScanCode(responseStruct, isMany, metas)
		}

		return generateAggregatedScanCode(responseStruct, isMany)
	}

	return generateSimpleScanCode(responseStruct, isMany)
}

// generateHierarchicalManyScan builds code lines that aggregate rows with __ hierarchical fields.
// Heuristic:
// - Parent key: first field whose JSON tag does not contain '__' and ends with 'id' (case-insensitive) or exact 'id'.
// - Child groups: prefix before '__'.
// - For each group we build a map[parentKey]parentStruct and append child struct instances.
// NOTE: hierarchical many aggregation for __ fields is deferred; future implementation

// generateSimpleScanCode generates simple scanning code without aggregation
func generateSimpleScanCode(responseStruct *responseStructData, isMany bool) ([]string, error) {
	var code []string

	if isMany {
		// Multiple rows
		code = append(code, "for rows.Next() {")
		code = append(code, "    var item "+responseStruct.Name)
		code = append(code, "    err := rows.Scan(")

		// Generate scan targets (always include trailing comma in multiline)
		for _, field := range responseStruct.Fields {
			code = append(code, fmt.Sprintf("        &item.%s,", field.Name))
		}

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
		for _, field := range responseStruct.Fields {
			code = append(code, fmt.Sprintf("    &result.%s,", field.Name))
		}

		code = append(code, ")")
		code = append(code, "if err != nil {")
		code = append(code, "    return result, fmt.Errorf(\"failed to scan row: %w\", err)")
		code = append(code, "}")
	}

	return code, nil
}

// generateIteratorBody builds the body of an iterator for non-aggregated many responses.
func generateIteratorBody(responseStruct *responseStructData, functionName string) ([]string, error) {
	if responseStruct == nil {
		return nil, ErrIteratorRequiresStruct
	}

	var code []string

	prefix := functionName + ": "

	code = append(code, "stmt, err := executor.PrepareContext(ctx, query)")
	code = append(code, "if err != nil {")
	code = append(code, fmt.Sprintf("\terr = fmt.Errorf(\"%sfailed to prepare statement: %%w (query: %%s)\", err, query)", prefix))
	code = append(code, "\tlogger.SetErr(err)")
	code = append(code, "\t_ = yield(nil, err)")
	code = append(code, "\treturn")
	code = append(code, "}")
	code = append(code, "defer stmt.Close()")
	code = append(code, "")
	code = append(code, "rows, err := stmt.QueryContext(ctx, args...)")
	code = append(code, "if err != nil {")
	code = append(code, fmt.Sprintf("\terr = fmt.Errorf(\"%sfailed to execute query: %%w\", err)", prefix))
	code = append(code, "\tlogger.SetErr(err)")
	code = append(code, "\t_ = yield(nil, err)")
	code = append(code, "\treturn")
	code = append(code, "}")
	code = append(code, "defer rows.Close()")
	code = append(code, "")
	code = append(code, "for rows.Next() {")
	code = append(code, fmt.Sprintf("\titem := new(%s)", responseStruct.Name))

	code = append(code, "\tif err := rows.Scan(")
	for _, field := range responseStruct.Fields {
		code = append(code, fmt.Sprintf("\t\t&item.%s,", field.Name))
	}

	code = append(code, "\t); err != nil {")
	code = append(code, fmt.Sprintf("\t\terr = fmt.Errorf(\"%sfailed to scan row: %%w\", err)", prefix))
	code = append(code, "\t\tlogger.SetErr(err)")
	code = append(code, "\t\t_ = yield(nil, err)")
	code = append(code, "\t\treturn")
	code = append(code, "\t}")
	code = append(code, "\tif !yield(item, nil) {")
	code = append(code, "\t\treturn")
	code = append(code, "\t}")
	code = append(code, "}")
	code = append(code, "")
	code = append(code, "if err := rows.Err(); err != nil {")
	code = append(code, fmt.Sprintf("\terr = fmt.Errorf(\"%serror iterating rows: %%w\", err)", prefix))
	code = append(code, "\tlogger.SetErr(err)")
	code = append(code, "\t_ = yield(nil, err)")
	code = append(code, "\treturn")
	code = append(code, "}")

	return code, nil
}

// generateAggregatedScanCode generates scanning code with __ field aggregation
func generateAggregatedScanCode(responseStruct *responseStructData, isMany bool) ([]string, error) {
	// Multi-level hierarchical aggregation.
	if responseStruct == nil || len(responseStruct.RawResponses) == 0 {
		return nil, snapsql.ErrHierarchicalNoRawResponses
	}

	type colMeta struct {
		Resp    intermediate.Response
		VarName string // scanned variable name
		ColName string // last segment for field assignment
	}

	// Build tree of nodes: path segments (groups) -> columns
	type node struct {
		Path     []string
		Columns  []colMeta // columns belonging to this node (excluding root fields)
		PKCols   []colMeta // subset of Columns that are PK columns of this node
		Children map[string]*node
	}

	newNode := func(path []string) *node {
		return &node{Path: append([]string{}, path...), Children: map[string]*node{}}
	}

	nodes := map[string]*node{}
	getNode := func(path []string) *node {
		key := strings.Join(path, "__")
		if n, ok := nodes[key]; ok {
			return n
		}

		n := newNode(path)
		nodes[key] = n

		return n
	}

	var (
		rootCols []colMeta
		allCols  []colMeta
	)
	// create scanned variable names first pass

	for _, r := range responseStruct.RawResponses {
		segs := strings.Split(r.Name, "__")

		varName := "col_" + strings.ToLower(celNameToGoName(r.Name))
		if len(segs) == 1 { // root field
			rootCols = append(rootCols, colMeta{Resp: r, VarName: varName, ColName: segs[0]})
			allCols = append(allCols, colMeta{Resp: r, VarName: varName, ColName: segs[0]})

			continue
		}
		// hierarchical field
		fieldName := segs[len(segs)-1]
		path := segs[:len(segs)-1]
		n := getNode(path)
		cm := colMeta{Resp: r, VarName: varName, ColName: fieldName}
		n.Columns = append(n.Columns, cm)
		allCols = append(allCols, cm)
	}

	if len(nodes) == 0 {
		return nil, snapsql.ErrHierarchicalNoGroups
	}

	// Determine PK columns per node
	isIDName := func(name string) bool {
		lower := strings.ToLower(name)
		return lower == "id" || strings.HasSuffix(lower, "_id")
	}

	for _, n := range nodes {
		// Prefer explicit hierarchy level match
		for _, c := range n.Columns {
			if c.Resp.HierarchyKeyLevel == len(n.Path)+1 {
				n.PKCols = append(n.PKCols, c)
			}
		}

		if len(n.PKCols) == 0 {
			// Fallback heuristic: pick columns named id or *_id
			for _, c := range n.Columns {
				if isIDName(c.ColName) {
					n.PKCols = append(n.PKCols, c)
				}
			}
		}
	}

	// Root parent PKs
	var parentPK []colMeta

	for _, c := range rootCols {
		if c.Resp.HierarchyKeyLevel == 1 {
			parentPK = append(parentPK, c)
		}
	}

	if len(parentPK) == 0 {
		// Fallback heuristic: first root field named id or *_id
		for _, c := range rootCols {
			if isIDName(c.ColName) {
				parentPK = append(parentPK, c)
				break
			}
		}
	}

	if len(parentPK) == 0 {
		return nil, snapsql.ErrHierarchicalNoParentPrimaryKey
	}

	// Build parent-child relationships between nodes (for traversal ordering)
	for _, n := range nodes {
		if len(n.Path) <= 1 {
			continue
		}

		parent := getNode(n.Path[:len(n.Path)-1])
		if parent.Children == nil {
			parent.Children = map[string]*node{}
		}

		parent.Children[n.Path[len(n.Path)-1]] = n
	}

	// Order nodes by depth ascending so parents handled before children in row processing
	orderedNodes := make([]*node, 0, len(nodes))
	for _, n := range nodes {
		orderedNodes = append(orderedNodes, n)
	}

	sort.Slice(orderedNodes, func(i, j int) bool {
		if len(orderedNodes[i].Path) == len(orderedNodes[j].Path) {
			return strings.Join(orderedNodes[i].Path, "__") < strings.Join(orderedNodes[j].Path, "__")
		}

		return len(orderedNodes[i].Path) < len(orderedNodes[j].Path)
	})

	// Code generation begins
	var code []string
	if isMany {
		code = append(code, "// Hierarchical many scan (multi-level)")
	} else {
		code = append(code, "// Hierarchical one scan (multi-level)")
	}

	code = append(code, "var _parentMap map[string]*"+responseStruct.Name)
	// Node instance maps (for fast lookup) keyed by chain keys
	for _, n := range orderedNodes {
		structName := responseStruct.Name
		for _, seg := range n.Path {
			structName += celNameToGoName(seg)
		}

		mapVar := "_nodeMap" + structName
		code = append(code, fmt.Sprintf("var %s map[string]*%s", mapVar, structName))
	}

	if !isMany {
		code = append(code, "// Re-executing as rows for aggregation")
		code = append(code, "rows, err := stmt.QueryContext(ctx, args...)")
		code = append(code, "if err != nil { return result, fmt.Errorf(\"failed to query rows: %w\", err) }")
		code = append(code, "defer rows.Close()")
	}

	code = append(code, "for rows.Next() {")
	// Declarations
	for _, c := range allCols {
		goType, _ := convertToGoType(c.Resp.Type)
		// 階層列（__含む）はLEFT JOINでNULLになりうるため、常にポインタ型で受ける
		if strings.Contains(c.Resp.Name, "__") && !strings.HasPrefix(goType, "*") {
			goType = "*" + goType
		}

		if c.Resp.IsNullable && !strings.HasPrefix(goType, "*") {
			goType = "*" + goType
		}

		code = append(code, fmt.Sprintf("    var %s %s", c.VarName, goType))
	}

	code = append(code, "    err := rows.Scan(")
	for _, c := range allCols {
		code = append(code, fmt.Sprintf("        &%s,", c.VarName))
	}

	code = append(code, "    )")
	code = append(code, "    if err != nil { return result, fmt.Errorf(\"failed to scan row: %w\", err) }")
	// parent key build (dereference pointer PKs for stable keys)
	//nolint:dupl // Different variable naming conventions for different contexts
	if len(parentPK) == 1 {
		if parentPK[0].Resp.IsNullable {
			code = append(code, "    var parentKey string")
			code = append(code, fmt.Sprintf("    if %s != nil { parentKey = fmt.Sprintf(\"%%v\", *%s) } else { parentKey = \"<nil>\" }", parentPK[0].VarName, parentPK[0].VarName))
		} else {
			code = append(code, fmt.Sprintf("    parentKey := fmt.Sprintf(\"%%v\", %s)", parentPK[0].VarName))
		}
	} else {
		// multiple key parts
		partVars := make([]string, len(parentPK))
		for i, pk := range parentPK {
			v := fmt.Sprintf("_pkp_%d", i)
			partVars[i] = v

			code = append(code, fmt.Sprintf("    var %s string", v))
			if pk.Resp.IsNullable {
				code = append(code, fmt.Sprintf("    if %s != nil { %s = fmt.Sprintf(\"%%v\", *%s) } else { %s = \"<nil>\" }", pk.VarName, v, pk.VarName, v))
			} else {
				code = append(code, fmt.Sprintf("    %s = fmt.Sprintf(\"%%v\", %s)", v, pk.VarName))
			}
		}

		code = append(code, fmt.Sprintf("    parentKey := strings.Join([]string{%s}, \"|\")", strings.Join(partVars, ", ")))
	}

	code = append(code, fmt.Sprintf("    if _parentMap == nil { _parentMap = make(map[string]*%s) }", responseStruct.Name))
	code = append(code, "    parentObj, _parentExists := _parentMap[parentKey]")
	code = append(code, "    if !_parentExists {")

	code = append(code, fmt.Sprintf("        parentObj = &%s{}", responseStruct.Name))
	for _, c := range rootCols { // assign root fields once (Name は既に PascalCase)
		code = append(code, fmt.Sprintf("        parentObj.%s = %s", celNameToGoName(c.Resp.Name), c.VarName))
	}
	// Initialize top-level slices (they are pointers slice fields)
	// Not strictly required; appends will allocate.
	code = append(code, "        _parentMap[parentKey] = parentObj")
	code = append(code, "    }")
	code = append(code, "    _chain_parent := parentKey")

	// Process nodes by depth
	for _, n := range orderedNodes {
		depthPathKey := strings.Join(n.Path, "__")

		structName := responseStruct.Name
		for _, seg := range n.Path {
			structName += celNameToGoName(seg)
		}

		mapVar := "_nodeMap_" + strings.Join(n.Path, "_")
		keyVar := "_k_" + strings.Join(n.Path, "_")
		chainVar := "_chain_" + strings.Join(n.Path, "_")
		parentMapVar := ""

		if len(n.Path) > 1 {
			parentStruct := responseStruct.Name
			for _, seg := range n.Path[:len(n.Path)-1] {
				parentStruct += celNameToGoName(seg)
			}

			parentMapVar = "_nodeMap" + parentStruct
		}
		// Build node presence predicate based on PK nil checks
		if len(n.PKCols) == 0 {
			// If no PK, we still append if any column is non-nil (heuristic)
			condParts := []string{}
			for _, c := range n.Columns {
				condParts = append(condParts, c.VarName+" != nil")
			}

			if len(condParts) == 0 {
				condParts = append(condParts, "true")
			}

			code = append(code, fmt.Sprintf("    // Node %s (no PK)", depthPathKey))
			code = append(code, fmt.Sprintf("    if %s {", strings.Join(condParts, " || ")))
		} else {
			nilConds := []string{}
			for _, pk := range n.PKCols {
				nilConds = append(nilConds, pk.VarName+" == nil")
			}

			code = append(code, "    // Node "+depthPathKey)
			code = append(code, fmt.Sprintf("    if !(%s) {", strings.Join(nilConds, " || ")))
		}
		// Build node key (hierarchical scan vars are pointers; nil guarded above)
		if len(n.PKCols) == 1 {
			code = append(code, fmt.Sprintf("        %s := fmt.Sprintf(\"%%v\", *%s)", keyVar, n.PKCols[0].VarName))
		} else if len(n.PKCols) > 1 {
			fmtParts := make([]string, len(n.PKCols))

			args := make([]string, len(n.PKCols))
			for i, pk := range n.PKCols {
				fmtParts[i] = "%v"
				args[i] = "*" + pk.VarName
			}

			code = append(code, fmt.Sprintf("        %s := fmt.Sprintf(\"%s\", %s)", keyVar, strings.Join(fmtParts, "|"), strings.Join(args, ", ")))
		} else {
			code = append(code, fmt.Sprintf("        %s := fmt.Sprintf(\"auto_%s_%s\")", keyVar, depthPathKey, "no_pk"))
		}
		// Chain key relative to current parent
		code = append(code, fmt.Sprintf("        %s := _chain_parent + \"|%s:\" + %s", chainVar, depthPathKey, keyVar))
		// Map init
		code = append(code, fmt.Sprintf("        if %s == nil { %s = make(map[string]*%s) }", mapVar, mapVar, structName))
		// Dedup using map directly (unique chain key)
		code = append(code, fmt.Sprintf("        node_%s, _nodeExists := %s[%s]", strings.ReplaceAll(depthPathKey, "__", "_"), mapVar, chainVar))
		code = append(code, "        if !_nodeExists {")
		code = append(code, fmt.Sprintf("            node_%s = &%s{}", strings.ReplaceAll(depthPathKey, "__", "_"), structName))
		// Assign columns for this node
		for _, c := range n.Columns {
			goField := celNameToGoName(c.ColName)
			if c.Resp.IsNullable {
				code = append(code, fmt.Sprintf("            node_%s.%s = %s", strings.ReplaceAll(depthPathKey, "__", "_"), goField, c.VarName))
			} else if strings.Contains(c.Resp.Name, "__") {
				code = append(code, fmt.Sprintf("            if %s != nil {", c.VarName))
				code = append(code, fmt.Sprintf("                node_%s.%s = *%s", strings.ReplaceAll(depthPathKey, "__", "_"), goField, c.VarName))
				code = append(code, "            }")
			} else {
				code = append(code, fmt.Sprintf("            node_%s.%s = %s", strings.ReplaceAll(depthPathKey, "__", "_"), goField, c.VarName))
			}
		}
		// Append to parent slice
		if len(n.Path) == 1 {
			// top-level slice on parentObj
			sliceField := celNameToGoName(n.Path[0])
			code = append(code, fmt.Sprintf("            parentObj.%s = append(parentObj.%s, node_%s)", sliceField, sliceField, strings.ReplaceAll(depthPathKey, "__", "_")))
		} else {
			parentSliceField := celNameToGoName(n.Path[len(n.Path)-1])
			// Lookup parent instance by current _chain_parent in its node map
			code = append(code, fmt.Sprintf("            if p, ok := %s[_chain_parent]; ok {", parentMapVar))
			code = append(code, fmt.Sprintf("                p.%s = append(p.%s, node_%s)", parentSliceField, parentSliceField, strings.ReplaceAll(depthPathKey, "__", "_")))
			code = append(code, "            }")
		}

		code = append(code, fmt.Sprintf("            %s[%s] = node_%s", mapVar, chainVar, strings.ReplaceAll(depthPathKey, "__", "_")))
		code = append(code, "        }")
		// Update parent chain for deeper level
		code = append(code, "        _chain_parent = "+chainVar)
		code = append(code, "    }")
	}

	code = append(code, "}")

	code = append(code, "if err = rows.Err(); err != nil { return result, fmt.Errorf(\"error iterating rows: %w\", err) }")
	if isMany {
		code = append(code, "for _, v := range _parentMap { result = append(result, *v) }")
	} else {
		code = append(code, "if len(_parentMap) == 0 { return result, snapsql.ErrNotFound }")
		code = append(code, "if len(_parentMap) > 1 { return result, snapsql.ErrHierarchicalMultipleParentsForOne }")
		code = append(code, "for _, v := range _parentMap { result = *v }")
	}

	return code, nil
}

// generateMetaDrivenAggregatedScanCode builds hierarchical aggregation scan code using precomputed metas.
// This avoids re-parsing response names and duplicates the logic with a simpler deterministic expansion.
func generateMetaDrivenAggregatedScanCode(responseStruct *responseStructData, isMany bool, metas []*hierarchicalNodeMeta) ([]string, error) {
	if responseStruct == nil || len(responseStruct.RawResponses) == 0 {
		return nil, snapsql.ErrHierarchicalNoRawResponses
	}

	if len(metas) == 0 {
		return nil, snapsql.ErrHierarchicalNoGroups
	}

	respByName := make(map[string]intermediate.Response, len(responseStruct.RawResponses))
	for _, r := range responseStruct.RawResponses {
		respByName[r.Name] = r
	}
	// Determine root fields (no __ in JSON tag). RawResponses keeps original names.
	type col struct {
		resp    intermediate.Response
		varName string
		last    string
	}

	var (
		rootCols []col
		allCols  []col
	)

	for _, r := range responseStruct.RawResponses {
		varName := "col_" + strings.ToLower(celNameToGoName(r.Name))

		segs := strings.Split(r.Name, "__")
		if len(segs) == 1 { // root
			rootCols = append(rootCols, col{resp: r, varName: varName, last: segs[0]})
			allCols = append(allCols, col{resp: r, varName: varName, last: segs[0]})

			continue
		}
		// hierarchical leaf column
		last := segs[len(segs)-1]
		allCols = append(allCols, col{resp: r, varName: varName, last: last})
	}
	// Build map depth -> metas to control ordering (parents before children per row).
	sort.Slice(metas, func(i, j int) bool {
		if metas[i].Depth == metas[j].Depth {
			return strings.Join(metas[i].Path, "__") < strings.Join(metas[j].Path, "__")
		}

		return metas[i].Depth < metas[j].Depth
	})
	// Collect parent PK root (HierarchyKeyLevel==1). Fallback: id or *_id
	var parentPK []col

	for _, c := range rootCols {
		if c.resp.HierarchyKeyLevel == 1 {
			parentPK = append(parentPK, c)
		}
	}

	if len(parentPK) == 0 {
		isIDName := func(name string) bool {
			lower := strings.ToLower(name)
			return lower == "id" || strings.HasSuffix(lower, "_id")
		}
		for _, c := range rootCols {
			if isIDName(c.last) {
				parentPK = append(parentPK, c)
				break
			}
		}
	}

	if len(parentPK) == 0 {
		return nil, snapsql.ErrHierarchicalNoParentPrimaryKey
	}

	mainStruct := responseStruct.Name

	var code []string
	if isMany {
		code = append(code, "// Meta-driven hierarchical many scan")
	} else {
		code = append(code, "// Meta-driven hierarchical one scan")
	}

	code = append(code, "rows, err := stmt.QueryContext(ctx, args...)")
	code = append(code, "if err != nil { return result, fmt.Errorf(\"failed to query rows: %w\", err) }")
	code = append(code, "defer rows.Close()")
	// Declare column vars
	for _, c := range allCols {
		goType, _ := convertToGoType(c.resp.Type)
		if strings.Contains(c.resp.Name, "__") && !strings.HasPrefix(goType, "*") {
			goType = "*" + goType
		}

		if c.resp.IsNullable && !strings.HasPrefix(goType, "*") {
			goType = "*" + goType
		}

		code = append(code, fmt.Sprintf("var %s %s", c.varName, goType))
	}
	// Maps per node: path chain key -> struct pointers
	code = append(code, "var _parentMap map[string]*"+mainStruct)
	for _, m := range metas {
		// Compose struct name same as generateHierarchicalStructs
		structName := mainStruct
		for _, seg := range m.Path {
			structName += celNameToGoName(seg)
		}

		code = append(code, fmt.Sprintf("var _nodeMap_%s map[string]*%s", strings.Join(m.Path, "_"), structName))
	}

	code = append(code, "for rows.Next() {")
	// Scan line
	code = append(code, "    err = rows.Scan(")
	for _, c := range allCols {
		code = append(code, fmt.Sprintf("        &%s,", c.varName))
	}

	code = append(code, "    )")
	code = append(code, "    if err != nil { return result, fmt.Errorf(\"failed to scan row: %w\", err) }")
	// Build parent key (dereference pointer PKs for stable keys)
	//nolint:dupl // Different variable naming conventions for different contexts
	if len(parentPK) == 1 {
		if parentPK[0].resp.IsNullable {
			code = append(code, "    var pk_parent string")
			code = append(code, fmt.Sprintf("    if %s != nil { pk_parent = fmt.Sprintf(\"%%v\", *%s) } else { pk_parent = \"<nil>\" }", parentPK[0].varName, parentPK[0].varName))
		} else {
			code = append(code, fmt.Sprintf("    pk_parent := fmt.Sprintf(\"%%v\", %s)", parentPK[0].varName))
		}
	} else {
		partVars := make([]string, len(parentPK))
		for i, pk := range parentPK {
			v := fmt.Sprintf("_pkp_%d", i)
			partVars[i] = v

			code = append(code, fmt.Sprintf("    var %s string", v))
			if pk.resp.IsNullable {
				code = append(code, fmt.Sprintf("    if %s != nil { %s = fmt.Sprintf(\"%%v\", *%s) } else { %s = \"<nil>\" }", pk.varName, v, pk.varName, v))
			} else {
				code = append(code, fmt.Sprintf("    %s = fmt.Sprintf(\"%%v\", %s)", v, pk.varName))
			}
		}

		code = append(code, fmt.Sprintf("    pk_parent := strings.Join([]string{%s}, \"|\")", strings.Join(partVars, ", ")))
	}

	code = append(code, fmt.Sprintf("    if _parentMap == nil { _parentMap = make(map[string]*%s) }", mainStruct))
	code = append(code, "    parentObj, _okParent := _parentMap[pk_parent]")
	code = append(code, "    if !_okParent {")

	code = append(code, fmt.Sprintf("        parentObj = &%s{}", mainStruct))
	for _, c := range rootCols {
		code = append(code, fmt.Sprintf("        parentObj.%s = %s", celNameToGoName(c.resp.Name), c.varName))
	}
	// Initialize child slices to empty arrays (depth 1)
	for _, m := range metas {
		if m.Depth == 1 {
			childStructName := mainStruct
			for _, seg := range m.Path {
				childStructName += celNameToGoName(seg)
			}

			code = append(code, fmt.Sprintf("        parentObj.%s = make([]*%s, 0)", celNameToGoName(m.Path[0]), childStructName))
		}
	}

	code = append(code, "        _parentMap[pk_parent] = parentObj")
	code = append(code, "    }")
	// Parent full key for child chain
	code = append(code, "    _chain_parent := pk_parent")
	// For each meta (ordered by depth)
	for _, m := range metas {
		// Key fields must all be non-nil
		condNil := []string{}
		for _, kf := range m.KeyFields {
			condNil = append(condNil, fmt.Sprintf("col_%s == nil", strings.ToLower(celNameToGoName(kf))))
		}

		code = append(code, "    // Node "+strings.Join(m.Path, "__"))
		code = append(code, fmt.Sprintf("    if !(%s) {", strings.Join(condNil, " || ")))
		// Build node key (key fields are scanned as pointers when nullable; safe to deref due to nil guard)
		if len(m.KeyFields) == 1 {
			kf := m.KeyFields[0]
			k := strings.ToLower(celNameToGoName(kf))

			expr := "col_" + k
			if strings.Contains(kf, "__") {
				expr = "*" + expr
			} else if resp, ok := respByName[kf]; ok {
				if resp.IsNullable {
					expr = "*" + expr
				}
			}

			code = append(code, fmt.Sprintf("        _k_%s := fmt.Sprintf(\"%%v\", %s)", strings.Join(m.Path, "_"), expr))
		} else {
			fmtParts := make([]string, len(m.KeyFields))

			args := make([]string, len(m.KeyFields))
			for i, kf := range m.KeyFields {
				fmtParts[i] = "%v"

				arg := "col_" + strings.ToLower(celNameToGoName(kf))
				if strings.Contains(kf, "__") {
					arg = "*" + arg
				} else if resp, ok := respByName[kf]; ok {
					if resp.IsNullable {
						arg = "*" + arg
					}
				}

				args[i] = arg
			}

			code = append(code, fmt.Sprintf("        _k_%s := fmt.Sprintf(\"%s\", %s)", strings.Join(m.Path, "_"), strings.Join(fmtParts, "|"), strings.Join(args, ", ")))
		}

		code = append(code, fmt.Sprintf("        if _nodeMap_%s == nil { _nodeMap_%s = make(map[string]*%s) }", strings.Join(m.Path, "_"), strings.Join(m.Path, "_"), mainStruct+joinCamel(m.Path)))
		code = append(code, fmt.Sprintf("        _chain_%s := _chain_parent + \"|%s:\" + _k_%s", strings.Join(m.Path, "_"), strings.Join(m.Path, "__"), strings.Join(m.Path, "_")))
		code = append(code, fmt.Sprintf("        node_%s, _exists_%s := _nodeMap_%s[_chain_%s]", strings.Join(m.Path, "_"), strings.Join(m.Path, "_"), strings.Join(m.Path, "_"), strings.Join(m.Path, "_")))
		code = append(code, fmt.Sprintf("        if ! _exists_%s {", strings.Join(m.Path, "_")))
		code = append(code, fmt.Sprintf("            node_%s = &%s{}", strings.Join(m.Path, "_"), mainStruct+joinCamel(m.Path)))
		// Assign key fields first (hierarchy keys like ID)
		for _, kf := range m.KeyFields {
			leaf := kf[strings.LastIndex(kf, "__")+1:]
			colVar := "col_" + strings.ToLower(celNameToGoName(kf))
			code = append(code, fmt.Sprintf("            node_%s.%s = %s", strings.Join(m.Path, "_"), celNameToGoName(leaf), colVar))
		}
		// Assign data fields (exclude keys already assigned)
		keySet := map[string]struct{}{}
		for _, kf := range m.KeyFields {
			keySet[kf] = struct{}{}
		}

		for _, df := range m.DataFields {
			if _, exists := keySet[df]; exists {
				continue
			}

			leaf := df[strings.LastIndex(df, "__")+1:]
			resp := respByName[df]

			colVar := "col_" + strings.ToLower(celNameToGoName(df))
			if resp.IsNullable {
				code = append(code, fmt.Sprintf("            node_%s.%s = %s", strings.Join(m.Path, "_"), celNameToGoName(leaf), colVar))
			} else if strings.Contains(df, "__") {
				code = append(code, fmt.Sprintf("            if %s != nil {", colVar))
				code = append(code, fmt.Sprintf("                node_%s.%s = *%s", strings.Join(m.Path, "_"), celNameToGoName(leaf), colVar))
				code = append(code, "            }")
			} else {
				code = append(code, fmt.Sprintf("            node_%s.%s = %s", strings.Join(m.Path, "_"), celNameToGoName(leaf), colVar))
			}
		}
		// Initialize child slices for this node
		for _, child := range metas {
			// Check if child is a direct child of current node (parent path matches current path)
			if len(child.ParentPath) == len(m.Path) {
				isChild := true

				for i, seg := range m.Path {
					if child.ParentPath[i] != seg {
						isChild = false
						break
					}
				}

				if isChild {
					childStructName := mainStruct
					for _, seg := range child.Path {
						childStructName += celNameToGoName(seg)
					}

					childFieldName := celNameToGoName(child.Path[len(child.Path)-1])
					code = append(code, fmt.Sprintf("            node_%s.%s = make([]*%s, 0)", strings.Join(m.Path, "_"), childFieldName, childStructName))
				}
			}
		}
		// Append to parent slice
		if len(m.ParentPath) == 0 { // top-level
			code = append(code, fmt.Sprintf("            parentObj.%s = append(parentObj.%s, node_%s)", celNameToGoName(m.Path[0]), celNameToGoName(m.Path[0]), strings.Join(m.Path, "_")))
		} else {
			// Lookup parent by current chain key and append to its slice
			parentMapVar := "_nodeMap_" + strings.Join(m.ParentPath, "_")
			sliceField := celNameToGoName(m.Path[len(m.Path)-1])

			code = append(code, fmt.Sprintf("            if p, ok := %s[_chain_parent]; ok {", parentMapVar))
			code = append(code, fmt.Sprintf("                p.%s = append(p.%s, node_%s)", sliceField, sliceField, strings.Join(m.Path, "_")))
			code = append(code, "            }")
		}

		code = append(code, fmt.Sprintf("            _nodeMap_%s[_chain_%s] = node_%s", strings.Join(m.Path, "_"), strings.Join(m.Path, "_"), strings.Join(m.Path, "_")))
		code = append(code, "        }")
		code = append(code, "        _chain_parent = _chain_"+strings.Join(m.Path, "_"))
		code = append(code, "    }")
	}

	code = append(code, "}")

	code = append(code, "if err = rows.Err(); err != nil { return result, fmt.Errorf(\"error iterating rows: %w\", err) }")
	if isMany {
		code = append(code, "for _, v := range _parentMap { result = append(result, *v) }")
	} else {
		code = append(code, "if len(_parentMap) == 0 { return result, snapsql.ErrNotFound }")
		code = append(code, "if len(_parentMap) > 1 { return result, snapsql.ErrHierarchicalMultipleParentsForOne }")
		code = append(code, "for _, v := range _parentMap { result = *v }")
	}

	return code, nil
}

// joinCamel combines path segments converting each to Go-style CamelCase.
func joinCamel(segs []string) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(celNameToGoName(s))
	}

	return b.String()
}
