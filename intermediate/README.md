# Intermediate Format

The intermediate format package provides a standardized JSON representation of parsed SQL templates and their metadata.

## Usage

### Basic Usage

```go
package main

import (
    "os"
    "github.com/shibukawa/snapsql/intermediate"
    "github.com/shibukawa/snapsql/parser"
)

func main() {
    // Create intermediate format
    format := intermediate.NewFormat()
    
    // Set source information
    format.SetSource("users.sql", "SELECT * FROM users WHERE active = /*= active */true")
    
    // Add interface schema (optional)
    schema := &parser.InterfaceSchema{
        Name:         "UserQuery",
        Description:  "Query active users",
        FunctionName: "queryUsers",
        OrderedParams: parser.NewOrderedParameters(),
    }
    schema.OrderedParams.Add("active", "bool")
    format.SetInterfaceSchema(schema)
    
    // Write JSON output
    err := format.WriteJSON(os.Stdout, true) // pretty-printed
    if err != nil {
        panic(err)
    }
}
```

### WriteJSON Method

The `WriteJSON(w io.Writer, pretty bool)` method provides flexible output options:

- **w**: Any `io.Writer` (file, buffer, stdout, network connection, etc.)
- **pretty**: `true` for formatted JSON with indentation, `false` for compact JSON

```go
// Write to file
file, err := os.Create("output.json")
if err != nil {
    panic(err)
}
defer file.Close()

err = format.WriteJSON(file, true)

// Write to buffer
var buf bytes.Buffer
err = format.WriteJSON(&buf, false) // compact

// Write to stdout
err = format.WriteJSON(os.Stdout, true) // pretty
```

### Complex Parameter Structures

The intermediate format supports recursive parameter structures:

```go
// Create complex nested parameters
schema := &parser.InterfaceSchema{
    Name:         "ComplexQuery",
    FunctionName: "complexQuery",
    OrderedParams: parser.NewOrderedParameters(),
}

// Add simple parameter
schema.OrderedParams.Add("active", "bool")

// Add nested object parameter
filtersMap := map[string]any{
    "name":        "string",
    "department":  []any{"string"},
    "permissions": map[string]any{
        "read":  "bool",
        "write": "bool",
    },
}
schema.OrderedParams.Add("filters", filtersMap)

// Add array parameter
schema.OrderedParams.Add("sort_fields", []any{"string"})

format.SetInterfaceSchema(schema)
```

This generates JSON output like:

```json
{
  "interface_schema": {
    "name": "ComplexQuery",
    "function_name": "complexQuery",
    "parameters": [
      {
        "name": "active",
        "type": "bool"
      },
      {
        "name": "filters",
        "type": "object",
        "children": [
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "department",
            "type": "array",
            "children": [
              {
                "name": "o1",
                "type": "string"
              }
            ]
          },
          {
            "name": "permissions",
            "type": "object",
            "children": [
              {
                "name": "read",
                "type": "bool"
              },
              {
                "name": "write",
                "type": "bool"
              }
            ]
          }
        ]
      },
      {
        "name": "sort_fields",
        "type": "array",
        "children": [
          {
            "name": "o1",
            "type": "string"
          }
        ]
      }
    ]
  }
}
```

## Output Format Structure

The intermediate format JSON contains the following sections:

### Source Information
```json
{
  "source": {
    "file": "path/to/file.sql",
    "content": "SQL template content"
  }
}
```

### Interface Schema (Optional)
```json
{
  "interface_schema": {
    "name": "QueryName",
    "description": "Query description",
    "function_name": "functionName",
    "parameters": [
      {
        "name": "param_name",
        "type": "param_type",
        "children": [] // For complex types
      }
    ]
  }
}
```

### AST (Optional)
```json
{
  "ast": {
    "type": "SELECT_STATEMENT",
    "pos": [line, column, offset],
    "Children": {
      // AST node children
    }
  }
}
```

## Backward Compatibility

The legacy methods `ToJSON()` and `ToJSONPretty()` are still available but deprecated:

```go
// Deprecated: use WriteJSON instead
jsonData, err := format.ToJSON()

// Deprecated: use WriteJSON instead  
prettyJSON, err := format.ToJSONPretty()
```

## Type System

The intermediate format supports the following parameter types:

- **Primitive types**: `string`, `bool`, `int`, `float`, etc.
- **Array types**: `[]any{element_type}`
- **Object types**: `map[string]any{field: type}`
- **Recursive structures**: Nested objects and arrays

Array elements are automatically named `o1`, `o2`, etc. for consistent JSON representation.
