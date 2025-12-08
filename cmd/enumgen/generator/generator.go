// Package generator provides enum code generation functionality.
package generator

import (
	"fmt"
	"strings"

	"github.com/tlipoca9/devgen/cmd/enumgen/rules"
	"github.com/tlipoca9/devgen/genkit"
)

// ToolName is the name of this tool, used in annotations.
const ToolName = "enumgen"

// Error codes for diagnostics.
const (
	ErrCodeUnsupportedType  = "E001"
	ErrCodeNameOnStringType = "E002"
	ErrCodeDuplicateName    = "E003"
	ErrCodeNameMissingParam = "E004"
)

// GenerateOption represents an enum generation option.
// enumgen:@enum(string, json, text, sql)
type GenerateOption int

const (
	// enumgen:@name(string)
	GenerateOptionString GenerateOption = iota + 1
	// enumgen:@name(json)
	GenerateOptionJSON
	// enumgen:@name(text)
	GenerateOptionText
	// enumgen:@name(sql)
	GenerateOptionSQL
)

// UnderlyingType represents supported underlying types for enums.
// enumgen:@enum(string)
type UnderlyingType string

const (
	UnderlyingTypeInt    UnderlyingType = "int"
	UnderlyingTypeInt8   UnderlyingType = "int8"
	UnderlyingTypeInt16  UnderlyingType = "int16"
	UnderlyingTypeInt32  UnderlyingType = "int32"
	UnderlyingTypeInt64  UnderlyingType = "int64"
	UnderlyingTypeUint   UnderlyingType = "uint"
	UnderlyingTypeUint8  UnderlyingType = "uint8"
	UnderlyingTypeUint16 UnderlyingType = "uint16"
	UnderlyingTypeUint32 UnderlyingType = "uint32"
	UnderlyingTypeUint64 UnderlyingType = "uint64"
	UnderlyingTypeString UnderlyingType = "string"
)

// Generator generates enum helper methods.
type Generator struct{}

// New creates a new Generator.
func New() *Generator {
	return &Generator{}
}

// Name returns the tool name.
func (eg *Generator) Name() string {
	return ToolName
}

// Config returns the tool configuration for VSCode extension integration.
func (eg *Generator) Config() genkit.ToolConfig {
	return eg.config()
}

