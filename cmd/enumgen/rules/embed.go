// Package rules contains embedded AI rules for enumgen.
package rules

import _ "embed"

//go:embed enumgen.md
var EnumgenRule string
