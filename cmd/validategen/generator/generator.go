// Package generator provides validation code generation functionality.
package generator

import (
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tlipoca9/devgen/genkit"
)

// ToolName is the name of this tool, used in annotations.
const ToolName = "validategen"

// Predefined regex patterns
const (
	regexEmail    = "email"
	regexUUID     = "uuid"
	regexAlpha    = "alpha"
	regexAlphanum = "alphanum"
	regexNumeric  = "numeric"
)

var regexPatterns = map[string]string{
	regexEmail:    `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
	regexUUID:     `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	regexAlpha:    `^[a-zA-Z]+$`,
	regexAlphanum: `^[a-zA-Z0-9]+$`,
	regexNumeric:  `^[0-9]+$`,
}

var regexVarNames = map[string]string{
	regexEmail:    "_validateRegexEmail",
	regexUUID:     "_validateRegexUUID",
	regexAlpha:    "_validateRegexAlpha",
	regexAlphanum: "_validateRegexAlphanum",
	regexNumeric:  "_validateRegexNumeric",
}

// Generator generates Validate() methods for structs.
type Generator struct{}

// New creates a new Generator.
func New() *Generator {
	return &Generator{}
}

// regexTracker tracks custom regex patterns and assigns variable names.
type regexTracker struct {
	patterns map[string]string // pattern -> variable name
	counter  int
}

func newRegexTracker() *regexTracker {
	return &regexTracker{
		patterns: make(map[string]string),
	}
}

// getVarName returns the variable name for a pattern, creating one if needed.
func (rt *regexTracker) getVarName(pattern string) string {
	if varName, ok := rt.patterns[pattern]; ok {
		return varName
	}
	rt.counter++
	varName := fmt.Sprintf("_validateRegex%d", rt.counter)
	rt.patterns[pattern] = varName
	return varName
}

// Name returns the tool name.
func (vg *Generator) Name() string {
	return ToolName
}

// Config returns the tool configuration for VSCode extension integration.
func (vg *Generator) Config() genkit.ToolConfig {
	return genkit.ToolConfig{
		OutputSuffix: "_validate.go",
		Annotations: []genkit.AnnotationConfig{
			{Name: "validate", Type: "type", Doc: "Generate Validate() method for struct"},
			{Name: "required", Type: "field", Doc: "Field must not be empty/zero"},
			{
				Name:   "min",
				Type:   "field",
				Doc:    "Minimum value or length",
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name:   "max",
				Type:   "field",
				Doc:    "Maximum value or length",
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name:   "len",
				Type:   "field",
				Doc:    "Exact length",
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name:   "eq",
				Type:   "field",
				Doc:    "Must equal specified value",
				Params: &genkit.AnnotationParams{Type: []string{"string", "number", "bool"}, Placeholder: "value"},
			},
			{
				Name:   "ne",
				Type:   "field",
				Doc:    "Must not equal specified value",
				Params: &genkit.AnnotationParams{Type: []string{"string", "number", "bool"}, Placeholder: "value"},
			},
			{
				Name:   "gt",
				Type:   "field",
				Doc:    "Must be greater than",
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name:   "gte",
				Type:   "field",
				Doc:    "Must be greater than or equal",
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name:   "lt",
				Type:   "field",
				Doc:    "Must be less than",
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name:   "lte",
				Type:   "field",
				Doc:    "Must be less than or equal",
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name:   "oneof",
				Type:   "field",
				Doc:    "Must be one of the specified values",
				Params: &genkit.AnnotationParams{Type: "list", Placeholder: "values"},
			},
			{Name: "email", Type: "field", Doc: "Must be a valid email address"},
			{Name: "url", Type: "field", Doc: "Must be a valid URL"},
			{Name: "uuid", Type: "field", Doc: "Must be a valid UUID"},
			{Name: "ip", Type: "field", Doc: "Must be a valid IP address"},
			{Name: "ipv4", Type: "field", Doc: "Must be a valid IPv4 address"},
			{Name: "ipv6", Type: "field", Doc: "Must be a valid IPv6 address"},
			{Name: "duration", Type: "field", Doc: "Must be a valid time.Duration string (e.g., 1h30m, 500ms)"},
			{
				Name:   "duration_min",
				Type:   "field",
				Doc:    "Minimum duration value (e.g., 1s, 5m, 1h)",
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "duration"},
			},
			{
				Name:   "duration_max",
				Type:   "field",
				Doc:    "Maximum duration value (e.g., 1h, 24h, 7d)",
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "duration"},
			},
			{Name: "alpha", Type: "field", Doc: "Must contain only letters"},
			{Name: "alphanum", Type: "field", Doc: "Must contain only letters and numbers"},
			{Name: "numeric", Type: "field", Doc: "Must contain only numbers"},
			{
				Name:   "contains",
				Type:   "field",
				Doc:    "Must contain the specified substring",
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "substring"},
			},
			{
				Name:   "excludes",
				Type:   "field",
				Doc:    "Must not contain the specified substring",
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "substring"},
			},
			{
				Name:   "startswith",
				Type:   "field",
				Doc:    "Must start with the specified prefix",
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "prefix"},
			},
			{
				Name:   "endswith",
				Type:   "field",
				Doc:    "Must end with the specified suffix",
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "suffix"},
			},
			{
				Name:   "method",
				Type:   "field",
				Doc:    "Call specified method for validation (for struct fields)",
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "MethodName"},
				LSP: &genkit.LSPConfig{
					Enabled:     true,
					Provider:    "gopls",
					Feature:     "method",
					Signature:   "func() error",
					ResolveFrom: "fieldType",
				},
			},
			{
				Name:   "regex",
				Type:   "field",
				Doc:    "Must match the specified regular expression",
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "pattern"},
			},
			{
				Name: "format",
				Type: "field",
				Doc:  "Must be valid format (json, yaml, toml, csv)",
				Params: &genkit.AnnotationParams{
					Values:  []string{"json", "yaml", "toml", "csv"},
					MaxArgs: 1,
					Docs: map[string]string{
						"json": "Validate JSON format",
						"yaml": "Validate YAML format",
						"toml": "Validate TOML format",
						"csv":  "Validate CSV format",
					},
				},
			},
		},
	}
}

