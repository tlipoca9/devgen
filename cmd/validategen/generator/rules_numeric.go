// Package generator provides validation code generation functionality.
package generator

func init() {
	DefaultRegistry.Register("min", PriorityRange, func() Rule { return &MinRule{} })
	DefaultRegistry.Register("max", PriorityRange+1, func() Rule { return &MaxRule{} })
	DefaultRegistry.Register("gt", PriorityRange+3, func() Rule { return &GtRule{} })
	DefaultRegistry.Register("gte", PriorityRange+4, func() Rule { return &GteRule{} })
	DefaultRegistry.Register("lt", PriorityRange+5, func() Rule { return &LtRule{} })
	DefaultRegistry.Register("lte", PriorityRange+6, func() Rule { return &LteRule{} })
	DefaultRegistry.Register("len", PriorityRange+2, func() Rule { return &LenRule{} })
	DefaultRegistry.Register("eq", PriorityEquality, func() Rule { return &EqRule{} })
	DefaultRegistry.Register("ne", PriorityEquality+1, func() Rule { return &NeRule{} })
	DefaultRegistry.Register("default", PriorityDefault, func() Rule { return &DefaultRule{} })
}

// comparisonRule is a base type for comparison rules.
type comparisonRule struct {
	name    string
	op      string // "<", ">", "<=", ">="
	msgVerb string // "at least", "at most", "greater than", "less than"
	lenOp   string // for string/slice length comparison
	lenVerb string
}

// RequiredRegex returns nil (no predefined regex needed).
func (r *comparisonRule) RequiredRegex() []string { return nil }

