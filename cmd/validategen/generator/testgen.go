// Package generator provides validation code generation functionality.
package generator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tlipoca9/devgen/genkit"
)

// GenerateValidateTest generates table-driven tests for a single type's _validate method.
func (vg *Generator) GenerateValidateTest(g *genkit.GeneratedFile, typ *genkit.Type, pkg *genkit.Package) {
	typeName := typ.Name

	var validatedFields []*FieldValidation
	for _, field := range typ.Fields {
		vrules := vg.parseFieldAnnotations(field)
		var nonMethodRules []*ValidateRule
		for _, rule := range vrules {
			if rule.Name != "method" {
				nonMethodRules = append(nonMethodRules, rule)
			}
		}
		if len(nonMethodRules) > 0 {
			validatedFields = append(validatedFields, &FieldValidation{
				Field: field,
				Rules: nonMethodRules,
			})
		}
	}

	if len(validatedFields) == 0 {
		return
	}

	for _, fv := range validatedFields {
		EnsureTypeImport(g, fv.Field.Type, typ.Pkg)
	}

	methodName := "_validate"
	testFuncName := "Test" + typeName + "__validate"

	g.P()
	g.P("func ", testFuncName, "(t *", genkit.GoImportPath("testing").Ident("T"), ") {")
	g.P("tests := []struct {")
	g.P("name    string")
	g.P("input   ", typeName)
	g.P("wantErr bool")
	g.P("}{")

	// Generate valid case
	g.P("{")
	g.P("name: \"valid\",")
	g.P("input: ", typeName, "{")
	for _, fv := range validatedFields {
		vg.generateValidFieldValue(g, fv, pkg)
	}
	g.P("},")
	g.P("wantErr: false,")
	g.P("},")

	// Generate invalid cases
	for _, fv := range validatedFields {
		vg.generateInvalidTestCases(g, typeName, fv, validatedFields, pkg)
	}

	g.P("}")
	g.P("for _, tt := range tests {")
	g.P("t.Run(tt.name, func(t *testing.T) {")
	g.P("errs := tt.input.", methodName, "()")
	g.P("hasErr := len(errs) > 0")
	g.P("if hasErr != tt.wantErr {")
	g.P("t.Errorf(\"", methodName, "() errors = %v, wantErr %v\", errs, tt.wantErr)")
	g.P("}")
	g.P("})")
	g.P("}")
	g.P("}")
}

// GenerateSetDefaultsTest generates table-driven tests for SetDefaults method.
func (vg *Generator) GenerateSetDefaultsTest(g *genkit.GeneratedFile, typ *genkit.Type) {
	typeName := typ.Name

	var defaultFields []*FieldValidation
	for _, field := range typ.Fields {
		vrules := vg.parseFieldAnnotations(field)
		for _, rule := range vrules {
			if rule.Name == "default" {
				defaultFields = append(defaultFields, &FieldValidation{
					Field: field,
					Rules: []*ValidateRule{rule},
				})
				break
			}
		}
	}

	if len(defaultFields) == 0 {
		return
	}

	testFuncName := "Test" + typeName + "_SetDefaults"

	g.P()
	g.P("func ", testFuncName, "(t *", genkit.GoImportPath("testing").Ident("T"), ") {")
	g.P("tests := []struct {")
	g.P("name   string")
	g.P("input  ", typeName)
	g.P("expect ", typeName)
	g.P("}{")

	g.P("{")
	g.P("name: \"all_zero_values\",")
	g.P("input: ", typeName, "{},")
	g.P("expect: ", typeName, "{")
	for _, fv := range defaultFields {
		vg.generateDefaultExpectValue(g, fv)
	}
	g.P("},")
	g.P("},")

	g.P("{")
	g.P("name: \"non_zero_values_preserved\",")
	g.P("input: ", typeName, "{")
	for _, fv := range defaultFields {
		vg.generateNonZeroValue(g, fv)
	}
	g.P("},")
	g.P("expect: ", typeName, "{")
	for _, fv := range defaultFields {
		vg.generateNonZeroValue(g, fv)
	}
	g.P("},")
	g.P("},")

	g.P("}")

	g.P("for _, tt := range tests {")
	g.P("t.Run(tt.name, func(t *testing.T) {")
	g.P("got := tt.input")
	g.P("got.SetDefaults()")

	for _, fv := range defaultFields {
		fieldName := fv.Field.Name
		fieldType := fv.Field.Type
		if IsPointerType(fieldType) {
			g.P("if got.", fieldName, " == nil || tt.expect.", fieldName, " == nil {")
			g.P("if got.", fieldName, " != tt.expect.", fieldName, " {")
			g.P("t.Errorf(\"", fieldName, " = %v, want %v\", got.", fieldName, ", tt.expect.", fieldName, ")")
			g.P("}")
			g.P("} else if *got.", fieldName, " != *tt.expect.", fieldName, " {")
			g.P("t.Errorf(\"", fieldName, " = %v, want %v\", *got.", fieldName, ", *tt.expect.", fieldName, ")")
			g.P("}")
		} else {
			g.P("if got.", fieldName, " != tt.expect.", fieldName, " {")
			g.P("t.Errorf(\"", fieldName, " = %v, want %v\", got.", fieldName, ", tt.expect.", fieldName, ")")
			g.P("}")
		}
	}

	g.P("})")
	g.P("}")
	g.P("}")
}

