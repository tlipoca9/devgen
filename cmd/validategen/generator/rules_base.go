// Package generator provides validation code generation functionality.
package generator

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tlipoca9/devgen/genkit"
)

// Error codes for diagnostics.
const (
	ErrCodeMethodMissingParam  = "E001"
	ErrCodeRegexMissingPattern = "E002"
	ErrCodeFormatMissingType   = "E003"
	ErrCodeFormatMultipleArgs  = "E004"
	ErrCodeFormatUnsupported   = "E005"
	ErrCodeOneofMissingValues  = "E006"
	ErrCodeMissingParam        = "E007"
	ErrCodeInvalidParamType    = "E008"
	ErrCodeInvalidFieldType    = "E009"
	ErrCodeMethodNotFound      = "E010"
	ErrCodeInvalidRegex        = "E011"
	ErrCodeInvalidOneofValue   = "E012"
)

// fmtSprintf returns the fmt.Sprintf identifier for code generation.
func fmtSprintf() genkit.GoIdent {
	return genkit.GoImportPath("fmt").Ident("Sprintf")
}

// isValidNumber checks if a string is a valid number (integer or float).
func isValidNumber(s string) bool {
	if s == "" {
		return false
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// isValidDuration checks if a string is a valid Go duration.
func isValidDuration(s string) bool {
	if s == "" {
		return false
	}
	_, err := time.ParseDuration(s)
	return err == nil
}

// escapeString escapes a string for safe embedding in generated Go code.
// It handles quotes, backslashes, and other special characters.
func escapeString(s string) string {
	// Use strconv.Quote and strip the surrounding quotes
	quoted := strconv.Quote(s)
	return quoted[1 : len(quoted)-1]
}

// isValidRegex checks if a string is a valid regular expression.
func isValidRegex(pattern string) bool {
	_, err := regexp.Compile(pattern)
	return err == nil
}

// splitAndClean splits a space-separated string and returns non-empty trimmed values.
func splitAndClean(param string) []string {
	var result []string
	for _, v := range strings.Split(param, " ") {
		if v = strings.TrimSpace(v); v != "" {
			result = append(result, v)
		}
	}
	return result
}

// IsScalarOrPointerType checks if the type is a scalar type (string, numeric, bool) or pointer to one.
func IsScalarOrPointerType(t string) bool {
	return IsStringType(t) || IsPointerToStringType(t) ||
		IsNumericType(t) || IsPointerToNumericType(t) ||
		IsBoolType(t) || IsPointerToBoolType(t)
}