func (eg *Generator) config() genkit.ToolConfig {
	return genkit.ToolConfig{
		OutputSuffix: "_enum.go",
		Annotations: []genkit.AnnotationConfig{
			{
				Name: "enum",
				Type: "type",
				Doc: `Generate enum helper methods for type-safe enums.

USAGE:
  // enumgen:@enum(string, json, text, sql)
  type Status int

SUPPORTED OPTIONS:
  - string: Generate String() method for fmt.Stringer interface
  - json:   Generate MarshalJSON/UnmarshalJSON for encoding/json
  - text:   Generate MarshalText/UnmarshalText for encoding.TextMarshaler
  - sql:    Generate Value/Scan for database/sql driver

GENERATED HELPERS:
  - StatusEnums.List()         - Get all valid enum values
  - StatusEnums.Contains(v)    - Check if value is valid
  - StatusEnums.Parse(s)       - Parse string to enum (returns error if invalid)
  - StatusEnums.Name(v)        - Get string name of enum value
  - Status.IsValid()           - Check if the enum value is valid

EXAMPLE:
  // enumgen:@enum(string, json)
  type OrderStatus int

  const (
      OrderStatusPending   OrderStatus = iota + 1  // String: "Pending"
      OrderStatusConfirmed                          // String: "Confirmed"
      OrderStatusShipped                            // String: "Shipped"
  )

  // Usage:
  status := OrderStatusPending
  fmt.Println(status.String())           // "Pending"
  fmt.Println(status.IsValid())          // true
  fmt.Println(OrderStatusEnums.List())   // [OrderStatusPending, OrderStatusConfirmed, OrderStatusShipped]
  parsed, _ := OrderStatusEnums.Parse("Pending")  // OrderStatusPending

UNDERLYING TYPES:
  Supported: int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, string
  For string underlying type, @name annotation is not supported (value itself is the name).`,
				Params: &genkit.AnnotationParams{
					Values: []string{"string", "json", "text", "sql"},
					Docs: map[string]string{
						"string": "Generate String() method for fmt.Stringer interface",
						"json":   "Generate MarshalJSON/UnmarshalJSON for JSON serialization",
						"text":   "Generate MarshalText/UnmarshalText for text serialization",
						"sql":    "Generate Value/Scan for database/sql driver interface",
					},
				},
			},
			{
				Name: "name",
				Type: "field",
				Doc: `Custom string name for an enum value (only for non-string underlying types).

USAGE:
  const (
      // enumgen:@name(custom_name)
      StatusActive Status = iota + 1
  )

DEFAULT BEHAVIOR:
  Without @name, the string name is derived by trimming the type prefix:
    StatusActive -> "Active"
    StatusPending -> "Pending"

EXAMPLE:
  // enumgen:@enum(string, json)
  type ErrorCode int

  const (
      // enumgen:@name(ERR_NOT_FOUND)
      ErrorCodeNotFound ErrorCode = 404

      // enumgen:@name(ERR_INTERNAL)
      ErrorCodeInternal ErrorCode = 500
  )

  // Usage:
  fmt.Println(ErrorCodeNotFound.String())  // "ERR_NOT_FOUND"
  parsed, _ := ErrorCodeEnums.Parse("ERR_INTERNAL")  // ErrorCodeInternal

NOTE:
  - @name is NOT supported for string underlying types (the string value is used directly)
  - Each @name value must be unique within the enum type`,
				Params: &genkit.AnnotationParams{
					Type:        "string",
					Placeholder: "custom_name",
				},
			},
		},
	}
}

// Rules implements genkit.RuleTool.
// Returns AI-friendly documentation for enumgen.
func (eg *Generator) Rules() []genkit.Rule {
	return []genkit.Rule{
		{
			Name:        "devgen-tool-enumgen",
			Description: "Go 枚举代码生成工具 enumgen 的使用指南。当用户需要定义类型安全的枚举、生成枚举辅助方法（String、JSON、SQL等）时使用此规则。",
			Globs:       []string{"*.go"},
			AlwaysApply: false,
			Content:     rules.EnumgenRule,
		},
	}
}

// Validate implements genkit.ValidatableTool.
// It checks for errors without generating files, returning diagnostics for IDE integration.
func (eg *Generator) Validate(gen *genkit.Generator, _ *genkit.Logger) []genkit.Diagnostic {
	c := genkit.NewDiagnosticCollector(ToolName)

	for _, pkg := range gen.Packages {
		for _, enum := range pkg.Enums {
			if !genkit.HasAnnotation(enum.Doc, ToolName, "enum") {
				continue
			}
			eg.validateEnum(c, enum)
		}
	}

	return c.Collect()
}

// validateEnum validates a single enum and collects diagnostics.
func (eg *Generator) validateEnum(c *genkit.DiagnosticCollector, enum *genkit.Enum) {
	typeName := enum.Name

	// Check underlying type
	if !UnderlyingTypeEnums.Contains(enum.UnderlyingType) {
		c.Errorf(ErrCodeUnsupportedType, enum.Values[0].Pos,
			"unsupported underlying type %q, must be one of: %v",
			enum.UnderlyingType, UnderlyingTypeEnums.List())
		return // Can't continue validation without valid type
	}

	isStringType := enum.UnderlyingType == "string"

	// For string types, @name annotation is not supported
	if isStringType {
		for _, v := range enum.Values {
			if genkit.HasAnnotation(v.Doc, ToolName, "name") {
				c.Errorf(ErrCodeNameOnStringType, v.Pos,
					"@name annotation is not supported for string underlying type (on %s)", v.Name)
			}
		}
		return
	}

	// For non-string types, check for @name issues
	nameSet := make(map[string]string) // name -> value name
	for _, v := range enum.Values {
		ann := genkit.GetAnnotation(v.Doc, ToolName, "name")
		if ann != nil && len(ann.Flags) == 0 {
			c.Error(ErrCodeNameMissingParam, "@name annotation requires a name parameter", v.Pos)
			continue
		}

		name := getValueNameFromAnnotation(ann, v.Name, typeName)
		if existing, ok := nameSet[name]; ok {
			c.Errorf(ErrCodeDuplicateName, v.Pos,
				"duplicate @name %q, already used by %s", name, existing)
		}
		nameSet[name] = v.Name
	}
}

