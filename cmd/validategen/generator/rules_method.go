// Package generator provides validation code generation functionality.
package generator

import "strings"

func init() {
	DefaultRegistry.Register("method", PriorityNested, func() Rule { return &MethodRule{} })
}

// MethodRule validates by calling a method on the field.
type MethodRule struct{}

func (r *MethodRule) Name() string              { return "method" }
func (r *MethodRule) RequiredRegex() []string   { return nil }

func (r *MethodRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fieldName := ctx.FieldName
	fieldType := ctx.FieldType
	methodName := ctx.Param
	fmtSprintf := fmtSprintf()
	g := ctx.G

	if IsSliceType(fieldType) {
		g.P("for _i, _v := range x.", fieldName, " {")
		elemType := strings.TrimPrefix(fieldType, "[]")
		if IsPointerType(elemType) {
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
	} else if IsMapType(fieldType) {
		g.P("for _k, _v := range x.", fieldName, " {")
		valueType := ExtractMapValueType(fieldType)
		if IsPointerType(valueType) {
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
	} else if IsPointerType(fieldType) {
		g.P("if x.", fieldName, " != nil {")
		g.P("if err := x.", fieldName, ".", methodName, "(); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, ": %v\", err))")
		g.P("}")
		g.P("}")
	} else {
		g.P("if err := x.", fieldName, ".", methodName, "(); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, ": %v\", err))")
		g.P("}")
	}
}

func (r *MethodRule) Validate(ctx *ValidateContext) {
	if ctx.Param == "" {
		ctx.Collector.Error(ErrCodeMethodMissingParam, "@method annotation requires a method name parameter", ctx.Field.Pos)
		return
	}
	if IsBuiltinType(ctx.Field.Type) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@method annotation can only be applied to custom types, got builtin type %s",
			ctx.Field.Type,
		)
	}
}
