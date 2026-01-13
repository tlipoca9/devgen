# delegatorgen Cache Delegator 设计

> 本文档详细描述 Cache Delegator 的设计和实现。

## 一、注解规范

```go
// delegatorgen:@cache                                     // 启用缓存（使用默认配置）
// delegatorgen:@cache(ttl=5m)                             // 自定义 TTL
// delegatorgen:@cache(key="user:{id}")                    // 自定义 key 后缀（双引号）
// delegatorgen:@cache(key='user:{id}')                    // 自定义 key 后缀（单引号）
// delegatorgen:@cache(prefix="myapp:")                    // 自定义 key 前缀
// delegatorgen:@cache(prefix="{INTERFACE}:", key="{id}")  // 组合使用
// delegatorgen:@cache(ttl=5m, jitter=15)                  // 自定义 jitter（±15%）
// delegatorgen:@cache(ttl=5m, refresh=30)                 // TTL 剩余 30% 时异步刷新
// delegatorgen:@cache(ttl=5m, refresh=0)                  // 禁用异步刷新

// delegatorgen:@cache_evict(key="user:{id}")                  // 驱逐单个 key
// delegatorgen:@cache_evict(keys="user:{id},user:list:{id}")  // 驱逐多个 key（逗号分隔需要引号包裹）
```

### 1.1 注解参数语法

参数值支持以下格式：
- **无引号**：`key=value`（值中不能包含 `,` `=` `)` 等特殊字符）
- **双引号**：`key="value"`（值中可包含特殊字符，`"` 需转义为 `\"`）
- **单引号**：`key='value'`（值中可包含特殊字符，`'` 需转义为 `\'`）

```go
// 简单值（无特殊字符）可省略引号
// delegatorgen:@cache(ttl=5m, key={id})

// 包含逗号的值必须用引号
// delegatorgen:@cache_evict(keys="user:{id},list:{id}")

// 包含等号的值必须用引号
// delegatorgen:@cache(key="type=user:id={id}")

// 单引号和双引号等效
// delegatorgen:@cache(key='user:{id}')
// delegatorgen:@cache(key="user:{id}")
```

### 1.2 注解参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `ttl` | 缓存过期时间 | `5m` |
| `prefix` | 缓存 key 前缀模板 | `{PKG}:{INTERFACE}:` |
| `key` | 缓存 key 后缀模板 | `{METHOD}:{base64_json()}` |
| `jitter` | TTL 抖动百分比 | `10`（±10%） |
| `refresh` | 异步刷新阈值百分比 | `20`（TTL 剩余 20% 时刷新），设为 `0` 禁用 |

### 1.3 Key 生成规则

Key 由两部分组成：**前缀（prefix）** + **后缀（key）**

完整 key = `prefix` + `key`

#### 默认值

| 部分 | 默认模板 | 示例结果 |
|------|----------|----------|
| `prefix` | `{PKG}:{INTERFACE}:` | `github.com/myapp/user:UserRepository:` |
| `key` | `{METHOD}:{base64_json()}` | `GetByID:eyJpZCI6IjEyMyJ9` |

### 1.4 模板语法

#### 内置变量

| 变量 | 说明 | 示例值 |
|------|------|--------|
| `{PKG}` | 完整包 import 路径 | `github.com/myapp/user` |
| `{INTERFACE}` | 接口名 | `UserRepository` |
| `{METHOD}` | 方法名 | `GetByID` |

#### 参数引用

| 语法 | 说明 | 示例 |
|------|------|------|
| `{param}` | 引用方法参数 | `{id}` → `123` |
| `{param.Field}` | 引用参数的字段（支持嵌套） | `{user.ID}` → `user123` |

#### 内置函数

| 函数 | 说明 |
|------|------|
| `{base64_json(...)}` | 将参数 JSON 序列化后 Base64 编码 |

**`base64_json` 函数参数说明**：

```go
// 方法签名：GetByID(ctx context.Context, id string) (*User, error)

// 使用所有非 context 参数（推荐用于默认 key）
{base64_json()}
// 生成代码：base64JSONEncode(id)
// 结果：eyJpZCI6IjEyMyJ9

// 指定单个参数
{base64_json(id)}
// 生成代码：base64JSONEncode(id)

// 指定多个参数
{base64_json(id, name)}
// 生成代码：base64JSONEncode(id, name)
```

