// Package generator provides struct converter code generation functionality.
package generator

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/tlipoca9/devgen/cmd/convertgen/rules"
	"github.com/tlipoca9/devgen/genkit"
)

// ToolName is the name of this tool, used in annotations.
const ToolName = "convertgen"

// Annotation names.
const (
	AnnotationConverter = "converter" // marks an interface as a converter
	AnnotationMap       = "map"       // maps source field to destination field
	AnnotationIgnore    = "ignore"    // ignores fields during conversion
	AnnotationShallow   = "shallow"   // uses shallow copy instead of deep copy
)

// Error codes for diagnostics.
const (
	ErrCodeInvalidMethodSig     = "E001" // method signature is invalid
	ErrCodeInvalidMapAnnotation = "E002" // @map annotation requires 2 parameters
	ErrCodeFieldNotFound        = "E003" // source/destination field not found
	ErrCodeTypeMismatch         = "E004" // field type mismatch

	WarnCodeMissingField = "W001" // destination field has no source mapping
)

// Generator generates struct converter implementations.
type Generator struct{}

// New creates a new Generator.
func New() *Generator {
	return &Generator{}
}

// Name returns the tool name.
func (g *Generator) Name() string {
	return ToolName
}

// Config returns the tool configuration for VSCode extension integration.
func (g *Generator) Config() genkit.ToolConfig {
	return genkit.ToolConfig{
		OutputSuffix: "_convertgen.go",
		Annotations: []genkit.AnnotationConfig{
			{
				Name: AnnotationConverter,
				Type: "type",
				Doc: `Mark an interface as a struct converter.

USAGE:
  // convertgen:@converter
  type UserConverter interface {
      Convert(src *User) *UserDTO
  }

The interface methods define conversion functions. Each method must have:
- Exactly 1 parameter (source type)
- Exactly 1 return value (destination type)

GENERATED CODE:
- A private implementation struct (e.g., userConverterImpl)
- A public singleton variable (e.g., DefaultUserConverter)
- Implementation of all interface methods with automatic field mapping`,
			},
			{
				Name: AnnotationMap,
				Type: "field",
				Doc: `Map a source field to a differently-named destination field.

USAGE:
  // convertgen:@map(SourceField, DestField)

EXAMPLE:
  // convertgen:@converter
  type UserConverter interface {
      // convertgen:@map(FullName, Name)
      // convertgen:@map(EmailAddress, Email)
      Convert(src *User) *UserDTO
  }

This will generate: dst.Name = src.FullName`,
				Params: &genkit.AnnotationParams{
					Type:        "string",
					Placeholder: "srcField, dstField",
					MaxArgs:     2,
				},
			},
			{
				Name: AnnotationIgnore,
				Type: "field",
				Doc: `Ignore specific fields during conversion.

USAGE:
  // convertgen:@ignore(Field1, Field2, ...)

EXAMPLE:
  // convertgen:@converter
  type UserConverter interface {
      // convertgen:@ignore(Password, InternalID)
      Convert(src *User) *UserDTO
  }

Ignored fields will not be copied to the destination struct.`,
				Params: &genkit.AnnotationParams{
					Type:        "string",
					Placeholder: "field1, field2, ...",
				},
			},
			{
				Name: AnnotationShallow,
				Type: "field",
				Doc: `Use shallow copy instead of deep copy for this method.

USAGE:
  // convertgen:@shallow

By default, convertgen generates deep copy code for:
- Pointers to structs (creates new instances)
- Slices (copies elements)
- Maps (copies key-value pairs)

With @shallow, all fields are assigned directly without deep copying.`,
			},
		},
	}
}

// Rules implements genkit.RuleTool.
// Returns AI-friendly documentation for convertgen.
func (g *Generator) Rules() []genkit.Rule {
	return []genkit.Rule{
		{
			Name:        "devgen-tool-convertgen",
			Description: "Go 结构体转换代码生成工具 convertgen 的使用指南。当用户需要生成类型安全的结构体转换代码（如 DTO 转换、API 模型转换）时使用此规则。",
			Globs:       []string{"*.go"},
			AlwaysApply: false,
			Content:     rules.ConvertgenRule,
		},
	}
}

// Validate implements genkit.ValidatableTool.
// It checks for errors without generating files, returning diagnostics for IDE integration.
func (g *Generator) Validate(gen *genkit.Generator, _ *genkit.Logger) []genkit.Diagnostic {
	c := genkit.NewDiagnosticCollector(ToolName)

	for _, pkg := range gen.Packages {
		converters := g.findConverters(pkg)
		for _, conv := range converters {
			g.validateConverter(c, pkg, conv)
		}
	}

	return c.Collect()
}

// validateConverter validates a single converter interface.
func (g *Generator) validateConverter(c *genkit.DiagnosticCollector, pkg *genkit.Package, conv *converter) {
	for _, method := range conv.methods {
		g.validateMethod(c, pkg, method)
	}
}

