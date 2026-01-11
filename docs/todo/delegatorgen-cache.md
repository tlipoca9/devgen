# delegatorgen Cache 中间件设计

> 本文档详细描述 Cache 中间件的设计和实现。

## 一、注解规范

```go
// delegatorgen:@cache                                     // 启用缓存（使用默认配置）
// delegatorgen:@cache(ttl=5m)                             // 自定义 TTL
// delegatorgen:@cache(key=user:{id})                      // 自定义 key 模板
// delegatorgen:@cache(ttl=5m, jitter=15)                  // 自定义 jitter（±15%）
// delegatorgen:@cache(ttl=5m, refresh=30)                 // TTL 剩余 30% 时异步刷新
// delegatorgen:@cache(ttl=5m, refresh=0)                  // 禁用异步刷新
// delegatorgen:@cache(null=ErrNotFound)                   // 缓存特定错误（空值缓存）
// delegatorgen:@cache(null=ErrNotFound|ErrDeleted)        // 缓存多个错误
// delegatorgen:@cache(null=ErrNotFound, null_ttl=10)      // 空值 TTL = 正常 TTL 的 10%（覆盖默认 20%）
// delegatorgen:@cache_evict(key=user:{id})             // 驱逐单个 key
// delegatorgen:@cache_evict(keys=user:{id},user:list)  // 驱逐多个 key（key 和 keys 互斥，只能用一个）
```

### 1.1 注解参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `ttl` | 缓存过期时间 | `5m` |
| `key` | 缓存 key 后缀模板（前缀固定为 `{import路径}:{接口名}:`） | `{方法名}:base64(json(参数))` |
| `jitter` | TTL 抖动百分比 | `10`（±10%） |
| `refresh` | 异步刷新阈值百分比 | `20`（TTL 剩余 20% 时刷新），设为 `0` 禁用 |
| `null` | 缓存的错误类型 | 无 |
| `null_ttl` | 空值 TTL 百分比 | `20`（正常 TTL 的 20%） |

### 1.2 Key 生成规则

Key 由两部分组成：**固定前缀** + **可配置后缀**

#### 固定前缀（不可配置）

```
{完整包import路径}:{接口名}:
```

例如：`github.com/myapp/user:UserRepository:`

#### 后缀（可配置）

| 配置方式 | 说明 | 示例 |
|----------|------|------|
| 默认 | `{方法名}:base64(json(参数))` | `GetByID:eyJpZCI6IjEyMyJ9` |
| `key=` | 自定义模板 | `key={id}` → `123` |

#### 完整示例

```go
// 默认 key（无 key= 参数）
// delegatorgen:@cache(ttl=5m)
// → github.com/myapp/user:UserRepository:GetByID:eyJpZCI6IjEyMyJ9

// 自定义后缀
// delegatorgen:@cache(ttl=5m, key={id})
// → github.com/myapp/user:UserRepository:123

// 复杂模板
// delegatorgen:@cache(ttl=5m, key={user.TenantID}:{user.ID})
// → github.com/myapp/user:UserRepository:tenant1:user123
```

#### 模板语法

- `{param}` - 引用方法参数
- `{param.Field}` - 引用参数的字段（支持嵌套）

### 1.3 空值缓存

通过 `null` 参数指定需要缓存的错误类型，防止缓存穿透。空值 TTL 默认为正常 TTL 的 20%，可通过 `null_ttl` 自定义。

---

## 二、高级特性