**多参数方法示例**：

```go
// 方法签名：List(ctx context.Context, tenantID string, filter *Filter, page int) ([]*User, error)

// 默认 key 使用 base64_json() 包含所有非 context 参数
// delegatorgen:@cache
// base64_json() → base64JSONEncode(tenantID, filter, page)
// 结果：eyJ0ZW5hbnRJRCI6InQxIiwiZmlsdGVyIjp7InN0YXR1cyI6ImFjdGl2ZSJ9LCJwYWdlIjoxfQ==

// 只使用部分参数
// delegatorgen:@cache(key="{METHOD}:{tenantID}:{base64_json(filter)}")
// 结果：List:t1:eyJzdGF0dXMiOiJhY3RpdmUifQ==

// 组合使用
// delegatorgen:@cache(key="{tenantID}:{page}:{base64_json(filter)}")
// 结果：t1:1:eyJzdGF0dXMiOiJhY3RpdmUifQ==
```

**生成的辅助函数**：

```go
// base64JSONEncode 将参数 JSON 序列化后 Base64 编码
func base64JSONEncode(args ...any) (string, error) {
    var data []byte
    var err error
    if len(args) == 1 {
        data, err = json.Marshal(args[0])
    } else {
        data, err = json.Marshal(args)
    }
    if err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(data), nil
}
```

### 1.5 Key 示例

```go
// 默认 key（使用默认 prefix 和 key）
// delegatorgen:@cache(ttl=5m)
// → github.com/myapp/user:UserRepository:GetByID:eyJpZCI6IjEyMyJ9

// 自定义后缀
// delegatorgen:@cache(ttl=5m, key="{id}")
// → github.com/myapp/user:UserRepository:123

// 自定义前缀
// delegatorgen:@cache(ttl=5m, prefix="myapp:user:")
// → myapp:user:GetByID:eyJpZCI6IjEyMyJ9

// 自定义前缀（使用内置变量）
// delegatorgen:@cache(ttl=5m, prefix="{INTERFACE}:")
// → UserRepository:GetByID:eyJpZCI6IjEyMyJ9

// 完全自定义
// delegatorgen:@cache(ttl=5m, prefix="cache:", key="user:{id}")
// → cache:user:123

// 复杂模板
// delegatorgen:@cache(ttl=5m, key="{user.TenantID}:{user.ID}")
// → github.com/myapp/user:UserRepository:tenant1:user123

// 使用 base64_json 函数
// delegatorgen:@cache(ttl=5m, key="{METHOD}:{base64_json(id, filter)}")
// → github.com/myapp/user:UserRepository:List:eyJpZCI6IjEiLCJmaWx0ZXIiOnsic3RhdHVzIjoiYWN0aXZlIn19
```

---

## 二、高级特性

| 特性 | 说明 | 默认值 | 启用条件 |
|------|------|--------|----------|
| **Jitter** | TTL 抖动，防止缓存雪崩 | ±10% | 始终启用 |
| **异步刷新** | TTL 剩余一定比例时异步刷新，避免请求阻塞 | 20% | Cache 实现了 `CacheAsyncExecutor` 接口 |
| **错误缓存** | 缓存特定错误，防止缓存穿透 | - | Cache 实现了 `SetError` 方法并返回需要缓存 |
| **分布式锁** | 缓存未命中时加锁，防止缓存击穿 | - | Cache 实现了 `CacheLocker` 接口 |

### 2.1 能力自动检测

高级特性通过**接口断言**在运行时自动检测启用，无需额外配置：

```go
// 如果 Cache 实现了 CacheLocker，自动启用分布式锁
if locker, ok := cache.(CacheLocker); ok {
    m.locker = locker
}

// 如果 Cache 实现了 CacheAsyncExecutor，自动启用异步刷新
if executor, ok := cache.(CacheAsyncExecutor); ok {
    m.asyncExecutor = executor
}
```

---

## 三、生成的接口

