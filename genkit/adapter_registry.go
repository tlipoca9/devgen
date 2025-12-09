package genkit

import "sort"

// AdapterRegistry manages available agent adapters.
// It provides a central registry for all supported AI agents,
// allowing tools to discover and use adapters dynamically.
type AdapterRegistry struct {
	adapters map[string]AgentAdapter
}

// NewAdapterRegistry creates a new registry with built-in adapters.
// The registry is pre-populated with adapters for Kiro, CodeBuddy, and Cursor.
func NewAdapterRegistry() *AdapterRegistry {
	registry := &AdapterRegistry{
		adapters: make(map[string]AgentAdapter),
	}

	// Register built-in adapters
	registry.Register(&KiroAdapter{})
	registry.Register(&CodeBuddyAdapter{})
	registry.Register(&CursorAdapter{})

	return registry
}

// Register adds an adapter to the registry.
// If an adapter with the same name already exists, it will be replaced.
// This allows users to override built-in adapters with custom implementations.
func (r *AdapterRegistry) Register(adapter AgentAdapter) {
	r.adapters[adapter.Name()] = adapter
}

// Get retrieves an adapter by name.
// Returns the adapter and true if found, nil and false otherwise.
//
// Example:
//
//	adapter, ok := registry.Get("kiro")
//	if !ok {
//	    return fmt.Errorf("unknown agent: kiro")
//	}
func (r *AdapterRegistry) Get(name string) (AgentAdapter, bool) {
	adapter, ok := r.adapters[name]
	return adapter, ok
}

// List returns all registered adapter names in alphabetical order.
// This is useful for displaying available agents to users.
//
// Example output: ["codebuddy", "cursor", "kiro"]
func (r *AdapterRegistry) List() []string {
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
