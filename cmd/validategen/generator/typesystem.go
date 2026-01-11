// Package generator provides validation code generation functionality.
package generator

import (
	"strings"

	"github.com/tlipoca9/devgen/genkit"
)

// Type classification functions for validation rules.

// IsStringType checks if t is a string type.
func IsStringType(t string) bool {
	return t == "string"
}

// IsPointerToStringType checks if t is a pointer to string type.
func IsPointerToStringType(t string) bool {
	return t == "*string"
}

// IsSliceOrMapType checks if t is a slice or map type.
func IsSliceOrMapType(t string) bool {
	return strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[")
}

// IsSliceType checks if t is a slice type.
func IsSliceType(t string) bool {
	return strings.HasPrefix(t, "[]")
}

// IsMapType checks if t is a map type.
func IsMapType(t string) bool {
	return strings.HasPrefix(t, "map[")
}

// IsPointerType checks if t is a pointer type.
func IsPointerType(t string) bool {
	return strings.HasPrefix(t, "*")
}

// IsBoolType checks if t is a bool type.
func IsBoolType(t string) bool {
	return t == "bool"
}

// IsPointerToBoolType checks if t is a pointer to bool type.
func IsPointerToBoolType(t string) bool {
	return t == "*bool"
}

// IsNumericType checks if t is a numeric type.
func IsNumericType(t string) bool {
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"byte", "rune", "uintptr":
		return true
	default:
		return false
	}
}

// IsPointerToNumericType checks if t is a pointer to a numeric type.
func IsPointerToNumericType(t string) bool {
	if !strings.HasPrefix(t, "*") {
		return false
	}
	return IsNumericType(strings.TrimPrefix(t, "*"))
}

// IsBuiltinType checks if the type is a Go builtin type that cannot have methods.
func IsBuiltinType(t string) bool {
	// Check pointer to builtin
	if strings.HasPrefix(t, "*") {
		return IsBuiltinType(strings.TrimPrefix(t, "*"))
	}
	// Check slice - recurse to check element type
	if strings.HasPrefix(t, "[]") {
		return IsBuiltinType(strings.TrimPrefix(t, "[]"))
	}
	// Check map - recurse to check value type
	if strings.HasPrefix(t, "map[") {
		valueType := ExtractMapValueType(t)
		if valueType == "" {
			return true // malformed map type
		}
		return IsBuiltinType(valueType)
	}
	// Builtin primitive types
	switch t {
	case "string", "bool", "error",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128",
		"byte", "rune", "uintptr", "any":
		return true
	}
	// interface{} is builtin
	if strings.HasPrefix(t, "interface") {
		return true
	}
	return false
}

// ExtractMapValueType extracts the value type from a map type string.
// e.g., "map[string]Address" -> "Address", "map[int]*User" -> "*User"
func ExtractMapValueType(t string) string {
	if !strings.HasPrefix(t, "map[") {
		return ""
	}
	depth := 0
	for i := 4; i < len(t); i++ {
		switch t[i] {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return t[i+1:]
			}
			depth--
		}
	}
	return ""
}

// EnsureTypeImport checks if a type string contains a cross-package reference
// and adds the necessary import.
func EnsureTypeImport(g *genkit.GeneratedFile, fieldType string, pkg *genkit.Package) {
	elemType := fieldType
	if strings.HasPrefix(elemType, "[]") {
		elemType = strings.TrimPrefix(elemType, "[]")
	} else if strings.HasPrefix(elemType, "map[") {
		elemType = ExtractMapValueType(elemType)
	}
	elemType = strings.TrimPrefix(elemType, "*")

	if dotIdx := strings.Index(elemType, "."); dotIdx != -1 {
		pkgAlias := elemType[:dotIdx]
		if pkg != nil && pkg.TypesPkg != nil {
			for _, imp := range pkg.TypesPkg.Imports() {
				if imp.Name() == pkgAlias {
					g.Import(genkit.GoImportPath(imp.Path()))
					return
				}
				path := imp.Path()
				parts := strings.Split(path, "/")
				if len(parts) > 0 && parts[len(parts)-1] == pkgAlias {
					g.Import(genkit.GoImportPath(imp.Path()))
					return
				}
			}
		}
	}
}