func (vg *Generator) generateDefaultExpectValue(g *genkit.GeneratedFile, fv *FieldValidation) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type
	param := fv.Rules[0].Param

	if IsStringType(fieldType) {
		g.P(fieldName, ": \"", param, "\",")
	} else if IsPointerToStringType(fieldType) {
		g.P(fieldName, ": func() *string { v := \"", param, "\"; return &v }(),")
	} else if IsBoolType(fieldType) {
		boolVal := param == "true" || param == "1"
		g.P(fieldName, ": ", boolVal, ",")
	} else if IsPointerToBoolType(fieldType) {
		boolVal := param == "true" || param == "1"
		g.P(fieldName, ": func() *bool { v := ", boolVal, "; return &v }(),")
	} else if IsNumericType(fieldType) {
		g.P(fieldName, ": ", param, ",")
	} else if IsPointerToNumericType(fieldType) {
		baseType := strings.TrimPrefix(fieldType, "*")
		g.P(fieldName, ": func() *", baseType, " { v := ", baseType, "(", param, "); return &v }(),")
	}
}

func (vg *Generator) generateNonZeroValue(g *genkit.GeneratedFile, fv *FieldValidation) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type

	if IsStringType(fieldType) {
		g.P(fieldName, ": \"custom\",")
	} else if IsPointerToStringType(fieldType) {
		g.P(fieldName, ": func() *string { v := \"custom\"; return &v }(),")
	} else if IsBoolType(fieldType) {
		g.P(fieldName, ": true,")
	} else if IsPointerToBoolType(fieldType) {
		g.P(fieldName, ": func() *bool { v := true; return &v }(),")
	} else if IsNumericType(fieldType) {
		g.P(fieldName, ": 999,")
	} else if IsPointerToNumericType(fieldType) {
		baseType := strings.TrimPrefix(fieldType, "*")
		g.P(fieldName, ": func() *", baseType, " { v := ", baseType, "(999); return &v }(),")
	}
}

