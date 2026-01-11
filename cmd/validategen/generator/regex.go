// Package generator provides validation code generation functionality.
package generator

import "fmt"

// Predefined regex pattern names.
const (
	RegexEmail    = "email"
	RegexUUID     = "uuid"
	RegexAlpha    = "alpha"
	RegexAlphanum = "alphanum"
	RegexNumeric  = "numeric"
	RegexDNS1123  = "dns1123_label"
)

// RegexPatterns maps pattern names to their regex patterns.
var RegexPatterns = map[string]string{
	RegexEmail:    `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
	RegexUUID:     `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	RegexAlpha:    `^[a-zA-Z]+$`,
	RegexAlphanum: `^[a-zA-Z0-9]+$`,
	RegexNumeric:  `^[0-9]+$`,
	RegexDNS1123:  `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`,
}

// RegexVarNames maps pattern names to their variable names.
var RegexVarNames = map[string]string{
	RegexEmail:    "_validateRegexEmail",
	RegexUUID:     "_validateRegexUUID",
	RegexAlpha:    "_validateRegexAlpha",
	RegexAlphanum: "_validateRegexAlphanum",
	RegexNumeric:  "_validateRegexNumeric",
	RegexDNS1123:  "_validateRegexDNS1123Label",
}

// RegexTracker tracks custom regex patterns and assigns variable names.
type RegexTracker struct {
	patterns map[string]string // pattern -> variable name
	counter  int
}

// NewRegexTracker creates a new regex tracker.
func NewRegexTracker() *RegexTracker {
	return &RegexTracker{
		patterns: make(map[string]string),
	}
}

// GetVarName returns the variable name for a pattern, creating one if needed.
func (rt *RegexTracker) GetVarName(pattern string) string {
	if varName, ok := rt.patterns[pattern]; ok {
		return varName
	}
	rt.counter++
	varName := fmt.Sprintf("_validateRegex%d", rt.counter)
	rt.patterns[pattern] = varName
	return varName
}

// Patterns returns all tracked patterns.
func (rt *RegexTracker) Patterns() map[string]string {
	return rt.patterns
}
