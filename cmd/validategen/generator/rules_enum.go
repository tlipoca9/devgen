// Package generator provides validation code generation functionality.
package generator

import (
	"fmt"
	"strings"

	"github.com/tlipoca9/devgen/genkit"
)

func init() {
	DefaultRegistry.Register("oneof", PriorityEquality+2, func() Rule { return &OneofRule{} })
	DefaultRegistry.Register("oneof_enum", PriorityEquality+3, func() Rule { return &OneofEnumRule{} })
}

// OneofRule validates field is one of specified values.
type OneofRule struct{}

func (r *OneofRule) Name() string              { return "oneof" }
func (r *OneofRule) RequiredRegex() []string   { return nil }

func (r *OneofRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}

	cleanValues := splitAndClean(ctx.Param)
	if len(cleanValues) == 0 {
		return
	}

	fieldName := ctx.FieldName
	fieldType := ctx.FieldType
	fmtSprintf := fmtSprintf()
	g := ctx.G

	if IsStringType(fieldType) {
		var quoted []string
		for _, v := range cleanValues {
			quoted = append(quoted, fmt.Sprintf("%q", v))
		}
		g.P("if !func() bool {")
		g.P("for _, v := range []string{", strings.Join(quoted, ", "), "} {")
		g.P("if x.", fieldName, " == v { return true }")
		g.P("}")
		g.P("return false")
		g.P("}() {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be one of [", strings.Join(cleanValues, ", "), "], got %q\", x.", fieldName, "))")
		g.P("}")
	} else {
		g.P("if !func() bool {")
		g.P("for _, v := range []", fieldType, "{", strings.Join(cleanValues, ", "), "} {")
		g.P("if x.", fieldName, " == v { return true }")
		g.P("}")
		g.P("return false")
		g.P("}() {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be one of [", strings.Join(cleanValues, ", "), "], got %v\", x.", fieldName, "))")
		g.P("}")
	}
}

func (r *OneofRule) Validate(ctx *ValidateContext) {
	underlyingType := ctx.UnderlyingType
	if !IsStringType(underlyingType) && !IsNumericType(underlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@oneof annotation requires string or numeric underlying type, got %s",
			underlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeOneofMissingValues, ctx.Field.Pos, "@oneof annotation requires at least one value")
		return
	}

	cleanValues := splitAndClean(ctx.Param)
	if len(cleanValues) == 0 {
		ctx.Collector.Errorf(ErrCodeOneofMissingValues, ctx.Field.Pos, "@oneof annotation requires at least one value")
		return
	}

	// Validate numeric values for numeric types
	if IsNumericType(underlyingType) {
		for _, v := range cleanValues {
			if !isValidNumber(v) {
				ctx.Collector.Errorf(
					ErrCodeInvalidOneofValue,
					ctx.Field.Pos,
					"@oneof value %q is not a valid number for numeric field type %s",
					v,
					underlyingType,
				)
			}
		}
	}
}

// OneofEnumRule validates field is a valid enum value.
type OneofEnumRule struct{}

func (r *OneofEnumRule) Name() string              { return "oneof_enum" }
func (r *OneofEnumRule) RequiredRegex() []string   { return nil }

