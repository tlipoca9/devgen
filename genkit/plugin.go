// Package genkit provides plugin loading functionality for devgen tools.
package genkit

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"
)

// PluginLoader loads and manages external tool plugins.
type PluginLoader struct {
	// cacheDir is the directory for compiled plugin cache.
	cacheDir string

	// loaded tracks loaded plugins to avoid duplicate loading.
	loaded map[string]Tool

	mu sync.Mutex
}

// NewPluginLoader creates a new plugin loader.
// cacheDir is used to store compiled plugins; if empty, uses system temp dir.
func NewPluginLoader(cacheDir string) *PluginLoader {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "devgen-plugins")
	}
	return &PluginLoader{
		cacheDir: cacheDir,
		loaded:   make(map[string]Tool),
	}
}

// LoadPlugin loads a plugin based on its configuration.
func (pl *PluginLoader) LoadPlugin(ctx context.Context, cfg PluginConfig) (Tool, error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	// Check if already loaded
	if tool, ok := pl.loaded[cfg.Name]; ok {
		return tool, nil
	}

	var tool Tool
	var err error

	switch cfg.Type {
	case PluginTypeSource, "":
		tool, err = pl.loadSourcePlugin(ctx, cfg)
	case PluginTypePlugin:
		tool, err = pl.loadGoPlugin(cfg)
	default:
		return nil, fmt.Errorf("unknown plugin type: %s", cfg.Type)
	}

	if err != nil {
		return nil, err
	}

	pl.loaded[cfg.Name] = tool
	return tool, nil
}

// LoadPlugins loads all plugins from the configuration.
func (pl *PluginLoader) LoadPlugins(ctx context.Context, cfg *Config) ([]Tool, error) {
	var tools []Tool
	for _, pluginCfg := range cfg.Plugins {
		tool, err := pl.LoadPlugin(ctx, pluginCfg)
		if err != nil {
			return nil, fmt.Errorf("load plugin %s: %w", pluginCfg.Name, err)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// loadSourcePlugin compiles and loads a Go source plugin.
func (pl *PluginLoader) loadSourcePlugin(ctx context.Context, cfg PluginConfig) (Tool, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(pl.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	// Check if source path exists
	srcPath := cfg.Path
	info, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("source path not found: %s", srcPath)
	}

	// Determine output .so path based on source modification time
	var modTime time.Time
	if info.IsDir() {
		// Get latest modification time from Go files in directory
		modTime, err = getLatestModTime(srcPath)
		if err != nil {
			return nil, err
		}
	} else {
		modTime = info.ModTime()
	}

	soName := fmt.Sprintf("%s_%d.so", cfg.Name, modTime.Unix())
	soPath := filepath.Join(pl.cacheDir, soName)

	// Check if cached version exists
	if _, err := os.Stat(soPath); os.IsNotExist(err) {
		// Compile the plugin
		if err := pl.compilePlugin(ctx, srcPath, soPath); err != nil {
			return nil, fmt.Errorf("compile plugin: %w", err)
		}
	}

	// Load the compiled plugin
	return pl.loadGoPluginFile(soPath, cfg.Name)
}

// compilePlugin compiles Go source to a plugin .so file.
func (pl *PluginLoader) compilePlugin(ctx context.Context, srcPath, outPath string) error {
	// Build command: go build -buildmode=plugin -o outPath srcPath
	cmd := exec.CommandContext(ctx, "go", "build", "-buildmode=plugin", "-o", outPath, srcPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, stderr.String())
	}

	return nil
}

// loadGoPlugin loads a pre-compiled Go plugin (.so file).
func (pl *PluginLoader) loadGoPlugin(cfg PluginConfig) (Tool, error) {
	return pl.loadGoPluginFile(cfg.Path, cfg.Name)
}

// loadGoPluginFile loads a .so file and extracts the Tool.
func (pl *PluginLoader) loadGoPluginFile(path, name string) (Tool, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open plugin %s: %w", path, err)
	}

	// Look for exported "Tool" symbol
	sym, err := p.Lookup("Tool")
	if err != nil {
		// Try looking for "New" function
		newSym, newErr := p.Lookup("New")
		if newErr != nil {
			return nil, fmt.Errorf("plugin %s: no 'Tool' variable or 'New' function found", name)
		}

		// Try different function signatures
		switch newFn := newSym.(type) {
		case func() Tool:
			return newFn(), nil
		case func() interface{}:
			if tool, ok := newFn().(Tool); ok {
				return tool, nil
			}
			return nil, fmt.Errorf("plugin %s: New() does not return Tool", name)
		default:
			return nil, fmt.Errorf("plugin %s: New has unexpected type %T", name, newSym)
		}
	}

	// Check if it's a Tool or *Tool
	switch t := sym.(type) {
	case Tool:
		return t, nil
	case *Tool:
		return *t, nil
	default:
		return nil, fmt.Errorf("plugin %s: Tool has unexpected type %T", name, sym)
	}
}

// getLatestModTime returns the latest modification time of Go files in a directory.
func getLatestModTime(dir string) (time.Time, error) {
	var latest time.Time

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			if info.ModTime().After(latest) {
				latest = info.ModTime()
			}
		}
		return nil
	})

	if err != nil {
		return time.Time{}, err
	}
	if latest.IsZero() {
		return time.Time{}, fmt.Errorf("no Go files found in %s", dir)
	}
	return latest, nil
}

// CleanCache removes old cached plugins.
func (pl *PluginLoader) CleanCache(maxAge time.Duration) error {
	entries, err := os.ReadDir(pl.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(pl.cacheDir, entry.Name()))
		}
	}
	return nil
}
