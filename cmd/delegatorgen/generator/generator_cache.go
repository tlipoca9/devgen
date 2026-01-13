package generator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tlipoca9/devgen/genkit"
)

// generateCacheDelegator generates the cache delegator implementation.
func (g *Generator) generateCacheDelegator(gf *genkit.GeneratedFile, iface *genkit.Interface, pkg *genkit.Package) {
	ifaceName := iface.Name
	delegatorName := toLowerFirst(ifaceName) + "CacheDelegator"

	gf.P()
	gf.P("// =============================================================================")
	gf.P("// Cache Delegator")
	gf.P("// =============================================================================")

	// Struct definition
	gf.P()
	gf.P("type ", delegatorName, " struct {")
	gf.P("next          ", ifaceName)
	gf.P("cache         ", ifaceName, "Cache")
	gf.P("locker        ", ifaceName, "CacheLocker")
	gf.P("asyncExecutor ", ifaceName, "CacheAsyncExecutor")
	gf.P("refreshing    ", genkit.GoImportPath("sync").Ident("Map"))
	gf.P("}")

	// Constructor
	gf.P()
	gf.P("func new", ifaceName, "CacheDelegator(next ", ifaceName, ", cache ", ifaceName, "Cache) *", delegatorName, " {")
	gf.P("m := &", delegatorName, "{")
	gf.P("next:  next,")
	gf.P("cache: cache,")
	gf.P("}")
	gf.P()
	gf.P("// Runtime detection of optional capabilities")
	gf.P("if locker, ok := cache.(", ifaceName, "CacheLocker); ok {")
	gf.P("m.locker = locker")
	gf.P("}")
	gf.P("if executor, ok := cache.(", ifaceName, "CacheAsyncExecutor); ok {")
	gf.P("m.asyncExecutor = executor")
	gf.P("}")
	gf.P()
	gf.P("return m")
	gf.P("}")

	// Generate methods
	for _, m := range iface.Methods {
		g.generateCacheMethod(gf, m, iface, pkg, delegatorName)
	}
}

// generateCacheMethod generates a single cache method.
func (g *Generator) generateCacheMethod(gf *genkit.GeneratedFile, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package, delegatorName string) {
	cacheAnn := genkit.GetAnnotation(m.Doc, ToolName, "cache")
	evictAnn := genkit.GetAnnotation(m.Doc, ToolName, "cache_evict")

	// Method signature
	gf.P()
	gf.P("func (d *", delegatorName, ") ", m.Name, "(", formatParams(m.Params), ")", formatResults(m.Results), " {")

	if cacheAnn == nil && evictAnn == nil {
		// No cache annotation - pass through
		gf.P("return d.next.", m.Name, "(", formatCallArgs(m.Params), ")")
		gf.P("}")
		return
	}

	ctxParam := findContextParam(m.Params)
	if ctxParam == "" {
		ctxParam = "ctx"
	}

	if cacheAnn != nil {
		g.generateCacheReadMethod(gf, m, iface, pkg, cacheAnn, ctxParam)
	} else if evictAnn != nil {
		g.generateCacheEvictMethod(gf, m, iface, pkg, evictAnn, ctxParam)
	}
}