// Run processes all packages and generates validation methods.
func (vg *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error {
	var totalCount int
	for _, pkg := range gen.Packages {
		types := vg.FindTypes(pkg)
		if len(types) == 0 {
			continue
		}
		log.Find("Found %v type(s) with validation in %v", len(types), pkg.GoImportPath())
		for _, t := range types {
			log.Item("%v", t.Name)
		}
		totalCount += len(types)
		if err := vg.ProcessPackage(gen, pkg); err != nil {
			return fmt.Errorf("process %s: %w", pkg.Name, err)
		}
	}

	if totalCount == 0 {
		return nil
	}

	return nil
}

// ProcessPackage processes a package and generates validation methods.
func (vg *Generator) ProcessPackage(gen *genkit.Generator, pkg *genkit.Package) error {
	types := vg.FindTypes(pkg)
	if len(types) == 0 {
		return nil
	}

	outPath := genkit.OutputPath(pkg.Dir, pkg.Name+"_validate.go")
	g := gen.NewGeneratedFile(outPath, pkg.GoImportPath())

	// Track which regex patterns are used
	usedRegex := make(map[string]bool)
	// Track custom regex patterns
	customRegex := newRegexTracker()

	// First pass: collect all used regex patterns
	for _, typ := range types {
		for _, field := range typ.Fields {
			rules := vg.parseFieldAnnotations(field)
			for _, rule := range rules {
				switch rule.Name {
				case "email":
					usedRegex[regexEmail] = true
				case "uuid":
					usedRegex[regexUUID] = true
				case "alpha":
					usedRegex[regexAlpha] = true
				case "alphanum":
					usedRegex[regexAlphanum] = true
				case "numeric":
					usedRegex[regexNumeric] = true
				case "regex":
					if rule.Param != "" {
						customRegex.getVarName(rule.Param)
					}
				}
			}
		}
	}

	vg.WriteHeader(g, pkg.Name, usedRegex, customRegex)
	for _, typ := range types {
		if err := vg.GenerateValidate(g, typ, customRegex); err != nil {
			return err
		}
	}

	return nil
}

// FindTypes finds all types with validategen:@validate annotation.
func (vg *Generator) FindTypes(pkg *genkit.Package) []*genkit.Type {
	var types []*genkit.Type
	for _, t := range pkg.Types {
		if genkit.HasAnnotation(t.Doc, ToolName, "validate") {
			types = append(types, t)
		}
	}
	return types
}

// WriteHeader writes the file header and global regex variables.
func (vg *Generator) WriteHeader(
	g *genkit.GeneratedFile,
	pkgName string,
	usedRegex map[string]bool,
	customRegex *regexTracker,
) {
	g.P("// Code generated by ", ToolName, ". DO NOT EDIT.")
	g.P()
	g.P("package ", pkgName)

	// Generate global regex variables if any are used
	hasBuiltin := len(usedRegex) > 0
	hasCustom := len(customRegex.patterns) > 0
	if hasBuiltin || hasCustom {
		g.P()
		g.P("// Precompiled regex patterns for validation.")
		g.P("var (")
		// Built-in patterns (sorted for deterministic output)
		var builtinNames []string
		for name := range usedRegex {
			builtinNames = append(builtinNames, name)
		}
		sort.Strings(builtinNames)
		for _, name := range builtinNames {
			varName := regexVarNames[name]
			pattern := regexPatterns[name]
			g.P(varName, " = ", genkit.GoImportPath("regexp").Ident("MustCompile"), "(`", pattern, "`)")
		}
		// Custom patterns (sorted for deterministic output)
		var customPatterns []string
		for pattern := range customRegex.patterns {
			customPatterns = append(customPatterns, pattern)
		}
		sort.Strings(customPatterns)
		for _, pattern := range customPatterns {
			varName := customRegex.patterns[pattern]
			g.P(varName, " = ", genkit.GoImportPath("regexp").Ident("MustCompile"), "(", genkit.RawString(pattern), ")")
		}
		g.P(")")
	}
}

// GenerateValidate generates Validate method for a single type.
func (vg *Generator) GenerateValidate(g *genkit.GeneratedFile, typ *genkit.Type, customRegex *regexTracker) error {
	typeName := typ.Name

	// Collect fields with validation annotations
	var validatedFields []*fieldValidation
	for _, field := range typ.Fields {
		rules := vg.parseFieldAnnotations(field)
		if len(rules) > 0 {
			validatedFields = append(validatedFields, &fieldValidation{
				Field: field,
				Rules: rules,
			})
		}
	}

	if len(validatedFields) == 0 {
		return nil
	}

	// Check if type has postValidate method
	hasPostValidate := vg.hasPostValidateMethod(typ)

	// Generate Validate method
	g.P()
	g.P(genkit.GoMethod{
		Doc:     genkit.GoDoc("Validate validates the " + typeName + " fields."),
		Recv:    genkit.GoReceiver{Name: "x", Type: typeName},
		Name:    "Validate",
		Results: genkit.GoResults{{Type: "error"}},
	}, " {")

	g.P("var errs []string")
	g.P()

	for _, fv := range validatedFields {
		vg.generateFieldValidation(g, fv, customRegex)
	}

	g.P()
	// Call postValidate if exists (always called, regardless of errs)
	if hasPostValidate {
		g.P("return x.postValidate(errs)")
	} else {
		g.P("if len(errs) > 0 {")
		g.P(
			"return ",
			genkit.GoImportPath("fmt").Ident("Errorf"),
			"(\"%s\", ",
			genkit.GoImportPath("strings").Ident("Join"),
			"(errs, \"; \"))",
		)
		g.P("}")
		g.P("return nil")
	}
	g.P("}")

	return nil
}

// hasPostValidateMethod checks if the type has a postValidate(errs []string) error method.
func (vg *Generator) hasPostValidateMethod(typ *genkit.Type) bool {
	if typ.Pkg == nil || typ.Pkg.TypesPkg == nil {
		return false
	}

	// Look up the type in the package scope
	obj := typ.Pkg.TypesPkg.Scope().Lookup(typ.Name)
	if obj == nil {
		return false
	}

	// Get the named type
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}

	// Check methods on the type (including pointer receiver)
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if method.Name() == "postValidate" {
			// Verify signature: func(errs []string) error
			sig, ok := method.Type().(*types.Signature)
			if !ok {
				continue
			}
			// One parameter of type []string
			if sig.Params().Len() != 1 {
				continue
			}
			param := sig.Params().At(0)
			slice, ok := param.Type().(*types.Slice)
			if !ok {
				continue
			}
			if basic, ok := slice.Elem().(*types.Basic); !ok || basic.Kind() != types.String {
				continue
			}
			// One result of type error
			if sig.Results().Len() != 1 {
				continue
			}
			if sig.Results().At(0).Type().String() == "error" {
				return true
			}
		}
	}

	return false
}

