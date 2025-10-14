package intermediate

import (
	"fmt"
	"strings"
)

// BuildTableReferenceMap builds a lookup map from various identifiers (names, aliases) to the
// underlying table reference metadata. Keys are stored in lower-case without quotation characters
// for case-insensitive matching.
func BuildTableReferenceMap(refs []TableReferenceInfo) map[string]TableReferenceInfo {
	if len(refs) == 0 {
		return nil
	}

	mapping := make(map[string]TableReferenceInfo)

	for _, ref := range refs {
		addReferenceKey(mapping, ref.Name, ref)
		addReferenceKey(mapping, ref.Alias, ref)
		addReferenceKey(mapping, ref.TableName, ref)
		addReferenceKey(mapping, ref.QueryName, ref)

		physical := strings.TrimSpace(ref.TableName)
		if physical != "" {
			addReferenceKey(mapping, physical, ref)

			if idx := strings.Index(physical, "."); idx > 0 && idx < len(physical)-1 {
				addReferenceKey(mapping, physical[idx+1:], ref)
			}
		}
	}

	if len(mapping) == 0 {
		return nil
	}

	return mapping
}

func addReferenceKey(mapping map[string]TableReferenceInfo, key string, ref TableReferenceInfo) {
	canonical := CanonicalIdentifier(key)
	if canonical == "" {
		return
	}

	if _, exists := mapping[canonical]; !exists {
		mapping[canonical] = ref
	}
}

// DescribeTable returns a human-readable description of the physical table and its alias/CTE context.
// Example: table 'lists' in 'done_stage'(CTE/subquery)
func DescribeTable(ref TableReferenceInfo, alias string) string {
	return DescribeTableWithResolver(ref, alias, nil)
}

// DescribeTableWithResolver extends DescribeTable by allowing the caller to determine whether a name
// already represents a physical table via the supplied checker.
func DescribeTableWithResolver(ref TableReferenceInfo, alias string, isPhysical func(string) bool) string {
	aliasName := strings.TrimSpace(alias)

	physical := strings.TrimSpace(ref.TableName)
	if physical == "" {
		physical = strings.TrimSpace(ref.Name)
	}

	if physical == "" {
		physical = aliasName
	}

	if strings.TrimSpace(physical) == "" {
		return "table '<unknown>'"
	}

	contextLabel := classifyContext(ref.Context)
	physicalDisplay := strings.TrimSpace(physical)
	aliasIsPhysical := aliasName != "" && isPhysical != nil && isPhysical(aliasName)

	switch contextLabel {
	case "CTE", "subquery":
		if aliasName == "" || strings.EqualFold(aliasName, physicalDisplay) {
			return fmt.Sprintf("table '%s' (%s)", physicalDisplay, contextLabel)
		}

		return fmt.Sprintf("table '%s' in '%s'(%s)", physicalDisplay, aliasName, contextLabel)
	case "join":
		if aliasName != "" && !strings.EqualFold(aliasName, physicalDisplay) && !aliasIsPhysical {
			return fmt.Sprintf("table '%s' in '%s'(%s)", physicalDisplay, aliasName, contextLabel)
		}

		return fmt.Sprintf("table '%s' (%s)", physicalDisplay, contextLabel)
	default:
		return fmt.Sprintf("table '%s'", physicalDisplay)
	}
}

// CanonicalIdentifier normalizes identifiers for case-insensitive comparisons.
func CanonicalIdentifier(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.Trim(trimmed, "`\"[]")
	if trimmed == "" {
		return ""
	}

	return strings.ToLower(trimmed)
}

func classifyContext(ctx string) string {
	switch strings.ToLower(strings.TrimSpace(ctx)) {
	case "cte":
		return "CTE"
	case "subquery":
		return "subquery"
	case "join":
		return "join"
	case "main", "":
		return ""
	default:
		return ctx
	}
}
