package generator

import "github.com/tlipoca9/devgen/genkit"

// generateHelpers generates helper functions.
func (g *Generator) generateHelpers(gf *genkit.GeneratedFile, iface *genkit.Interface, hasCache bool) {
	ifaceName := iface.Name

	gf.P()
	gf.P("// =============================================================================")
	gf.P("// Helper Functions")
	gf.P("// =============================================================================")

	if hasCache {
		// Generate base64JSONEncode helper
		gf.P()
		gf.P("// base64JSONEncode encodes arguments as JSON then Base64.")
		gf.P("func base64JSONEncode(args ...any) (string, error) {")
		gf.P("var data []byte")
		gf.P("var err error")
		gf.P("if len(args) == 1 {")
		gf.P("data, err = ", genkit.GoImportPath("encoding/json").Ident("Marshal"), "(args[0])")
		gf.P("} else {")
		gf.P("data, err = ", genkit.GoImportPath("encoding/json").Ident("Marshal"), "(args)")
		gf.P("}")
		gf.P("if err != nil {")
		gf.P("return \"\", err")
		gf.P("}")
		gf.P("return ", genkit.GoImportPath("encoding/base64").Ident("StdEncoding"), ".EncodeToString(data), nil")
		gf.P("}")

		// Generate calculateTTL helper
		gf.P()
		gf.P("// ", toLowerFirst(ifaceName), "CalculateTTL calculates TTL with jitter.")
		gf.P("func ", toLowerFirst(ifaceName), "CalculateTTL(baseTTL ", genkit.GoImportPath("time").Ident("Duration"), ", jitterPercent int) ", genkit.GoImportPath("time").Ident("Duration"), " {")
		gf.P("if jitterPercent <= 0 {")
		gf.P("return baseTTL")
		gf.P("}")
		gf.P("jitter := float64(jitterPercent) / 100.0")
		gf.P("factor := 1.0 + (", genkit.GoImportPath("math/rand").Ident("Float64"), "()*2-1)*jitter")
		gf.P("return ", genkit.GoImportPath("time").Ident("Duration"), "(float64(baseTTL) * factor)")
		gf.P("}")
	}
}