// hasMethodOnFieldType checks if a method exists on the field's type.
// It handles qualified types (e.g., "common.NetworkConfiguration"), pointers, slices, and maps.
func (vg *Generator) hasMethodOnFieldType(pkg *genkit.Package, fieldType, methodName string) bool {
	if pkg == nil || pkg.TypesInfo == nil {
		return false
	}

	// Strip slice prefix
	baseType := strings.TrimPrefix(fieldType, "[]")
	// Strip map prefix (extract value type)
	if strings.HasPrefix(baseType, "map[") {
		// Find the value type after ]
		idx := strings.Index(baseType, "]")
		if idx != -1 && idx+1 < len(baseType) {
			baseType = baseType[idx+1:]
		}
	}
	// Strip pointer prefix
	baseType = strings.TrimPrefix(baseType, "*")

	// Handle qualified types (e.g., "common.NetworkConfiguration")
	var typeName string
	var lookupPkg *types.Package
	if strings.Contains(baseType, ".") {
		parts := strings.SplitN(baseType, ".", 2)
		pkgAlias := parts[0]
		typeName = parts[1]
		// Find the imported package by alias
		lookupPkg = vg.findImportedPackage(pkg, pkgAlias)
		if lookupPkg == nil {
			return false
		}
	} else {
		typeName = baseType
		lookupPkg = pkg.TypesPkg
	}

	if lookupPkg == nil {
		return false
	}

	// Look up the type in the package scope
	obj := lookupPkg.Scope().Lookup(typeName)
	if obj == nil {
		return false
	}

	// Get the named type
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}

	// Check methods on the type (value receiver)
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if method.Name() == methodName {
			return true
		}
	}

	// Also check methods on pointer receiver
	ptrType := types.NewPointer(named)
	methodSet := types.NewMethodSet(ptrType)
	for i := 0; i < methodSet.Len(); i++ {
		sel := methodSet.At(i)
		if sel.Obj().Name() == methodName {
			return true
		}
	}

	return false
}

// findImportedPackage finds an imported package by its alias name.
func (vg *Generator) findImportedPackage(pkg *genkit.Package, alias string) *types.Package {
	if pkg.TypesPkg == nil {
		return nil
	}

	// Check all imports
	for _, imp := range pkg.TypesPkg.Imports() {
		// Check if the import name matches the alias
		if imp.Name() == alias {
			return imp
		}
		// Also check the last part of the path (default import name)
		path := imp.Path()
		parts := strings.Split(path, "/")
		if len(parts) > 0 && parts[len(parts)-1] == alias {
			return imp
		}
	}

	return nil
}

