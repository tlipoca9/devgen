# delegatorgen - 接口委托器代码生成工具

delegatorgen 是一个 Go 代码生成工具，用于为接口生成装饰器模式的委托器代码，支持缓存和链路追踪功能。

## 快速开始

### 1. 定义接口

```go
// delegatorgen:@delegator
type UserRepository interface {
    // delegatorgen:@cache(ttl=5m)
    // delegatorgen:@trace(attrs=id)
    GetByID(ctx context.Context, id string) (*User, error)

    // delegatorgen:@cache_evict(key="user:{user.ID}")
    // delegatorgen:@trace
    Save(ctx context.Context, user *User) error
}
```

### 2. 运行生成器

```bash
delegatorgen ./...
```

### 3. 使用生成的代码

```go
import "go.opentelemetry.io/otel"

// 创建基础实现
baseRepo := NewUserRepositoryImpl(db)

// 创建缓存实现
cache := NewRedisCache(redisClient)

// 获取 tracer
tracer := otel.Tracer("user-repo")

// 组装委托器
repo := NewUserRepositoryDelegator(baseRepo).
    WithCache(cache).
    WithTracing(tracer).
    Build()
```

## 注解说明

### @delegator（类型注解）

标记接口需要生成委托器。

```go
// delegatorgen:@delegator
type UserRepository interface { ... }
```

### @cache（方法注解）

为方法启用缓存功能。

```go
// delegatorgen:@cache                                     // 默认配置
// delegatorgen:@cache(ttl=10m)                            // 自定义 TTL
// delegatorgen:@cache(key="user:{id}")                    // 自定义 key
// delegatorgen:@cache(prefix="myapp:", key="{id}")        // 自定义前缀和 key
// delegatorgen:@cache(ttl=5m, jitter=15, refresh=30)      // 完整配置
```

**参数说明：**

| 参数 | 说明 | 默认值 |
|------|------|--------|
| ttl | 缓存过期时间 | 5m |
| prefix | key 前缀模板 | {PKG}:{INTERFACE}: |
| key | key 后缀模板 | {METHOD}:{base64_json()} |
| jitter | TTL 抖动百分比 | 10 |
| refresh | 异步刷新阈值百分比 | 20 |

**Key 模板变量：**

- `{PKG}` - 包路径
- `{INTERFACE}` - 接口名
- `{METHOD}` - 方法名
- `{param}` - 方法参数
- `{param.Field}` - 参数字段
- `{base64_json()}` - 所有非 context 参数的 JSON+Base64
- `{base64_json(p1, p2)}` - 指定参数的 JSON+Base64

### @cache_evict（方法注解）

方法执行后驱逐缓存。

```go
// delegatorgen:@cache_evict(key="user:{id}")
// delegatorgen:@cache_evict(keys="user:{id},user:list")
```

### @trace（方法注解）

为方法启用链路追踪（使用 OpenTelemetry）。

```go
// delegatorgen:@trace                      // 默认 span 名
// delegatorgen:@trace(span="CustomName")   // 自定义 span 名
// delegatorgen:@trace(attrs=id,name)       // 记录参数为属性
```

## 生成的接口

### Cache 接口

用户需要实现以下接口来集成缓存：

```go
type UserRepositoryCache interface {
    Get(ctx context.Context, key string) (result UserRepositoryCachedResult, ok bool)
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    SetError(ctx context.Context, key string, err error, ttl time.Duration) (shouldCache bool, cacheErr error)
    Delete(ctx context.Context, keys ...string) error
}

type UserRepositoryCachedResult interface {
    Value() any
    ExpiresAt() time.Time
    IsError() bool
}
```

### 可选接口

```go
// 分布式锁（防止缓存击穿）
type UserRepositoryCacheLocker interface {
    Lock(ctx context.Context, key string) (release func(), acquired bool)
}

// 异步执行（用于异步刷新）
type UserRepositoryCacheAsyncExecutor interface {
    Submit(task func())
}
```

## 高级特性

### TTL 抖动（Jitter）

防止缓存雪崩，默认 ±10%。

### 异步刷新

当 TTL 剩余不足阈值时，异步刷新缓存，避免请求阻塞。需要 Cache 实现 `CacheAsyncExecutor` 接口。

### 错误缓存

通过 `SetError` 方法，用户可以决定是否缓存特定错误（如 NotFound），防止缓存穿透。

### 分布式锁

缓存未命中时加锁，防止缓存击穿。需要 Cache 实现 `CacheLocker` 接口。

## 完整示例

### 用户代码

```go
package user

import "context"

type User struct {
    ID    string
    Name  string
    Email string
}

// delegatorgen:@delegator
type UserRepository interface {
    // delegatorgen:@cache(ttl=5m)
    // delegatorgen:@trace(attrs=id)
    GetByID(ctx context.Context, id string) (*User, error)

    // delegatorgen:@cache(ttl=10m, key="{tenantID}:{base64_json(filter)}")
    // delegatorgen:@trace
    List(ctx context.Context, tenantID string, filter *Filter) ([]*User, error)

    // delegatorgen:@cache_evict(key="user:{user.ID}")
    // delegatorgen:@trace
    Save(ctx context.Context, user *User) error

    // 无注解 = 直接透传
    Count(ctx context.Context) (int, error)
}
```

### Redis Cache 实现示例

```go
type RedisCache struct {
    client redis.UniversalClient
}

func (c *RedisCache) Get(ctx context.Context, key string) (UserRepositoryCachedResult, bool) {
    val, err := c.client.Get(ctx, key).Result()
    if err != nil {
        return nil, false
    }
    // 反序列化并返回
    // ...
}

func (c *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
    data, _ := json.Marshal(value)
    return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) SetError(ctx context.Context, key string, err error, ttl time.Duration) (bool, error) {
    // 只缓存 NotFound 错误
    if errors.Is(err, ErrNotFound) {
        // 使用较短的 TTL
        return true, c.client.Set(ctx, key, "NOT_FOUND", ttl/5).Err()
    }
    return false, nil
}

func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
    return c.client.Del(ctx, keys...).Err()
}
```
