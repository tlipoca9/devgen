// Package generator provides validation code generation functionality.
package generator

func init() {
	DefaultRegistry.Register("required", PriorityRequired, func() Rule { return &RequiredRule{} })
}

// RequiredRule validates that a field is not empty/zero.
type RequiredRule struct{}

func (r *RequiredRule) Name() string              { return "required" }
func (r *RequiredRule) RequiredRegex() []string   { return nil }

func (r *RequiredRule) Generate(ctx *GenerateContext) {
	fieldName := ctx.FieldName
	fieldType := ctx.FieldType
	g := ctx.G

	if IsStringType(fieldType) {
		g.P("if x.", fieldName, " == \"\" {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if IsSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") == 0 {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if IsPointerType(fieldType) {
		g.P("if x.", fieldName, " == nil {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if IsBoolType(fieldType) {
		g.P("if !x.", fieldName, " {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if IsNumericType(fieldType) {
		g.P("if x.", fieldName, " == 0 {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	}
}

func (r *RequiredRule) Validate(ctx *ValidateContext) {
	underlyingType := ctx.UnderlyingType
	if !IsStringType(underlyingType) && !IsSliceOrMapType(underlyingType) &&
		!IsPointerType(underlyingType) && !IsBoolType(underlyingType) &&
		!IsNumericType(underlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@required annotation requires string, slice, map, pointer, bool, or numeric underlying type, got %s",
			underlyingType,
		)
	}
}
