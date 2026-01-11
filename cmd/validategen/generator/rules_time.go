// Package generator provides validation code generation functionality.
package generator

import (
	"time"

	"github.com/tlipoca9/devgen/genkit"
)

func init() {
	DefaultRegistry.Register("duration", PriorityFormat+7, func() Rule { return &DurationRule{} })
	DefaultRegistry.Register("duration_min", PriorityFormat+7, func() Rule { return &DurationMinRule{} })
	DefaultRegistry.Register("duration_max", PriorityFormat+8, func() Rule { return &DurationMaxRule{} })
}

// DurationRule validates duration format.
type DurationRule struct{}

func (r *DurationRule) Name() string              { return "duration" }
func (r *DurationRule) RequiredRegex() []string   { return nil }

func (r *DurationRule) Generate(ctx *GenerateContext) {
	// Duration generation is handled by GenerateDurationCombined
}

func (r *DurationRule) Validate(ctx *ValidateContext) {
	if !IsStringType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@duration annotation requires string underlying type, got %s",
			ctx.UnderlyingType,
		)
	}
}

// DurationMinRule validates minimum duration.
type DurationMinRule struct{}

func (r *DurationMinRule) Name() string              { return "duration_min" }
func (r *DurationMinRule) RequiredRegex() []string   { return nil }

func (r *DurationMinRule) Generate(ctx *GenerateContext) {
	// Duration generation is handled by GenerateDurationCombined
}

func (r *DurationMinRule) Validate(ctx *ValidateContext) {
	if !IsStringType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@duration_min annotation requires string underlying type, got %s",
			ctx.UnderlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@duration_min annotation requires a duration parameter")
	} else if !isValidDuration(ctx.Param) {
		ctx.Collector.Errorf(ErrCodeInvalidParamType, ctx.Field.Pos, "@duration_min parameter must be a valid duration (e.g., 1h, 30m, 500ms), got %q", ctx.Param)
	}
}

// DurationMaxRule validates maximum duration.
type DurationMaxRule struct{}

func (r *DurationMaxRule) Name() string              { return "duration_max" }
func (r *DurationMaxRule) RequiredRegex() []string   { return nil }

func (r *DurationMaxRule) Generate(ctx *GenerateContext) {
	// Duration generation is handled by GenerateDurationCombined
}

func (r *DurationMaxRule) Validate(ctx *ValidateContext) {
	if !IsStringType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@duration_max annotation requires string underlying type, got %s",
			ctx.UnderlyingType,
		)
	}
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@duration_max annotation requires a duration parameter")
	} else if !isValidDuration(ctx.Param) {
		ctx.Collector.Errorf(ErrCodeInvalidParamType, ctx.Field.Pos, "@duration_max parameter must be a valid duration (e.g., 1h, 30m, 500ms), got %q", ctx.Param)
	}
}

// GenerateDurationCombined generates combined duration validation code.
func GenerateDurationCombined(
	g *genkit.GeneratedFile,
	fieldName string,
	checkFormat, hasMin bool,
	minParam string,
	hasMax bool,
	maxParam string,
) {
	var minDur, maxDur time.Duration
	if hasMin && minParam != "" {
		if dur, err := time.ParseDuration(minParam); err == nil {
			minDur = dur
		} else {
			hasMin = false
		}
	}
	if hasMax && maxParam != "" {
		if dur, err := time.ParseDuration(maxParam); err == nil {
			maxDur = dur
		} else {
			hasMax = false
		}
	}

	fmtSprintf := fmtSprintf()
	timePkg := genkit.GoImportPath("time")

	if checkFormat && !hasMin && !hasMax {
		g.P("if x.", fieldName, " != \"\" {")
		g.P("if _, err := ", timePkg.Ident("ParseDuration"), "(x.", fieldName, "); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid duration (e.g., 1h30m, 500ms), got %q\", x.", fieldName, "))")
		g.P("}")
		g.P("}")
		return
	}

	g.P("if x.", fieldName, " != \"\" {")
	g.P("if _dur, _err := ", timePkg.Ident("ParseDuration"), "(x.", fieldName, "); _err != nil {")
	if checkFormat {
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid duration (e.g., 1h30m, 500ms), got %q\", x.", fieldName, "))")
	}
	g.P("} else {")
	if hasMin {
		minArgs := []any{"if _dur < "}
		minArgs = append(minArgs, durationToExpr(minDur)...)
		minArgs = append(minArgs, " {")
		g.P(minArgs...)
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", minParam, ", got %s\", x.", fieldName, "))")
		g.P("}")
	}
	if hasMax {
		maxArgs := []any{"if _dur > "}
		maxArgs = append(maxArgs, durationToExpr(maxDur)...)
		maxArgs = append(maxArgs, " {")
		g.P(maxArgs...)
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", maxParam, ", got %s\", x.", fieldName, "))")
		g.P("}")
	}
	g.P("}")
	g.P("}")
}

// durationToExpr converts a duration to a readable Go expression.
func durationToExpr(d time.Duration) []any {
	timePkg := genkit.GoImportPath("time")

	switch {
	case d%time.Hour == 0:
		hours := int64(d / time.Hour)
		if hours == 1 {
			return []any{timePkg.Ident("Hour")}
		}
		return []any{hours, "*", timePkg.Ident("Hour")}
	case d%time.Minute == 0:
		minutes := int64(d / time.Minute)
		if minutes == 1 {
			return []any{timePkg.Ident("Minute")}
		}
		return []any{minutes, "*", timePkg.Ident("Minute")}
	case d%time.Second == 0:
		seconds := int64(d / time.Second)
		if seconds == 1 {
			return []any{timePkg.Ident("Second")}
		}
		return []any{seconds, "*", timePkg.Ident("Second")}
	case d%time.Millisecond == 0:
		ms := int64(d / time.Millisecond)
		if ms == 1 {
			return []any{timePkg.Ident("Millisecond")}
		}
		return []any{ms, "*", timePkg.Ident("Millisecond")}
	case d%time.Microsecond == 0:
		us := int64(d / time.Microsecond)
		if us == 1 {
			return []any{timePkg.Ident("Microsecond")}
		}
		return []any{us, "*", timePkg.Ident("Microsecond")}
	default:
		ns := d.Nanoseconds()
		if ns == 1 {
			return []any{timePkg.Ident("Nanosecond")}
		}
		return []any{ns, "*", timePkg.Ident("Nanosecond")}
	}
}