### 3.1 CachedResult 接口

```go
// CachedResult represents a cached value with metadata.
// Users must implement this interface to integrate with their cache library.
type CachedResult interface {
	// Value returns the cached data.
	// If IsError() returns true, this returns the cached error.
	Value() any
	// ExpiresAt returns the expiration time (used for async refresh decision).
	ExpiresAt() time.Time
	// IsError returns true if this is an error cache entry (for cache penetration prevention).
	IsError() bool
}
```

### 3.2 Cache 接口（通用接口，必须实现）

```go
// Cache is a generic caching interface.
// Implement this interface to integrate with your cache library (e.g., Redis, in-memory).
// The same implementation can be reused across all interfaces.
type Cache interface {
	// Get retrieves cached result by key.
	// Returns the cached result and whether the key was found.
	// If the cached result is an error (IsError() == true), the delegator will
	// return Value().(error) directly without calling the downstream.
	Get(ctx context.Context, key string) (result CachedResult, ok bool)

	// Set stores a value with the given TTL.
	// Implementation should wrap value into CachedResult with ExpiresAt = time.Now().Add(ttl).
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// SetError decides whether to cache an error and stores it if needed.
	// This method gives full control to the user to decide:
	//   1. Whether this error should be cached (return shouldCache=false to skip)
	//   2. What TTL to use for the error cache
	//   3. How to serialize/store the error
	//
	// Common use cases:
	//   - Cache ErrNotFound with short TTL to prevent cache penetration
	//   - Skip caching transient errors (network timeout, rate limit, etc.)
	//   - Use different TTLs for different error types
	//
	// Parameters:
	//   - key: the cache key
	//   - err: the error returned by downstream
	//   - ttl: the suggested TTL (same as normal cache TTL, user can override)
	//
	// Returns:
	//   - shouldCache: true if the error was cached, false if skipped
	//   - cacheErr: any error that occurred during caching (nil if successful or skipped)
	SetError(ctx context.Context, key string, err error, ttl time.Duration) (shouldCache bool, cacheErr error)

	// Delete removes one or more keys from the cache.
	Delete(ctx context.Context, keys ...string) error
}
```

> **设计说明**：
> - `CachedResult` 为接口：用户可自定义实现，灵活适配不同缓存库
> - `SetError` 替代 `SetNull`：将错误缓存的决策权完全交给用户
>   - 用户决定哪些错误需要缓存（如 `ErrNotFound`）
>   - 用户决定错误缓存的 TTL（可以比正常 TTL 短）
>   - 用户决定如何序列化错误（可以只存储错误类型标识）
> - 类型断言：`Get` 返回后通过 `result.Value().(*User)` 获取实际数据
> - 序列化由实现决定：Cache 实现自行处理序列化（JSON、Gob、Protobuf 等）

### 3.3 CacheLocker 接口（可选能力）

```go
// CacheLocker is an optional interface for distributed locking.
// If your cache implementation also implements this interface,
// the cache delegator will automatically use it to prevent cache stampede
// (lock on cache miss to avoid multiple concurrent requests hitting the backend).
//
// Note: This is a non-blocking lock. If the lock is not acquired, the request
// will proceed without waiting (graceful degradation). This design avoids
// blocking requests when the lock service is unavailable.
type CacheLocker interface {
	// Lock acquires a distributed lock for the given key.
	// Returns a release function and whether the lock was acquired.
	// If acquired is false, another process holds the lock, and the caller
	// should proceed without the lock (graceful degradation).
	Lock(ctx context.Context, key string) (release func(), acquired bool)
}
```

### 3.4 CacheAsyncExecutor 接口（可选能力）

```go
// CacheAsyncExecutor is an optional interface for async cache refresh.
// If your cache implementation also implements this interface,
// the cache delegator will automatically use it to refresh cache entries
// in the background when they are about to expire (controlled by refresh parameter in annotation).
type CacheAsyncExecutor interface {
	// Submit submits a task for async execution.
	// The executor should handle task queuing and concurrency control.
	Submit(task func())
}
```

---

## 四、生成的 Delegator 实现

### 4.1 Builder 方法