// validateMethod validates a single conversion method.
func (g *Generator) validateMethod(c *genkit.DiagnosticCollector, pkg *genkit.Package, method *convertMethod) {
	srcFieldsMap := g.getStructFieldsMap(method.srcType)
	dstFieldsMap := g.getStructFieldsMap(method.dstType)

	// Validate @map annotations
	for srcField, dstField := range method.fieldMaps {
		// Check source field exists
		if _, ok := srcFieldsMap[srcField]; !ok {
			c.Errorf(ErrCodeFieldNotFound, method.pos,
				"@map: source field %q not found in %s", srcField, g.typeString(pkg, method.srcType))
		}
		// Check destination field exists
		if _, ok := dstFieldsMap[dstField]; !ok {
			c.Errorf(ErrCodeFieldNotFound, method.pos,
				"@map: destination field %q not found in %s", dstField, g.typeString(pkg, method.dstType))
		}
	}

	// Build reverse map: dst field -> src field
	dstToSrc := make(map[string]string)
	for src, dst := range method.fieldMaps {
		dstToSrc[dst] = src
	}

	// Check field coverage (warnings)
	for dstName := range dstFieldsMap {
		if method.ignoreFields[dstName] {
			continue
		}

		srcName := dstName
		if mapped, ok := dstToSrc[dstName]; ok {
			srcName = mapped
		}

		if _, ok := srcFieldsMap[srcName]; !ok {
			c.Warningf(WarnCodeMissingField, method.pos,
				"destination field %q has no corresponding source field (consider @ignore or @map)", dstName)
		}
	}
}

// Run processes all packages and generates converter implementations.
func (g *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error {
	var totalCount int

	for _, pkg := range gen.Packages {
		converters := g.findConverters(pkg)
		if len(converters) == 0 {
			continue
		}

		log.Find("Found %v converter(s) in %s", len(converters), pkg.GoImportPath())
		for _, c := range converters {
			log.Item("%s", c.name)
		}
		totalCount += len(converters)

		if err := g.processPackage(gen, pkg, converters); err != nil {
			return fmt.Errorf("process %s: %w", pkg.Name, err)
		}
	}

	if totalCount == 0 {
		log.Info("no converters found")
	}

	return nil
}

// methodIndex indexes all conversion methods for quick lookup.
type methodIndex struct {
	// bySignature maps type signature to method name
	// key: "srcType->dstType", value: method name
	bySignature map[string]string

	// byElemSignature maps element type signature to method name (for slice conversion)
	// key: "srcElemType->dstElemType", value: method name
	byElemSignature map[string]string
}

// converter represents a parsed converter interface.
type converter struct {
	name    string           // interface name
	methods []*convertMethod // conversion methods
	index   *methodIndex     // method index for nested conversion lookup
}

// buildMethodIndex builds the method index for nested conversion lookup.
func (c *converter) buildMethodIndex(g *Generator, pkg *genkit.Package) {
	c.index = &methodIndex{
		bySignature:     make(map[string]string),
		byElemSignature: make(map[string]string),
	}

	for _, method := range c.methods {
		// Index by full signature
		key := g.fullSignatureKey(method.srcType, method.dstType)
		c.index.bySignature[key] = method.name

		// For non-slice methods, also index as element signature
		if !g.isSliceType(method.srcType) && !g.isSliceType(method.dstType) {
			c.index.byElemSignature[key] = method.name
		}
	}
}

// convertMethod represents a single conversion method.
type convertMethod struct {
	name         string            // method name
	srcType      types.Type        // source type
	dstType      types.Type        // destination type
	fieldMaps    map[string]string // source field -> dest field mapping
	ignoreFields map[string]bool   // fields to ignore
	shallow      bool              // use shallow copy
	pos          token.Position    // method definition position
}

// findConverters finds all interfaces with convertgen:@converter annotation.
func (g *Generator) findConverters(pkg *genkit.Package) []*converter {
	var converters []*converter

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				iface, ok := ts.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}

				// Check for @converter annotation
				doc := docText(gd.Doc)
				if !genkit.HasAnnotation(doc, ToolName, AnnotationConverter) {
					continue
				}

				conv := g.parseConverter(pkg, ts.Name.Name, iface)
				if conv != nil && len(conv.methods) > 0 {
					converters = append(converters, conv)
				}
			}
		}
	}

	return converters
}

// parseConverter parses a converter interface.
func (g *Generator) parseConverter(pkg *genkit.Package, name string, iface *ast.InterfaceType) *converter {
	conv := &converter{
		name: name,
	}

	if iface.Methods == nil {
		return conv
	}

	for _, method := range iface.Methods.List {
		if len(method.Names) == 0 {
			continue
		}

		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		pos := pkg.Fset.Position(method.Names[0].Pos())
		cm := g.parseMethod(pkg, method.Names[0].Name, funcType, docText(method.Doc), pos)
		if cm != nil {
			conv.methods = append(conv.methods, cm)
		}
	}

	return conv
}

