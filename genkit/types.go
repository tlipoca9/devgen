package genkit

import (
	"fmt"
	"go/token"
	"regexp"
	"strings"
)

// DiagnosticSeverity represents the severity of a diagnostic.
type DiagnosticSeverity string

const (
	DiagnosticError   DiagnosticSeverity = "error"
	DiagnosticWarning DiagnosticSeverity = "warning"
	DiagnosticInfo    DiagnosticSeverity = "info"
)

// Diagnostic represents a single error or warning with source location.
// Used for reporting validation errors that can be displayed in IDEs.
type Diagnostic struct {
	Severity DiagnosticSeverity `json:"severity"`
	Message  string             `json:"message"`
	File     string             `json:"file"`
	Line     int                `json:"line"`
	Column   int                `json:"column"`
	EndLine  int                `json:"endLine,omitempty"`
	EndCol   int                `json:"endColumn,omitempty"`
	Tool     string             `json:"tool"`
	Code     string             `json:"code,omitempty"` // e.g., "E001"
}

// NewDiagnostic creates a new diagnostic from a token.Position.
func NewDiagnostic(severity DiagnosticSeverity, tool, code, message string, pos token.Position) Diagnostic {
	return Diagnostic{
		Severity: severity,
		Message:  message,
		File:     pos.Filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Tool:     tool,
		Code:     code,
	}
}

// DryRunResult contains the result of a dry-run execution.
type DryRunResult struct {
	Success     bool              `json:"success"`
	Files       map[string]string `json:"files,omitempty"` // filename -> content preview
	Diagnostics []Diagnostic      `json:"diagnostics,omitempty"`
	Stats       DryRunStats       `json:"stats"`
}

// DryRunStats contains statistics from a dry-run execution.
type DryRunStats struct {
	PackagesLoaded int `json:"packagesLoaded"`
	FilesGenerated int `json:"filesGenerated"`
	ErrorCount     int `json:"errorCount"`
	WarningCount   int `json:"warningCount"`
}

// AddDiagnostic adds a diagnostic to the result and updates stats.
func (r *DryRunResult) AddDiagnostic(d Diagnostic) {
	r.Diagnostics = append(r.Diagnostics, d)
	switch d.Severity {
	case DiagnosticError:
		r.Stats.ErrorCount++
		r.Success = false
	case DiagnosticWarning:
		r.Stats.WarningCount++
	}
}

// AddError is a convenience method to add an error diagnostic.
func (r *DryRunResult) AddError(tool, code, message string, pos token.Position) {
	r.AddDiagnostic(NewDiagnostic(DiagnosticError, tool, code, message, pos))
}

// AddWarning is a convenience method to add a warning diagnostic.
func (r *DryRunResult) AddWarning(tool, code, message string, pos token.Position) {
	r.AddDiagnostic(NewDiagnostic(DiagnosticWarning, tool, code, message, pos))
}

// DiagnosticCollector provides a fluent API for collecting diagnostics.
// It simplifies validation code by providing chainable methods.
type DiagnosticCollector struct {
	tool        string
	diagnostics []Diagnostic
}

// NewDiagnosticCollector creates a new collector for the given tool.
func NewDiagnosticCollector(tool string) *DiagnosticCollector {
	return &DiagnosticCollector{tool: tool}
}

// Error adds an error diagnostic.
func (c *DiagnosticCollector) Error(code, message string, pos token.Position) *DiagnosticCollector {
	c.diagnostics = append(c.diagnostics, NewDiagnostic(DiagnosticError, c.tool, code, message, pos))
	return c
}

// Errorf adds an error diagnostic with formatted message.
func (c *DiagnosticCollector) Errorf(code string, pos token.Position, format string, args ...any) *DiagnosticCollector {
	return c.Error(code, fmt.Sprintf(format, args...), pos)
}

// Warning adds a warning diagnostic.
func (c *DiagnosticCollector) Warning(code, message string, pos token.Position) *DiagnosticCollector {
	c.diagnostics = append(c.diagnostics, NewDiagnostic(DiagnosticWarning, c.tool, code, message, pos))
	return c
}

// Warningf adds a warning diagnostic with formatted message.
func (c *DiagnosticCollector) Warningf(
	code string,
	pos token.Position,
	format string,
	args ...any,
) *DiagnosticCollector {
	return c.Warning(code, fmt.Sprintf(format, args...), pos)
}