```go
// WithCache adds caching delegator.
// Advanced features (distributed lock, async refresh) are automatically enabled
// if the cache implementation also implements CacheLocker or CacheAsyncExecutor.
func (d *UserRepositoryDelegator) WithCache(cache Cache) *UserRepositoryDelegator {
	return d.Use(func(next UserRepository) UserRepository {
		return newUserRepositoryCacheDelegator(next, cache)
	})
}
```

### 4.2 Delegator 结构

```go
type userRepositoryCacheDelegator struct {
	next          UserRepository
	cache         Cache
	locker        CacheLocker        // 运行时检测填充
	asyncExecutor CacheAsyncExecutor // 运行时检测填充
	refreshing    sync.Map           // 记录正在刷新的 key，防止重复提交
}

func newUserRepositoryCacheDelegator(next UserRepository, cache Cache) *userRepositoryCacheDelegator {
	m := &userRepositoryCacheDelegator{
		next:  next,
		cache: cache,
	}

	// 运行时检测可选能力
	if locker, ok := cache.(CacheLocker); ok {
		m.locker = locker
	}
	if executor, ok := cache.(CacheAsyncExecutor); ok {
		m.asyncExecutor = executor
	}

	return m
}
```

### 4.3 方法实现示例

```go
// GetByID: @cache(ttl=5m, jitter=10, refresh=20)
func (m *userRepositoryCacheDelegator) GetByID(ctx context.Context, id string) (*User, error) {
	// 编译时常量（从注解生成）
	const baseTTL = 5 * time.Minute
	const jitterPercent = 10
	const refreshPercent = 20

	key, err := m.buildGetByIDKey(id)
	if err != nil {
		return nil, fmt.Errorf("build cache key: %w", err)
	}

	// 缓存命中检查
	if res, ok := m.cache.Get(ctx, key); ok {
		// 检查是否为错误缓存
		if res.IsError() {
			if err, ok := res.Value().(error); ok {
				return nil, err
			}
			// 类型不匹配，视为缓存未命中
			goto cacheMiss
		}

		// 类型断言获取实际数据
		value, ok := res.Value().(*User)
		if !ok {
			// 类型不匹配，视为缓存未命中，继续调用下游
			goto cacheMiss
		}

		// 异步刷新检查（仅当实现了 CacheAsyncExecutor 且 refresh > 0）
		if m.asyncExecutor != nil && refreshPercent > 0 {
			remaining := time.Until(res.ExpiresAt())
			threshold := baseTTL * refreshPercent / 100
			if remaining > 0 && remaining < threshold {
				// 使用 LoadOrStore 防止重复提交
				if _, loaded := m.refreshing.LoadOrStore(key, struct{}{}); !loaded {
					m.asyncExecutor.Submit(func() {
						defer m.refreshing.Delete(key)
						m.refreshGetByIDCache(context.Background(), key, id)
					})
				}
			}
		}

		return value, nil
	}

cacheMiss:
	// 缓存未命中 - 加锁（仅当实现了 CacheLocker）
	if m.locker != nil {
		release, acquired := m.locker.Lock(ctx, key)
		if acquired {
			defer release()
			// Double-check
			if res, ok := m.cache.Get(ctx, key); ok {
				if res.IsError() {
					if err, ok := res.Value().(error); ok {
						return nil, err
					}
				} else if value, ok := res.Value().(*User); ok {
					return value, nil
				}
			}
		}
	}

	// 调用下游
	result, err := m.next.GetByID(ctx, id)

	// 计算 TTL（带 jitter）
	ttl := calculateTTL(baseTTL, jitterPercent)

	if err != nil {
		// 错误缓存：由用户实现决定是否缓存此错误
		m.cache.SetError(ctx, key, err, ttl)
		return nil, err
	}

	// 存入缓存
	m.cache.Set(ctx, key, result, ttl)
	return result, nil
}

func (m *userRepositoryCacheDelegator) refreshGetByIDCache(ctx context.Context, key string, id string) {
	// 编译时常量
	const baseTTL = 5 * time.Minute
	const jitterPercent = 10
	const refreshPercent = 20

	// 加锁防止多个 goroutine 同时刷新（与正常请求共用同一把锁）
	if m.locker != nil {
		release, acquired := m.locker.Lock(ctx, key)
		if !acquired {
			return // 其他 goroutine 正在刷新或处理请求
		}
		defer release()

		// Double-check: 获取锁后检查是否还需要刷新
		if res, ok := m.cache.Get(ctx, key); ok {
			remaining := time.Until(res.ExpiresAt())
			threshold := baseTTL * refreshPercent / 100
			if remaining >= threshold {
				return // 已被其他 goroutine 刷新，无需再刷新
			}
		}
	}

	result, err := m.next.GetByID(ctx, id)
	if err != nil {
		return // 刷新失败，保留旧缓存
	}

	ttl := calculateTTL(baseTTL, jitterPercent)
	m.cache.Set(ctx, key, result, ttl)
}

// Save: @cache_evict(key=user:{user.ID})
func (m *userRepositoryCacheDelegator) Save(ctx context.Context, user *User) error {
	err := m.next.Save(ctx, user)
	if err == nil {
		// 失效缓存
		key := fmt.Sprintf("user:%s", user.ID)
		m.cache.Delete(ctx, key)
	}
	return err
}

// Delete: @cache_evict(keys=user:{id},user:list:{id})
func (m *userRepositoryCacheDelegator) Delete(ctx context.Context, id string) error {
	err := m.next.Delete(ctx, id)
	if err == nil {
		// 失效多个缓存 key
		m.cache.Delete(ctx,
			fmt.Sprintf("user:%s", id),
			fmt.Sprintf("user:list:%s", id),
		)
	}
	return err
}

func (m *userRepositoryCacheDelegator) buildGetByIDKey(id string) (string, error) {
	// 前缀（从注解 prefix 参数生成，默认为 {PKG}:{INTERFACE}:）
	const prefix = "github.com/myapp/user:UserRepository:"

	// 后缀（从注解 key 参数生成，默认为 {METHOD}:{base64_json()}）
	// base64_json() 无参数时，自动包含所有非 context 参数
	encoded, err := base64JSONEncode(id)
	if err != nil {
		return "", fmt.Errorf("failed to encode cache key: %w", err)
	}
	suffix := "GetByID:" + encoded

	return prefix + suffix, nil
}

// base64JSONEncode 将参数 JSON 序列化后 Base64 编码
// 单个参数时直接序列化，多个参数时序列化为数组
func base64JSONEncode(args ...any) (string, error) {
	var data []byte
	var err error
	if len(args) == 1 {
		data, err = json.Marshal(args[0])
	} else {
		data, err = json.Marshal(args)
	}
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func calculateTTL(baseTTL time.Duration, jitterPercent int) time.Duration {
	if jitterPercent <= 0 {
		return baseTTL
	}
	// 添加随机抖动: TTL * (1 ± JitterPercent/100)
	jitter := float64(jitterPercent) / 100.0
	factor := 1.0 + (rand.Float64()*2-1)*jitter // [1-jitter, 1+jitter]
	return time.Duration(float64(baseTTL) * factor)
}
```