func (r *OneofEnumRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}

	fieldName := ctx.FieldName
	enumType := strings.TrimSpace(ctx.Param)
	fmtSprintf := fmtSprintf()
	g := ctx.G
	pkg := ctx.Pkg

	var enumsVar string
	var enumValues []string
	var isStringEnum bool

	// Check for alias format
	var importAlias string
	if colonIdx := strings.Index(enumType, ":"); colonIdx != -1 {
		importAlias = enumType[:colonIdx]
		enumType = enumType[colonIdx+1:]
	}

	if lastDot := strings.LastIndex(enumType, "."); lastDot != -1 {
		// Cross-package enum
		beforeDot := enumType[:lastDot]
		typeName := enumType[lastDot+1:]

		importPath := genkit.GoImportPath(beforeDot)

		var pkgName string
		if importAlias != "" {
			g.ImportAs(importPath, genkit.GoPackageName(importAlias))
			pkgName = importAlias
		} else {
			pkgName = string(g.Import(importPath))
		}

		enumsVar = pkgName + "." + typeName + "Enums"

		if enum := ctx.Generator.FindEnum(importPath, typeName); enum != nil {
			isStringEnum = IsStringType(enum.UnderlyingType)
			for _, v := range enum.Values {
				enumValues = append(enumValues, pkgName+"."+v.Name)
			}
		}
	} else {
		// Same package enum
		enumsVar = enumType + "Enums"

		for _, e := range pkg.Enums {
			if e.Name == enumType {
				isStringEnum = IsStringType(e.UnderlyingType)
				for _, v := range e.Values {
					enumValues = append(enumValues, v.Name)
				}
				break
			}
		}
	}

	// Generate comment with enum values
	if len(enumValues) > 0 {
		g.P("// Valid values:")
		for _, v := range enumValues {
			g.P("//   - ", v)
		}
	}

	// Generate validation code
	if isStringEnum {
		fieldValue := "x." + fieldName
		if ctx.Field.Type != "string" && IsStringType(ctx.Field.UnderlyingType) {
			fieldValue = "string(x." + fieldName + ")"
		}
		g.P("if x.", fieldName, " != \"\" && !", enumsVar, ".Contains(", fieldValue, ") {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be one of %v, got %v\", ", enumsVar, ".List(), x.", fieldName, "))")
		g.P("}")
	} else {
		g.P("if !", enumsVar, ".ContainsName(", fmtSprintf, "(\"%v\", x.", fieldName, ")) {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be one of %v, got %v\", ", enumsVar, ".Names(), x.", fieldName, "))")
		g.P("}")
	}
}

func (r *OneofEnumRule) Validate(ctx *ValidateContext) {
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@oneof_enum annotation requires an enum type parameter")
		return
	}

	enumType := strings.TrimSpace(ctx.Param)

	// Strip alias prefix
	if colonIdx := strings.Index(enumType, ":"); colonIdx != -1 {
		enumType = enumType[colonIdx+1:]
	}

	// Check if cross-package enum
	if lastDot := strings.LastIndex(enumType, "."); lastDot != -1 && strings.Contains(enumType[:lastDot], "/") {
		importPath := genkit.GoImportPath(enumType[:lastDot])
		typeName := enumType[lastDot+1:]

		enum := ctx.Generator.FindEnum(importPath, typeName)
		if enum != nil {
			if ctx.Field.Type != typeName && !IsStringType(ctx.UnderlyingType) && ctx.UnderlyingType != enum.UnderlyingType {
				ctx.Collector.Errorf(
					ErrCodeInvalidFieldType,
					ctx.Field.Pos,
					"@oneof_enum(%s) requires field underlying type to be %s or string, got %s",
					ctx.Param,
					enum.UnderlyingType,
					ctx.UnderlyingType,
				)
			}
		}
	} else {
		// Same package enum
		var enum *genkit.Enum
		for _, e := range ctx.Pkg.Enums {
			if e.Name == enumType {
				enum = e
				break
			}
		}

		if enum == nil {
			ctx.Collector.Errorf(
				ErrCodeInvalidFieldType,
				ctx.Field.Pos,
				"@oneof_enum(%s): enum type %s not found in current package",
				enumType,
				enumType,
			)
			return
		}

		if ctx.Field.Type != enumType && !IsStringType(ctx.UnderlyingType) && ctx.UnderlyingType != enum.UnderlyingType {
			ctx.Collector.Errorf(
				ErrCodeInvalidFieldType,
				ctx.Field.Pos,
				"@oneof_enum(%s) requires field type to be %s or have underlying type %s, got %s (underlying: %s)",
				enumType,
				enumType,
				enum.UnderlyingType,
				ctx.Field.Type,
				ctx.UnderlyingType,
			)
		}
	}
}