// Run processes all packages and generates enum helpers.
func (eg *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error {
	var totalCount int
	for _, pkg := range gen.Packages {
		enums := eg.FindEnums(pkg)
		if len(enums) == 0 {
			continue
		}
		log.Find("Found %v enum(s) in %v", len(enums), pkg.GoImportPath())
		for _, e := range enums {
			log.Item("%v", e.Name)
		}
		totalCount += len(enums)
		if err := eg.ProcessPackage(gen, pkg); err != nil {
			return fmt.Errorf("process %s: %w", pkg.Name, err)
		}
	}

	if totalCount == 0 {
		return nil
	}

	return nil
}

// ProcessPackage processes a package and generates enum helpers.
func (eg *Generator) ProcessPackage(gen *genkit.Generator, pkg *genkit.Package) error {
	enums := eg.FindEnums(pkg)
	if len(enums) == 0 {
		return nil
	}

	outPath := genkit.OutputPath(pkg.Dir, pkg.Name+"_enum.go")
	g := gen.NewGeneratedFile(outPath, pkg.GoImportPath())

	eg.WriteHeader(g, pkg.Name)
	for _, enum := range enums {
		if err := eg.GenerateEnum(g, enum); err != nil {
			return err
		}
	}

	return nil
}

// FindEnums finds all enums with enumgen:@enum annotation.
func (eg *Generator) FindEnums(pkg *genkit.Package) []*genkit.Enum {
	var enums []*genkit.Enum
	for _, e := range pkg.Enums {
		if genkit.HasAnnotation(e.Doc, ToolName, "enum") {
			enums = append(enums, e)
		}
	}
	return enums
}

// WriteHeader writes the file header.
func (eg *Generator) WriteHeader(g *genkit.GeneratedFile, pkgName string) {
	g.P("// Code generated by ", ToolName, ". DO NOT EDIT.")
	g.P()
	g.P("package ", pkgName)
}

// GenerateEnum generates helper code for a single enum.
func (eg *Generator) GenerateEnum(g *genkit.GeneratedFile, enum *genkit.Enum) error {
	// Validate first using collector
	c := genkit.NewDiagnosticCollector(ToolName)
	eg.validateEnum(c, enum)
	if c.HasErrors() {
		// Return first error message
		for _, d := range c.Collect() {
			if d.Severity == genkit.DiagnosticError {
				return fmt.Errorf("%s: %s", enum.Name, d.Message)
			}
		}
	}

	ann := genkit.GetAnnotation(enum.Doc, ToolName, "enum")

	// Options from annotation
	genString := ann.Has(GenerateOptionString.String())
	genJSON := ann.Has(GenerateOptionJSON.String())
	genText := ann.Has(GenerateOptionText.String())
	genSQL := ann.Has(GenerateOptionSQL.String())

	typeName := enum.Name
	enumsType := "_" + typeName + "Enums" // e.g., _StatusEnums
	enumsVar := typeName + "Enums"        // e.g., StatusEnums

	// Check if underlying type is string
	isStringType := enum.UnderlyingType == "string"

	// 1. Enum type methods (IsValid, String, MarshalJSON, etc.) - at the top

	// IsValid is always generated
	g.P()
	g.P(genkit.GoMethod{
		Doc:     genkit.GoDoc("IsValid reports whether x is a valid " + typeName + "."),
		Recv:    genkit.GoReceiver{Name: "x", Type: typeName},
		Name:    "IsValid",
		Results: genkit.GoResults{{Type: "bool"}},
	}, " {")
	if isStringType {
		g.P("return ", enumsVar, ".Contains(string(x))")
	} else {
		g.P("return ", enumsVar, ".Contains(x)")
	}
	g.P("}")

	if genString {
		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("String returns the string representation of " + typeName + "."),
			Recv:    genkit.GoReceiver{Name: "x", Type: typeName},
			Name:    "String",
			Results: genkit.GoResults{{Type: "string"}},
		}, " {")
		if isStringType {
			g.P("return string(x)")
		} else {
			g.P("return ", enumsVar, ".Name(x)")
		}
		g.P("}")
	}

	if genJSON {
		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("MarshalJSON implements json.Marshaler."),
			Recv:    genkit.GoReceiver{Name: "x", Type: typeName},
			Name:    "MarshalJSON",
			Results: genkit.GoResults{{Type: "[]byte"}, {Type: "error"}},
		}, " {")
		if isStringType {
			g.P("return ", genkit.GoImportPath("encoding/json").Ident("Marshal"), "(string(x))")
		} else {
			g.P("return ", genkit.GoImportPath("encoding/json").Ident("Marshal"), "(", enumsVar, ".Name(x))")
		}
		g.P("}")

		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("UnmarshalJSON implements json.Unmarshaler."),
			Recv:    genkit.GoReceiver{Name: "x", Type: "*" + typeName},
			Name:    "UnmarshalJSON",
			Params:  genkit.GoParams{List: []genkit.GoParam{{Name: "data", Type: "[]byte"}}},
			Results: genkit.GoResults{{Type: "error"}},
		}, " {")
		g.P("var s string")
		g.P("if err := ", genkit.GoImportPath("encoding/json").Ident("Unmarshal"), "(data, &s); err != nil {")
		g.P("return err")
		g.P("}")
		g.P("v, err := ", enumsVar, ".Parse(s)")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("*x = v")
		g.P("return nil")
		g.P("}")
	}

	if genText {
		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("MarshalText implements encoding.TextMarshaler."),
			Recv:    genkit.GoReceiver{Name: "x", Type: typeName},
			Name:    "MarshalText",
			Results: genkit.GoResults{{Type: "[]byte"}, {Type: "error"}},
		}, " {")
		if isStringType {
			g.P("return []byte(x), nil")
		} else {
			g.P("return []byte(", enumsVar, ".Name(x)), nil")
		}
		g.P("}")

		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("UnmarshalText implements encoding.TextUnmarshaler."),
			Recv:    genkit.GoReceiver{Name: "x", Type: "*" + typeName},
			Name:    "UnmarshalText",
			Params:  genkit.GoParams{List: []genkit.GoParam{{Name: "data", Type: "[]byte"}}},
			Results: genkit.GoResults{{Type: "error"}},
		}, " {")
		g.P("v, err := ", enumsVar, ".Parse(string(data))")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("*x = v")
		g.P("return nil")
		g.P("}")
	}

	if genSQL {
		g.P()
		g.P(genkit.GoMethod{
			Doc:  genkit.GoDoc("Value implements driver.Valuer."),
			Recv: genkit.GoReceiver{Name: "x", Type: typeName},
			Name: "Value",
			Results: genkit.GoResults{
				{Type: genkit.GoImportPath("database/sql/driver").Ident("Value")},
				{Type: "error"},
			},
		}, " {")
		if isStringType {
			g.P("return string(x), nil")
		} else {
			g.P("return ", enumsVar, ".Name(x), nil")
		}
		g.P("}")

		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("Scan implements sql.Scanner."),
			Recv:    genkit.GoReceiver{Name: "x", Type: "*" + typeName},
			Name:    "Scan",
			Params:  genkit.GoParams{List: []genkit.GoParam{{Name: "src", Type: "any"}}},
			Results: genkit.GoResults{{Type: "error"}},
		}, " {")
		g.P("if src == nil {")
		g.P("return nil")
		g.P("}")
		g.P("var s string")
		g.P("switch v := src.(type) {")
		g.P("case string:")
		g.P("s = v")
		g.P("case []byte:")
		g.P("s = string(v)")
		g.P("default:")
		g.P("return ", genkit.GoImportPath("fmt").Ident("Errorf"), "(\"cannot scan %T into ", typeName, "\", src)")
		g.P("}")
		g.P("v, err := ", enumsVar, ".Parse(s)")
		g.P("if err != nil {")
		g.P("return err")
		g.P("}")
		g.P("*x = v")
		g.P("return nil")
		g.P("}")
	}

	// 2. Global variable XxxEnums
	g.P()
	g.P("// ", enumsVar, " is the enum helper for ", typeName, ".")
	g.P("var ", enumsVar, " = ", enumsType, "{")

	// values slice
	g.P("values: []", typeName, "{")
	for _, v := range enum.Values {
		g.P(v.Name, ",")
	}
	g.P("},")

	if isStringType {
		// For string type, use a set for fast lookup
		g.P("set: map[", typeName, "]struct{}{")
		for _, v := range enum.Values {
			g.P(v.Name, ": {},")
		}
		g.P("},")
	} else {
		// names map (only for non-string types)
		g.P("names: map[", typeName, "]string{")
		for _, v := range enum.Values {
			name := GetValueName(v, typeName)
			g.P(v.Name, ": ", fmt.Sprintf("%q", name), ",")
		}
		g.P("},")

		// byName map (case-sensitive, only for non-string types)
		g.P("byName: map[string]", typeName, "{")
		for _, v := range enum.Values {
			name := GetValueName(v, typeName)
			g.P(fmt.Sprintf("%q", name), ": ", v.Name, ",")
		}
		g.P("},")
	}
	g.P("}")

	// 3. _XxxEnums type definition
	g.P()
	g.P("// ", enumsType, " provides enum metadata and validation for ", typeName, ".")
	g.P("type ", enumsType, " struct {")
	g.P("values []", typeName)
	if isStringType {
		g.P("set map[", typeName, "]struct{}")
	} else {
		g.P("names  map[", typeName, "]string")
		g.P("byName map[string]", typeName)
	}
	g.P("}")

	// 4. _XxxEnums methods
	g.P()
	g.P(genkit.GoMethod{
		Doc:     genkit.GoDoc("List returns all valid " + typeName + " values."),
		Recv:    genkit.GoReceiver{Name: "e", Type: enumsType},
		Name:    "List",
		Results: genkit.GoResults{{Type: "[]" + typeName}},
	}, " {")
	g.P("return e.values")
	g.P("}")

	g.P()
	if isStringType {
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("Contains reports whether v is a valid " + typeName + "."),
			Recv:    genkit.GoReceiver{Name: "e", Type: enumsType},
			Name:    "Contains",
			Params:  genkit.GoParams{List: []genkit.GoParam{{Name: "v", Type: "string"}}},
			Results: genkit.GoResults{{Type: "bool"}},
		}, " {")
		g.P("_, ok := e.set[", typeName, "(v)]")
	} else {
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("Contains reports whether v is a valid " + typeName + "."),
			Recv:    genkit.GoReceiver{Name: "e", Type: enumsType},
			Name:    "Contains",
			Params:  genkit.GoParams{List: []genkit.GoParam{{Name: "v", Type: typeName}}},
			Results: genkit.GoResults{{Type: "bool"}},
		}, " {")
		g.P("_, ok := e.names[v]")
	}
	g.P("return ok")
	g.P("}")

	g.P()
	g.P(genkit.GoMethod{
		Doc:     genkit.GoDoc("Parse parses a string into " + typeName + "."),
		Recv:    genkit.GoReceiver{Name: "e", Type: enumsType},
		Name:    "Parse",
		Params:  genkit.GoParams{List: []genkit.GoParam{{Name: "s", Type: "string"}}},
		Results: genkit.GoResults{{Type: typeName}, {Type: "error"}},
	}, " {")
	if isStringType {
		g.P("v := ", typeName, "(s)")
		g.P("if _, ok := e.set[v]; ok {")
		g.P("return v, nil")
		g.P("}")
		g.P("return \"\", ", genkit.GoImportPath("fmt").Ident("Errorf"), "(\"invalid ", typeName, ": %q\", s)")
	} else {
		g.P("if v, ok := e.byName[s]; ok {")
		g.P("return v, nil")
		g.P("}")
		g.P("return 0, ", genkit.GoImportPath("fmt").Ident("Errorf"), "(\"invalid ", typeName, ": %q\", s)")
	}
	g.P("}")

	// Only generate Name/Names/ContainsName for non-string types
	if !isStringType {
		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("ContainsName reports whether name is a valid " + typeName + " name."),
			Recv:    genkit.GoReceiver{Name: "e", Type: enumsType},
			Name:    "ContainsName",
			Params:  genkit.GoParams{List: []genkit.GoParam{{Name: "name", Type: "string"}}},
			Results: genkit.GoResults{{Type: "bool"}},
		}, " {")
		g.P("_, ok := e.byName[name]")
		g.P("return ok")
		g.P("}")

		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("Name returns the string name of v."),
			Recv:    genkit.GoReceiver{Name: "e", Type: enumsType},
			Name:    "Name",
			Params:  genkit.GoParams{List: []genkit.GoParam{{Name: "v", Type: typeName}}},
			Results: genkit.GoResults{{Type: "string"}},
		}, " {")
		g.P("if name, ok := e.names[v]; ok {")
		g.P("return name")
		g.P("}")
		g.P("return ", genkit.GoImportPath("fmt").Ident("Sprintf"), "(\"", typeName, "(%d)\", v)")
		g.P("}")

		g.P()
		g.P(genkit.GoMethod{
			Doc:     genkit.GoDoc("Names returns all valid " + typeName + " names."),
			Recv:    genkit.GoReceiver{Name: "e", Type: enumsType},
			Name:    "Names",
			Results: genkit.GoResults{{Type: "[]string"}},
		}, " {")
		g.P("names := make([]string, len(e.values))")
		g.P("for i, v := range e.values {")
		g.P("names[i] = e.names[v]")
		g.P("}")
		g.P("return names")
		g.P("}")
	}

	return nil
}

// TrimPrefix removes the type name prefix from an enum value name.
// Unlike strings.TrimPrefix, it returns the original name if the result would be empty.
// Example: TrimPrefix("StatusActive", "Status") -> "Active"
//
//	TrimPrefix("Status", "Status") -> "Status" (not "")
func TrimPrefix(name, prefix string) string {
	if s, found := strings.CutPrefix(name, prefix); found && s != "" {
		return s
	}
	return name
}

// GetValueName returns the display name for an enum value (exported for testing).
// It checks for enumgen:@name annotation first, otherwise uses TrimPrefix.
func GetValueName(v *genkit.EnumValue, typeName string) string {
	ann := genkit.GetAnnotation(v.Doc, ToolName, "name")
	return getValueNameFromAnnotation(ann, v.Name, typeName)
}

// getValueNameFromAnnotation extracts the name from annotation or falls back to TrimPrefix.
// This is the core logic shared by validation and generation.
func getValueNameFromAnnotation(ann *genkit.Annotation, valueName, typeName string) string {
	if ann != nil && len(ann.Flags) > 0 {
		return ann.Flags[0]
	}
	return TrimPrefix(valueName, typeName)
}
