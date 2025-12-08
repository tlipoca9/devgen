// Package rules contains embedded AI rules for validategen.
package rules

import _ "embed"

//go:embed validategen.md
var ValidategenRule string
