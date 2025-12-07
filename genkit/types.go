package genkit

import (
	"regexp"
	"strings"
)

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
