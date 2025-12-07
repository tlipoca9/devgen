// Command vscgen generates VSCode extension configuration from devgen.toml files.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ToolConfig represents a devgen.toml file.
type ToolConfig struct {
	Tool        Tool         `toml:"tool"`
	Annotations []Annotation `toml:"annotations"`
}

// Tool represents the [tool] section.
type Tool struct {
	Name         string `toml:"name"`
	OutputSuffix string `toml:"output_suffix"`
}

// Annotation represents an [[annotations]] entry.
type Annotation struct {
	Name   string           `toml:"name"`
	Type   string           `toml:"type"` // "type" or "field"
	Doc    string           `toml:"doc"`
	Params *AnnotationParam `toml:"params"`
}

// AnnotationParam represents annotation parameters.
type AnnotationParam struct {
	Type        any               `toml:"type"`        // "string", "number", "list", "bool" or array of types
	Placeholder string            `toml:"placeholder"` // placeholder for snippet
	Values      []string          `toml:"values"`      // for enum-like params
	Docs        map[string]string `toml:"docs"`        // documentation for each value
	MaxArgs     int               `toml:"maxArgs"`     // maximum number of arguments allowed
}

// GetTypes returns the type(s) as a slice.
func (p *AnnotationParam) GetTypes() []string {
	if p.Type == nil {
		return nil
	}
	switch v := p.Type.(type) {
	case string:
		return []string{v}
	case []any:
		types := make([]string, 0, len(v))
		for _, t := range v {
			if s, ok := t.(string); ok {
				types = append(types, s)
			}
		}
		return types
	}
	return nil
}

// VSCodeToolConfig is the output format for VSCode extension.
type VSCodeToolConfig struct {
	TypeAnnotations  []string                    `json:"typeAnnotations"`
	FieldAnnotations []string                    `json:"fieldAnnotations"`
	OutputSuffix     string                      `json:"outputSuffix"`
	Annotations      map[string]VSCodeAnnotation `json:"annotations"`
}

// VSCodeAnnotation contains annotation metadata for VSCode.
type VSCodeAnnotation struct {
	Doc         string            `json:"doc"`
	ParamType   any               `json:"paramType,omitempty"`   // "string", "number", "list", "enum", "bool" or array of types
	Placeholder string            `json:"placeholder,omitempty"` // for snippet
	Values      []string          `json:"values,omitempty"`      // for enum params
	MaxArgs     int               `json:"maxArgs,omitempty"`     // maximum number of arguments allowed
	ValueDocs   map[string]string `json:"valueDocs,omitempty"`   // docs for each value
}

func main() {
	var (
		inputDir  string
		outputDir string
	)
	flag.StringVar(&inputDir, "input", "cmd", "Directory containing generator subdirectories with devgen.toml files")
	flag.StringVar(&outputDir, "output", "vscode-devgen/src", "Output directory for generated files")
	flag.Parse()

	// Find all devgen.toml files
	configs := make(map[string]*ToolConfig)
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "devgen.toml" {
			cfg, err := parseToolConfig(path)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
			configs[cfg.Tool.Name] = cfg
			fmt.Printf("Loaded: %s (%s)\n", path, cfg.Tool.Name)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(configs) == 0 {
		fmt.Fprintf(os.Stderr, "No devgen.toml files found in %s\n", inputDir)
		os.Exit(1)
	}

	// Convert to VSCode format
	vscodeConfigs := make(map[string]VSCodeToolConfig)
	for name, cfg := range configs {
		vscodeConfigs[name] = convertToVSCode(cfg)
	}

	// Write tools-config.json
	outputPath := filepath.Join(outputDir, "tools-config.json")
	if err := writeJSON(outputPath, vscodeConfigs); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputPath, err)
		os.Exit(1)
	}
	fmt.Printf("Generated: %s\n", outputPath)
}

func parseToolConfig(path string) (*ToolConfig, error) {
	var cfg ToolConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func convertToVSCode(cfg *ToolConfig) VSCodeToolConfig {
	vsc := VSCodeToolConfig{
		OutputSuffix: cfg.Tool.OutputSuffix,
		Annotations:  make(map[string]VSCodeAnnotation),
	}

	for _, ann := range cfg.Annotations {
		if ann.Type == "type" {
			vsc.TypeAnnotations = append(vsc.TypeAnnotations, ann.Name)
		} else {
			vsc.FieldAnnotations = append(vsc.FieldAnnotations, ann.Name)
		}

		vsAnn := VSCodeAnnotation{
			Doc: ann.Doc,
		}

		if ann.Params != nil {
			if len(ann.Params.Values) > 0 {
				vsAnn.ParamType = "enum"
				vsAnn.Values = ann.Params.Values
				vsAnn.ValueDocs = ann.Params.Docs
				vsAnn.MaxArgs = ann.Params.MaxArgs
			} else {
				types := ann.Params.GetTypes()
				if len(types) == 1 {
					vsAnn.ParamType = types[0]
				} else if len(types) > 1 {
					vsAnn.ParamType = types
				}
				vsAnn.Placeholder = ann.Params.Placeholder
			}
		}

		vsc.Annotations[ann.Name] = vsAnn
	}

	return vsc
}

func writeJSON(path string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
