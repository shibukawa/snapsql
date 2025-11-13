package parsercommon

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

// GeneratePlaceholderData creates zero-value data tree from parameter definitions.
func GeneratePlaceholderData(params map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(params))

	for k, v := range params {
		switch val := v.(type) {
		case string:
			result[k] = generatePlaceholderValueFromString(val)
		case map[string]any:
			if typeVal, hasType := val["type"]; hasType {
				if typeStr, ok := typeVal.(string); ok {
					result[k] = generatePlaceholderValueFromString(typeStr)
				} else {
					result[k] = generatePlaceholderValueFromString("string")
				}
			} else {
				d, err := GeneratePlaceholderData(val)
				if err != nil {
					return nil, err
				}

				result[k] = d
			}
		case []any:
			if len(val) == 1 {
				switch elem := val[0].(type) {
				case string:
					if strings.HasPrefix(elem, "./") {
						result[k] = []any{elem}
					} else {
						result[k] = []any{generatePlaceholderValueFromString(elem)}
					}
				case map[string]any:
					d, err := GeneratePlaceholderData(elem)
					if err != nil {
						return nil, err
					}

					result[k] = []any{d}
				default:
					result[k] = []any{elem}
				}
			} else {
				result[k] = []any{}
			}
		default:
			return nil, fmt.Errorf("%w: %T", snapsql.ErrUnsupportedParameterType, v)
		}
	}

	return result, nil
}

func generatePlaceholderValueFromString(typeStr string) any {
	t := strings.TrimSpace(typeStr)
	switch t {
	case "string", "text", "varchar", "str":
		return "dummy"
	case "int":
		return int64(1)
	case "int32":
		return int32(2)
	case "int16":
		return int16(3)
	case "int8":
		return int8(4)
	case "float":
		return 1.1
	case "float32":
		return float32(2.2)
	case "decimal":
		return "1.0"
	case "bool":
		return true
	case "date":
		return "2024-01-01"
	case "datetime":
		return "2024-01-01 00:00:00"
	case "timestamp":
		return "2024-01-02 00:00:00"
	case "email":
		return "user@example.com"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "json":
		return map[string]any{"#": "json"}
	case "any":
		return map[string]any{"#": "any"}
	case "object":
		return map[string]any{"#": "object"}
	}

	if strings.HasSuffix(t, "[]") {
		base := t[:len(t)-2]
		return []any{generatePlaceholderValueFromString(base)}
	}

	if strings.HasPrefix(t, "./") {
		return t
	}

	return ""
}

// InferTypeStringFromDummyValue infers type string from a placeholder value generated above.
func InferTypeStringFromDummyValue(val any) string {
	switch v := val.(type) {
	case int64:
		if v == 1 {
			return "int"
		}
	case int32:
		if v == 2 {
			return "int32"
		}
	case int16:
		if v == 3 {
			return "int16"
		}
	case int8:
		if v == 4 {
			return "int8"
		}
	case float64:
		return "float"
	case float32:
		return "float32"
	case bool:
		return "bool"
	case string:
		return "string"
	case map[string]any:
		if tag, ok := v["#"]; ok {
			if tagStr, ok := tag.(string); ok {
				return tagStr
			}
		}

		return "object"
	default:
		return ""
	}

	return ""
}

// InferTypeStringFromActualValue infers type string from a runtime value (used for constants, etc.).
func InferTypeStringFromActualValue(val any) string {
	switch val.(type) {
	case int64, int32, int16, int8, int, uint, uint64, uint32, uint16, uint8:
		return "int"
	case float32, float64:
		return "float"
	case string:
		return "string"
	case bool:
		return "bool"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case *snapsqlgo.Decimal:
		return "decimal"
	default:
		return ""
	}
}