// generateCacheReadMethod generates a cache read method with all advanced features.
func (g *Generator) generateCacheReadMethod(gf *genkit.GeneratedFile, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package, ann *genkit.Annotation, ctxParam string) {
	ifaceName := iface.Name

	// Parse configuration
	ttl := ann.GetOr("ttl", DefaultCacheTTL)
	jitter := parseIntOr(ann.Get("jitter"), DefaultCacheJitter)
	refresh := parseIntOr(ann.Get("refresh"), DefaultCacheRefresh)
	prefix := ann.GetOr("prefix", DefaultCacheKeyPrefix)
	keySuffix := ann.GetOr("key", DefaultCacheKeySuffix)

	// Get return type (first non-error result)
	returnType := getReturnType(m.Results)

	// Generate constants
	gf.P("// Compile-time constants from annotation")
	gf.P("const (")
	writeDurationConst(gf, "baseTTL", ttl)
	gf.P("jitterPercent = ", jitter)
	gf.P("refreshPercent = ", refresh)
	gf.P(")")
	gf.P()

	// Generate key building
	gf.P("// Build cache key")
	g.generateKeyBuilding(gf, m, iface, pkg, prefix, keySuffix, returnType)
	gf.P()

	// Cache hit check
	gf.P("// Check cache")
	gf.P("if res, ok := d.cache.Get(", ctxParam, ", key); ok {")
	gf.P("// Check if error cache")
	gf.P("if res.IsError() {")
	gf.P("if err, ok := res.Value().(error); ok {")
	gf.P("return ", zeroValue(returnType), ", err")
	gf.P("}")
	gf.P("goto cacheMiss")
	gf.P("}")
	gf.P()
	gf.P("// Type assertion")
	gf.P("value, ok := res.Value().(", returnType, ")")
	gf.P("if !ok {")
	gf.P("goto cacheMiss")
	gf.P("}")
	gf.P()

	// Async refresh check
	gf.P("// Async refresh check")
	gf.P("if d.asyncExecutor != nil && refreshPercent > 0 {")
	gf.P("remaining := ", genkit.GoImportPath("time").Ident("Until"), "(res.ExpiresAt())")
	gf.P("threshold := baseTTL * ", genkit.GoImportPath("time").Ident("Duration"), "(refreshPercent) / 100")
	gf.P("if remaining > 0 && remaining < threshold {")
	gf.P("if _, loaded := d.refreshing.LoadOrStore(key, struct{}{}); !loaded {")
	gf.P("d.asyncExecutor.Submit(func() {")
	gf.P("defer d.refreshing.Delete(key)")
	nonCtxArgs := formatNonContextArgs(m.Params)
	if nonCtxArgs != "" {
		gf.P("d.refresh", m.Name, "Cache(", genkit.GoImportPath("context").Ident("Background"), "(), key, ", nonCtxArgs, ")")
	} else {
		gf.P("d.refresh", m.Name, "Cache(", genkit.GoImportPath("context").Ident("Background"), "(), key)")
	}
	gf.P("})")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()
	gf.P("return value, nil")
	gf.P("}")
	gf.P()

	// Cache miss
	gf.P("cacheMiss:")

	// Distributed lock
	gf.P("// Distributed lock (if available)")
	gf.P("if d.locker != nil {")
	gf.P("release, acquired := d.locker.Lock(", ctxParam, ", key)")
	gf.P("if acquired {")
	gf.P("defer release()")
	gf.P("// Double-check after acquiring lock")
	gf.P("if res, ok := d.cache.Get(", ctxParam, ", key); ok {")
	gf.P("if res.IsError() {")
	gf.P("if err, ok := res.Value().(error); ok {")
	gf.P("return ", zeroValue(returnType), ", err")
	gf.P("}")
	gf.P("} else if value, ok := res.Value().(", returnType, "); ok {")
	gf.P("return value, nil")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()

	// Call downstream
	gf.P("// Call downstream")
	gf.P("result, err := d.next.", m.Name, "(", formatCallArgs(m.Params), ")")
	gf.P()

	// Calculate TTL with jitter
	gf.P("// Calculate TTL with jitter")
	gf.P("ttl := ", toLowerFirst(ifaceName), "CalculateTTL(baseTTL, jitterPercent)")
	gf.P()

	// Handle error
	gf.P("if err != nil {")
	gf.P("// Let cache implementation decide whether to cache this error")
	gf.P("d.cache.SetError(", ctxParam, ", key, err, ttl)")
	gf.P("return ", zeroValue(returnType), ", err")
	gf.P("}")
	gf.P()

	// Store in cache
	gf.P("// Store in cache")
	gf.P("d.cache.Set(", ctxParam, ", key, result, ttl)")
	gf.P("return result, nil")
	gf.P("}") // Close the method

	// Generate refresh method
	g.generateRefreshMethod(gf, m, iface, pkg, ann, ctxParam)
}