// parseMethod parses a conversion method.
func (g *Generator) parseMethod(pkg *genkit.Package, name string, funcType *ast.FuncType, doc string, pos token.Position) *convertMethod {
	// Validate: must have exactly 1 param and 1 result
	if funcType.Params == nil || len(funcType.Params.List) != 1 {
		return nil
	}
	if funcType.Results == nil || len(funcType.Results.List) != 1 {
		return nil
	}

	srcExpr := funcType.Params.List[0].Type
	dstExpr := funcType.Results.List[0].Type

	srcType := pkg.TypesInfo.TypeOf(srcExpr)
	dstType := pkg.TypesInfo.TypeOf(dstExpr)

	if srcType == nil || dstType == nil {
		return nil
	}

	cm := &convertMethod{
		name:         name,
		srcType:      srcType,
		dstType:      dstType,
		fieldMaps:    make(map[string]string),
		ignoreFields: make(map[string]bool),
		pos:          pos,
	}

	// Parse annotations
	annotations := genkit.ParseAnnotations(doc)
	for _, ann := range annotations {
		if ann.Tool != ToolName {
			continue
		}

		switch ann.Name {
		case AnnotationMap:
			// @map(srcField, dstField)
			if len(ann.Flags) >= 2 {
				srcField := strings.TrimSpace(ann.Flags[0])
				dstField := strings.TrimSpace(ann.Flags[1])
				cm.fieldMaps[srcField] = dstField
			}
		case AnnotationIgnore:
			// @ignore(field1, field2, ...)
			for _, field := range ann.Flags {
				cm.ignoreFields[strings.TrimSpace(field)] = true
			}
		case AnnotationShallow:
			cm.shallow = true
		}
	}

	return cm
}

// processPackage generates converter code for a package.
func (g *Generator) processPackage(gen *genkit.Generator, pkg *genkit.Package, converters []*converter) error {
	outPath := genkit.OutputPath(pkg.Dir, pkg.Name+"_convertgen.go")
	gf := gen.NewGeneratedFile(outPath, pkg.GoImportPath())

	// Write header
	gf.P("// Code generated by ", ToolName, ". DO NOT EDIT.")
	gf.P()
	gf.P("package ", pkg.Name)

	// Generate each converter
	for _, conv := range converters {
		g.generateConverter(gf, pkg, conv)
	}

	// Generate test file if requested
	if gen.IncludeTests() {
		testPath := genkit.OutputPath(pkg.Dir, pkg.Name+"_convertgen_test.go")
		tg := gen.NewGeneratedFile(testPath, pkg.GoImportPath())
		g.writeTestHeader(tg, pkg.Name)
		for _, conv := range converters {
			g.generateConverterTest(tg, pkg, conv)
		}
	}

	return nil
}

// generateConverter generates code for a single converter.
func (g *Generator) generateConverter(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter) {
	implName := toLowerFirst(conv.name) + "Impl"
	varName := "Default" + conv.name

	// Build method index for nested conversion lookup
	conv.buildMethodIndex(g, pkg)

	// Generate impl struct
	gf.P()
	gf.P("type ", implName, " struct{}")

	// Generate singleton variable
	gf.P()
	gf.P("// ", varName, " is the default implementation of ", conv.name, ".")
	gf.P("var ", varName, " = &", implName, "{}")

	// Ensure impl satisfies interface
	gf.P()
	gf.P("var _ ", conv.name, " = (*", implName, ")(nil)")

	// Generate methods
	for _, method := range conv.methods {
		g.generateMethod(gf, pkg, implName, conv, method)
	}
}

// fullSignatureKey returns a unique key for a type signature including package path.
func (g *Generator) fullSignatureKey(srcType, dstType types.Type) string {
	return g.fullTypeString(srcType) + "->" + g.fullTypeString(dstType)
}

// fullTypeString returns the full type string including package path.
func (g *Generator) fullTypeString(t types.Type) string {
	if ptr, ok := t.(*types.Pointer); ok {
		return "*" + g.fullTypeString(ptr.Elem())
	}
	if slice, ok := t.(*types.Slice); ok {
		return "[]" + g.fullTypeString(slice.Elem())
	}
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if obj.Pkg() != nil {
			return obj.Pkg().Path() + "." + obj.Name()
		}
		return obj.Name()
	}
	return t.String()
}

// findNestedConvertMethod finds a conversion method for nested types.
func (g *Generator) findNestedConvertMethod(conv *converter, srcType, dstType types.Type) (methodName string, found bool) {
	if conv.index == nil {
		return "", false
	}

	// Try exact match
	key := g.fullSignatureKey(srcType, dstType)
	if method, ok := conv.index.bySignature[key]; ok {
		return method, true
	}

	// For slices, try element type match
	srcSlice, srcIsSlice := srcType.Underlying().(*types.Slice)
	dstSlice, dstIsSlice := dstType.Underlying().(*types.Slice)
	if srcIsSlice && dstIsSlice {
		elemKey := g.fullSignatureKey(srcSlice.Elem(), dstSlice.Elem())
		if method, ok := conv.index.byElemSignature[elemKey]; ok {
			return method, true
		}
	}

	return "", false
}

