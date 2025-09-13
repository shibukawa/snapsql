package query

import (
    "errors"
    "fmt"
    "sort"
    "strings"
    "github.com/shibukawa/snapsql/intermediate"
)

var (
    // ErrMissingRequiredParam is returned when one or more required parameters are missing.
    ErrMissingRequiredParam = errors.New("missing required parameter")
)

// ValidateParameters checks that all required parameters defined in the
// intermediate format are present in the provided params map.
// It ignores implicit parameters supplied by runtime context.
func ValidateParameters(format *intermediate.IntermediateFormat, params map[string]any) error {
    if format == nil {
        return nil
    }
    provided := map[string]struct{}{}
    for k := range params {
        provided[k] = struct{}{}
    }

    // Collect implicit parameter names to exclude from required checking
    implicit := map[string]struct{}{}
    for _, ip := range format.ImplicitParameters {
        implicit[ip.Name] = struct{}{}
    }

    var missing []string
    for _, p := range format.Parameters {
        if p.Optional {
            continue
        }
        if _, isImplicit := implicit[p.Name]; isImplicit {
            continue
        }
        if _, ok := provided[p.Name]; !ok {
            // include type if available
            typ := strings.TrimSpace(p.Type)
            if typ != "" {
                missing = append(missing, fmt.Sprintf("%s (%s)", p.Name, typ))
            } else {
                missing = append(missing, p.Name)
            }
        }
    }

    if len(missing) == 0 {
        return nil
    }
    sort.Strings(missing)
    return fmt.Errorf("%w: %s", ErrMissingRequiredParam, strings.Join(missing, ", "))
}