---

## 五、用户适配示例

### 5.1 简单 Redis 缓存（不缓存错误）

```go
package adapters

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// cachedResult 是 CachedResult 接口的实现
type cachedResult struct {
	data      any
	expiresAt time.Time
	isError   bool
}

func (r *cachedResult) Value() any           { return r.data }
func (r *cachedResult) ExpiresAt() time.Time { return r.expiresAt }
func (r *cachedResult) IsError() bool        { return r.isError }

// cachedResultJSON 用于 JSON 序列化
type cachedResultJSON struct {
	Data      json.RawMessage `json:"data,omitempty"`
	ExpiresAt time.Time       `json:"expires_at"`
	IsError   bool            `json:"is_error,omitempty"`
	ErrorMsg  string          `json:"error_msg,omitempty"` // 用于存储错误信息
}

// SimpleRedisCache 只实现基础缓存接口，不缓存任何错误。
type SimpleRedisCache struct {
	client redis.UniversalClient
}

func NewSimpleRedisCache(client redis.UniversalClient) *SimpleRedisCache {
	return &SimpleRedisCache{client: client}
}

func (c *SimpleRedisCache) Get(ctx context.Context, key string) (CachedResult, bool) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var stored cachedResultJSON
	if err := json.Unmarshal([]byte(val), &stored); err != nil {
		return nil, false
	}

	return &cachedResult{
		data:      stored.Data, // 保持为 json.RawMessage，由调用方解析
		expiresAt: stored.ExpiresAt,
		isError:   stored.IsError,
	}, true
}

func (c *SimpleRedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	dataBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	stored := cachedResultJSON{
		Data:      dataBytes,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// SetError 不缓存任何错误
func (c *SimpleRedisCache) SetError(ctx context.Context, key string, err error, ttl time.Duration) (bool, error) {
	return false, nil // 不缓存错误
}

func (c *SimpleRedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}
```