// generateRefreshMethod generates the async refresh method.
func (g *Generator) generateRefreshMethod(gf *genkit.GeneratedFile, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package, ann *genkit.Annotation, ctxParam string) {
	ifaceName := iface.Name

	ttl := ann.GetOr("ttl", DefaultCacheTTL)
	jitter := parseIntOr(ann.Get("jitter"), DefaultCacheJitter)
	refresh := parseIntOr(ann.Get("refresh"), DefaultCacheRefresh)

	gf.P()
	nonCtxParams := formatNonContextParams(m.Params)
	if nonCtxParams != "" {
		gf.P("func (d *", toLowerFirst(ifaceName), "CacheDelegator) refresh", m.Name, "Cache(", ctxParam, " ", genkit.GoImportPath("context").Ident("Context"), ", key string, ", nonCtxParams, ") {")
	} else {
		gf.P("func (d *", toLowerFirst(ifaceName), "CacheDelegator) refresh", m.Name, "Cache(", ctxParam, " ", genkit.GoImportPath("context").Ident("Context"), ", key string) {")
	}
	gf.P("const (")
	writeDurationConst(gf, "baseTTL", ttl)
	gf.P("jitterPercent = ", jitter)
	gf.P("refreshPercent = ", refresh)
	gf.P(")")
	gf.P()

	gf.P("// Lock to prevent concurrent refresh")
	gf.P("if d.locker != nil {")
	gf.P("release, acquired := d.locker.Lock(", ctxParam, ", key)")
	gf.P("if !acquired {")
	gf.P("return")
	gf.P("}")
	gf.P("defer release()")
	gf.P()
	gf.P("// Double-check if still needs refresh")
	gf.P("if res, ok := d.cache.Get(", ctxParam, ", key); ok {")
	gf.P("remaining := ", genkit.GoImportPath("time").Ident("Until"), "(res.ExpiresAt())")
	gf.P("threshold := baseTTL * ", genkit.GoImportPath("time").Ident("Duration"), "(refreshPercent) / 100")
	gf.P("if remaining >= threshold {")
	gf.P("return")
	gf.P("}")
	gf.P("}")
	gf.P("}")
	gf.P()

	refreshArgs := formatNonContextArgs(m.Params)
	if refreshArgs != "" {
		gf.P("result, err := d.next.", m.Name, "(", ctxParam, ", ", refreshArgs, ")")
	} else {
		gf.P("result, err := d.next.", m.Name, "(", ctxParam, ")")
	}
	gf.P("if err != nil {")
	gf.P("return // Keep old cache on refresh failure")
	gf.P("}")
	gf.P()
	gf.P("ttl := ", toLowerFirst(ifaceName), "CalculateTTL(baseTTL, jitterPercent)")
	gf.P("d.cache.Set(", ctxParam, ", key, result, ttl)")
	gf.P("}")
}

// generateCacheEvictMethod generates a cache evict method.
func (g *Generator) generateCacheEvictMethod(gf *genkit.GeneratedFile, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package, ann *genkit.Annotation, ctxParam string) {
	// Call downstream first
	resultVars := formatResultVars(m.Results)
	if resultVars != "" {
		gf.P(resultVars, " := d.next.", m.Name, "(", formatCallArgs(m.Params), ")")
	} else {
		gf.P("d.next.", m.Name, "(", formatCallArgs(m.Params), ")")
	}

	// Evict on success
	hasError := hasErrorReturn(m.Results)
	if hasError {
		gf.P("if err == nil {")
	}

	// Generate evict keys
	key := ann.Get("key")
	keys := ann.Get("keys")
	if key != "" {
		keyParts := g.generateKeyExpressionParts(key, m, iface, pkg, gf)
		args := []any{"d.cache.Delete(", ctxParam, ", "}
		args = append(args, keyParts...)
		args = append(args, ")")
		gf.P(args...)
	} else if keys != "" {
		keyList := strings.Split(keys, ",")
		gf.P("d.cache.Delete(", ctxParam, ",")
		for _, k := range keyList {
			k = strings.TrimSpace(k)
			keyParts := g.generateKeyExpressionParts(k, m, iface, pkg, gf)
			args := keyParts
			args = append(args, ",")
			gf.P(args...)
		}
		gf.P(")")
	}

	if hasError {
		gf.P("}")
	}

	// Return
	if resultVars != "" {
		gf.P("return ", resultVars)
	}
	gf.P("}") // Close the method
}

