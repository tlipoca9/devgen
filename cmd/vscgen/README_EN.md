# vscgen

VSCode extension configuration generator that creates configuration from `devgen.toml` files.

## Overview

`vscgen` is a core component of the devgen toolchain. It reads `devgen.toml` configuration files from each code generator (such as enumgen, validategen) and generates a unified `tools-config.json` for the VSCode extension.

This design implements the **configuration as documentation** philosophy:
- Generator developers only need to maintain a single `devgen.toml` configuration
- VSCode extension automatically gains annotation completion, parameter validation, documentation hints, etc.
- Adding new generators requires no changes to the extension code

## Installation

```bash
go install github.com/tlipoca9/devgen/cmd/vscgen@latest
```

## Usage

```bash
# Default: scan cmd directory, output to vscode-devgen/src
vscgen

# Custom input/output directories
vscgen -input ./generators -output ./extension/src
```

### Command Line Arguments

| Argument | Default | Description |
|----------|---------|-------------|
| `-input` | `cmd` | Directory containing generator subdirectories (each should have `devgen.toml`) |
| `-output` | `vscode-devgen/src` | Directory to output `tools-config.json` |

## devgen.toml Configuration Reference

`devgen.toml` is the core configuration file of the devgen ecosystem, defining the annotations, parameter types, and documentation supported by code generators.

### Complete Configuration Structure

```toml
# DevGen tool configuration
# This file is used to generate VSCode extension configuration

[tool]
name = "toolname"           # Tool name, used as annotation prefix (toolname:@annotation)
output_suffix = "_gen.go"   # Suffix for generated files

[[annotations]]
name = "annotationName"     # Annotation name
type = "type"               # Annotation type: "type" (type-level) or "field" (field-level)
doc = "Annotation documentation"  # Documentation shown in completion and hover

[annotations.params]        # Parameter configuration (optional)
type = "string"             # Parameter type
placeholder = "value"       # Snippet placeholder
```

### Parameter Types Reference

#### 1. No-Parameter Annotations

For annotations that don't require parameters:

```toml
[[annotations]]
name = "required"
type = "field"
doc = "Field must not be empty/zero"
# No [annotations.params] section
```

Usage: `// validategen:@required`

#### 2. Single-Type Parameters

```toml
[[annotations]]
name = "min"
type = "field"
doc = "Minimum value or length"

[annotations.params]
type = "string"         # Options: "string", "number", "bool", "list"
placeholder = "value"   # Placeholder shown in snippet
```

Supported parameter types:
- `"string"` - String value
- `"number"` - Numeric value (integer or float)
- `"bool"` - Boolean value (true/false)
- `"list"` - Comma-separated list of values

Usage: `// validategen:@min(10)`

#### 3. Multi-Type Parameters

Parameters that support multiple types:

```toml
[[annotations]]
name = "eq"
type = "field"
doc = "Must equal specified value (supports string, number, bool)"

[annotations.params]
type = ["string", "number", "bool"]  # Array format for multiple types
placeholder = "value"
```

Usage:
- `// validategen:@eq(hello)` - String
- `// validategen:@eq(42)` - Number
- `// validategen:@eq(true)` - Boolean

#### 4. Enum Parameters

Predefined list of options:

```toml
[[annotations]]
name = "enum"
type = "type"
doc = "Generate enum helper methods (options: string, json, text, sql)"

[annotations.params]
values = ["string", "json", "text", "sql"]  # List of valid values

[annotations.params.docs]                    # Documentation for each option (optional)
string = "Generate String() method"
json = "Generate MarshalJSON/UnmarshalJSON methods"
text = "Generate MarshalText/UnmarshalText methods"
sql = "Generate Value/Scan methods for database/sql"
```

Usage: `// enumgen:@enum(string, json, sql)`

#### 5. Enum with Argument Limit

```toml
[[annotations]]
name = "format"
type = "field"
doc = "Must be valid format (json, yaml, toml, csv)"

[annotations.params]
values = ["json", "yaml", "toml", "csv"]
maxArgs = 1                                  # Maximum 1 argument allowed

[annotations.params.docs]
json = "Validate JSON format"
yaml = "Validate YAML format"
toml = "Validate TOML format"
csv = "Validate CSV format"
```

Usage: `// validategen:@format(json)` (only one option allowed)

### Complete Example: enumgen

```toml
# DevGen tool configuration for enumgen
# This file is used to generate VSCode extension configuration

[tool]
name = "enumgen"
output_suffix = "_enum.go"

[[annotations]]
name = "enum"
type = "type"
doc = "Generate enum helper methods (options: string, json, text, sql)"

[annotations.params]
values = ["string", "json", "text", "sql"]

[annotations.params.docs]
string = "Generate String() method"
json = "Generate MarshalJSON/UnmarshalJSON methods"
text = "Generate MarshalText/UnmarshalText methods"
sql = "Generate Value/Scan methods for database/sql"

[[annotations]]
name = "name"
type = "field"
doc = "Custom name for enum value"

[annotations.params]
type = "string"
placeholder = "name"
```

