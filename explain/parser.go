package explain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	errPlanJSONEmpty          = errors.New("explain: plan json is empty")
	errMySQLMissingQueryBlock = errors.New("explain: mysql plan missing query_block")
)

// ParsePlanJSON converts the raw JSON bytes into PlanNode structures.
func ParsePlanJSON(dialect string, data []byte) ([]*PlanNode, error) {
	if len(data) == 0 {
		return nil, errPlanJSONEmpty
	}

	switch strings.ToLower(dialect) {
	case "postgres", "postgresql", "pgx":
		return parsePostgresPlan(data)
	case "mysql", "mariadb":
		return parseMySQLPlan(data)
	default:
		return nil, ErrUnsupportedDialect
	}
}

func parsePostgresPlan(data []byte) ([]*PlanNode, error) {
	var container []map[string]any
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, fmt.Errorf("failed to unmarshal postgres plan: %w", err)
	}

	var nodes []*PlanNode

	for idx, entry := range container {
		planRaw, ok := entry["Plan"].(map[string]any)
		if !ok {
			continue
		}

		node := buildPostgresNode(planRaw, fmt.Sprintf("main[%d]", idx))
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func buildPostgresNode(plan map[string]any, path string) *PlanNode {
	node := &PlanNode{
		NodeType:        getString(plan, "Node Type"),
		Relation:        getString(plan, "Relation Name"),
		Schema:          getString(plan, "Schema"),
		Alias:           getString(plan, "Alias"),
		QueryPath:       path,
		ActualRows:      getFloat(plan, "Actual Rows"),
		PlanRows:        getFloat(plan, "Plan Rows"),
		ActualTotalTime: getFloat(plan, "Actual Total Time"),
		EstimatedCost:   getFloat(plan, "Total Cost"),
	}

	if strings.Contains(strings.ToLower(node.NodeType), "seq scan") {
		node.AccessType = "ALL"
	}

	if rawChildren, ok := plan["Plans"].([]any); ok {
		for idx, child := range rawChildren {
			if childMap, ok := child.(map[string]any); ok {
				childNode := buildPostgresNode(childMap, fmt.Sprintf("%s.child[%d]", path, idx))
				node.Children = append(node.Children, childNode)
			}
		}
	}

	return node
}

func parseMySQLPlan(data []byte) ([]*PlanNode, error) {
	var container map[string]any
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mysql plan: %w", err)
	}

	block, ok := container["query_block"].(map[string]any)
	if !ok {
		return nil, errMySQLMissingQueryBlock
	}

	node := buildMySQLNode(block, "main")

	return []*PlanNode{node}, nil
}

func buildMySQLNode(block map[string]any, path string) *PlanNode {
	node := &PlanNode{
		NodeType:        getString(block, "select_id"),
		QueryPath:       path,
		ActualRows:      getFloat(block, "actual_rows"),
		PlanRows:        getFloat(block, "rows"),
		ActualTotalTime: getFloat(block, "actual_total_time"),
	}

	if tableRaw, ok := block["table"].(map[string]any); ok {
		tableNode := &PlanNode{
			NodeType:        "Table",
			Relation:        getString(tableRaw, "table_name"),
			AccessType:      strings.ToUpper(getString(tableRaw, "access_type")),
			ActualRows:      getFloat(tableRaw, "actual_rows"),
			PlanRows:        getFloat(tableRaw, "rows"),
			ActualTotalTime: getFloat(tableRaw, "actual_total_time"),
			QueryPath:       path + ".table",
		}
		node.Children = append(node.Children, tableNode)
	}

	if nested, ok := block["nested_loop"].([]any); ok {
		for idx, child := range nested {
			if childMap, ok := child.(map[string]any); ok {
				node.Children = append(node.Children, buildMySQLNode(childMap, fmt.Sprintf("%s.nested[%d]", path, idx)))
			}
		}
	}

	return node
}

func getString(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}

	if v, ok := obj[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

func getFloat(obj map[string]any, key string) float64 {
	if obj == nil {
		return 0
	}

	if v, ok := obj[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case json.Number:
			f, err := n.Float64()
			if err == nil {
				return f
			}
		}
	}

	return 0
}