// generateMethod generates code for a single conversion method.
func (g *Generator) generateMethod(gf *genkit.GeneratedFile, pkg *genkit.Package, implName string, conv *converter, method *convertMethod) {
	srcTypeStr := g.typeString(pkg, method.srcType)
	dstTypeStr := g.typeString(pkg, method.dstType)

	gf.P()
	gf.P("func (c *", implName, ") ", method.name, "(src ", srcTypeStr, ") ", dstTypeStr, " {")

	// Check if it's a slice conversion
	if g.isSliceType(method.srcType) && g.isSliceType(method.dstType) {
		g.generateSliceConversion(gf, pkg, conv, method)
	} else {
		g.generateStructConversion(gf, pkg, conv, method)
	}

	gf.P("}")
}

// generateSliceConversion generates code for slice conversion.
func (g *Generator) generateSliceConversion(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod) {
	dstTypeStr := g.typeString(pkg, method.dstType)

	srcSlice := method.srcType.Underlying().(*types.Slice)
	dstSlice := method.dstType.Underlying().(*types.Slice)

	elemSrcType := srcSlice.Elem()
	elemDstType := dstSlice.Elem()

	gf.P("if src == nil {")
	gf.P("return nil")
	gf.P("}")

	gf.P("dst := make(", dstTypeStr, ", len(src))")
	gf.P("for i, v := range src {")

	// Check if there's a method we can reuse for element conversion
	if methodName, found := g.findNestedConvertMethod(conv, elemSrcType, elemDstType); found {
		gf.P("dst[i] = c.", methodName, "(v)")
	} else {
		// Generate inline conversion
		g.generateInlineElementConversion(gf, pkg, conv, method, elemSrcType, elemDstType)
	}

	gf.P("}")
	gf.P("return dst")
}

// generateStructConversion generates code for struct conversion.
func (g *Generator) generateStructConversion(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod) {
	dstTypeStr := g.typeString(pkg, method.dstType)

	_, srcIsPtr := method.srcType.(*types.Pointer)
	_, dstIsPtr := method.dstType.(*types.Pointer)

	if srcIsPtr {
		gf.P("if src == nil {")
		gf.P("return nil")
		gf.P("}")
	}

	if dstIsPtr {
		dstTypeStrWithoutPtr := g.typeStringWithoutPointer(pkg, method.dstType)
		gf.P("dst := &", dstTypeStrWithoutPtr, "{}")
	} else {
		gf.P("var dst ", dstTypeStr)
	}

	g.generateFieldAssignments(gf, pkg, conv, method, method.srcType, method.dstType, "src", "dst", "")

	gf.P("return dst")
}

// generateInlineElementConversion generates inline conversion for slice elements when no reusable method exists.
func (g *Generator) generateInlineElementConversion(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod, elemSrcType, elemDstType types.Type) {
	_, srcIsPtr := elemSrcType.(*types.Pointer)
	_, dstIsPtr := elemDstType.(*types.Pointer)

	if srcIsPtr {
		gf.P("if v == nil {")
		if dstIsPtr {
			gf.P("dst[i] = nil")
		} else {
			elemDstTypeStr := g.typeString(pkg, elemDstType)
			gf.P("dst[i] = ", elemDstTypeStr, "{}")
		}
		gf.P("continue")
		gf.P("}")
	}

	if dstIsPtr {
		elemDstTypeStr := g.typeStringWithoutPointer(pkg, elemDstType)
		gf.P("dst[i] = &", elemDstTypeStr, "{}")
	} else {
		elemDstTypeStr := g.typeString(pkg, elemDstType)
		gf.P("dst[i] = ", elemDstTypeStr, "{}")
	}

	// Generate field assignments
	g.generateFieldAssignments(gf, pkg, conv, method, elemSrcType, elemDstType, "v", "dst[i]", "\t")
}

// generateFieldAssignments generates field assignment statements.
func (g *Generator) generateFieldAssignments(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod, srcType, dstType types.Type, srcVar, dstVar, indent string) {
	srcFieldsMap := g.getStructFieldsMap(srcType)
	dstFields := g.getStructFields(dstType) // ordered slice

	// Build reverse map: dst field -> src field
	dstToSrc := make(map[string]string)
	for src, dst := range method.fieldMaps {
		dstToSrc[dst] = src
	}

	// Generate assignments for each destination field in definition order
	for _, df := range dstFields {
		dstName := df.field.Name()
		if method.ignoreFields[dstName] {
			continue
		}

		// Determine source field name
		srcName := dstName
		if mappedSrc, ok := dstToSrc[dstName]; ok {
			srcName = mappedSrc
		}

		srcField, exists := srcFieldsMap[srcName]
		if !exists {
			continue
		}

		g.generateFieldAssignment(gf, pkg, conv, method, srcVar, dstVar, srcName, dstName, srcField.Type(), df.field.Type(), indent)
	}
}

