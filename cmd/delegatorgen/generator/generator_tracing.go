package generator

import (
	"fmt"
	"strings"

	"github.com/tlipoca9/devgen/genkit"
)

// generateTracingDelegator generates the tracing delegator implementation.
func (g *Generator) generateTracingDelegator(gf *genkit.GeneratedFile, iface *genkit.Interface, pkg *genkit.Package) {
	ifaceName := iface.Name
	delegatorName := toLowerFirst(ifaceName) + "TracingDelegator"

	gf.P()
	gf.P("// =============================================================================")
	gf.P("// Tracing Delegator")
	gf.P("// =============================================================================")

	// Struct definition
	gf.P()
	gf.P("type ", delegatorName, " struct {")
	gf.P("next   ", ifaceName)
	gf.P("tracer ", genkit.GoImportPath("go.opentelemetry.io/otel/trace").Ident("Tracer"))
	gf.P("}")

	// Generate methods
	for _, m := range iface.Methods {
		g.generateTracingMethod(gf, m, iface, pkg, delegatorName)
	}
}

// generateTracingMethod generates a single tracing method.
func (g *Generator) generateTracingMethod(gf *genkit.GeneratedFile, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package, delegatorName string) {
	ann := genkit.GetAnnotation(m.Doc, ToolName, "trace")

	// Method signature
	gf.P()
	gf.P("func (d *", delegatorName, ") ", m.Name, "(", formatParams(m.Params), ")", formatResults(m.Results), " {")

	if ann == nil {
		// No @trace annotation - pass through
		gf.P("return d.next.", m.Name, "(", formatCallArgs(m.Params), ")")
		gf.P("}")
		return
	}

	// Get span name: default is "pkg.Interface.Method"
	spanName := pkg.PkgPath + "." + iface.Name + "." + m.Name
	if customSpan := ann.Get("span"); customSpan != "" {
		spanName = customSpan
	}

	// Get attributes
	var attrs []string
	if attrsStr := ann.Get("attrs"); attrsStr != "" {
		attrs = strings.Split(attrsStr, ",")
		for i := range attrs {
			attrs[i] = strings.TrimSpace(attrs[i])
		}
	}

	// Find context parameter
	ctxParam := findContextParam(m.Params)
	if ctxParam == "" {
		ctxParam = "ctx" // fallback
	}

	// Generate span start
	if len(attrs) > 0 {
		gf.P(ctxParam, ", span := d.tracer.Start(", ctxParam, ", ", genkit.RawString(spanName), ",")
		gf.P(genkit.GoImportPath("go.opentelemetry.io/otel/trace").Ident("WithAttributes"), "(")
		for _, attr := range attrs {
			paramType := getParamType(m.Params, attr)
			writeAttribute(gf, attr, paramType)
		}
		gf.P("))")
	} else {
		gf.P(ctxParam, ", span := d.tracer.Start(", ctxParam, ", ", genkit.RawString(spanName), ")")
	}
	gf.P("defer span.End()")
	gf.P()

	// Call next
	resultVars := formatResultVars(m.Results)
	if resultVars != "" {
		gf.P(resultVars, " := d.next.", m.Name, "(", formatCallArgs(m.Params), ")")
	} else {
		gf.P("d.next.", m.Name, "(", formatCallArgs(m.Params), ")")
		gf.P("}")
		return
	}

	// Check for error and record it
	if hasErrorReturn(m.Results) {
		gf.P("if err != nil {")
		gf.P("span.RecordError(err)")
		gf.P("span.SetStatus(", genkit.GoImportPath("go.opentelemetry.io/otel/codes").Ident("Error"), ", err.Error())")
		gf.P("}")
	}

	// Return
	gf.P("return ", resultVars)
	gf.P("}")
}

// formatParams formats method parameters for signature.
func formatParams(params []*genkit.Param) string {
	var parts []string
	for _, p := range params {
		if p.Name != "" {
			parts = append(parts, p.Name+" "+p.Type)
		} else {
			parts = append(parts, p.Type)
		}
	}
	return strings.Join(parts, ", ")
}

// formatResults formats method results for signature.
func formatResults(results []*genkit.Param) string {
	if len(results) == 0 {
		return ""
	}
	if len(results) == 1 && results[0].Name == "" {
		return " " + results[0].Type
	}
	var parts []string
	for _, r := range results {
		if r.Name != "" {
			parts = append(parts, r.Name+" "+r.Type)
		} else {
			parts = append(parts, r.Type)
		}
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// formatCallArgs formats arguments for method call.
func formatCallArgs(params []*genkit.Param) string {
	var parts []string
	for _, p := range params {
		if p.Name != "" {
			parts = append(parts, p.Name)
		}
	}
	return strings.Join(parts, ", ")
}

// formatResultVars formats result variable names.
func formatResultVars(results []*genkit.Param) string {
	if len(results) == 0 {
		return ""
	}
	var parts []string
	for i, r := range results {
		if r.Name != "" {
			parts = append(parts, r.Name)
		} else if r.Type == "error" {
			parts = append(parts, "err")
		} else {
			parts = append(parts, resultVarName(i))
		}
	}
	return strings.Join(parts, ", ")
}

// resultVarName generates a result variable name.
func resultVarName(index int) string {
	if index == 0 {
		return "result"
	}
	return fmt.Sprintf("result%d", index)
}

// findContextParam finds the context parameter name.
func findContextParam(params []*genkit.Param) string {
	for _, p := range params {
		if p.Type == "context.Context" || strings.HasSuffix(p.Type, ".Context") {
			return p.Name
		}
	}
	return ""
}

// getParamType returns the type of a parameter by name.
func getParamType(params []*genkit.Param, name string) string {
	for _, p := range params {
		if p.Name == name {
			return p.Type
		}
	}
	return "string"
}

// writeAttribute writes an attribute for OTel to the generated file.
func writeAttribute(gf *genkit.GeneratedFile, name, paramType string) {
	attrPkg := genkit.GoImportPath("go.opentelemetry.io/otel/attribute")
	switch paramType {
	case "string":
		gf.P(attrPkg.Ident("String"), "(\"", name, "\", ", name, "),")
	case "int", "int32", "int64":
		gf.P(attrPkg.Ident("Int64"), "(\"", name, "\", int64(", name, ")),")
	case "bool":
		gf.P(attrPkg.Ident("Bool"), "(\"", name, "\", ", name, "),")
	case "float32", "float64":
		gf.P(attrPkg.Ident("Float64"), "(\"", name, "\", float64(", name, ")),")
	default:
		// For complex types, use fmt.Sprintf
		gf.P(attrPkg.Ident("String"), "(\"", name, "\", ", genkit.GoImportPath("fmt").Ident("Sprintf"), "(\"%v\", ", name, ")),")
	}
}

// hasErrorReturn checks if the method has an error return.
func hasErrorReturn(results []*genkit.Param) bool {
	for _, r := range results {
		if r.Type == "error" {
			return true
		}
	}
	return false
}