func (vg *Generator) generateValidFieldValue(g *genkit.GeneratedFile, fv *FieldValidation, pkg *genkit.Package) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type
	underlyingType := fv.Field.UnderlyingType

	var hasRequired, hasMin, hasLen bool
	var hasEmail, hasURL, hasUUID, hasIP, hasIPv4, hasIPv6, hasDuration bool
	var hasAlpha, hasAlphanum, hasNumeric, hasOneof, hasOneofEnum bool
	var hasContains, hasStartsWith, hasEndsWith, hasRegex bool
	var hasEq, hasNe, hasFormat, hasDNS1123 bool
	var hasDurationMin, hasDurationMax bool
	var hasCPU, hasMemory, hasDisk bool
	var hasGt, hasGte, hasLt, hasLte bool
	var minVal, lenVal string
	var oneofValues, containsVal, startsWithVal, endsWithVal string
	var eqVal, neVal, regexVal, formatVal string
	var durationMinVal, durationMaxVal string
	var oneofEnumParam string
	var gtVal, gteVal, ltVal, lteVal string

	for _, rule := range fv.Rules {
		switch rule.Name {
		case "required":
			hasRequired = true
		case "min":
			hasMin = true
			minVal = rule.Param
		case "len":
			hasLen = true
			lenVal = rule.Param
		case "gt":
			hasGt = true
			gtVal = rule.Param
		case "gte":
			hasGte = true
			gteVal = rule.Param
		case "lt":
			hasLt = true
			ltVal = rule.Param
		case "lte":
			hasLte = true
			lteVal = rule.Param
		case "email":
			hasEmail = true
		case "url":
			hasURL = true
		case "uuid":
			hasUUID = true
		case "ip":
			hasIP = true
		case "ipv4":
			hasIPv4 = true
		case "ipv6":
			hasIPv6 = true
		case "duration":
			hasDuration = true
		case "duration_min":
			hasDurationMin = true
			durationMinVal = rule.Param
		case "duration_max":
			hasDurationMax = true
			durationMaxVal = rule.Param
		case "dns1123_label":
			hasDNS1123 = true
		case "alpha":
			hasAlpha = true
		case "alphanum":
			hasAlphanum = true
		case "numeric":
			hasNumeric = true
		case "oneof":
			hasOneof = true
			oneofValues = rule.Param
		case "oneof_enum":
			hasOneofEnum = true
			oneofEnumParam = rule.Param
		case "contains":
			hasContains = true
			containsVal = rule.Param
		case "startswith":
			hasStartsWith = true
			startsWithVal = rule.Param
		case "endswith":
			hasEndsWith = true
			endsWithVal = rule.Param
		case "regex":
			hasRegex = true
			regexVal = rule.Param
		case "eq":
			hasEq = true
			eqVal = rule.Param
		case "ne":
			hasNe = true
			neVal = rule.Param
		case "format":
			hasFormat = true
			formatVal = rule.Param
		case "cpu":
			hasCPU = true
		case "memory":
			hasMemory = true
		case "disk":
			hasDisk = true
		}
	}

	if IsStringType(underlyingType) {
		var value string
		if hasOneofEnum {
			vg.generateEnumTestValue(g, fieldName, fieldType, oneofEnumParam, 1, pkg)
			return
		} else if hasEmail {
			value = "test@example.com"
		} else if hasURL {
			value = "https://example.com"
		} else if hasUUID {
			value = "550e8400-e29b-41d4-a716-446655440000"
		} else if hasIP || hasIPv4 {
			value = "192.168.1.1"
		} else if hasIPv6 {
			value = "::1"
		} else if hasDuration || hasDurationMin || hasDurationMax {
			if hasDurationMin {
				value = durationMinVal
			} else if hasDurationMax {
				value = durationMaxVal
			} else {
				value = "1h"
			}
		} else if hasCPU {
			value = "100m"
		} else if hasMemory {
			value = "128Mi"
		} else if hasDisk {
			value = "10Gi"
		} else if hasDNS1123 {
			value = "example-name"
		} else if hasAlpha {
			value = "abcdef"
		} else if hasAlphanum {
			value = "abc123"
		} else if hasNumeric {
			value = "123456"
		} else if hasOneof {
			parts := strings.Split(oneofValues, " ")
			if len(parts) > 0 {
				value = strings.TrimSpace(parts[0])
			}
		} else if hasRegex {
			value = vg.generateValidRegexValue(regexVal)
		} else if hasFormat {
			value = vg.generateValidFormatValue(formatVal)
		} else if hasEq {
			value = eqVal
		} else if hasStartsWith && hasEndsWith {
			value = startsWithVal + "middle" + endsWithVal
		} else if hasStartsWith {
			value = startsWithVal + "test"
		} else if hasEndsWith {
			value = "test" + endsWithVal
		} else if hasContains {
			value = "test" + containsVal + "test"
		} else if hasLen {
			n, _ := strconv.Atoi(lenVal)
			value = strings.Repeat("a", n)
		} else if hasMin {
			n, _ := strconv.Atoi(minVal)
			value = strings.Repeat("a", n)
		} else if hasRequired {
			value = "test"
		} else {
			value = "test"
		}
		g.P(fieldName, ": \"", value, "\",")
	} else if IsNumericType(underlyingType) {
		var value string
		if hasOneofEnum {
			vg.generateEnumTestValue(g, fieldName, fieldType, oneofEnumParam, 1, pkg)
			return
		} else if hasOneof {
			parts := strings.Split(oneofValues, " ")
			if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
				value = strings.TrimSpace(parts[0])
			} else {
				value = "1"
			}
		} else if hasEq {
			value = eqVal
		} else if hasGt {
			n, _ := strconv.ParseFloat(gtVal, 64)
			value = fmt.Sprintf("%v", int(n)+1)
		} else if hasGte {
			value = gteVal
		} else if hasLt {
			n, _ := strconv.ParseFloat(ltVal, 64)
			value = fmt.Sprintf("%v", int(n)-1)
		} else if hasLte {
			value = lteVal
		} else if hasMin {
			value = minVal
		} else if hasNe {
			value = "1"
		} else if hasRequired {
			value = "1"
		} else {
			value = "1"
		}
		g.P(fieldName, ": ", value, ",")
	} else if IsBoolType(underlyingType) {
		if hasEq {
			g.P(fieldName, ": ", eqVal, ",")
		} else if hasNe {
			if neVal == "true" {
				g.P(fieldName, ": false,")
			} else {
				g.P(fieldName, ": true,")
			}
		} else if hasRequired {
			g.P(fieldName, ": true,")
		} else {
			g.P(fieldName, ": true,")
		}
	} else if IsSliceType(fieldType) {
		if hasLen {
			n, _ := strconv.Atoi(lenVal)
			g.P(fieldName, ": make(", fieldType, ", ", n, "),")
		} else if hasMin {
			n, _ := strconv.Atoi(minVal)
			g.P(fieldName, ": make(", fieldType, ", ", n, "),")
		} else if hasRequired {
			g.P(fieldName, ": make(", fieldType, ", 1),")
		}
	} else if IsMapType(fieldType) {
		if hasRequired {
			g.P(fieldName, ": ", fieldType, "{\"key\": 0},")
		} else {
			g.P(fieldName, ": make(", fieldType, "),")
		}
	} else if IsPointerType(fieldType) {
		if hasRequired {
			elemType := strings.TrimPrefix(fieldType, "*")
			g.P(fieldName, ": &", elemType, "{},")
		}
	} else if hasOneofEnum {
		vg.generateEnumTestValue(g, fieldName, fieldType, oneofEnumParam, 1, pkg)
	}
}

