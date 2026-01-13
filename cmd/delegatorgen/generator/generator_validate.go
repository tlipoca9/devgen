package generator

import (
	"strings"
	"time"

	"github.com/tlipoca9/devgen/genkit"
)

// Validate implements genkit.ValidatableTool.
func (g *Generator) Validate(gen *genkit.Generator, _ *genkit.Logger) []genkit.Diagnostic {
	c := genkit.NewDiagnosticCollector(ToolName)

	for _, pkg := range gen.Packages {
		for _, iface := range pkg.Interfaces {
			if !genkit.HasAnnotation(iface.Doc, ToolName, "delegator") {
				continue
			}
			g.validateInterface(c, iface)
		}
	}

	return c.Collect()
}

// validateInterface validates a single interface and collects diagnostics.
func (g *Generator) validateInterface(c *genkit.DiagnosticCollector, iface *genkit.Interface) {
	if len(iface.Methods) == 0 {
		c.Error(ErrCodeNoMethods, "interface has no methods", iface.Pos)
		return
	}

	for _, m := range iface.Methods {
		g.validateMethod(c, m, iface)
	}
}

// validateMethod validates a single method and collects diagnostics.
func (g *Generator) validateMethod(c *genkit.DiagnosticCollector, m *genkit.Method, iface *genkit.Interface) {
	// Validate @cache annotation
	if ann := genkit.GetAnnotation(m.Doc, ToolName, "cache"); ann != nil {
		// @cache requires at least one return value (besides error)
		nonErrorResults := countNonErrorResults(m.Results)
		if nonErrorResults == 0 {
			c.Errorf(ErrCodeCacheNoReturn, m.Pos,
				"@cache requires method to have a return value (besides error)")
		} else if nonErrorResults > 1 {
			c.Warningf(ErrCodeCacheNoReturn, m.Pos,
				"@cache on method with multiple return values will only cache the first non-error value")
		}

		// Validate TTL format if specified
		if ttl := ann.Get("ttl"); ttl != "" {
			if !isValidDuration(ttl) {
				c.Errorf(ErrCodeCacheInvalidTTL, m.Pos,
					"invalid TTL format %q, expected duration like 5m, 1h, 30s", ttl)
			}
		}

		// Validate key template if specified
		if key := ann.Get("key"); key != "" {
			g.validateKeyTemplate(c, key, m, "key")
		}
		if prefix := ann.Get("prefix"); prefix != "" {
			g.validateKeyTemplate(c, prefix, m, "prefix")
		}
	}

	// Validate @cache_evict annotation
	if ann := genkit.GetAnnotation(m.Doc, ToolName, "cache_evict"); ann != nil {
		key := ann.Get("key")
		keys := ann.Get("keys")
		if key == "" && keys == "" {
			c.Errorf(ErrCodeEvictInvalidKey, m.Pos,
				"@cache_evict requires key or keys parameter")
		}
		if key != "" && keys != "" {
			c.Errorf(ErrCodeEvictInvalidKey, m.Pos,
				"@cache_evict cannot have both key and keys parameters")
		}
		if key != "" {
			g.validateKeyTemplate(c, key, m, "key")
		}
		if keys != "" {
			for _, k := range strings.Split(keys, ",") {
				g.validateKeyTemplate(c, strings.TrimSpace(k), m, "keys")
			}
		}
	}

	// Validate @trace annotation
	if ann := genkit.GetAnnotation(m.Doc, ToolName, "trace"); ann != nil {
		// Validate attrs if specified
		if attrs := ann.Get("attrs"); attrs != "" {
			paramNames := make(map[string]bool)
			for _, p := range m.Params {
				if p.Name != "" {
					paramNames[p.Name] = true
				}
			}
			for _, attr := range strings.Split(attrs, ",") {
				attr = strings.TrimSpace(attr)
				if attr != "" && !paramNames[attr] {
					c.Errorf(ErrCodeTraceInvalidAttr, m.Pos,
						"@trace(attrs=%s) references unknown parameter %q, available: %v",
						attrs, attr, getParamNames(m))
				}
			}
		}
	}
}

// validateKeyTemplate validates a cache key template.
func (g *Generator) validateKeyTemplate(c *genkit.DiagnosticCollector, template string, m *genkit.Method, paramName string) {
	// Extract variable references from template
	// Variables are in format: {varName} or {varName.field}
	vars := extractTemplateVars(template)

	// Build set of valid variable names
	validVars := map[string]bool{
		"PKG":       true,
		"INTERFACE": true,
		"METHOD":    true,
	}
	for _, p := range m.Params {
		if p.Name != "" {
			validVars[p.Name] = true
		}
	}

	for _, v := range vars {
		// Skip function calls like base64_json()
		if strings.Contains(v, "(") {
			continue
		}
		// Get the base variable name (before any dot)
		baseName := v
		if idx := strings.Index(v, "."); idx > 0 {
			baseName = v[:idx]
		}
		if !validVars[baseName] {
			c.Errorf(ErrCodeInvalidKeyTemplate, m.Pos,
				"@cache(%s=%q) references unknown variable %q", paramName, template, baseName)
		}
	}
}

// extractTemplateVars extracts variable names from a template string.
// Template format: {varName} or {varName.field} or {func(args)}
func extractTemplateVars(template string) []string {
	var vars []string
	i := 0
	for i < len(template) {
		if template[i] == '{' {
			j := i + 1
			depth := 1
			for j < len(template) && depth > 0 {
				if template[j] == '{' {
					depth++
				} else if template[j] == '}' {
					depth--
				}
				j++
			}
			if depth == 0 {
				vars = append(vars, template[i+1:j-1])
			}
			i = j
		} else {
			i++
		}
	}
	return vars
}

// isValidDuration checks if a string is a valid Go duration.
func isValidDuration(s string) bool {
	_, err := time.ParseDuration(s)
	return err == nil
}

// getParamNames returns a list of parameter names for error messages.
func getParamNames(m *genkit.Method) []string {
	var names []string
	for _, p := range m.Params {
		if p.Name != "" {
			names = append(names, p.Name)
		}
	}
	return names
}

// countNonErrorResults counts the number of non-error return values.
func countNonErrorResults(results []*genkit.Param) int {
	count := 0
	for _, r := range results {
		if r.Type != "error" {
			count++
		}
	}
	return count
}