// parseFieldAnnotations parses validation annotations from field doc/comment.
// Supported annotations:
//   - validategen:@required
//   - validategen:@min(n)
//   - validategen:@max(n)
//   - validategen:@len(n)
//   - validategen:@gt(n)
//   - validategen:@gte(n)
//   - validategen:@lt(n)
//   - validategen:@lte(n)
//   - validategen:@eq(v)
//   - validategen:@ne(v)
//   - validategen:@oneof(a, b, c)
//   - validategen:@email
//   - validategen:@url
//   - validategen:@uuid
//   - validategen:@ip
//   - validategen:@ipv4
//   - validategen:@ipv6
//   - validategen:@duration
//   - validategen:@alpha
//   - validategen:@alphanum
//   - validategen:@numeric
//   - validategen:@contains(s)
//   - validategen:@excludes(s)
//   - validategen:@startswith(s)
//   - validategen:@endswith(s)
//   - validategen:@regex(pattern)
func (vg *Generator) parseFieldAnnotations(field *genkit.Field) []*validateRule {
	var rules []*validateRule

	// Parse from both Doc and Comment
	doc := field.Doc + "\n" + field.Comment
	annotations := genkit.ParseAnnotations(doc)

	for _, ann := range annotations {
		if ann.Tool != ToolName {
			continue
		}

		rule := &validateRule{Name: ann.Name}

		// Get parameter from Flags (positional args)
		if len(ann.Flags) > 0 {
			rule.Param = strings.Join(ann.Flags, " ")
		}

		rules = append(rules, rule)
	}

	return rules
}

type fieldValidation struct {
	Field *genkit.Field
	Rules []*validateRule
}

type validateRule struct {
	Name  string
	Param string
}

// rulePriority defines the execution order for validation rules.
// Lower numbers execute first.
var rulePriority = map[string]int{
	// 1. Required check - must come first
	"required": 10,

	// 2. Range/length checks
	"min": 20,
	"max": 21,
	"len": 22,
	"gt":  23,
	"gte": 24,
	"lt":  25,
	"lte": 26,

	// 3. Equality checks
	"eq":    30,
	"ne":    31,
	"oneof": 32,

	// 4. Format checks
	"email":    40,
	"url":      41,
	"uuid":     42,
	"ip":           43,
	"ipv4":         44,
	"ipv6":         45,
	"duration_min": 46,
	"duration_max": 47,
	"alpha":    46,
	"alphanum": 47,
	"numeric":  48,
	"regex":    49,
	"format":   50,

	// 5. String content checks
	"contains":   60,
	"excludes":   61,
	"startswith": 62,
	"endswith":   63,

	// 6. Nested validation - should come last
	"method": 70,
}

func (vg *Generator) generateFieldValidation(g *genkit.GeneratedFile, fv *fieldValidation, customRegex *regexTracker) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type

	// Sort rules by priority for deterministic output
	rules := make([]*validateRule, len(fv.Rules))
	copy(rules, fv.Rules)
	sort.SliceStable(rules, func(i, j int) bool {
		pi := rulePriority[rules[i].Name]
		pj := rulePriority[rules[j].Name]
		if pi != pj {
			return pi < pj
		}
		// Same priority: maintain original order (stable sort)
		return false
	})

	// Collect duration-related rules to generate them together
	var hasDuration, hasDurationMin, hasDurationMax bool
	var durationMinParam, durationMaxParam string
	for _, rule := range rules {
		switch rule.Name {
		case "duration":
			hasDuration = true
		case "duration_min":
			hasDurationMin = true
			durationMinParam = rule.Param
		case "duration_max":
			hasDurationMax = true
			durationMaxParam = rule.Param
		}
	}

	// Track if duration block has been generated
	durationGenerated := false

	for _, rule := range rules {
		switch rule.Name {
		case "required":
			vg.genRequired(g, fieldName, fieldType)
		case "min":
			vg.genMin(g, fieldName, fieldType, rule.Param)
		case "max":
			vg.genMax(g, fieldName, fieldType, rule.Param)
		case "len":
			vg.genLen(g, fieldName, fieldType, rule.Param)
		case "eq":
			vg.genEq(g, fieldName, fieldType, rule.Param)
		case "ne":
			vg.genNe(g, fieldName, fieldType, rule.Param)
		case "gt":
			vg.genGt(g, fieldName, fieldType, rule.Param)
		case "gte":
			vg.genGte(g, fieldName, fieldType, rule.Param)
		case "lt":
			vg.genLt(g, fieldName, fieldType, rule.Param)
		case "lte":
			vg.genLte(g, fieldName, fieldType, rule.Param)
		case "oneof":
			vg.genOneof(g, fv.Field, rule.Param)
		case "email":
			vg.genEmail(g, fieldName)
		case "url":
			vg.genURL(g, fieldName)
		case "uuid":
			vg.genUUID(g, fieldName)
		case "alpha":
			vg.genAlpha(g, fieldName)
		case "alphanum":
			vg.genAlphanum(g, fieldName)
		case "numeric":
			vg.genNumeric(g, fieldName)
		case "contains":
			vg.genContains(g, fieldName, rule.Param)
		case "excludes":
			vg.genExcludes(g, fieldName, rule.Param)
		case "startswith":
			vg.genStartsWith(g, fieldName, rule.Param)
		case "endswith":
			vg.genEndsWith(g, fieldName, rule.Param)
		case "ip":
			vg.genIP(g, fieldName)
		case "ipv4":
			vg.genIPv4(g, fieldName)
		case "ipv6":
			vg.genIPv6(g, fieldName)
		case "duration", "duration_min", "duration_max":
			// Generate all duration validations together, only once
			if !durationGenerated {
				vg.genDurationCombined(g, fieldName, hasDuration, hasDurationMin, durationMinParam, hasDurationMax, durationMaxParam)
				durationGenerated = true
			}
		case "method":
			vg.genMethod(g, fv.Field, rule.Param)
		case "regex":
			vg.genRegex(g, fv.Field, rule.Param, customRegex)
		case "format":
			vg.genFormat(g, fv.Field, rule.Param)
		}
	}
}