| 特性 | 说明 | 默认值 | 启用条件 |
|------|------|--------|----------|
| **Jitter** | TTL 抖动，防止缓存雪崩 | ±10% | 始终启用 |
| **异步刷新** | TTL 剩余一定比例时异步刷新，避免请求阻塞 | 20% | Cache 实现了 `CacheAsyncExecutor` 接口 |
| **空值缓存** | 缓存特定错误，防止缓存穿透 | TTL 的 20% | 注解指定 `null=` 参数 |
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
	Value() any
	// ExpiresAt returns the expiration time (used for async refresh decision).
	ExpiresAt() time.Time
	// IsNull returns true if this is a null-value cache entry (for cache penetration prevention).
	IsNull() bool
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
	Get(ctx context.Context, key string) (result CachedResult, ok bool)

	// Set stores a value with the given TTL.
	// Implementation should wrap value into CachedResult with ExpiresAt = time.Now().Add(ttl).
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// SetNull stores a null-value cache entry with the given TTL.
	// Used for cache penetration prevention.
	SetNull(ctx context.Context, key string, ttl time.Duration) error

	// Delete removes one or more keys from the cache.
	Delete(ctx context.Context, keys ...string) error
}
```

> **设计说明**：
> - `CachedResult` 为接口：用户可自定义实现，灵活适配不同缓存库
> - `Set` 和 `SetNull` 分离：语义更清晰，避免混淆
> - 类型断言：`Get` 返回后通过 `result.Value().(*User)` 获取实际数据
> - 序列化由实现决定：Cache 实现自行处理序列化（JSON、Gob、Protobuf 等）

### 3.3 CacheLocker 接口（可选能力）

```go
// CacheLocker is an optional interface for distributed locking.
// If your cache implementation also implements this interface,
// the cache middleware will automatically use it to prevent cache stampede
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
// the cache middleware will automatically use it to refresh cache entries
// in the background when they are about to expire (controlled by refresh parameter in annotation).
type CacheAsyncExecutor interface {
	// Submit submits a task for async execution.
	// The executor should handle task queuing and concurrency control.
	Submit(task func())
}
```

---

## 四、生成的 Middleware 实现

### 4.1 Builder 方法

```go
// WithCache adds caching middleware.
// Advanced features (distributed lock, async refresh) are automatically enabled
// if the cache implementation also implements CacheLocker or CacheAsyncExecutor.
func (d *UserRepositoryDelegator) WithCache(cache Cache) *UserRepositoryDelegator {
	return d.Use(func(next UserRepository) UserRepository {
		return newUserRepositoryCacheMiddleware(next, cache)
	})
}
```

### 4.2 Middleware 结构

```go
type userRepositoryCacheMiddleware struct {
	next          UserRepository
	cache         Cache
	locker        CacheLocker        // 运行时检测填充
	asyncExecutor CacheAsyncExecutor // 运行时检测填充
	refreshing    sync.Map           // 记录正在刷新的 key，防止重复提交
}

