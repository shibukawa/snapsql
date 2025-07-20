generators:
  go:
    package: "user"
    output: "./generated/user.go"
    use_struct: true
    input_type: "map[string]any"
    output_type: "map[string]any"
  typescript:
    output: "./generated/user.ts"
    use_interface: true

# FunctionDefinition Specification

## Overview

FunctionDefinition is a structure that comprehensively manages function definition information for generating application code from SQL templates and query definitions. It contains attributes, parameters, metadata, and language-specific generator settings required for generating each function.

## Required Fields

- **FunctionName** (`string`)
  - The name of the generated function. Extracted from file names or frontmatter and converted according to language-specific naming conventions.
- **Description** (`string`)
  - Function description. Used for documentation and code comments.
- **Parameters** (`map[string]any`)
  - Input parameter definitions. Contains type information (int, str, bool, arrays, nested structures, etc.) and preserves YAML definition order. Used for type generation, validation, and argument ordering.
  - If it includes common type references (type names starting with uppercase), they are resolved during Finalize.

## Optional Fields

- **Generators** (`map[string]map[string]any`)
  - Additional attributes for each language generator. The first key is the language name (e.g., "go", "typescript"), and the value contains language-specific attributes (e.g., for Go: package name, output path, whether to generate structs, input/output type specifications).
- **CommonTypesPath** (`string`)
  - Path to the common type definition file. Defaults to `_common.yaml` in the same directory as the processed file.
- **CommonTypes** (`map[string]CommonTypeDefinition`)
  - Cache of resolved common type definitions. Built during Finalize.

### Generators Example

```yaml
generators:
  go:
    package: "user"
    output: "./generated/user.go"
    use_struct: true
    input_type: "map[string]any"
    output_type: "map[string]any"
  typescript:
    output: "./generated/user.ts"
    use_interface: true
```

## Design Principles & Notes

- Parameters support hierarchical structures (nesting, arrays, objects)
- Parameters are extracted from YAML nodes in order and reflected in type generation and argument ordering
- Language-specific generator attributes are extensible
- Balances template definition flexibility with type safety

## Mapping with Markdown Files

FunctionDefinition is extracted by combining the following three information sources from Markdown query definition files (.snap.md):

- **FrontMatter**
  - Extracts metadata such as FunctionName and generator settings from the YAML frontmatter (section enclosed by ---) at the beginning of the file.
- **Overview Section**
  - The text in the `## Overview` section is assigned to the Description.
- **Parameters Section**
  - The YAML code block in the `## Parameters` section is assigned to Parameters.

These three pieces of information are combined to extract and integrate the information required for the FunctionDefinition specification.

## Common Type Definition Feature

### Overview

The common type definition feature allows reuse of the same parameter structures (e.g., user information, department information) across multiple queries. Common types are defined in `_common.yaml` files in each directory and can be referenced using relative paths.

### Common Type Definition Files

- Place `_common.yaml` files in each directory
- Identifiers starting with uppercase letters are recognized as struct types
- Types can reference each other within the same file
- Comments can be added to type definitions and fields

### Common Type Definition Example

```yaml
# _common.yaml
User: # Common type representing user information
  id: int # User ID
  name: string # User name
  email: string # Email address
  department: Department # Department

Department: # Common type representing department information
  id: int # Department ID
  name: string # Department name
  manager_id: int # Manager ID
```

### How to Reference Common Types

Common types can be referenced in parameter definitions using the following formats:

1. **Referencing types in the same directory**:
   ```yaml
   parameters:
     user: User        # Reference from _common.yaml in the same directory
     admin: .User      # Explicitly specify the same directory (starts with .)
   ```

2. **Referencing types in different directories**:
   ```yaml
   parameters:
     user: ../User     # Reference from _common.yaml in the parent directory
     member: ./sub/User # Reference from _common.yaml in a subdirectory
   ```

3. **Referencing as array types**:
   ```yaml
   parameters:
     users: User[]     # Array of User
     departments: ../Department[] # Array of Department from the parent directory
   ```

### Type Resolution Mechanism

1. If a parameter type starts with an uppercase letter, it is interpreted as a common type
2. If a path is specified (e.g., `./`, `../`), the `_common.yaml` in that path is referenced
3. If no path is specified, the `_common.yaml` in the current directory is referenced
4. If the common type is not found, an error occurs
5. Circular references are allowed (as some languages permit circular references)

### Implementation Notes

- Common type resolution is performed all at once in the `FunctionDefinition.Finalize()` method
- Common type reference resolution is based on relative paths
- Comments on common types are utilized during documentation generation
- If a common type is changed, all function definitions that reference it need to be regenerated

---

(Updated 2025-07-20: Added common type definition feature)

## Common Type Definition Implementation

The following functionality is added to the FunctionDefinition struct to process common type definitions:

```go
type FunctionDefinition struct {
    // Existing fields
    Name           string
    Description    string
    FunctionName   string
    Parameters     map[string]any
    ParameterOrder []string
    RawParameters  yaml.MapSlice
    Generators     map[string]map[string]any
    
    // Common type related fields
    commonTypes    map[string]map[string]any  // Loaded common type definitions (internal use)
    basePath       string                     // Base path for resolving relative paths
}
```

### Common Type Resolution Process

1. Load necessary `_common.yaml` files in the `FunctionDefinition.Finalize()` method
2. Detect common type references (type names starting with uppercase) in parameters
3. Resolve relative paths and retrieve corresponding common type definitions
4. Replace parameter definitions with common type definitions
5. Recursively resolve references within common types

### State After Processing

After common type resolution is complete, the `Parameters` field contains type definitions with expanded common types. This allows subsequent processing (dummy data generation, code generation, etc.) to use existing logic without being aware of common types.

## Common Type Definition Files

### File Format

Common type definition files are written in YAML format. The file name is `_common.yaml` and is placed in each directory.

```yaml
# _common.yaml
User: # Common type representing user information
  id: int # User ID
  name: string # User name
  email: string # Email address
  department: Department # Department (can reference other types in the same file)

Department: # Common type representing department information
  id: int # Department ID
  name: string # Department name
  manager_id: int # Manager ID
```

### Loading Method

Common type definition files are loaded all at once when the `FunctionDefinition.Finalize()` method is executed:

1. Load `_common.yaml` from the current directory
2. Detect relative path references (e.g., `../User`) in parameters and also load `_common.yaml` from the corresponding directories
3. Loaded common type definitions are stored in the commonTypes map of FunctionDefinition

### Path Resolution

Common type reference path resolution follows these rules:

1. If no path is specified (e.g., `User`)
   - Search for the type in `_common.yaml` in the current directory
2. If a relative path is specified (e.g., `../User`, `./sub/User`)
   - Search for the type in `_common.yaml` in the specified path
3. Absolute paths are not supported

### Error Handling

Errors occur in the following cases:

1. When a referenced common type is not found
2. When a common type definition file does not exist
3. When the format of a common type definition file is invalid

Error messages include the location (file path, line number) and cause of the problem.