func (vg *Generator) genRequired(g *genkit.GeneratedFile, fieldName, fieldType string) {
	if isStringType(fieldType) {
		g.P("if x.", fieldName, " == \"\" {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") == 0 {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if isPointerType(fieldType) {
		g.P("if x.", fieldName, " == nil {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if isBoolType(fieldType) {
		// For bool, required means must be true
		g.P("if !x.", fieldName, " {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if isNumericType(fieldType) {
		// For numeric types, check zero value
		g.P("if x.", fieldName, " == 0 {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	}
	// Other types (struct, interface, etc.) are not supported for required check
}

func (vg *Generator) genMin(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") < ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") < ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have at least ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genMax(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") > ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") > ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have at most ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genLen(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") != ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be exactly ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") != ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have exactly ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	}
}

func (vg *Generator) genEq(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if x.", fieldName, " != \"", param, "\" {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must equal ",
			param,
			", got %q\", x.",
			fieldName,
			"))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " != \"", param, "\" {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must equal ",
			param,
			", got %q\", *x.",
			fieldName,
			"))",
		)
		g.P("}")
	} else if isNumericType(fieldType) || isBoolType(fieldType) {
		g.P("if x.", fieldName, " != ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must equal ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) || isPointerToBoolType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " != ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must equal ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genNe(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	if isStringType(fieldType) {
		g.P("if x.", fieldName, " == \"", param, "\" {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", param, "\")")
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " == \"", param, "\" {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", param, "\")")
		g.P("}")
	} else if isNumericType(fieldType) || isBoolType(fieldType) {
		g.P("if x.", fieldName, " == ", param, " {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", param, "\")")
		g.P("}")
	} else if isPointerToNumericType(fieldType) || isPointerToBoolType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " == ", param, " {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", param, "\")")
		g.P("}")
	}
}

func (vg *Generator) genGt(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") <= ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be more than ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") <= ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be more than ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") <= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have more than ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " <= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be greater than ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " <= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be greater than ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genGte(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") < ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") < ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have at least ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genLt(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") >= ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be less than ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") >= ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be less than ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") >= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have less than ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " >= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be less than ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " >= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be less than ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genLte(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") > ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") > ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have at most ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genOneof(g *genkit.GeneratedFile, field *genkit.Field, param string) {
	// Validation already done in validateRule, skip if invalid
	if param == "" {
		return
	}

	values := strings.Split(param, " ")
	// Clean up values
	var cleanValues []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			cleanValues = append(cleanValues, v)
		}
	}
	if len(cleanValues) == 0 {
		return // Validation already done in validateRule
	}

	fieldName := field.Name
	fieldType := field.Type
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
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
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be one of [",
			strings.Join(cleanValues, ", "),
			"], got %q\", x.",
			fieldName,
			"))",
		)
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

func (vg *Generator) genEmail(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexEmail], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be a valid email address, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genURL(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" {")
	g.P("if _, err := ", genkit.GoImportPath("net/url").Ident("ParseRequestURI"), "(x.", fieldName, "); err != nil {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid URL, got %q\", x.", fieldName, "))")
	g.P("}")
	g.P("}")
}

func (vg *Generator) genUUID(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexUUID], ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid UUID, got %q\", x.", fieldName, "))")
	g.P("}")
}