### 5.2 支持错误缓存的 Redis 实现

```go
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// 定义需要缓存的错误
var (
	ErrNotFound = errors.New("not found")
	ErrDeleted  = errors.New("deleted")
)

// ErrorCachingRedisCache 支持错误缓存的 Redis 实现
type ErrorCachingRedisCache struct {
	client       redis.UniversalClient
	errorTTLRate float64 // 错误 TTL 相对于正常 TTL 的比例，如 0.2 表示 20%
}

func NewErrorCachingRedisCache(client redis.UniversalClient, errorTTLRate float64) *ErrorCachingRedisCache {
	if errorTTLRate <= 0 {
		errorTTLRate = 0.2 // 默认 20%
	}
	return &ErrorCachingRedisCache{
		client:       client,
		errorTTLRate: errorTTLRate,
	}
}

func (c *ErrorCachingRedisCache) Get(ctx context.Context, key string) (CachedResult, bool) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var stored cachedResultJSON
	if err := json.Unmarshal([]byte(val), &stored); err != nil {
		return nil, false
	}

	result := &cachedResult{
		expiresAt: stored.ExpiresAt,
		isError:   stored.IsError,
	}

	if stored.IsError {
		// 根据错误消息还原错误
		result.data = c.restoreError(stored.ErrorMsg)
	} else {
		result.data = stored.Data
	}

	return result, true
}

func (c *ErrorCachingRedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	dataBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	stored := cachedResultJSON{
		Data:      dataBytes,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// SetError 决定是否缓存错误
func (c *ErrorCachingRedisCache) SetError(ctx context.Context, key string, err error, ttl time.Duration) (bool, error) {
	// 只缓存特定的错误类型
	if !c.shouldCacheError(err) {
		return false, nil
	}

	// 使用较短的 TTL
	errorTTL := time.Duration(float64(ttl) * c.errorTTLRate)

	stored := cachedResultJSON{
		ExpiresAt: time.Now().Add(errorTTL),
		IsError:   true,
		ErrorMsg:  c.errorToString(err),
	}

	data, jsonErr := json.Marshal(stored)
	if jsonErr != nil {
		return false, jsonErr
	}

	if setErr := c.client.Set(ctx, key, data, errorTTL).Err(); setErr != nil {
		return false, setErr
	}

	return true, nil
}

// shouldCacheError 决定哪些错误需要缓存
func (c *ErrorCachingRedisCache) shouldCacheError(err error) bool {
	// 只缓存 "not found" 类型的错误，防止缓存穿透
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrDeleted)
}

// errorToString 将错误转换为可存储的字符串
func (c *ErrorCachingRedisCache) errorToString(err error) string {
	switch {
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrDeleted):
		return "deleted"
	default:
		return err.Error()
	}
}

// restoreError 从字符串还原错误
func (c *ErrorCachingRedisCache) restoreError(msg string) error {
	switch msg {
	case "not_found":
		return ErrNotFound
	case "deleted":
		return ErrDeleted
	default:
		return errors.New(msg)
	}
}

func (c *ErrorCachingRedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}
```

### 5.3 完整 Redis 缓存（支持所有高级功能）