func (vg *Generator) generateEnumTestValue(
	g *genkit.GeneratedFile,
	fieldName, fieldType, enumParam string,
	value int,
	pkg *genkit.Package,
) {
	enumType := strings.TrimSpace(enumParam)

	var importAlias string
	if colonIdx := strings.Index(enumType, ":"); colonIdx != -1 {
		importAlias = enumType[:colonIdx]
		enumType = enumType[colonIdx+1:]
	}

	isBasicFieldType := IsStringType(fieldType) || IsNumericType(fieldType)

	if lastDot := strings.LastIndex(enumType, "."); lastDot != -1 && strings.Contains(enumType[:lastDot], "/") {
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

		enum := vg.FindEnum(importPath, typeName)

		if isBasicFieldType {
			if IsStringType(fieldType) {
				if value > 0 && value < 100 && enum != nil && len(enum.Values) > 0 {
					g.P(fieldName, ": ", pkgName, ".", enum.Values[0].Name, ".String(),")
				} else {
					g.P(fieldName, ": \"__invalid__\",")
				}
			} else {
				if value > 0 && value < 100 && enum != nil && len(enum.Values) > 0 {
					g.P(fieldName, ": int(", pkgName, ".", enum.Values[0].Name, "),")
				} else {
					g.P(fieldName, ": ", value, ",")
				}
			}
		} else {
			if enum != nil && len(enum.Values) > 0 && value > 0 && value < 100 {
				g.P(fieldName, ": ", pkgName, ".", enum.Values[0].Name, ",")
			} else {
				if enum != nil && IsStringType(enum.UnderlyingType) {
					g.P(fieldName, ": ", pkgName, ".", typeName, "(\"__invalid__\"),")
				} else {
					g.P(fieldName, ": ", pkgName, ".", typeName, "(", value, "),")
				}
			}
		}
	} else {
		var enum *genkit.Enum
		for _, e := range pkg.Enums {
			if e.Name == enumType {
				enum = e
				break
			}
		}

		if isBasicFieldType {
			if IsStringType(fieldType) {
				if value > 0 && value < 100 && enum != nil && len(enum.Values) > 0 {
					g.P(fieldName, ": ", enum.Values[0].Name, ".String(),")
				} else {
					g.P(fieldName, ": \"__invalid__\",")
				}
			} else {
				if value > 0 && value < 100 && enum != nil && len(enum.Values) > 0 {
					g.P(fieldName, ": int(", enum.Values[0].Name, "),")
				} else {
					g.P(fieldName, ": ", value, ",")
				}
			}
		} else {
			if enum != nil && len(enum.Values) > 0 && value > 0 && value < 100 {
				g.P(fieldName, ": ", enum.Values[0].Name, ",")
			} else {
				if enum != nil && IsStringType(enum.UnderlyingType) {
					g.P(fieldName, ": ", fieldType, "(\"__invalid__\"),")
				} else {
					g.P(fieldName, ": ", fieldType, "(", value, "),")
				}
			}
		}
	}
}