// generateFieldAssignment generates a single field assignment.
func (g *Generator) generateFieldAssignment(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod, srcVar, dstVar, srcName, dstName string, srcType, dstType types.Type, indent string) {
	srcAccess := fmt.Sprintf("%s.%s", srcVar, srcName)
	dstAccess := fmt.Sprintf("%s.%s", dstVar, dstName)

	// For shallow copy, direct assignment
	if method.shallow {
		gf.P(indent, dstAccess, " = ", srcAccess)
		return
	}

	// Check if there's a nested convert method we can use
	if methodName, found := g.findNestedConvertMethod(conv, srcType, dstType); found {
		gf.P(indent, dstAccess, " = c.", methodName, "(", srcAccess, ")")
		return
	}

	// Deep copy for complex types
	switch t := srcType.Underlying().(type) {
	case *types.Basic:
		// Basic types - direct assignment
		gf.P(indent, dstAccess, " = ", srcAccess)

	case *types.Pointer:
		g.generatePointerFieldAssignment(gf, pkg, conv, method, srcAccess, dstAccess, srcType, dstType, t, indent)

	case *types.Slice:
		g.generateSliceFieldAssignment(gf, pkg, conv, method, srcAccess, dstAccess, srcType, dstType, t, indent)

	case *types.Map:
		g.generateMapFieldAssignment(gf, pkg, method, srcAccess, dstAccess, t, dstType, indent)

	default:
		gf.P(indent, dstAccess, " = ", srcAccess)
	}
}

// generatePointerFieldAssignment generates deep copy for pointer fields.
func (g *Generator) generatePointerFieldAssignment(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod, srcAccess, dstAccess string, srcType, dstType types.Type, srcPtrType *types.Pointer, indent string) {
	elemType := srcPtrType.Elem()
	dstPtr, isDstPtr := dstType.(*types.Pointer)

	gf.P(indent, "if ", srcAccess, " != nil {")

	// Check if there's a nested convert method for the element types
	if isDstPtr {
		if methodName, found := g.findNestedConvertMethod(conv, srcType, dstType); found {
			gf.P(indent, "\t", dstAccess, " = c.", methodName, "(", srcAccess, ")")
			gf.P(indent, "}")
			return
		}
	}

	switch elemType.Underlying().(type) {
	case *types.Basic:
		// Pointer to basic type (e.g., *int, *string) - deep copy
		if isDstPtr {
			gf.P(indent, "\t", "tmp := *", srcAccess)
			gf.P(indent, "\t", dstAccess, " = &tmp")
		} else {
			gf.P(indent, "\t", dstAccess, " = *", srcAccess)
		}

	case *types.Struct:
		// Pointer to struct - deep copy with nested fields
		if isDstPtr {
			dstTypeStr := g.typeStringWithoutPointer(pkg, dstType)
			gf.P(indent, "\t", dstAccess, " = &", dstTypeStr, "{}")
			g.generateNestedFieldAssignments(gf, pkg, conv, method, srcAccess, dstAccess, elemType, dstPtr.Elem(), indent+"\t")
		} else {
			g.generateNestedFieldAssignments(gf, pkg, conv, method, srcAccess, dstAccess, elemType, dstType, indent+"\t")
		}

	default:
		// Other pointer types - direct assignment (shallow)
		gf.P(indent, "\t", dstAccess, " = ", srcAccess)
	}

	gf.P(indent, "}")
}

// generateNestedFieldAssignments generates assignments for nested struct fields.
func (g *Generator) generateNestedFieldAssignments(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod, srcVar, dstVar string, srcType, dstType types.Type, indent string) {
	srcFieldsMap := g.getStructFieldsMap(srcType)
	dstFields := g.getStructFields(dstType) // ordered slice

	for _, df := range dstFields {
		dstName := df.field.Name()
		srcField, exists := srcFieldsMap[dstName]
		if !exists {
			continue
		}
		g.generateFieldAssignment(gf, pkg, conv, method, srcVar, dstVar, dstName, dstName, srcField.Type(), df.field.Type(), indent)
	}
}

// generateSliceFieldAssignment generates deep copy for slice fields.
func (g *Generator) generateSliceFieldAssignment(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod, srcAccess, dstAccess string, srcType, dstType types.Type, srcSliceType *types.Slice, indent string) {
	dstTypeStr := g.typeString(pkg, dstType)
	elemSrcType := srcSliceType.Elem()

	// Get destination element type
	var elemDstType types.Type
	if dstSlice, ok := dstType.Underlying().(*types.Slice); ok {
		elemDstType = dstSlice.Elem()
	} else {
		elemDstType = elemSrcType // fallback
	}

	gf.P(indent, "if ", srcAccess, " != nil {")
	gf.P(indent, "\t", dstAccess, " = make(", dstTypeStr, ", len(", srcAccess, "))")

	// Check if there's a nested convert method for element conversion
	if methodName, found := g.findNestedConvertMethod(conv, elemSrcType, elemDstType); found {
		gf.P(indent, "\tfor i, v := range ", srcAccess, " {")
		gf.P(indent, "\t\t", dstAccess, "[i] = c.", methodName, "(v)")
		gf.P(indent, "\t}")
	} else if _, isPtr := elemSrcType.Underlying().(*types.Pointer); isPtr && !method.shallow {
		// Deep copy pointer elements
		gf.P(indent, "\tfor i, v := range ", srcAccess, " {")
		gf.P(indent, "\t\tif v != nil {")
		gf.P(indent, "\t\t\ttmp := *v")
		gf.P(indent, "\t\t\t", dstAccess, "[i] = &tmp")
		gf.P(indent, "\t\t}")
		gf.P(indent, "\t}")
	} else {
		gf.P(indent, "\tcopy(", dstAccess, ", ", srcAccess, ")")
	}

	gf.P(indent, "}")
}