```go
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// AdvancedRedisCache 实现完整的缓存接口，包括分布式锁、异步执行和错误缓存。
type AdvancedRedisCache struct {
	client       redis.UniversalClient
	lockTTL      time.Duration
	taskQueue    chan func()
	errorTTLRate float64
}

func NewAdvancedRedisCache(client redis.UniversalClient, lockTTL time.Duration, workers, queueSize int, errorTTLRate float64) *AdvancedRedisCache {
	if errorTTLRate <= 0 {
		errorTTLRate = 0.2
	}

	c := &AdvancedRedisCache{
		client:       client,
		lockTTL:      lockTTL,
		taskQueue:    make(chan func(), queueSize),
		errorTTLRate: errorTTLRate,
	}

	// 启动 worker goroutines
	for i := 0; i < workers; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// 记录 panic，防止 worker 退出
				}
			}()
			for task := range c.taskQueue {
				func() {
					defer func() {
						if r := recover(); r != nil {
							// 记录 panic
						}
					}()
					task()
				}()
			}
		}()
	}

	return c
}

// ============ 基础缓存接口 ============

func (c *AdvancedRedisCache) Get(ctx context.Context, key string) (CachedResult, bool) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var stored cachedResultJSON
	if err := json.Unmarshal([]byte(val), &stored); err != nil {
		return nil, false
	}

	result := &cachedResult{
		expiresAt: stored.ExpiresAt,
		isError:   stored.IsError,
	}

	if stored.IsError {
		result.data = c.restoreError(stored.ErrorMsg)
	} else {
		result.data = stored.Data
	}

	return result, true
}

func (c *AdvancedRedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	dataBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	stored := cachedResultJSON{
		Data:      dataBytes,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *AdvancedRedisCache) SetError(ctx context.Context, key string, err error, ttl time.Duration) (bool, error) {
	if !c.shouldCacheError(err) {
		return false, nil
	}

	errorTTL := time.Duration(float64(ttl) * c.errorTTLRate)

	stored := cachedResultJSON{
		ExpiresAt: time.Now().Add(errorTTL),
		IsError:   true,
		ErrorMsg:  c.errorToString(err),
	}

	data, jsonErr := json.Marshal(stored)
	if jsonErr != nil {
		return false, jsonErr
	}

	if setErr := c.client.Set(ctx, key, data, errorTTL).Err(); setErr != nil {
		return false, setErr
	}

	return true, nil
}

func (c *AdvancedRedisCache) shouldCacheError(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrDeleted)
}

func (c *AdvancedRedisCache) errorToString(err error) string {
	switch {
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrDeleted):
		return "deleted"
	default:
		return err.Error()
	}
}

func (c *AdvancedRedisCache) restoreError(msg string) error {
	switch msg {
	case "not_found":
		return ErrNotFound
	case "deleted":
		return ErrDeleted
	default:
		return errors.New(msg)
	}
}

func (c *AdvancedRedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// ============ CacheLocker 接口 - 自动启用分布式锁 ============

func (c *AdvancedRedisCache) Lock(ctx context.Context, key string) (release func(), acquired bool) {
	lockKey := "lock:" + key

	// 使用 SET NX 获取锁
	ok, err := c.client.SetNX(ctx, lockKey, "1", c.lockTTL).Result()
	if err != nil || !ok {
		return nil, false
	}

	return func() {
		c.client.Del(context.Background(), lockKey)
	}, true
}

// ============ CacheAsyncExecutor 接口 - 自动启用异步刷新 ============

func (c *AdvancedRedisCache) Submit(task func()) {
	select {
	case c.taskQueue <- task:
	default:
		// 队列已满，丢弃任务（或记录警告）
	}
}

func (c *AdvancedRedisCache) Close() {
	close(c.taskQueue)
}
```

---

## 六、使用示例

### 6.1 简单使用（不缓存错误）

```go
// 创建简单缓存（不缓存任何错误）
cache := adapters.NewSimpleRedisCache(redisClient)

// 组装委托器
repo := user.NewUserRepositoryDelegator(baseRepo).
	WithCache(cache).
	Build()
```

### 6.2 缓存特定错误（防止缓存穿透）

