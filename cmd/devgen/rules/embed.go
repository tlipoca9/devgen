// Package rules contains embedded rule content for devgen.
package rules

import _ "embed"

//go:embed devgen.md
var DevgenRule string

//go:embed devgen-plugin.md
var DevgenPluginRule string

//go:embed devgen-genkit.md
var DevgenGenkitRule string

//go:embed devgen-rules.md
var DevgenRulesRule string