func (vg *Generator) genAlpha(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexAlpha], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must contain only letters, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genAlphanum(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexAlphanum], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must contain only letters and numbers, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genNumeric(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexNumeric], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must contain only numbers, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genContains(g *genkit.GeneratedFile, fieldName, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if !", genkit.GoImportPath("strings").Ident("Contains"), "(x.", fieldName, ", \"", param, "\") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must contain '",
		param,
		"', got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genExcludes(g *genkit.GeneratedFile, fieldName, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if ", genkit.GoImportPath("strings").Ident("Contains"), "(x.", fieldName, ", \"", param, "\") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must not contain '",
		param,
		"', got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genStartsWith(g *genkit.GeneratedFile, fieldName, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if !", genkit.GoImportPath("strings").Ident("HasPrefix"), "(x.", fieldName, ", \"", param, "\") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must start with '",
		param,
		"', got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genEndsWith(g *genkit.GeneratedFile, fieldName, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if !", genkit.GoImportPath("strings").Ident("HasSuffix"), "(x.", fieldName, ", \"", param, "\") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must end with '",
		param,
		"', got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genIP(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ") == nil {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be a valid IP address, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genIPv4(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" {")
	g.P("ip := ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ")")
	g.P("if ip == nil || ip.To4() == nil {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be a valid IPv4 address, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
	g.P("}")
}

func (vg *Generator) genIPv6(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" {")
	g.P("ip := ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ")")
	g.P("if ip == nil || ip.To4() != nil {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be a valid IPv6 address, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
	g.P("}")
}

func (vg *Generator) genDurationCombined(g *genkit.GeneratedFile, fieldName string, checkFormat, hasMin bool, minParam string, hasMax bool, maxParam string) {
	// Parse min/max durations at generation time
	var minNanos, maxNanos int64
	if hasMin && minParam != "" {
		if dur, err := time.ParseDuration(minParam); err == nil {
			minNanos = dur.Nanoseconds()
		} else {
			hasMin = false // Invalid duration, skip
		}
	}
	if hasMax && maxParam != "" {
		if dur, err := time.ParseDuration(maxParam); err == nil {
			maxNanos = dur.Nanoseconds()
		} else {
			hasMax = false // Invalid duration, skip
		}
	}

	// If only format check, use simple validation
	if checkFormat && !hasMin && !hasMax {
		fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
		g.P("if x.", fieldName, " != \"\" {")
		g.P("if _, err := ", genkit.GoImportPath("time").Ident("ParseDuration"), "(x.", fieldName, "); err != nil {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be a valid duration (e.g., 1h30m, 500ms), got %q\", x.",
			fieldName,
			"))",
		)
		g.P("}")
		g.P("}")
		return
	}

	// Combined validation with min/max
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	timePkg := genkit.GoImportPath("time")

	g.P("if x.", fieldName, " != \"\" {")
	g.P("if _dur, _err := ", timePkg.Ident("ParseDuration"), "(x.", fieldName, "); _err != nil {")
	if checkFormat {
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be a valid duration (e.g., 1h30m, 500ms), got %q\", x.",
			fieldName,
			"))",
		)
	}
	g.P("} else {")
	if hasMin {
		g.P("if _dur < ", minNanos, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			minParam,
			", got %s\", x.",
			fieldName,
			"))",
		)
		g.P("}")
	}
	if hasMax {
		g.P("if _dur > ", maxNanos, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			maxParam,
			", got %s\", x.",
			fieldName,
			"))",
		)
		g.P("}")
	}
	g.P("}")
	g.P("}")
}

func (vg *Generator) genMethod(g *genkit.GeneratedFile, field *genkit.Field, methodName string) {
	// Validation already done in validateRule, skip if invalid
	if methodName == "" {
		return
	}
	fieldName := field.Name
	fieldType := field.Type
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")

	if isSliceType(fieldType) {
		// For slice types, iterate over elements and call method on each
		g.P("for _i, _v := range x.", fieldName, " {")
		elemType := strings.TrimPrefix(fieldType, "[]")
		if isPointerType(elemType) {
			g.P("if _v != nil {")
			g.P("if err := _v.", methodName, "(); err != nil {")
			g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, "[%d]: %v\", _i, err))")
			g.P("}")
			g.P("}")
		} else {
			g.P("if err := _v.", methodName, "(); err != nil {")
			g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, "[%d]: %v\", _i, err))")
			g.P("}")
		}
		g.P("}")
	} else if isMapType(fieldType) {
		// For map types, iterate over values and call method on each
		g.P("for _k, _v := range x.", fieldName, " {")
		valueType := extractMapValueType(fieldType)
		if isPointerType(valueType) {
			g.P("if _v != nil {")
			g.P("if err := _v.", methodName, "(); err != nil {")
			g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, "[%v]: %v\", _k, err))")
			g.P("}")
			g.P("}")
		} else {
			g.P("if err := _v.", methodName, "(); err != nil {")
			g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, "[%v]: %v\", _k, err))")
			g.P("}")
		}
		g.P("}")
	} else if isPointerType(fieldType) {
		// For pointer types, check nil first
		g.P("if x.", fieldName, " != nil {")
		g.P("if err := x.", fieldName, ".", methodName, "(); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, ": %v\", err))")
		g.P("}")
		g.P("}")
	} else {
		// For value types (struct, etc.), call directly
		g.P("if err := x.", fieldName, ".", methodName, "(); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, ": %v\", err))")
		g.P("}")
	}
}

func (vg *Generator) genRegex(g *genkit.GeneratedFile, field *genkit.Field, pattern string, customRegex *regexTracker) {
	// Validation already done in validateRule, skip if invalid
	if pattern == "" {
		return
	}
	fieldName := field.Name
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	varName := customRegex.getVarName(pattern)
	g.P("if x.", fieldName, " != \"\" && !", varName, ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must match pattern %s, got %q\", ",
		genkit.RawString(pattern),
		", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

// Supported format types for @format annotation.
var supportedFormats = map[string]bool{
	"json": true,
	"yaml": true,
	"toml": true,
	"csv":  true,
}

func (vg *Generator) genFormat(g *genkit.GeneratedFile, field *genkit.Field, format string) {
	// Validation already done in validateRule, skip if invalid
	if format == "" || strings.Contains(format, " ") {
		return
	}
	format = strings.ToLower(format)
	if !supportedFormats[format] {
		return
	}

	fieldName := field.Name
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")

	g.P("if x.", fieldName, " != \"\" {")
	switch format {
	case "json":
		g.P("if !", genkit.GoImportPath("encoding/json").Ident("Valid"), "([]byte(x.", fieldName, ")) {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be valid JSON format\"))",
		)
		g.P("}")
	case "yaml":
		// Import gopkg.in/yaml.v3 with alias "yaml" since the path base is "v3"
		yamlImport := genkit.GoImportPath("gopkg.in/yaml.v3")
		g.ImportAs(yamlImport, "yaml")
		g.P("var _yamlVal", fieldName, " interface{}")
		g.P(
			"if err := ",
			yamlImport.Ident("Unmarshal"),
			"([]byte(x.",
			fieldName,
			"), &_yamlVal",
			fieldName,
			"); err != nil {",
		)
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be valid YAML format: %v\", err))",
		)
		g.P("}")
	case "toml":
		g.P("var _tomlVal", fieldName, " interface{}")
		g.P(
			"if err := ",
			genkit.GoImportPath("github.com/BurntSushi/toml").Ident("Unmarshal"),
			"([]byte(x.",
			fieldName,
			"), &_tomlVal",
			fieldName,
			"); err != nil {",
		)
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be valid TOML format: %v\", err))",
		)
		g.P("}")
	case "csv":
		g.P(
			"_csvReader",
			fieldName,
			" := ",
			genkit.GoImportPath("encoding/csv").Ident("NewReader"),
			"(",
			genkit.GoImportPath("strings").Ident("NewReader"),
			"(x.",
			fieldName,
			"))",
		)
		g.P("if _, err := _csvReader", fieldName, ".ReadAll(); err != nil {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be valid CSV format: %v\", err))",
		)
		g.P("}")
	}
	g.P("}")
}