```go
// 创建支持错误缓存的实现（错误 TTL = 正常 TTL 的 20%）
cache := adapters.NewErrorCachingRedisCache(redisClient, 0.2)

// 组装委托器
repo := user.NewUserRepositoryDelegator(baseRepo).
	WithCache(cache).
	Build()
```

### 6.3 完整使用（所有高级功能）

```go
// 创建高级缓存（实现了 CacheLocker 和 CacheAsyncExecutor）
cache := adapters.NewAdvancedRedisCache(
	redisClient,
	10*time.Second, // lock TTL
	10,             // workers
	1000,           // queue size
	0.2,            // error TTL rate
)
defer cache.Close()

// 组装委托器 - 自动启用分布式锁和异步刷新
repo := user.NewUserRepositoryDelegator(baseRepo).
	WithCache(cache).
	Build()
```

---

## 七、ToolConfig 配置

```go
func (g *Generator) Config() genkit.ToolConfig {
	return genkit.ToolConfig{
		OutputSuffix: "_delegator.go",
		Annotations: []genkit.AnnotationConfig{
			{
				Name: "cache",
				Type: "field",
				Doc: `配置方法的缓存行为（仅适用于有返回值的方法）。

用法：
  // delegatorgen:@cache                                     // 启用（默认配置）
  // delegatorgen:@cache(ttl=10m)                            // 自定义 TTL
  // delegatorgen:@cache(key=user:{id})                      // 自定义 key 后缀
  // delegatorgen:@cache(prefix=myapp:)                      // 自定义 key 前缀
  // delegatorgen:@cache(ttl=5m, jitter=15)                  // 自定义 jitter（±15%）
  // delegatorgen:@cache(ttl=5m, refresh=30)                 // TTL 剩余 30% 时异步刷新
  // delegatorgen:@cache(ttl=5m, refresh=0)                  // 禁用异步刷新

Key 生成：
  完整 key = prefix + key
  默认 prefix：{PKG}:{INTERFACE}:
  默认 key：{METHOD}:{base64_json(ARGS)}
  例如：github.com/myapp/user:UserRepository:GetByID:eyJpZCI6IjEyMyJ9

内置变量：
  {PKG}       - 完整包 import 路径
  {INTERFACE} - 接口名
  {METHOD}    - 方法名
  {ARGS}      - 所有参数（用于函数调用）

参数引用：
  {param}       - 引用方法参数
  {param.Field} - 引用参数的字段

内置函数：
  {base64_json(ARGS)}      - 所有参数 JSON+Base64
  {base64_json(p1, p2)}    - 指定参数 JSON+Base64

高级特性（自动检测启用）：
  - Jitter: TTL 抖动，防止缓存雪崩（默认 ±10%）
  - 异步刷新: Cache 实现 CacheAsyncExecutor 接口时自动启用
  - 错误缓存: Cache.SetError() 方法决定是否缓存错误，防止缓存穿透
  - 分布式锁: Cache 实现 CacheLocker 接口时自动启用，防止缓存击穿`,
				Params: &genkit.AnnotationParams{
					Docs: map[string]string{
						"prefix":  "缓存 key 前缀模板（默认 {PKG}:{INTERFACE}:）",
						"key":     "缓存 key 后缀模板（默认 {METHOD}:{base64_json(ARGS)}）",
						"ttl":     "缓存过期时间（如 5m, 1h，默认 5m）",
						"jitter":  "TTL 抖动百分比（默认 10，即 ±10%）",
						"refresh": "异步刷新阈值百分比（默认 20，设为 0 禁用）",
					},
				},
			},
			{
				Name: "cache_evict",
				Type: "field",
				Doc: `配置方法执行后驱逐的缓存 key。

用法：
  // delegatorgen:@cache_evict(key=user:{id})
  // delegatorgen:@cache_evict(keys=user:{id},user:list)

注意：key 和 keys 参数互斥，只能使用其中一个。`,
				Params: &genkit.AnnotationParams{
					Docs: map[string]string{
						"key":  "要驱逐的单个缓存 key 模板",
						"keys": "要驱逐的多个缓存 key 模板（逗号分隔）",
					},
				},
			},
		},
	}
}
```
