// Package rules provides embedded AI rules for delegatorgen.
package rules

import _ "embed"

// DelegatorgenRule contains the AI-friendly documentation for delegatorgen.
//
//go:embed delegatorgen.md
var DelegatorgenRule string