// generateKeyBuilding generates the key building code.
func (g *Generator) generateKeyBuilding(gf *genkit.GeneratedFile, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package, prefix, keySuffix string, returnType string) {
	// Check if we need error handling (base64_json is used)
	usesBase64Json := strings.Contains(prefix, "{base64_json(") || strings.Contains(keySuffix, "{base64_json(")

	if usesBase64Json {
		// Generate key with error handling
		g.generateKeyBuildingWithError(gf, m, iface, pkg, prefix, keySuffix, returnType)
	} else {
		// Generate simple key assignment
		prefixParts := g.generateKeyExpressionParts(prefix, m, iface, pkg, gf)
		suffixParts := g.generateKeyExpressionParts(keySuffix, m, iface, pkg, gf)

		parts := []any{"key := "}
		parts = append(parts, prefixParts...)
		parts = append(parts, " + ")
		parts = append(parts, suffixParts...)
		gf.P(parts...)
	}
}

// generateKeyBuildingWithError generates key building code with error handling for base64JSONEncode.
func (g *Generator) generateKeyBuildingWithError(gf *genkit.GeneratedFile, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package, prefix, keySuffix string, returnType string) {
	// Generate prefix
	prefixExpr := g.generateKeyExpressionPartsForBase64(prefix, m, iface, pkg)
	suffixExpr := g.generateKeyExpressionPartsForBase64(keySuffix, m, iface, pkg)

	// Check which parts use base64JSONEncode
	prefixUsesBase64 := strings.Contains(prefix, "{base64_json(")
	suffixUsesBase64 := strings.Contains(keySuffix, "{base64_json(")

	if prefixUsesBase64 && suffixUsesBase64 {
		// Both use base64JSONEncode
		gf.P("keyPrefix, err := ", prefixExpr)
		gf.P("if err != nil {")
		gf.P("return ", zeroValue(returnType), ", err")
		gf.P("}")
		gf.P("keySuffix, err := ", suffixExpr)
		gf.P("if err != nil {")
		gf.P("return ", zeroValue(returnType), ", err")
		gf.P("}")
		gf.P("key := keyPrefix + keySuffix")
	} else if prefixUsesBase64 {
		// Only prefix uses base64JSONEncode
		gf.P("keyPrefix, err := ", prefixExpr)
		gf.P("if err != nil {")
		gf.P("return ", zeroValue(returnType), ", err")
		gf.P("}")
		suffixParts := g.generateKeyExpressionParts(keySuffix, m, iface, pkg, gf)
		parts := []any{"key := keyPrefix + "}
		parts = append(parts, suffixParts...)
		gf.P(parts...)
	} else {
		// Only suffix uses base64JSONEncode
		prefixParts := g.generateKeyExpressionParts(prefix, m, iface, pkg, gf)
		gf.P("keySuffix, err := ", suffixExpr)
		gf.P("if err != nil {")
		gf.P("return ", zeroValue(returnType), ", err")
		gf.P("}")
		parts := []any{"key := "}
		parts = append(parts, prefixParts...)
		parts = append(parts, " + keySuffix")
		gf.P(parts...)
	}
}

// generateKeyExpressionPartsForBase64 generates a key expression that returns (string, error).
func (g *Generator) generateKeyExpressionPartsForBase64(template string, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package) string {
	if template == "" {
		return `"", nil`
	}

	// Replace built-in variables
	result := template
	result = strings.ReplaceAll(result, "{PKG}", pkg.PkgPath)
	result = strings.ReplaceAll(result, "{INTERFACE}", iface.Name)
	result = strings.ReplaceAll(result, "{METHOD}", m.Name)

	// Check for base64_json function
	if strings.Contains(result, "{base64_json(") {
		return g.generateBase64JsonKeyWithError(result, m)
	}

	// No base64_json - this shouldn't happen in this code path
	return fmt.Sprintf("%q, nil", result)
}

// generateKeyExpressionParts generates key expression parts that can be used with gf.P().
// This properly handles fmt.Sprintf by using genkit.GoImportPath.
func (g *Generator) generateKeyExpressionParts(template string, m *genkit.Method, iface *genkit.Interface, pkg *genkit.Package, gf *genkit.GeneratedFile) []any {
	if template == "" {
		return []any{`""`}
	}

	// Replace built-in variables
	result := template
	result = strings.ReplaceAll(result, "{PKG}", pkg.PkgPath)
	result = strings.ReplaceAll(result, "{INTERFACE}", iface.Name)
	result = strings.ReplaceAll(result, "{METHOD}", m.Name)

	// Check for base64_json function
	if strings.Contains(result, "{base64_json(") {
		// generateBase64JsonKey returns a Go expression string
		return []any{g.generateBase64JsonKey(result, m)}
	}

	// Check for simple variable references
	if strings.Contains(result, "{") {
		return g.generateFormattedKeyParts(result, m)
	}

	// Static string - return quoted
	return []any{fmt.Sprintf("%q", result)}
}