// Helper functions

func isStringType(t string) bool {
	return t == "string"
}

// isPointerToStringType checks if t is a pointer to string type (e.g., "*string").
func isPointerToStringType(t string) bool {
	return t == "*string"
}

func isSliceOrMapType(t string) bool {
	return strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[")
}

func isSliceType(t string) bool {
	return strings.HasPrefix(t, "[]")
}

func isMapType(t string) bool {
	return strings.HasPrefix(t, "map[")
}

// extractMapValueType extracts the value type from a map type string.
// e.g., "map[string]Address" -> "Address", "map[int]*User" -> "*User"
func extractMapValueType(t string) string {
	// Find the closing bracket of the key type
	if !strings.HasPrefix(t, "map[") {
		return ""
	}
	depth := 0
	for i := 4; i < len(t); i++ {
		switch t[i] {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				// Found the closing bracket, value type starts after it
				return t[i+1:]
			}
			depth--
		}
	}
	return ""
}

func isPointerType(t string) bool {
	return strings.HasPrefix(t, "*")
}

func isBoolType(t string) bool {
	return t == "bool"
}

// isPointerToBoolType checks if t is a pointer to bool type (e.g., "*bool").
func isPointerToBoolType(t string) bool {
	return t == "*bool"
}

func isNumericType(t string) bool {
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"byte", "rune", "uintptr":
		return true
	default:
		return false
	}
}

// isPointerToNumericType checks if t is a pointer to a numeric type (e.g., "*int", "*float64").
func isPointerToNumericType(t string) bool {
	if !strings.HasPrefix(t, "*") {
		return false
	}
	return isNumericType(strings.TrimPrefix(t, "*"))
}

// isBuiltinType checks if the type is a Go builtin type that cannot have methods.
func isBuiltinType(t string) bool {
	// Check pointer to builtin
	if strings.HasPrefix(t, "*") {
		return isBuiltinType(strings.TrimPrefix(t, "*"))
	}
	// Check slice - recurse to check element type
	if strings.HasPrefix(t, "[]") {
		return isBuiltinType(strings.TrimPrefix(t, "[]"))
	}
	// Check map - recurse to check value type
	if strings.HasPrefix(t, "map[") {
		valueType := extractMapValueType(t)
		if valueType == "" {
			return true // malformed map type
		}
		return isBuiltinType(valueType)
	}
	// Builtin primitive types
	switch t {
	case "string", "bool", "error",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128",
		"byte", "rune", "uintptr", "any":
		return true
	}
	// interface{} is builtin
	if strings.HasPrefix(t, "interface") {
		return true
	}
	return false
}