// Collect returns all collected diagnostics.
func (c *DiagnosticCollector) Collect() []Diagnostic {
	return c.diagnostics
}

// HasErrors returns true if any error diagnostics were collected.
func (c *DiagnosticCollector) HasErrors() bool {
	for _, d := range c.diagnostics {
		if d.Severity == DiagnosticError {
			return true
		}
	}
	return false
}

// Merge adds diagnostics from another collector.
func (c *DiagnosticCollector) Merge(other *DiagnosticCollector) *DiagnosticCollector {
	if other != nil {
		c.diagnostics = append(c.diagnostics, other.diagnostics...)
	}
	return c
}

// MergeSlice adds diagnostics from a slice.
func (c *DiagnosticCollector) MergeSlice(diagnostics []Diagnostic) *DiagnosticCollector {
	c.diagnostics = append(c.diagnostics, diagnostics...)
	return c
}

// Annotation represents a parsed annotation from comments.
// Annotations follow the format: tool:@name or tool:@name(arg1, arg2, key=value)
// Example: enumgen:@enum(string, json)
type Annotation struct {
	Tool  string            // tool name (e.g., "enumgen")
	Name  string            // annotation name (e.g., "enum")
	Args  map[string]string // key=value args
	Flags []string          // positional args without =
	Raw   string
}

// Has checks if the annotation has a flag or arg (case-sensitive).
func (a *Annotation) Has(name string) bool {
	for k := range a.Args {
		if k == name {
			return true
		}
	}
	for _, f := range a.Flags {
		if f == name {
			return true
		}
	}
	return false
}

// Get returns an arg value or empty string.
func (a *Annotation) Get(name string) string {
	return a.Args[name]
}

// GetOr returns an arg value or the default.
func (a *Annotation) GetOr(name, def string) string {
	if v, ok := a.Args[name]; ok {
		return v
	}
	return def
}

// ParseAnnotations extracts annotations from a doc comment.
// Supports format: tool:@name or tool:@name(args) or tool:@name.subname(args)
func ParseAnnotations(doc string) []*Annotation {
	var annotations []*Annotation
	// Match: word:@word or word:@word.word or word:@word(...) or word:@word.word(...)
	re := regexp.MustCompile(`(\w+):@([\w.]+)(?:\(([^)]*)\))?`)
	matches := re.FindAllStringSubmatch(doc, -1)

	for _, match := range matches {
		ann := &Annotation{
			Tool: match[1],
			Name: match[2],
			Args: make(map[string]string),
			Raw:  match[0],
		}

		if len(match) > 3 && match[3] != "" {
			for _, arg := range strings.Split(match[3], ",") {
				arg = strings.TrimSpace(arg)
				if arg == "" {
					continue
				}
				if strings.Contains(arg, "=") {
					parts := strings.SplitN(arg, "=", 2)
					key := strings.TrimSpace(parts[0])
					val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
					ann.Args[key] = val
				} else {
					ann.Flags = append(ann.Flags, arg)
				}
			}
		}
		annotations = append(annotations, ann)
	}
	return annotations
}

// HasAnnotation checks if doc contains a specific annotation.
// Format: tool:@name (e.g., HasAnnotation(doc, "enumgen", "enum"))
func HasAnnotation(doc, tool, name string) bool {
	return GetAnnotation(doc, tool, name) != nil
}

// GetAnnotation returns the first annotation with the given tool and name.
func GetAnnotation(doc, tool, name string) *Annotation {
	for _, ann := range ParseAnnotations(doc) {
		if ann.Tool == tool && ann.Name == name {
			return ann
		}
	}
	return nil
}

// Annotations is a slice of annotations with helper methods.
type Annotations []*Annotation

// ParseDoc parses all annotations from a doc comment.
func ParseDoc(doc string) Annotations {
	return ParseAnnotations(doc)
}

// Has checks if any annotation with the tool and name exists.
func (a Annotations) Has(tool, name string) bool {
	for _, ann := range a {
		if ann.Tool == tool && ann.Name == name {
			return true
		}
	}
	return false
}

// Get returns the first annotation with the tool and name.
func (a Annotations) Get(tool, name string) *Annotation {
	for _, ann := range a {
		if ann.Tool == tool && ann.Name == name {
			return ann
		}
	}
	return nil
}
