// Package generator provides validation code generation functionality.
package generator

import (
	"strings"

	"github.com/tlipoca9/devgen/genkit"
)

func init() {
	DefaultRegistry.Register("format", PriorityFormat+13, func() Rule { return &FormatRule{} })
}

// SupportedFormats defines the supported format types.
var SupportedFormats = map[string]bool{
	"json": true,
	"yaml": true,
	"toml": true,
	"csv":  true,
}

// FormatRule validates string is valid format (json, yaml, toml, csv).
type FormatRule struct{}

func (r *FormatRule) Name() string              { return "format" }
func (r *FormatRule) RequiredRegex() []string   { return nil }

func (r *FormatRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" || strings.Contains(ctx.Param, " ") {
		return
	}
	format := strings.ToLower(ctx.Param)
	if !SupportedFormats[format] {
		return
	}

	fieldName := ctx.FieldName
	fmtSprintf := fmtSprintf()
	g := ctx.G

	g.P("if x.", fieldName, " != \"\" {")
	switch format {
	case "json":
		g.P("if !", genkit.GoImportPath("encoding/json").Ident("Valid"), "([]byte(x.", fieldName, ")) {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be valid JSON format\"))")
		g.P("}")
	case "yaml":
		yamlImport := genkit.GoImportPath("gopkg.in/yaml.v3")
		g.ImportAs(yamlImport, "yaml")
		g.P("var _yamlVal", fieldName, " interface{}")
		g.P("if err := ", yamlImport.Ident("Unmarshal"), "([]byte(x.", fieldName, "), &_yamlVal", fieldName, "); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be valid YAML format: %v\", err))")
		g.P("}")
	case "toml":
		g.P("var _tomlVal", fieldName, " interface{}")
		g.P("if err := ", genkit.GoImportPath("github.com/BurntSushi/toml").Ident("Unmarshal"), "([]byte(x.", fieldName, "), &_tomlVal", fieldName, "); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be valid TOML format: %v\", err))")
		g.P("}")
	case "csv":
		g.P("_csvReader", fieldName, " := ", genkit.GoImportPath("encoding/csv").Ident("NewReader"), "(", genkit.GoImportPath("strings").Ident("NewReader"), "(x.", fieldName, "))")
		g.P("if _, err := _csvReader", fieldName, ".ReadAll(); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be valid CSV format: %v\", err))")
		g.P("}")
	}
	g.P("}")
}

func (r *FormatRule) Validate(ctx *ValidateContext) {
	if !IsStringType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@format annotation requires string underlying type, got %s",
			ctx.UnderlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Error(ErrCodeFormatMissingType, "@format annotation requires a format type parameter", ctx.Field.Pos)
	} else if strings.Contains(ctx.Param, " ") {
		ctx.Collector.Error(ErrCodeFormatMultipleArgs, "@format annotation only accepts one parameter", ctx.Field.Pos)
	} else if !SupportedFormats[strings.ToLower(ctx.Param)] {
		ctx.Collector.Errorf(ErrCodeFormatUnsupported, ctx.Field.Pos, "unsupported format %q, supported: json, yaml, toml, csv", ctx.Param)
	}
}