// isValidNumber checks if a string is a valid number (integer or float).
func isValidNumber(s string) bool {
	if s == "" {
		return false
	}
	// Try parsing as integer first
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	// Try parsing as float
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// isValidDuration checks if a string is a valid Go duration (e.g., "1h", "30m", "500ms").
func isValidDuration(s string) bool {
	if s == "" {
		return false
	}
	_, err := time.ParseDuration(s)
	return err == nil
}

// Error codes for diagnostics.
const (
	ErrCodeMethodMissingParam  = "E001"
	ErrCodeRegexMissingPattern = "E002"
	ErrCodeFormatMissingType   = "E003"
	ErrCodeFormatMultipleArgs  = "E004"
	ErrCodeFormatUnsupported   = "E005"
	ErrCodeOneofMissingValues  = "E006"
	ErrCodeMissingParam        = "E007" // Generic missing parameter error
	ErrCodeInvalidParamType    = "E008" // Invalid parameter type
	ErrCodeInvalidFieldType    = "E009" // Annotation not applicable to field type
	ErrCodeMethodNotFound      = "E010" // Method not found on type
)

// Validate implements genkit.ValidatableTool.
// It checks for errors without generating files, returning diagnostics for IDE integration.
func (vg *Generator) Validate(gen *genkit.Generator, _ *genkit.Logger) []genkit.Diagnostic {
	c := genkit.NewDiagnosticCollector(ToolName)

	for _, pkg := range gen.Packages {
		for _, typ := range pkg.Types {
			if !genkit.HasAnnotation(typ.Doc, ToolName, "validate") {
				continue
			}
			vg.validateType(c, typ)
		}
	}

	return c.Collect()
}

// validateType validates a single type and collects diagnostics.
func (vg *Generator) validateType(c *genkit.DiagnosticCollector, typ *genkit.Type) {
	for _, field := range typ.Fields {
		rules := vg.parseFieldAnnotations(field)
		for _, rule := range rules {
			vg.validateRule(c, typ, field, rule)
		}
	}
}

// validateRule validates a single rule and collects diagnostics.
func (vg *Generator) validateRule(c *genkit.DiagnosticCollector, typ *genkit.Type, field *genkit.Field, rule *validateRule) {
	// Use UnderlyingType for validation (supports custom types like `type Email string`)
	underlyingType := field.UnderlyingType

	switch rule.Name {
	// Annotations that require string underlying type
	case "email", "url", "uuid", "alpha", "alphanum", "numeric", "regex", "format":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		// Additional validation for specific annotations
		switch rule.Name {
		case "regex":
			if rule.Param == "" {
				c.Error(ErrCodeRegexMissingPattern, "@regex annotation requires a pattern parameter", field.Pos)
			}
		case "format":
			if rule.Param == "" {
				c.Error(ErrCodeFormatMissingType, "@format annotation requires a format type parameter", field.Pos)
			} else if strings.Contains(rule.Param, " ") {
				c.Error(ErrCodeFormatMultipleArgs, "@format annotation only accepts one parameter", field.Pos)
			} else if !supportedFormats[strings.ToLower(rule.Param)] {
				c.Errorf(ErrCodeFormatUnsupported, field.Pos,
					"unsupported format %q, supported: json, yaml, toml, csv", rule.Param)
			}
		}

	// Annotations that require string underlying type with parameter
	case "contains", "excludes", "startswith", "endswith":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@%s annotation requires a string parameter", rule.Name)
		}

	// IP validation annotations - require string underlying type
	case "ip", "ipv4", "ipv6", "duration":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}

	// Duration range validation - require string underlying type and valid duration parameter
	case "duration_min", "duration_max":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@%s annotation requires a duration parameter", rule.Name)
		} else if !isValidDuration(rule.Param) {
			c.Errorf(ErrCodeInvalidParamType, field.Pos, "@%s parameter must be a valid duration (e.g., 1h, 30m, 500ms), got %q", rule.Name, rule.Param)
		}

	// Annotations that work on string/slice/map (length) or numeric (value)
	case "min", "max", "gt", "gte", "lt", "lte":
		if !isStringType(underlyingType) && !isPointerToStringType(underlyingType) &&
			!isSliceOrMapType(underlyingType) &&
			!isNumericType(underlyingType) && !isPointerToNumericType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string, slice, map, or numeric underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@%s annotation requires a value parameter", rule.Name)
		} else if !isValidNumber(rule.Param) {
			c.Errorf(ErrCodeInvalidParamType, field.Pos, "@%s parameter must be a number, got %q", rule.Name, rule.Param)
		}

	// len annotation - only for string/slice/map
	case "len":
		if !isStringType(underlyingType) && !isSliceOrMapType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@len annotation requires string, slice, or map underlying type, got %s",
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@len annotation requires a value parameter")
		} else if !isValidNumber(rule.Param) {
			c.Errorf(ErrCodeInvalidParamType, field.Pos, "@len parameter must be a number, got %q", rule.Param)
		}

	// eq/ne - string, numeric, or bool
	case "eq", "ne":
		if !isStringType(underlyingType) && !isPointerToStringType(underlyingType) &&
			!isNumericType(underlyingType) && !isPointerToNumericType(underlyingType) &&
			!isBoolType(underlyingType) && !isPointerToBoolType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string, numeric, or bool underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@%s annotation requires a value parameter", rule.Name)
		}

	// required - string, slice, map, pointer, bool, numeric
	case "required":
		if !isStringType(underlyingType) && !isSliceOrMapType(underlyingType) && !isPointerType(underlyingType) &&
			!isBoolType(underlyingType) && !isNumericType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@required annotation requires string, slice, map, pointer, bool, or numeric underlying type, got %s",
				underlyingType,
			)
		}

	// method - must be a custom type (not builtin types like string, int, bool, etc.)
	case "method":
		if rule.Param == "" {
			c.Error(ErrCodeMethodMissingParam, "@method annotation requires a method name parameter", field.Pos)
			return
		}
		// For method, check declared type (not underlying) - custom types can have methods
		if isBuiltinType(field.Type) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@method annotation can only be applied to custom types, got builtin type %s",
				field.Type,
			)
			return
		}
		// Check if the method exists on the field type
		if typ.Pkg != nil && typ.Pkg.TypesInfo != nil {
			if !vg.hasMethodOnFieldType(typ.Pkg, field.Type, rule.Param) {
				c.Errorf(
					ErrCodeMethodNotFound,
					field.Pos,
					"method '%s' not found on type '%s'",
					rule.Param,
					field.Type,
				)
			}
		}

	// oneof - string or numeric
	case "oneof":
		if !isStringType(underlyingType) && !isNumericType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@oneof annotation requires string or numeric underlying type, got %s",
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeOneofMissingValues, field.Pos,
				"@oneof annotation requires at least one value")
		} else {
			// Check if all values are empty after cleanup
			hasValue := false
			for _, v := range strings.Split(rule.Param, " ") {
				if strings.TrimSpace(v) != "" {
					hasValue = true
					break
				}
			}
			if !hasValue {
				c.Errorf(ErrCodeOneofMissingValues, field.Pos,
					"@oneof annotation requires at least one value")
			}
		}
	}
}
