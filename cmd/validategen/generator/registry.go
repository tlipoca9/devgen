// Package generator provides validation code generation functionality.
package generator

import (
	"sort"
	"sync"
)

// Registry manages validation rules.
type Registry struct {
	mu       sync.RWMutex
	rules    map[string]RuleFactory
	priority map[string]int
}

// NewRegistry creates a new rule registry.
func NewRegistry() *Registry {
	return &Registry{
		rules:    make(map[string]RuleFactory),
		priority: make(map[string]int),
	}
}

// Register registers a rule factory with a priority.
// Lower priority numbers execute first.
func (r *Registry) Register(name string, priority int, factory RuleFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[name] = factory
	r.priority[name] = priority
}

// Get returns a rule instance by name.
func (r *Registry) Get(name string) Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if factory, ok := r.rules[name]; ok {
		return factory()
	}
	return nil
}

// Priority returns the priority of a rule.
func (r *Registry) Priority(name string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.priority[name]
}

// SortRules sorts rules by priority.
func (r *Registry) SortRules(rules []*ValidateRule) []*ValidateRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sorted := make([]*ValidateRule, len(rules))
	copy(sorted, rules)

	sort.SliceStable(sorted, func(i, j int) bool {
		pi := r.priority[sorted[i].Name]
		pj := r.priority[sorted[j].Name]
		return pi < pj
	})

	return sorted
}

// DefaultRegistry is the global rule registry.
var DefaultRegistry = NewRegistry()

// Rule priorities define execution order.
// Lower numbers execute first. Use 100 intervals for extensibility.
const (
	PriorityRequired = 100
	PriorityRange    = 200
	PriorityEquality = 300
	PriorityFormat   = 400
	PriorityString   = 500
	PriorityNested   = 600
	PriorityDefault  = 1000
)