func newUserRepositoryCacheMiddleware(next UserRepository, cache Cache) *userRepositoryCacheMiddleware {
	m := &userRepositoryCacheMiddleware{
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
// GetByID: @cache(ttl=5m, jitter=10, refresh=20, null=ErrNotFound, null_ttl=20)
func (m *userRepositoryCacheMiddleware) GetByID(ctx context.Context, id string) (*User, error) {
	// 编译时常量（从注解生成）
	const baseTTL = 5 * time.Minute
	const jitterPercent = 10
	const refreshPercent = 20
	const nullTTLPercent = 20

	key := m.buildGetByIDKey(id)

	// 缓存命中检查
	if res, ok := m.cache.Get(ctx, key); ok {
		// 检查是否为空值缓存
		if res.IsNull() {
			return nil, ErrNotFound // 返回注解中定义的错误
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
				if res.IsNull() {
					return nil, ErrNotFound
				}
				if value, ok := res.Value().(*User); ok {
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
		// 空值缓存（从注解生成的错误检查）
		if errors.Is(err, ErrNotFound) {
			nullTTL := calculateTTL(baseTTL*nullTTLPercent/100, jitterPercent)
			m.cache.SetNull(ctx, key, nullTTL)
		}
		return nil, err
	}

	// 存入缓存
	m.cache.Set(ctx, key, result, ttl)
	return result, nil
}

func (m *userRepositoryCacheMiddleware) refreshGetByIDCache(ctx context.Context, key string, id string) {
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
func (m *userRepositoryCacheMiddleware) Save(ctx context.Context, user *User) error {
	err := m.next.Save(ctx, user)
	if err == nil {
		// 失效缓存
		key := fmt.Sprintf("user:%s", user.ID)
		m.cache.Delete(ctx, key)
	}
	return err
}

// Delete: @cache_evict(keys=user:{id},user:list:{id})
func (m *userRepositoryCacheMiddleware) Delete(ctx context.Context, id string) error {
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

func (m *userRepositoryCacheMiddleware) buildGetByIDKey(id string) string {
	// 固定前缀：{import路径}:{接口名}:
	const prefix = "github.com/myapp/user:UserRepository:"

	// 默认后缀：{方法名}:base64(json(参数))
	data, _ := json.Marshal([]any{id})
	suffix := "GetByID:" + base64.StdEncoding.EncodeToString(data)

	return prefix + suffix
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

### 5.1 简单 Redis 缓存（只有基础功能）

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
	isNull    bool
}

func (r *cachedResult) Value() any         { return r.data }
func (r *cachedResult) ExpiresAt() time.Time { return r.expiresAt }
func (r *cachedResult) IsNull() bool       { return r.isNull }

// cachedResultJSON 用于 JSON 序列化
type cachedResultJSON struct {
	Data      json.RawMessage `json:"data,omitempty"`
	ExpiresAt time.Time       `json:"expires_at"`
	IsNull    bool            `json:"is_null,omitempty"`
}

// SimpleRedisCache 只实现基础缓存接口。
// 不支持分布式锁和异步刷新。
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
		isNull:    stored.IsNull,
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

func (c *SimpleRedisCache) SetNull(ctx context.Context, key string, ttl time.Duration) error {
	stored := cachedResultJSON{
		ExpiresAt: time.Now().Add(ttl),
		IsNull:    true,
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *SimpleRedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}
```

### 5.2 完整 Redis 缓存（支持所有高级功能）

```go
package adapters

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// AdvancedRedisCache 实现完整的缓存接口，包括分布式锁和异步执行。
// 通过实现 CacheLocker 和 CacheAsyncExecutor 接口，自动启用高级功能。
type AdvancedRedisCache struct {
	client    redis.UniversalClient
	lockTTL   time.Duration
	taskQueue chan func()
}

func NewAdvancedRedisCache(client redis.UniversalClient, lockTTL time.Duration, workers, queueSize int) *AdvancedRedisCache {
	c := &AdvancedRedisCache{
		client:    client,
		lockTTL:   lockTTL,
		taskQueue: make(chan func(), queueSize),
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

	return &cachedResult{
		data:      stored.Data,
		expiresAt: stored.ExpiresAt,
		isNull:    stored.IsNull,
	}, true
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

func (c *AdvancedRedisCache) SetNull(ctx context.Context, key string, ttl time.Duration) error {
	stored := cachedResultJSON{
		ExpiresAt: time.Now().Add(ttl),
		IsNull:    true,
	}

	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, ttl).Err()
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

### 6.1 简单使用（只有基础缓存）

```go
// 创建简单缓存
cache := adapters.NewSimpleRedisCache(redisClient)

// 组装委托器 - 只有基础缓存功能
repo := user.NewUserRepositoryDelegator(baseRepo).
	WithCache(cache).
	Build()
```

### 6.2 完整使用（所有高级功能）

```go
// 创建高级缓存（实现了 CacheLocker 和 CacheAsyncExecutor）
cache := adapters.NewAdvancedRedisCache(
	redisClient,
	10*time.Second, // lock TTL
	10,             // workers
	1000,           // queue size
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
  // delegatorgen:@cache(key=user:{id})                      // 自定义 key 模板
  // delegatorgen:@cache(ttl=5m, jitter=15)                  // 自定义 jitter（±15%）
  // delegatorgen:@cache(ttl=5m, refresh=30)                 // TTL 剩余 30% 时异步刷新
  // delegatorgen:@cache(ttl=5m, refresh=0)                  // 禁用异步刷新
  // delegatorgen:@cache(null=ErrNotFound)                   // 缓存特定错误
  // delegatorgen:@cache(null=ErrNotFound, null_ttl=10)      // 空值 TTL = 正常 TTL 的 10%

默认 Key 生成：
  前缀（固定）：{完整包import路径}:{接口名}:
  后缀（默认）：{方法名}:base64(json(参数))
  例如：github.com/myapp/user:UserRepository:GetByID:eyJpZCI6IjEyMyJ9

自定义 Key 后缀（key= 参数只配置后缀部分）：
  {param}       - 引用方法参数
  {param.Field} - 引用参数的字段
  例如：key={id} → github.com/myapp/user:UserRepository:123

高级特性（自动检测启用）：
  - Jitter: TTL 抖动，防止缓存雪崩（默认 ±10%）
  - 异步刷新: Cache 实现 CacheAsyncExecutor 接口时自动启用
  - 空值缓存: 注解指定 null= 参数时启用，防止缓存穿透
  - 分布式锁: Cache 实现 CacheLocker 接口时自动启用，防止缓存击穿`,
				Params: &genkit.AnnotationParams{
					Docs: map[string]string{
						"key":      "缓存 key 后缀模板（前缀固定为 {import路径}:{接口名}:）",
						"ttl":      "缓存过期时间（如 5m, 1h，默认 5m）",
						"jitter":   "TTL 抖动百分比（默认 10，即 ±10%）",
						"refresh":  "异步刷新阈值百分比（默认 20，设为 0 禁用）",
						"null":     "需要缓存的错误类型（用 | 分隔多个错误）",
						"null_ttl": "空值 TTL 百分比（默认 20，即正常 TTL 的 20%）",
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