// generateMapFieldAssignment generates deep copy for map fields.
func (g *Generator) generateMapFieldAssignment(gf *genkit.GeneratedFile, pkg *genkit.Package, method *convertMethod, srcAccess, dstAccess string, srcMapType *types.Map, dstType types.Type, indent string) {
	dstTypeStr := g.typeString(pkg, dstType)
	valueType := srcMapType.Elem()

	gf.P(indent, "if ", srcAccess, " != nil {")
	gf.P(indent, "\t", dstAccess, " = make(", dstTypeStr, ", len(", srcAccess, "))")
	gf.P(indent, "\tfor k, v := range ", srcAccess, " {")

	// Check if values need deep copy
	if _, isPtr := valueType.Underlying().(*types.Pointer); isPtr && !method.shallow {
		// Deep copy pointer values
		gf.P(indent, "\t\tif v != nil {")
		gf.P(indent, "\t\t\ttmp := *v")
		gf.P(indent, "\t\t\t", dstAccess, "[k] = &tmp")
		gf.P(indent, "\t\t} else {")
		gf.P(indent, "\t\t\t", dstAccess, "[k] = nil")
		gf.P(indent, "\t\t}")
	} else {
		gf.P(indent, "\t\t", dstAccess, "[k] = v")
	}

	gf.P(indent, "\t}")
	gf.P(indent, "}")
}

// typeString returns the string representation of a type.
func (g *Generator) typeString(pkg *genkit.Package, t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p.Path() == pkg.PkgPath {
			return ""
		}
		return p.Name()
	})
}

// typeStringWithoutPointer returns the type string without leading pointer.
func (g *Generator) typeStringWithoutPointer(pkg *genkit.Package, t types.Type) string {
	if ptr, ok := t.(*types.Pointer); ok {
		return g.typeString(pkg, ptr.Elem())
	}
	return g.typeString(pkg, t)
}

// isSliceType returns true if the type is a slice type.
func (g *Generator) isSliceType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Slice)
	return ok
}

// structField represents a struct field with its index for stable ordering.
type structField struct {
	index int
	field *types.Var
}

// getStructFields returns the exported fields of a struct type as an ordered slice.
func (g *Generator) getStructFields(t types.Type) []structField {
	return g.getStructFieldsFromType(t)
}

// getStructFieldsFromType returns struct fields from a type in definition order.
func (g *Generator) getStructFieldsFromType(t types.Type) []structField {
	var fields []structField

	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Get underlying struct
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return fields
	}

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field.Exported() {
			fields = append(fields, structField{index: i, field: field})
		}
	}

	return fields
}

// getStructFieldsMap returns a map of field name to field for quick lookup.
func (g *Generator) getStructFieldsMap(t types.Type) map[string]*types.Var {
	fields := make(map[string]*types.Var)

	// Unwrap pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Get underlying struct
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return fields
	}

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field.Exported() {
			fields[field.Name()] = field
		}
	}

	return fields
}

// toLowerFirst converts the first character to lowercase.
func toLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// docText extracts text from a comment group.
func docText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return cg.Text()
}

// writeTestHeader writes the test file header.
func (g *Generator) writeTestHeader(gf *genkit.GeneratedFile, pkgName string) {
	gf.P("// Code generated by ", ToolName, ". DO NOT EDIT.")
	gf.P()
	gf.P("package ", pkgName)
}

// generateConverterTest generates tests for a single converter.
func (g *Generator) generateConverterTest(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter) {
	varName := "Default" + conv.name

	// Build method index for nested conversion lookup
	conv.buildMethodIndex(g, pkg)

	for _, method := range conv.methods {
		g.generateMethodTest(gf, pkg, conv, varName, method)
	}
}

