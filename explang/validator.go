package explang

import (
	"fmt"
	"strings"
)

// ValidationError represents a mismatch between an explang Step and the declared parameters.
type ValidationError struct {
	StepIndex int
	Step      Step
	Message   string
}

func (e ValidationError) Error() string {
	return e.Message
}

// ValidatorOptions configures ValidateStepsAgainstParameters behaviour.
type ValidatorOptions struct {
	// AdditionalRoots allows callers to inject synthetic root symbols (e.g., system values).
	AdditionalRoots map[string]any
}

// ValidateStepsAgainstParameters ensures that every step is compatible with the given parameters map.
// params には FunctionDefinition.Parameters を渡す。
func ValidateStepsAgainstParameters(steps []Step, params map[string]any, opts *ValidatorOptions) []ValidationError {
	if len(steps) == 0 {
		return nil
	}

	rootTypes := make(map[string]*typeNode, len(params))
	for name, value := range params {
		rootTypes[name] = describeValue(value)
	}

	if opts != nil {
		for name, value := range opts.AdditionalRoots {
			rootTypes[name] = describeValue(value)
		}
	}

	var (
		errs         []ValidationError
		currentNode  *typeNode
		path         string
		contextKnown bool
	)

	setUnknown := func() {
		currentNode = unknownType()
		contextKnown = false
	}

	for idx, step := range steps {
		switch step.Kind {
		case StepIdentifier:
			path = step.Identifier

			node, ok := rootTypes[step.Identifier]
			if !ok {
				errs = append(errs, ValidationError{
					StepIndex: idx,
					Step:      step,
					Message:   fmt.Sprintf("unknown root parameter %q", step.Identifier),
				})

				setUnknown()

				continue
			}

			currentNode = node
			contextKnown = currentNode != nil && currentNode.kind != kindUnknown
		case StepMember:
			path = joinPath(path, step.Property)

			if !contextKnown || currentNode == nil {
				continue
			}

			if currentNode.kind != kindObject {
				errs = append(errs, ValidationError{
					StepIndex: idx,
					Step:      step,
					Message:   fmt.Sprintf("cannot access member %q on %s", step.Property, describeCurrent(path, currentNode)),
				})

				setUnknown()

				continue
			}

			child, ok := currentNode.fields[step.Property]
			if !ok {
				errs = append(errs, ValidationError{
					StepIndex: idx,
					Step:      step,
					Message:   fmt.Sprintf("unknown field %q on parameter %q", step.Property, path),
				})

				setUnknown()

				continue
			}

			currentNode = child
			contextKnown = currentNode != nil && currentNode.kind != kindUnknown
		case StepIndex:
			parentPath := path
			path = fmt.Sprintf("%s[%d]", path, step.Index)

			if !contextKnown || currentNode == nil {
				continue
			}

			if currentNode.kind != kindArray {
				errs = append(errs, ValidationError{
					StepIndex: idx,
					Step:      step,
					Message:   fmt.Sprintf("parameter %q is not an array (type %s)", parentPath, currentNode.describe()),
				})

				setUnknown()

				continue
			}

			if currentNode.elem == nil {
				currentNode = unknownType()
				contextKnown = false

				continue
			}

			currentNode = currentNode.elem
			contextKnown = currentNode.kind != kindUnknown
		}
	}

	return errs
}

func joinPath(base, property string) string {
	if base == "" {
		return property
	}

	return base + "." + property
}

func describeCurrent(path string, node *typeNode) string {
	if node == nil {
		return fmt.Sprintf("parameter %q (unknown)", path)
	}

	return fmt.Sprintf("parameter %q (type %s)", path, node.describe())
}

type typeKind int

const (
	kindUnknown typeKind = iota
	kindScalar
	kindObject
	kindArray
)

type typeNode struct {
	kind     typeKind
	typeName string
	elem     *typeNode
	fields   map[string]*typeNode
}

func unknownType() *typeNode {
	return &typeNode{kind: kindUnknown}
}

func describeValue(v any) *typeNode {
	switch val := v.(type) {
	case string:
		return describeStringType(val)
	case map[string]any:
		fields := make(map[string]*typeNode, len(val))
		for k, child := range val {
			fields[k] = describeValue(child)
		}

		return &typeNode{kind: kindObject, fields: fields}
	case []any:
		if len(val) == 0 {
			return &typeNode{kind: kindArray, elem: unknownType()}
		}

		return &typeNode{kind: kindArray, elem: describeValue(val[0])}
	case nil:
		return unknownType()
	default:
		return &typeNode{kind: kindScalar, typeName: inferLiteralType(val)}
	}
}

func describeStringType(typeName string) *typeNode {
	t := strings.TrimSpace(typeName)
	if t == "" {
		return unknownType()
	}

	depth := 0
	for strings.HasSuffix(t, "[]") {
		depth++
		t = strings.TrimSuffix(t, "[]")
		t = strings.TrimSpace(t)
	}

	if t == "" {
		t = "any"
	}

	var node = &typeNode{kind: kindScalar, typeName: t}

	for depth > 0 {
		depth--
		node = &typeNode{kind: kindArray, elem: node}
	}

	return node
}

func inferLiteralType(v any) string {
	switch v.(type) {
	case int, int64, int32, int16, int8:
		return "int"
	case uint, uint64, uint32, uint16, uint8:
		return "uint"
	case float32, float64:
		return "float"
	case bool:
		return "bool"
	case string:
		return "string"
	default:
		return "any"
	}
}

func (n *typeNode) describe() string {
	if n == nil {
		return "unknown"
	}

	switch n.kind {
	case kindScalar:
		if n.typeName != "" {
			return n.typeName
		}

		return "scalar"
	case kindObject:
		return "object"
	case kindArray:
		if n.elem == nil {
			return "array"
		}

		return "array of " + n.elem.describe()
	default:
		return "unknown"
	}
}
