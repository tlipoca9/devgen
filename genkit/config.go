// Package genkit provides configuration types for devgen tools and plugins.
package genkit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the project-level devgen.toml configuration.
type Config struct {
	// Plugins defines external tool plugins to load.
	Plugins []PluginConfig `toml:"plugins"`

	// Tools contains tool-specific configurations (annotations, output suffix, etc.)
	Tools map[string]ToolConfig `toml:"tools"`
}

// PluginConfig defines an external plugin to load.
type PluginConfig struct {
	// Name is the plugin/tool name (e.g., "customgen").
	Name string `toml:"name"`

	// Path is the path to the plugin.
	// For "source" type: Go package directory path
	// For "plugin" type: path to .so file
	Path string `toml:"path"`

	// Type specifies how to load the plugin.
	// - "source": compile Go source code at runtime (default)
	// - "plugin": load as Go plugin (.so)
	Type PluginType `toml:"type"`
}

// PluginType defines how a plugin is loaded.
type PluginType string

const (
	// PluginTypeSource compiles Go source code at runtime.
	PluginTypeSource PluginType = "source"

	// PluginTypePlugin loads a pre-compiled Go plugin (.so).
	PluginTypePlugin PluginType = "plugin"
)

// ToolConfig defines configuration for a specific tool.
type ToolConfig struct {
	// OutputSuffix is the suffix for generated files (e.g., "_enum.go").
	OutputSuffix string `toml:"output_suffix"`

	// Annotations defines the annotations supported by this tool.
	Annotations []AnnotationConfig `toml:"annotations"`
}

// AnnotationConfig defines a single annotation's metadata.
type AnnotationConfig struct {
	// Name is the annotation name (e.g., "enum", "validate").
	Name string `toml:"name"`

	// Type is where the annotation can be applied: "type" or "field".
	Type string `toml:"type"`

	// Doc is the documentation for this annotation.
	Doc string `toml:"doc"`

	// Params defines parameter configuration.
	Params *AnnotationParams `toml:"params"`

	// LSP defines LSP integration configuration.
	LSP *LSPConfig `toml:"lsp"`
}

// AnnotationParams defines annotation parameter configuration.
type AnnotationParams struct {
	// Type is the parameter type: "string", "number", "bool", "list", or "enum".
	// Can also be an array of types for multiple accepted types.
	Type any `toml:"type"`

	// Values is the list of allowed values for enum type.
	Values []string `toml:"values"`

	// Placeholder is the placeholder text for the parameter.
	Placeholder string `toml:"placeholder"`

	// MaxArgs is the maximum number of arguments allowed.
	MaxArgs int `toml:"maxArgs"`

	// Docs provides documentation for each enum value.
	Docs map[string]string `toml:"docs"`
}

// LSPConfig defines LSP integration for an annotation.
type LSPConfig struct {
	// Enabled indicates whether LSP integration is enabled.
	Enabled bool `toml:"enabled"`

	// Provider is the LSP provider (e.g., "gopls").
	Provider string `toml:"provider"`

	// Feature is the LSP feature type: "method", "type", "symbol".
	Feature string `toml:"feature"`

	// Signature is the required method signature (e.g., "func() error").
	Signature string `toml:"signature"`

	// ResolveFrom specifies where to find the type: "fieldType", "receiverType".
	ResolveFrom string `toml:"resolveFrom"`
}

// LoadConfig loads the devgen.toml configuration from the given directory.
// It searches for devgen.toml in the directory and its parents up to the root.
func LoadConfig(dir string) (*Config, error) {
	configPath, err := FindConfig(dir)
	if err != nil {
		return nil, err
	}
	if configPath == "" {
		return &Config{}, nil // No config found, return empty
	}
	return LoadConfigFile(configPath)
}

// LoadConfigFile loads configuration from a specific file path.
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	// Resolve relative paths based on config file location
	configDir := filepath.Dir(path)
	for i := range cfg.Plugins {
		if !filepath.IsAbs(cfg.Plugins[i].Path) {
			cfg.Plugins[i].Path = filepath.Join(configDir, cfg.Plugins[i].Path)
		}
		// Default type is source
		if cfg.Plugins[i].Type == "" {
			cfg.Plugins[i].Type = PluginTypeSource
		}
	}

	return &cfg, nil
}

// FindConfig searches for devgen.toml starting from dir and going up to root.
func FindConfig(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, "devgen.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return "", nil
		}
		dir = parent
	}
}

// ToVSCodeConfig converts the tool configuration to VSCode extension format.
func (tc *ToolConfig) ToVSCodeConfig() map[string]any {
	typeAnnotations := []string{}
	fieldAnnotations := []string{}
	annotations := make(map[string]any)

	for _, ann := range tc.Annotations {
		annConfig := map[string]any{
			"doc": ann.Doc,
		}

		if ann.Params != nil {
			if ann.Params.Type != nil {
				annConfig["paramType"] = ann.Params.Type
			} else if len(ann.Params.Values) > 0 {
				// If values are provided but no type, default to "enum"
				annConfig["paramType"] = "enum"
			}
			if len(ann.Params.Values) > 0 {
				annConfig["values"] = ann.Params.Values
			}
			if ann.Params.Placeholder != "" {
				annConfig["placeholder"] = ann.Params.Placeholder
			}
			if ann.Params.MaxArgs > 0 {
				annConfig["maxArgs"] = ann.Params.MaxArgs
			}
			if len(ann.Params.Docs) > 0 {
				annConfig["valueDocs"] = ann.Params.Docs
			}
		}

		if ann.LSP != nil && ann.LSP.Enabled {
			annConfig["lsp"] = map[string]any{
				"enabled":     ann.LSP.Enabled,
				"provider":    ann.LSP.Provider,
				"feature":     ann.LSP.Feature,
				"signature":   ann.LSP.Signature,
				"resolveFrom": ann.LSP.ResolveFrom,
			}
		}

		annotations[ann.Name] = annConfig

		switch ann.Type {
		case "type":
			typeAnnotations = append(typeAnnotations, ann.Name)
		case "field":
			fieldAnnotations = append(fieldAnnotations, ann.Name)
		}
	}

	return map[string]any{
		"typeAnnotations":  typeAnnotations,
		"fieldAnnotations": fieldAnnotations,
		"outputSuffix":     tc.OutputSuffix,
		"annotations":      annotations,
	}
}

// GetToolConfig extracts ToolConfig from a Tool.
// If the tool implements ConfigurableTool, returns its Config().
// Otherwise returns an empty ToolConfig.
func GetToolConfig(t Tool) ToolConfig {
	if ct, ok := t.(ConfigurableTool); ok {
		return ct.Config()
	}
	return ToolConfig{}
}

// MergeToolConfigs merges tool configurations from multiple sources.
// Later sources override earlier ones for the same tool name.
func MergeToolConfigs(configs ...map[string]ToolConfig) map[string]ToolConfig {
	result := make(map[string]ToolConfig)
	for _, cfg := range configs {
		for name, tc := range cfg {
			result[name] = tc
		}
	}
	return result
}

// CollectToolConfigs collects configurations from a list of tools.
func CollectToolConfigs(tools []Tool) map[string]ToolConfig {
	result := make(map[string]ToolConfig)
	for _, t := range tools {
		if cfg := GetToolConfig(t); len(cfg.Annotations) > 0 || cfg.OutputSuffix != "" {
			result[t.Name()] = cfg
		}
	}
	return result
}