// generateMethodTest generates tests for a single conversion method.
func (g *Generator) generateMethodTest(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, varName string, method *convertMethod) {
	srcTypeStr := g.typeString(pkg, method.srcType)
	dstTypeStr := g.typeString(pkg, method.dstType)

	_, srcIsPtr := method.srcType.(*types.Pointer)
	_, dstIsPtr := method.dstType.(*types.Pointer)
	srcIsSlice := g.isSliceType(method.srcType)
	dstIsSlice := g.isSliceType(method.dstType)

	testFuncName := "Test" + conv.name + "_" + method.name

	// Generate test function
	gf.P()
	gf.P("func ", testFuncName, "(t *", genkit.GoImportPath("testing").Ident("T"), ") {")

	// Test nil input
	if srcIsPtr || srcIsSlice {
		gf.P("// Test nil input")
		gf.P("t.Run(\"nil_input\", func(t *testing.T) {")
		gf.P("var src ", srcTypeStr)
		gf.P("dst := ", varName, ".", method.name, "(src)")
		if dstIsPtr || dstIsSlice {
			gf.P("if dst != nil {")
			gf.P("t.Errorf(\"expected nil, got %v\", dst)")
			gf.P("}")
		}
		gf.P("})")
		gf.P()
	}

	// Test basic conversion
	gf.P("// Test basic conversion")
	gf.P("t.Run(\"basic\", func(t *testing.T) {")

	if srcIsSlice {
		g.generateSliceTestCase(gf, pkg, conv, varName, method, srcTypeStr, dstTypeStr)
	} else {
		g.generateStructTestCase(gf, pkg, conv, varName, method, srcTypeStr, dstTypeStr, srcIsPtr, dstIsPtr)
	}

	gf.P("})")
	gf.P("}")
}

// generateStructTestCase generates test case for struct conversion.
func (g *Generator) generateStructTestCase(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, varName string, method *convertMethod, srcTypeStr, dstTypeStr string, srcIsPtr, dstIsPtr bool) {
	srcFields := g.getStructFields(method.srcType)

	// Build reverse map: dst field -> src field
	dstToSrc := make(map[string]string)
	for src, dst := range method.fieldMaps {
		dstToSrc[dst] = src
	}

	// Create source struct
	if srcIsPtr {
		srcTypeStrWithoutPtr := g.typeStringWithoutPointer(pkg, method.srcType)
		gf.P("src := &", srcTypeStrWithoutPtr, "{")
	} else {
		gf.P("src := ", srcTypeStr, "{")
	}

	// Generate test values for each field
	for _, sf := range srcFields {
		fieldName := sf.field.Name()
		testValue := g.getTestValue(pkg, sf.field.Type(), fieldName)
		if testValue != "" {
			gf.P(fieldName, ": ", testValue, ",")
		}
	}
	gf.P("}")

	// Call conversion method
	gf.P("dst := ", varName, ".", method.name, "(src)")
	gf.P()

	// Verify result is not nil for pointer return types
	if dstIsPtr {
		gf.P("if dst == nil {")
		gf.P("t.Fatal(\"expected non-nil result\")")
		gf.P("}")
		gf.P()
	}

	// Verify field mappings
	dstFields := g.getStructFields(method.dstType)
	for _, df := range dstFields {
		dstName := df.field.Name()
		if method.ignoreFields[dstName] {
			continue
		}

		srcName := dstName
		if mapped, ok := dstToSrc[dstName]; ok {
			srcName = mapped
		}

		// Check if source field exists
		srcFieldsMap := g.getStructFieldsMap(method.srcType)
		if _, exists := srcFieldsMap[srcName]; !exists {
			continue
		}

		// Generate assertion based on field type
		g.generateFieldAssertion(gf, pkg, conv, method, srcName, dstName, df.field.Type(), srcIsPtr, dstIsPtr)
	}
}

// generateSliceTestCase generates test case for slice conversion.
func (g *Generator) generateSliceTestCase(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, varName string, method *convertMethod, srcTypeStr, dstTypeStr string) {
	srcSlice := method.srcType.Underlying().(*types.Slice)
	elemSrcType := srcSlice.Elem()

	_, elemIsPtr := elemSrcType.(*types.Pointer)

	// Create source slice
	gf.P("src := ", srcTypeStr, "{")

	// Generate test element
	if elemIsPtr {
		elemTypeStr := g.typeStringWithoutPointer(pkg, elemSrcType)
		gf.P("&", elemTypeStr, "{},")
	} else {
		elemTypeStr := g.typeString(pkg, elemSrcType)
		gf.P(elemTypeStr, "{},")
	}
	gf.P("}")

	// Call conversion method
	gf.P("dst := ", varName, ".", method.name, "(src)")
	gf.P()

	// Verify length
	gf.P("if len(dst) != len(src) {")
	gf.P("t.Errorf(\"expected length %d, got %d\", len(src), len(dst))")
	gf.P("}")
}

