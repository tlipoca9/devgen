// Package generator provides delegator code generation functionality.
package generator

// ToolName is the name of this tool, used in annotations.
const ToolName = "delegatorgen"

// Error codes for diagnostics.
const (
	ErrCodeNoMethods          = "E001"
	ErrCodeCacheNoReturn      = "E002"
	ErrCodeCacheInvalidTTL    = "E003"
	ErrCodeTraceInvalidAttr   = "E004"
	ErrCodeEvictInvalidKey    = "E005"
	ErrCodeInvalidKeyTemplate = "E006"
)

// Default values for cache configuration.
const (
	DefaultCacheTTL       = "5m"
	DefaultCacheJitter    = 10
	DefaultCacheRefresh   = 20
	DefaultCacheKeyPrefix = "{PKG}:{INTERFACE}:"
	DefaultCacheKeySuffix = "{METHOD}:{base64_json()}"
)
