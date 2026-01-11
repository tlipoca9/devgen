// Package generator provides validation code generation functionality.
package generator

import "github.com/tlipoca9/devgen/genkit"

func init() {
	DefaultRegistry.Register("cpu", PriorityFormat+14, func() Rule {
		return &CPURule{k8sResourceRule{
			name:       "cpu",
			validUnits: `[]string{"", "m"}`,
			unitDesc:   "only divisor's values 1m and 1 are supported with the cpu resource",
		}}
	})
	DefaultRegistry.Register("memory", PriorityFormat+15, func() Rule {
		return &MemoryRule{k8sResourceRule{
			name:       "memory",
			validUnits: `[]string{"", "K", "M", "G", "T", "P", "E", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei"}`,
			unitDesc:   "only divisor's values 1, 1K, 1M, 1G, 1T, 1P, 1E, 1Ki, 1Mi, 1Gi, 1Ti, 1Pi, 1Ei are supported with the memory resource",
		}}
	})
	DefaultRegistry.Register("disk", PriorityFormat+16, func() Rule {
		return &DiskRule{k8sResourceRule{
			name:       "disk",
			validUnits: `[]string{"", "K", "M", "G", "T", "P", "E", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei"}`,
			unitDesc:   "only divisor's values 1, 1K, 1M, 1G, 1T, 1P, 1E, 1Ki, 1Mi, 1Gi, 1Ti, 1Pi, 1Ei are supported with the disk resource",
		}}
	})
}

// k8sResourceRule is a base type for Kubernetes resource validation rules.
// This eliminates the code duplication between cpu, memory, and disk rules.
type k8sResourceRule struct {
	name       string
	validUnits string
	unitDesc   string
}

// RequiredRegex returns nil (no predefined regex needed).
func (r *k8sResourceRule) RequiredRegex() []string { return nil }

func (r *k8sResourceRule) generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	stringsTrimLeft := genkit.GoImportPath("strings").Ident("TrimLeft")
	slicesContains := genkit.GoImportPath("slices").Ident("Contains")
	fieldName := ctx.FieldName
	g := ctx.G

	g.P("if x.", fieldName, " != \"\" {")
	g.P("_qty, err := ", genkit.GoImportPath("k8s.io/apimachinery/pkg/api/resource").Ident("ParseQuantity"), "(x.", fieldName, ")")
	g.P("if err != nil {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " invalid quantity: %v\", err))")
	g.P("} else if _qty.Sign() == -1 {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be non-negative, got %s\", x.", fieldName, "))")
	g.P("} else {")
	g.P("// https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation.go#L2978")
	g.P("_validUnits := ", r.validUnits)
	g.P("_unit := ", stringsTrimLeft, "(_qty.String(), \"0123456789.\")")
	g.P("if !", slicesContains, "(_validUnits, _unit) {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " invalid format: ", r.unitDesc, ", got %s\", x.", fieldName, "))")
	g.P("}")
	g.P("}")
	g.P("}")
}

func (r *k8sResourceRule) validate(ctx *ValidateContext) {
	if !IsStringType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@%s annotation requires string underlying type, got %s",
			r.name,
			ctx.UnderlyingType,
		)
	}
}

// CPURule validates Kubernetes CPU resource format.
type CPURule struct{ k8sResourceRule }

func (r *CPURule) Name() string { return "cpu" }

func (r *CPURule) Generate(ctx *GenerateContext) {
	r.k8sResourceRule.generate(ctx)
}

func (r *CPURule) Validate(ctx *ValidateContext) {
	r.k8sResourceRule.validate(ctx)
}

// MemoryRule validates Kubernetes memory resource format.
type MemoryRule struct{ k8sResourceRule }

func (r *MemoryRule) Name() string { return "memory" }

func (r *MemoryRule) Generate(ctx *GenerateContext) {
	r.k8sResourceRule.generate(ctx)
}

func (r *MemoryRule) Validate(ctx *ValidateContext) {
	r.k8sResourceRule.validate(ctx)
}

// DiskRule validates Kubernetes disk resource format.
type DiskRule struct{ k8sResourceRule }

func (r *DiskRule) Name() string { return "disk" }

func (r *DiskRule) Generate(ctx *GenerateContext) {
	r.k8sResourceRule.generate(ctx)
}

func (r *DiskRule) Validate(ctx *ValidateContext) {
	r.k8sResourceRule.validate(ctx)
}