// generateFormattedKeyParts generates key parts with fmt.Sprintf using proper imports.
func (g *Generator) generateFormattedKeyParts(template string, m *genkit.Method) []any {
	// Replace {var} with %v and collect args
	var args []string
	result := template

	for _, p := range m.Params {
		if p.Name != "" {
			placeholder := "{" + p.Name + "}"
			if strings.Contains(result, placeholder) {
				result = strings.ReplaceAll(result, placeholder, "%v")
				args = append(args, p.Name)
			}
			// Also handle {param.Field} patterns
			for strings.Contains(result, "{"+p.Name+".") {
				start := strings.Index(result, "{"+p.Name+".")
				end := strings.Index(result[start:], "}")
				if end == -1 {
					break
				}
				fieldPath := result[start+1 : start+end]
				result = result[:start] + "%v" + result[start+end+1:]
				args = append(args, fieldPath)
			}
		}
	}

	if len(args) == 0 {
		return []any{fmt.Sprintf("%q", result)}
	}

	// Return parts that use genkit.GoImportPath for fmt
	// The string after Ident is printed directly without quotes
	return []any{
		genkit.GoImportPath("fmt").Ident("Sprintf"),
		fmt.Sprintf("(%q, %s)", result, strings.Join(args, ", ")),
	}
}

// generateBase64JsonKey generates key with base64_json function (legacy, returns expression without error).
func (g *Generator) generateBase64JsonKey(template string, m *genkit.Method) string {
	// Find base64_json(...) and extract args
	start := strings.Index(template, "{base64_json(")
	if start == -1 {
		return fmt.Sprintf("%q", template)
	}

	end := strings.Index(template[start:], ")}")
	if end == -1 {
		return fmt.Sprintf("%q", template)
	}
	end += start + 2

	funcCall := template[start:end]
	argsStr := funcCall[len("{base64_json(") : len(funcCall)-2]

	// Determine arguments
	var args []string
	if argsStr == "" {
		// Use all non-context parameters
		for _, p := range m.Params {
			if p.Name != "" && !isContextType(p.Type) {
				args = append(args, p.Name)
			}
		}
	} else {
		// Use specified parameters
		for _, arg := range strings.Split(argsStr, ",") {
			args = append(args, strings.TrimSpace(arg))
		}
	}

	// Build the expression - this is used for refresh method where we already have the key
	prefix := template[:start]
	suffix := template[end:]

	var parts []string
	if prefix != "" {
		parts = append(parts, fmt.Sprintf("%q", prefix))
	}

	// For refresh method, we pass the key directly, so we don't need base64JSONEncode here
	if len(args) == 1 {
		parts = append(parts, fmt.Sprintf("base64JSONEncode(%s)", args[0]))
	} else {
		parts = append(parts, fmt.Sprintf("base64JSONEncode(%s)", strings.Join(args, ", ")))
	}

	if suffix != "" {
		parts = append(parts, fmt.Sprintf("%q", suffix))
	}

	return strings.Join(parts, " + ")
}