// generateFieldAssertion generates assertion for a single field.
func (g *Generator) generateFieldAssertion(gf *genkit.GeneratedFile, pkg *genkit.Package, conv *converter, method *convertMethod, srcName, dstName string, dstType types.Type, srcIsPtr, dstIsPtr bool) {
	srcAccess := "src." + srcName
	dstAccess := "dst." + dstName

	// Check if there's a nested convert method
	srcFieldsMap := g.getStructFieldsMap(method.srcType)
	srcField := srcFieldsMap[srcName]
	if srcField == nil {
		return
	}

	srcType := srcField.Type()

	// For nested conversions, just check non-nil
	if _, found := g.findNestedConvertMethod(conv, srcType, dstType); found {
		if _, isPtr := dstType.(*types.Pointer); isPtr {
			gf.P("if ", dstAccess, " == nil {")
			gf.P("t.Error(\"expected ", dstName, " to be non-nil\")")
			gf.P("}")
		}
		return
	}

	// For basic types, compare values
	switch dstType.Underlying().(type) {
	case *types.Basic:
		gf.P("if ", dstAccess, " != ", srcAccess, " {")
		gf.P("t.Errorf(\"expected ", dstName, " = %v, got %v\", ", srcAccess, ", ", dstAccess, ")")
		gf.P("}")
	case *types.Pointer:
		// For pointers, check deep copy
		gf.P("if ", srcAccess, " != nil {")
		gf.P("if ", dstAccess, " == nil {")
		gf.P("t.Error(\"expected ", dstName, " to be non-nil\")")
		gf.P("}")
		if !method.shallow {
			gf.P("if ", dstAccess, " == ", srcAccess, " {")
			gf.P("t.Error(\"expected ", dstName, " to be deep copied\")")
			gf.P("}")
		}
		gf.P("}")
	case *types.Slice:
		gf.P("if len(", dstAccess, ") != len(", srcAccess, ") {")
		gf.P("t.Errorf(\"expected ", dstName, " length %d, got %d\", len(", srcAccess, "), len(", dstAccess, "))")
		gf.P("}")
	case *types.Map:
		gf.P("if len(", dstAccess, ") != len(", srcAccess, ") {")
		gf.P("t.Errorf(\"expected ", dstName, " length %d, got %d\", len(", srcAccess, "), len(", dstAccess, "))")
		gf.P("}")
	}
}

// getTestValue returns a test value for a given type.
func (g *Generator) getTestValue(pkg *genkit.Package, t types.Type, fieldName string) string {
	switch ut := t.Underlying().(type) {
	case *types.Basic:
		switch ut.Kind() {
		case types.Bool:
			return "true"
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			return "42"
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			return "42"
		case types.Float32, types.Float64:
			return "3.14"
		case types.String:
			return fmt.Sprintf("%q", "test_"+strings.ToLower(fieldName))
		}
	case *types.Pointer:
		elemValue := g.getTestValue(pkg, ut.Elem(), fieldName)
		if elemValue != "" {
			// For basic types, we can use a helper
			if _, isBasic := ut.Elem().Underlying().(*types.Basic); isBasic {
				return "func() *" + g.typeString(pkg, ut.Elem()) + " { v := " + elemValue + "; return &v }()"
			}
			// For struct types, return pointer to struct
			return "&" + g.typeString(pkg, ut.Elem()) + "{}"
		}
	case *types.Slice:
		elemTypeStr := g.typeString(pkg, ut.Elem())
		elemValue := g.getSliceElemTestValue(pkg, ut.Elem(), fieldName)
		return "[]" + elemTypeStr + "{" + elemValue + "}"
	case *types.Map:
		keyTypeStr := g.typeString(pkg, ut.Key())
		valTypeStr := g.typeString(pkg, ut.Elem())
		keyValue := g.getMapKeyTestValue(pkg, ut.Key())
		valValue := g.getMapValueTestValue(pkg, ut.Elem(), fieldName)
		return "map[" + keyTypeStr + "]" + valTypeStr + "{" + keyValue + ": " + valValue + "}"
	case *types.Struct:
		return g.typeString(pkg, t) + "{}"
	}
	return ""
}

// getSliceElemTestValue returns a test value for slice element.
func (g *Generator) getSliceElemTestValue(pkg *genkit.Package, elemType types.Type, fieldName string) string {
	switch ut := elemType.Underlying().(type) {
	case *types.Basic:
		return g.getTestValue(pkg, elemType, fieldName)
	case *types.Pointer:
		if _, isBasic := ut.Elem().Underlying().(*types.Basic); isBasic {
			return g.getTestValue(pkg, elemType, fieldName)
		}
		return "&" + g.typeString(pkg, ut.Elem()) + "{}"
	case *types.Struct:
		return g.typeString(pkg, elemType) + "{}"
	}
	return ""
}

// getMapKeyTestValue returns a test value for map key.
func (g *Generator) getMapKeyTestValue(pkg *genkit.Package, keyType types.Type) string {
	if basic, ok := keyType.Underlying().(*types.Basic); ok {
		switch basic.Kind() {
		case types.String:
			return `"key"`
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			return "1"
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			return "1"
		}
	}
	return `"key"`
}

// getMapValueTestValue returns a test value for map value.
func (g *Generator) getMapValueTestValue(pkg *genkit.Package, valType types.Type, fieldName string) string {
	switch ut := valType.Underlying().(type) {
	case *types.Basic:
		return g.getTestValue(pkg, valType, fieldName)
	case *types.Pointer:
		if _, isBasic := ut.Elem().Underlying().(*types.Basic); isBasic {
			return g.getTestValue(pkg, valType, fieldName)
		}
		return "&" + g.typeString(pkg, ut.Elem()) + "{}"
	}
	return "nil"
}
