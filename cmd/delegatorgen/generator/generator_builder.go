package generator

import "github.com/tlipoca9/devgen/genkit"

// generateBuilder generates the builder pattern code.
func (g *Generator) generateBuilder(gf *genkit.GeneratedFile, iface *genkit.Interface, hasCache, hasTracing bool) {
	ifaceName := iface.Name
	delegatorType := ifaceName + "Delegator"
	delegatorFunc := ifaceName + "DelegatorFunc"

	gf.P()
	gf.P("// =============================================================================")
	gf.P("// Builder")
	gf.P("// =============================================================================")

	// DelegatorFunc type
	gf.P()
	gf.P("// ", delegatorFunc, " is a function that wraps a ", ifaceName, ".")
	gf.P("type ", delegatorFunc, " func(", ifaceName, ") ", ifaceName)

	// Delegator struct
	gf.P()
	gf.P("// ", delegatorType, " builds a ", ifaceName, " with delegators.")
	gf.P("type ", delegatorType, " struct {")
	gf.P("base       ", ifaceName)
	gf.P("delegators []", delegatorFunc)
	gf.P("}")

	// Constructor
	gf.P()
	gf.P("// New", delegatorType, " creates a new delegator builder.")
	gf.P("func New", delegatorType, "(base ", ifaceName, ") *", delegatorType, " {")
	gf.P("return &", delegatorType, "{base: base}")
	gf.P("}")

	// Use method
	gf.P()
	gf.P("// Use adds a custom delegator.")
	gf.P("// Delegators are applied in order: first added = outermost (executes first).")
	gf.P("func (d *", delegatorType, ") Use(mw ", delegatorFunc, ") *", delegatorType, " {")
	gf.P("d.delegators = append(d.delegators, mw)")
	gf.P("return d")
	gf.P("}")

	// WithCache method
	if hasCache {
		gf.P()
		gf.P("// WithCache adds caching delegator.")
		gf.P("// Advanced features (distributed lock, async refresh) are automatically enabled")
		gf.P("// if the cache implementation also implements CacheLocker or CacheAsyncExecutor.")
		gf.P("func (d *", delegatorType, ") WithCache(cache ", ifaceName, "Cache) *", delegatorType, " {")
		gf.P("return d.Use(func(next ", ifaceName, ") ", ifaceName, " {")
		gf.P("return new", ifaceName, "CacheDelegator(next, cache)")
		gf.P("})")
		gf.P("}")
	}

	// WithTracing method
	if hasTracing {
		gf.P()
		gf.P("// WithTracing adds tracing delegator using OpenTelemetry.")
		gf.P("func (d *", delegatorType, ") WithTracing(tracer ", genkit.GoImportPath("go.opentelemetry.io/otel/trace").Ident("Tracer"), ") *", delegatorType, " {")
		gf.P("return d.Use(func(next ", ifaceName, ") ", ifaceName, " {")
		gf.P("return &", toLowerFirst(ifaceName), "TracingDelegator{next: next, tracer: tracer}")
		gf.P("})")
		gf.P("}")
	}

	// Build method
	gf.P()
	gf.P("// Build creates the final ", ifaceName, " with all delegators applied.")
	gf.P("// Delegators are applied in reverse order so that the first added delegator")
	gf.P("// is the outermost (executes first).")
	gf.P("func (d *", delegatorType, ") Build() ", ifaceName, " {")
	gf.P("result := d.base")
	gf.P("for i := len(d.delegators) - 1; i >= 0; i-- {")
	gf.P("result = d.delegators[i](result)")
	gf.P("}")
	gf.P("return result")
	gf.P("}")
}

// toLowerFirst converts the first character of a string to lowercase.
func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] = r[0] - 'A' + 'a'
	}
	return string(r)
}