// generateBase64JsonKeyWithError generates key expression that returns (string, error).
func (g *Generator) generateBase64JsonKeyWithError(template string, m *genkit.Method) string {
	// Find base64_json(...) and extract args
	start := strings.Index(template, "{base64_json(")
	if start == -1 {
		return fmt.Sprintf("%q, nil", template)
	}

	end := strings.Index(template[start:], ")}")
	if end == -1 {
		return fmt.Sprintf("%q, nil", template)
	}
	end += start + 2

	funcCall := template[start:end]
	argsStr := funcCall[len("{base64_json(") : len(funcCall)-2]

	// Determine arguments
	var args []string
	if argsStr == "" {
		// Use all non-context parameters
		for _, p := range m.Params {
			if p.Name != "" && !isContextType(p.Type) {
				args = append(args, p.Name)
			}
		}
	} else {
		// Use specified parameters
		for _, arg := range strings.Split(argsStr, ",") {
			args = append(args, strings.TrimSpace(arg))
		}
	}

	prefix := template[:start]
	suffix := template[end:]

	// Generate function that builds the key with error handling
	var base64Call string
	if len(args) == 1 {
		base64Call = fmt.Sprintf("base64JSONEncode(%s)", args[0])
	} else {
		base64Call = fmt.Sprintf("base64JSONEncode(%s)", strings.Join(args, ", "))
	}

	// Build inline function that returns (string, error)
	if prefix == "" && suffix == "" {
		return base64Call
	} else if prefix == "" {
		return fmt.Sprintf("func() (string, error) { s, err := %s; if err != nil { return \"\", err }; return s + %q, nil }()", base64Call, suffix)
	} else if suffix == "" {
		return fmt.Sprintf("func() (string, error) { s, err := %s; if err != nil { return \"\", err }; return %q + s, nil }()", base64Call, prefix)
	}
	return fmt.Sprintf("func() (string, error) { s, err := %s; if err != nil { return \"\", err }; return %q + s + %q, nil }()", base64Call, prefix, suffix)
}

// formatNonContextParams formats non-context parameters for signature.
func formatNonContextParams(params []*genkit.Param) string {
	var parts []string
	for _, p := range params {
		if p.Name != "" && !isContextType(p.Type) {
			parts = append(parts, p.Name+" "+p.Type)
		}
	}
	return strings.Join(parts, ", ")
}

// formatNonContextArgs formats non-context arguments for call.
func formatNonContextArgs(params []*genkit.Param) string {
	var parts []string
	for _, p := range params {
		if p.Name != "" && !isContextType(p.Type) {
			parts = append(parts, p.Name)
		}
	}
	return strings.Join(parts, ", ")
}

// isContextType checks if a type is context.Context.
func isContextType(t string) bool {
	return t == "context.Context" || strings.HasSuffix(t, ".Context")
}

// getReturnType returns the first non-error return type.
func getReturnType(results []*genkit.Param) string {
	for _, r := range results {
		if r.Type != "error" {
			return r.Type
		}
	}
	return "any"
}

// zeroValue returns the zero value for a type.
func zeroValue(t string) string {
	if strings.HasPrefix(t, "*") || strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[") {
		return "nil"
	}
	switch t {
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "0"
	case "bool":
		return "false"
	default:
		return "nil"
	}
}

// parseIntOr parses an int or returns default.
func parseIntOr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// writeDurationConst writes a duration constant definition to the generated file.
// Examples: "5m" -> "baseTTL = 5 * time.Minute", "1h" -> "baseTTL = time.Hour"
func writeDurationConst(gf *genkit.GeneratedFile, name, s string) {
	if s == "" {
		gf.P(name, " = 0")
		return
	}

	// Parse the duration to validate and get components
	d, err := time.ParseDuration(s)
	if err != nil {
		gf.P(name, " = 0")
		return
	}

	timePkg := genkit.GoImportPath("time")

	// Convert to the most appropriate unit
	switch {
	case d >= time.Hour && d%time.Hour == 0:
		hours := int64(d / time.Hour)
		if hours == 1 {
			gf.P(name, " = ", timePkg.Ident("Hour"))
		} else {
			gf.P(name, " = ", hours, " * ", timePkg.Ident("Hour"))
		}
	case d >= time.Minute && d%time.Minute == 0:
		minutes := int64(d / time.Minute)
		if minutes == 1 {
			gf.P(name, " = ", timePkg.Ident("Minute"))
		} else {
			gf.P(name, " = ", minutes, " * ", timePkg.Ident("Minute"))
		}
	case d >= time.Second && d%time.Second == 0:
		seconds := int64(d / time.Second)
		if seconds == 1 {
			gf.P(name, " = ", timePkg.Ident("Second"))
		} else {
			gf.P(name, " = ", seconds, " * ", timePkg.Ident("Second"))
		}
	case d >= time.Millisecond && d%time.Millisecond == 0:
		ms := int64(d / time.Millisecond)
		if ms == 1 {
			gf.P(name, " = ", timePkg.Ident("Millisecond"))
		} else {
			gf.P(name, " = ", ms, " * ", timePkg.Ident("Millisecond"))
		}
	default:
		// Fallback to nanoseconds
		gf.P(name, " = ", int64(d))
	}
}