### Complete Example: validategen

```toml
# DevGen tool configuration for validategen
# This file is used to generate VSCode extension configuration

[tool]
name = "validategen"
output_suffix = "_validate.go"

[[annotations]]
name = "validate"
type = "type"
doc = "Generate Validate() method for struct"

# No-parameter annotation
[[annotations]]
name = "required"
type = "field"
doc = "Field must not be empty/zero"

# Number parameter
[[annotations]]
name = "min"
type = "field"
doc = "Minimum value or length"

[annotations.params]
type = "number"
placeholder = "value"

[[annotations]]
name = "max"
type = "field"
doc = "Maximum value or length"

[annotations.params]
type = "number"
placeholder = "value"

# Multi-type parameter
[[annotations]]
name = "eq"
type = "field"
doc = "Must equal specified value (supports string, number, bool)"

[annotations.params]
type = ["string", "number", "bool"]
placeholder = "value"

# List parameter
[[annotations]]
name = "oneof"
type = "field"
doc = "Must be one of the specified values"

[annotations.params]
type = "list"
placeholder = "values"

# String parameter
[[annotations]]
name = "regex"
type = "field"
doc = "Must match the specified regular expression"

[annotations.params]
type = "string"
placeholder = "pattern"

# Enum with argument limit
[[annotations]]
name = "format"
type = "field"
doc = "Must be valid format (json, yaml, toml, csv)"

[annotations.params]
values = ["json", "yaml", "toml", "csv"]
maxArgs = 1

[annotations.params.docs]
json = "Validate JSON format"
yaml = "Validate YAML format"
toml = "Validate TOML format"
csv = "Validate CSV format"
```

## Generated tools-config.json

`vscgen` merges all `devgen.toml` files into a single JSON file:

```json
{
  "enumgen": {
    "typeAnnotations": ["enum"],
    "fieldAnnotations": ["name"],
    "outputSuffix": "_enum.go",
    "annotations": {
      "enum": {
        "doc": "Generate enum helper methods (options: string, json, text, sql)",
        "paramType": "enum",
        "values": ["string", "json", "text", "sql"],
        "valueDocs": {
          "string": "Generate String() method",
          "json": "Generate MarshalJSON/UnmarshalJSON methods",
          "text": "Generate MarshalText/UnmarshalText methods",
          "sql": "Generate Value/Scan methods for database/sql"
        }
      },
      "name": {
        "doc": "Custom name for enum value",
        "paramType": "string",
        "placeholder": "name"
      }
    }
  },
  "validategen": {
    "typeAnnotations": ["validate"],
    "fieldAnnotations": ["required", "min", "max", ...],
    "outputSuffix": "_validate.go",
    "annotations": {
      "validate": {
        "doc": "Generate Validate() method for struct"
      },
      "required": {
        "doc": "Field must not be empty/zero"
      },
      "min": {
        "doc": "Minimum value or length",
        "paramType": "number",
        "placeholder": "value"
      }
      // ...
    }
  }
}
```

## Adding a New Generator

1. Create generator directory: `cmd/mygengen/`

2. Create `devgen.toml`:
```toml
[tool]
name = "mygengen"
output_suffix = "_mygen.go"

[[annotations]]
name = "generate"
type = "type"
doc = "Generate custom code"

[[annotations]]
name = "option"
type = "field"
doc = "Custom option"

[annotations.params]
type = "string"
placeholder = "value"
```

3. Re-run `vscgen`:
```bash
vscgen
```

4. Rebuild VSCode extension:
```bash
cd vscode-devgen
npm run compile
npm run package
```

The VSCode extension will automatically support the new annotation completion, parameter validation, and documentation hints.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ cmd/enumgen/    │     │ cmd/validategen/│     │ cmd/mygengen/   │
│ devgen.toml     │     │ devgen.toml     │     │ devgen.toml     │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                                 ▼
                        ┌────────────────┐
                        │    vscgen      │
                        │  (this tool)   │
                        └────────┬───────┘
                                 │
                                 ▼
                    ┌────────────────────────┐
                    │ vscode-devgen/src/     │
                    │ tools-config.json      │
                    └────────────┬───────────┘
                                 │
                                 ▼
                    ┌────────────────────────┐
                    │ VSCode Extension       │
                    │ - Syntax Highlighting  │
                    │ - Auto-completion      │
                    │ - Parameter Validation │
                    │ - Diagnostics          │
                    └────────────────────────┘
```

## License

MIT
