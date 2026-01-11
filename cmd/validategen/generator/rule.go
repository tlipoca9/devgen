// Package generator provides validation code generation functionality.
package generator

import (
	"github.com/tlipoca9/devgen/genkit"
)

// Rule represents a validation rule that can generate validation code.
type Rule interface {
	// Name returns the rule name (e.g., "required", "email", "min").
	Name() string

	// Generate generates validation code for the given field.
	// It writes the validation code to the GeneratedFile.
	Generate(ctx *GenerateContext)

	// Validate validates the rule configuration and returns diagnostics.
	// It checks if the rule is applicable to the field type and if parameters are valid.
	Validate(ctx *ValidateContext)

	// RequiredRegex returns the list of predefined regex names required by this rule.
	// Returns nil if no predefined regex is needed.
	RequiredRegex() []string
}

// GenerateContext provides context for code generation.
type GenerateContext struct {
	G           *genkit.GeneratedFile
	FieldName   string
	FieldType   string
	Param       string
	Pkg         *genkit.Package
	Field       *genkit.Field
	CustomRegex *RegexTracker
	Generator   *Generator // For accessing package index
}

// ValidateContext provides context for validation.
type ValidateContext struct {
	Collector      *genkit.DiagnosticCollector
	Field          *genkit.Field
	Param          string
	UnderlyingType string
	Pkg            *genkit.Package
	Generator      *Generator // For accessing package index
}

// RuleFactory creates a rule instance.
type RuleFactory func() Rule

// ValidateRule represents a parsed validation annotation.
type ValidateRule struct {
	Name  string
	Param string
}

// FieldValidation groups a field with its validation rules.
type FieldValidation struct {
	Field *genkit.Field
	Rules []*ValidateRule
}