func (r *comparisonRule) generateNumeric(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fmtSprintf := fmtSprintf()
	fieldName := ctx.FieldName
	fieldType := ctx.FieldType
	g := ctx.G

	if IsStringType(fieldType) {
		g.P("if len(x.", fieldName, ") ", r.lenOp, " ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be ", r.lenVerb, " ", ctx.Param, " characters, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if IsPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") ", r.lenOp, " ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be ", r.lenVerb, " ", ctx.Param, " characters, got %d\", len(*x.", fieldName, ")))")
		g.P("}")
	} else if IsSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") ", r.lenOp, " ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have ", r.lenVerb, " ", ctx.Param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if IsNumericType(fieldType) {
		g.P("if x.", fieldName, " ", r.op, " ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be ", r.msgVerb, " ", ctx.Param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if IsPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " ", r.op, " ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be ", r.msgVerb, " ", ctx.Param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (r *comparisonRule) validateNumeric(ctx *ValidateContext) {
	underlyingType := ctx.UnderlyingType
	if !IsStringType(underlyingType) && !IsPointerToStringType(underlyingType) &&
		!IsSliceOrMapType(underlyingType) &&
		!IsNumericType(underlyingType) && !IsPointerToNumericType(underlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@%s annotation requires string, slice, map, or numeric underlying type, got %s",
			r.name,
			underlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@%s annotation requires a value parameter", r.name)
	} else if !isValidNumber(ctx.Param) {
		ctx.Collector.Errorf(ErrCodeInvalidParamType, ctx.Field.Pos, "@%s parameter must be a number, got %q", r.name, ctx.Param)
	}
}

// MinRule validates minimum value/length.
type MinRule struct{}

func (r *MinRule) Name() string              { return "min" }
func (r *MinRule) RequiredRegex() []string   { return nil }

func (r *MinRule) Generate(ctx *GenerateContext) {
	cr := &comparisonRule{name: "min", op: "<", msgVerb: "at least", lenOp: "<", lenVerb: "at least"}
	cr.generateNumeric(ctx)
}

func (r *MinRule) Validate(ctx *ValidateContext) {
	cr := &comparisonRule{name: "min"}
	cr.validateNumeric(ctx)
}

// MaxRule validates maximum value/length.
type MaxRule struct{}

func (r *MaxRule) Name() string              { return "max" }
func (r *MaxRule) RequiredRegex() []string   { return nil }

func (r *MaxRule) Generate(ctx *GenerateContext) {
	cr := &comparisonRule{name: "max", op: ">", msgVerb: "at most", lenOp: ">", lenVerb: "at most"}
	cr.generateNumeric(ctx)
}

func (r *MaxRule) Validate(ctx *ValidateContext) {
	cr := &comparisonRule{name: "max"}
	cr.validateNumeric(ctx)
}

// GtRule validates greater than.
type GtRule struct{}

func (r *GtRule) Name() string              { return "gt" }
func (r *GtRule) RequiredRegex() []string   { return nil }

func (r *GtRule) Generate(ctx *GenerateContext) {
	cr := &comparisonRule{name: "gt", op: "<=", msgVerb: "greater than", lenOp: "<=", lenVerb: "more than"}
	cr.generateNumeric(ctx)
}

func (r *GtRule) Validate(ctx *ValidateContext) {
	cr := &comparisonRule{name: "gt"}
	cr.validateNumeric(ctx)
}

// GteRule validates greater than or equal.
type GteRule struct{}

func (r *GteRule) Name() string              { return "gte" }
func (r *GteRule) RequiredRegex() []string   { return nil }

func (r *GteRule) Generate(ctx *GenerateContext) {
	cr := &comparisonRule{name: "gte", op: "<", msgVerb: "at least", lenOp: "<", lenVerb: "at least"}
	cr.generateNumeric(ctx)
}

func (r *GteRule) Validate(ctx *ValidateContext) {
	cr := &comparisonRule{name: "gte"}
	cr.validateNumeric(ctx)
}

// LtRule validates less than.
type LtRule struct{}

func (r *LtRule) Name() string              { return "lt" }
func (r *LtRule) RequiredRegex() []string   { return nil }

func (r *LtRule) Generate(ctx *GenerateContext) {
	cr := &comparisonRule{name: "lt", op: ">=", msgVerb: "less than", lenOp: ">=", lenVerb: "less than"}
	cr.generateNumeric(ctx)
}

func (r *LtRule) Validate(ctx *ValidateContext) {
	cr := &comparisonRule{name: "lt"}
	cr.validateNumeric(ctx)
}

// LteRule validates less than or equal.
type LteRule struct{}

func (r *LteRule) Name() string              { return "lte" }
func (r *LteRule) RequiredRegex() []string   { return nil }

func (r *LteRule) Generate(ctx *GenerateContext) {
	cr := &comparisonRule{name: "lte", op: ">", msgVerb: "at most", lenOp: ">", lenVerb: "at most"}
	cr.generateNumeric(ctx)
}

func (r *LteRule) Validate(ctx *ValidateContext) {
	cr := &comparisonRule{name: "lte"}
	cr.validateNumeric(ctx)
}

// LenRule validates exact length.
type LenRule struct{}

func (r *LenRule) Name() string              { return "len" }
func (r *LenRule) RequiredRegex() []string   { return nil }

func (r *LenRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fmtSprintf := fmtSprintf()
	fieldName := ctx.FieldName
	fieldType := ctx.FieldType
	g := ctx.G

	if IsStringType(fieldType) {
		g.P("if len(x.", fieldName, ") != ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be exactly ", ctx.Param, " characters, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if IsSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") != ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have exactly ", ctx.Param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	}
}

func (r *LenRule) Validate(ctx *ValidateContext) {
	underlyingType := ctx.UnderlyingType
	if !IsStringType(underlyingType) && !IsSliceOrMapType(underlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@len annotation requires string, slice, or map underlying type, got %s",
			underlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@len annotation requires a value parameter")
	} else if !isValidNumber(ctx.Param) {
		ctx.Collector.Errorf(ErrCodeInvalidParamType, ctx.Field.Pos, "@len parameter must be a number, got %q", ctx.Param)
	}
}

// EqRule validates equality.
type EqRule struct{}

func (r *EqRule) Name() string              { return "eq" }
func (r *EqRule) RequiredRegex() []string   { return nil }

func (r *EqRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fmtSprintf := fmtSprintf()
	fieldName := ctx.FieldName
	fieldType := ctx.FieldType
	g := ctx.G

	// Escape string parameters for safe embedding in generated code
	escapedParam := escapeString(ctx.Param)

	if IsStringType(fieldType) {
		g.P("if x.", fieldName, " != \"", escapedParam, "\" {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must equal ", escapedParam, ", got %q\", x.", fieldName, "))")
		g.P("}")
	} else if IsPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " != \"", escapedParam, "\" {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must equal ", escapedParam, ", got %q\", *x.", fieldName, "))")
		g.P("}")
	} else if IsNumericType(fieldType) || IsBoolType(fieldType) {
		g.P("if x.", fieldName, " != ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must equal ", ctx.Param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if IsPointerToNumericType(fieldType) || IsPointerToBoolType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " != ", ctx.Param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must equal ", ctx.Param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (r *EqRule) Validate(ctx *ValidateContext) {
	if !IsScalarOrPointerType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@eq annotation requires string, numeric, or bool underlying type, got %s",
			ctx.UnderlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@eq annotation requires a value parameter")
	}
}

// NeRule validates inequality.
type NeRule struct{}

func (r *NeRule) Name() string              { return "ne" }
func (r *NeRule) RequiredRegex() []string   { return nil }

func (r *NeRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fieldName := ctx.FieldName
	fieldType := ctx.FieldType
	g := ctx.G

	// Escape string parameters for safe embedding in generated code
	escapedParam := escapeString(ctx.Param)

	if IsStringType(fieldType) {
		g.P("if x.", fieldName, " == \"", escapedParam, "\" {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", escapedParam, "\")")
		g.P("}")
	} else if IsPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " == \"", escapedParam, "\" {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", escapedParam, "\")")
		g.P("}")
	} else if IsNumericType(fieldType) || IsBoolType(fieldType) {
		g.P("if x.", fieldName, " == ", ctx.Param, " {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", ctx.Param, "\")")
		g.P("}")
	} else if IsPointerToNumericType(fieldType) || IsPointerToBoolType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " == ", ctx.Param, " {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", ctx.Param, "\")")
		g.P("}")
	}
}

