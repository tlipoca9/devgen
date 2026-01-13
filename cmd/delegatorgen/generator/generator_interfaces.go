package generator

import "github.com/tlipoca9/devgen/genkit"

// generateInterfaces generates the interface definitions for delegators.
func (g *Generator) generateInterfaces(gf *genkit.GeneratedFile, iface *genkit.Interface, hasCache, hasTracing bool) {
	ifaceName := iface.Name

	gf.P()
	gf.P("// =============================================================================")
	gf.P("// Delegator Interfaces")
	gf.P("// =============================================================================")

	if hasCache {
		g.generateCacheInterfaces(gf, ifaceName)
	}

	// Note: Tracing uses OpenTelemetry interfaces directly, no custom interfaces needed
}

// generateCacheInterfaces generates cache-related interface definitions.
func (g *Generator) generateCacheInterfaces(gf *genkit.GeneratedFile, ifaceName string) {
	gf.P()
	gf.P("// ", ifaceName, "CachedResult represents a cached value with metadata.")
	gf.P("// Users must implement this interface to integrate with their cache library.")
	gf.P("type ", ifaceName, "CachedResult interface {")
	gf.P("// Value returns the cached data.")
	gf.P("// If IsError() returns true, this returns the cached error.")
	gf.P("Value() any")
	gf.P("// ExpiresAt returns the expiration time (used for async refresh decision).")
	gf.P("ExpiresAt() ", genkit.GoImportPath("time").Ident("Time"))
	gf.P("// IsError returns true if this is an error cache entry (for cache penetration prevention).")
	gf.P("IsError() bool")
	gf.P("}")

	gf.P()
	gf.P("// ", ifaceName, "Cache is a generic caching interface.")
	gf.P("// Implement this interface to integrate with your cache library (e.g., Redis, in-memory).")
	gf.P("type ", ifaceName, "Cache interface {")
	gf.P("// Get retrieves cached result by key.")
	gf.P("// Returns the cached result and whether the key was found.")
	gf.P("Get(ctx ", genkit.GoImportPath("context").Ident("Context"), ", key string) (result ", ifaceName, "CachedResult, ok bool)")
	gf.P()
	gf.P("// Set stores a value with the given TTL.")
	gf.P("Set(ctx ", genkit.GoImportPath("context").Ident("Context"), ", key string, value any, ttl ", genkit.GoImportPath("time").Ident("Duration"), ") error")
	gf.P()
	gf.P("// SetError decides whether to cache an error and stores it if needed.")
	gf.P("// Returns shouldCache=true if the error was cached.")
	gf.P("SetError(ctx ", genkit.GoImportPath("context").Ident("Context"), ", key string, err error, ttl ", genkit.GoImportPath("time").Ident("Duration"), ") (shouldCache bool, cacheErr error)")
	gf.P()
	gf.P("// Delete removes one or more keys from the cache.")
	gf.P("Delete(ctx ", genkit.GoImportPath("context").Ident("Context"), ", keys ...string) error")
	gf.P("}")

	gf.P()
	gf.P("// ", ifaceName, "CacheLocker is an optional interface for distributed locking.")
	gf.P("// If your cache implementation also implements this interface,")
	gf.P("// the cache delegator will automatically use it to prevent cache stampede.")
	gf.P("type ", ifaceName, "CacheLocker interface {")
	gf.P("// Lock acquires a distributed lock for the given key.")
	gf.P("// Returns a release function and whether the lock was acquired.")
	gf.P("Lock(ctx ", genkit.GoImportPath("context").Ident("Context"), ", key string) (release func(), acquired bool)")
	gf.P("}")

	gf.P()
	gf.P("// ", ifaceName, "CacheAsyncExecutor is an optional interface for async cache refresh.")
	gf.P("// If your cache implementation also implements this interface,")
	gf.P("// the cache delegator will automatically use it to refresh cache entries in the background.")
	gf.P("type ", ifaceName, "CacheAsyncExecutor interface {")
	gf.P("// Submit submits a task for async execution.")
	gf.P("Submit(task func())")
	gf.P("}")
}