func (vg *Generator) generateValidRegexValue(pattern string) string {
	switch {
	case strings.Contains(pattern, "[A-Z]") && strings.Contains(pattern, "\\d"):
		return "AB-1234"
	case strings.Contains(pattern, "[a-z]") && strings.Contains(pattern, "[0-9]"):
		return "abc123"
	case strings.Contains(pattern, "[0-9]") || strings.Contains(pattern, "\\d"):
		return "12345"
	case strings.Contains(pattern, "[a-zA-Z]"):
		return "abcdef"
	default:
		return "test123"
	}
}

func (vg *Generator) generateValidFormatValue(format string) string {
	switch strings.ToLower(format) {
	case "json":
		return `{\"key\":\"value\"}`
	case "yaml":
		return "key: value"
	case "toml":
		return "key = \\\"value\\\""
	case "csv":
		return "a,b,c"
	default:
		return "test"
	}
}

func (vg *Generator) generateInvalidTestCases(
	g *genkit.GeneratedFile,
	typeName string,
	fv *FieldValidation,
	allFields []*FieldValidation,
	pkg *genkit.Package,
) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type
	underlyingType := fv.Field.UnderlyingType

	for _, rule := range fv.Rules {
		switch rule.Name {
		case "required":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_required\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "min":
			if IsStringType(underlyingType) {
				n, _ := strconv.Atoi(rule.Param)
				if n > 0 {
					g.P("{")
					g.P("name: \"invalid_", fieldName, "_min\",")
					g.P("input: ", typeName, "{")
					for _, otherFv := range allFields {
						if otherFv.Field.Name != fieldName {
							vg.generateValidFieldValue(g, otherFv, pkg)
						} else {
							g.P(fieldName, ": \"", strings.Repeat("a", n-1), "\",")
						}
					}
					g.P("},")
					g.P("wantErr: true,")
					g.P("},")
				}
			} else if IsNumericType(underlyingType) {
				n, _ := strconv.ParseFloat(rule.Param, 64)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_min\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)-1), ",")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if IsSliceOrMapType(fieldType) {
				n, _ := strconv.Atoi(rule.Param)
				if n > 0 {
					g.P("{")
					g.P("name: \"invalid_", fieldName, "_min\",")
					g.P("input: ", typeName, "{")
					for _, otherFv := range allFields {
						if otherFv.Field.Name != fieldName {
							vg.generateValidFieldValue(g, otherFv, pkg)
						} else {
							g.P(fieldName, ": make(", fieldType, ", ", n-1, "),")
						}
					}
					g.P("},")
					g.P("wantErr: true,")
					g.P("},")
				}
			}

		case "max":
			if IsStringType(underlyingType) {
				n, _ := strconv.Atoi(rule.Param)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_max\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": \"", strings.Repeat("a", n+1), "\",")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if IsNumericType(underlyingType) {
				n, _ := strconv.ParseFloat(rule.Param, 64)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_max\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)+1), ",")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			}

		case "email", "url", "uuid", "ip", "ipv4", "ipv6", "duration", "alpha", "alphanum", "numeric", "dns1123_label":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_", rule.Name, "\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"invalid-value\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "oneof":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_oneof\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					if IsStringType(underlyingType) {
						g.P(fieldName, ": \"__invalid_value__\",")
					} else {
						g.P(fieldName, ": -99999,")
					}
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "oneof_enum":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_oneof_enum\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					vg.generateEnumTestValue(g, fieldName, fieldType, rule.Param, 99999, pkg)
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")
		}
	}
}