func (r *NeRule) Validate(ctx *ValidateContext) {
	if !IsScalarOrPointerType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@ne annotation requires string, numeric, or bool underlying type, got %s",
			ctx.UnderlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@ne annotation requires a value parameter")
	}
}

// DefaultRule sets default values.
type DefaultRule struct{}

func (r *DefaultRule) Name() string              { return "default" }
func (r *DefaultRule) RequiredRegex() []string   { return nil }

func (r *DefaultRule) Generate(ctx *GenerateContext) {
	// Default is handled separately in SetDefaults generation
}

func (r *DefaultRule) Validate(ctx *ValidateContext) {
	if !IsScalarOrPointerType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@default annotation requires string, numeric, or bool underlying type, got %s",
			ctx.UnderlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@default annotation requires a value parameter")
	} else if IsNumericType(ctx.UnderlyingType) || IsPointerToNumericType(ctx.UnderlyingType) {
		if !isValidNumber(ctx.Param) {
			ctx.Collector.Errorf(ErrCodeInvalidParamType, ctx.Field.Pos, "@default parameter must be a number for numeric field, got %q", ctx.Param)
		}
	} else if IsBoolType(ctx.UnderlyingType) || IsPointerToBoolType(ctx.UnderlyingType) {
		if ctx.Param != "true" && ctx.Param != "false" && ctx.Param != "1" && ctx.Param != "0" {
			ctx.Collector.Errorf(ErrCodeInvalidParamType, ctx.Field.Pos, "@default parameter must be true/false for bool field, got %q", ctx.Param)
		}
	}
}
