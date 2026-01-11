# delegatorgen Retry 中间件设计

> 本文档详细描述 Retry 中间件的设计和实现。

## 一、注解规范

```go
// delegatorgen:@retry                      // 启用（默认 3 次，指数退避）
// delegatorgen:@retry(max=5)               // 最大重试次数
// 无注解 = 跳过（直接透传）
```

### 1.1 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `max` | 最大重试次数 | 3 |

---

## 二、Builder 方法

```go
// WithRetry adds retry middleware.
// maxRetries: maximum number of retry attempts (0 = no retries).
// backoff: function that returns the delay before attempt N (starting from 1).
func (d *UserRepositoryDelegator) WithRetry(maxRetries int, backoff func(attempt int) time.Duration) *UserRepositoryDelegator {
	return d.Use(func(next UserRepository) UserRepository {
		return &userRepositoryRetryMiddleware{next: next, maxRetries: maxRetries, backoff: backoff}
	})
}
```

### 2.1 退避策略

退避策略通过 `backoff` 函数参数传入，生成器提供常用的退避策略辅助函数：

```go
// ExponentialBackoff returns a backoff function with exponential delay.
// base: initial delay (e.g., 100ms)
// factor: multiplier for each attempt (e.g., 2.0)
// max: maximum delay (e.g., 10s)
func UserRepositoryExponentialBackoff(base time.Duration, factor float64, max time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		delay := base
		for i := 1; i < attempt; i++ {
			delay = time.Duration(float64(delay) * factor)
			if delay > max {
				return max
			}
		}
		return delay
	}
}

// ConstantBackoff returns a backoff function with constant delay.
func UserRepositoryConstantBackoff(delay time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		return delay
	}
}

// LinearBackoff returns a backoff function with linear delay.
// base: initial delay
// increment: delay increment per attempt
func UserRepositoryLinearBackoff(base, increment time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		return base + time.Duration(attempt-1)*increment
	}
}
```

---

## 三、生成的 Middleware 实现

```go
type userRepositoryRetryMiddleware struct {
	next       UserRepository
	maxRetries int
	backoff    func(attempt int) time.Duration
}

// GetByID: 无 @retry 注解 - 直接透传
func (m *userRepositoryRetryMiddleware) GetByID(ctx context.Context, id string) (*User, error) {
	return m.next.GetByID(ctx, id)
}

// Save: @retry(max=3)
func (m *userRepositoryRetryMiddleware) Save(ctx context.Context, user *User) error {
	maxRetries := 3 // from annotation, or use m.maxRetries as default

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := m.backoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := m.next.Save(ctx, user)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}
```

### 3.1 生成逻辑

1. **有 `@retry` 注解**：
   - 使用注解中的 `max` 值，或默认值 3
   - 循环调用下游方法，直到成功或达到最大重试次数
   - 每次重试前等待 `backoff(attempt)` 时间
   - 支持 context 取消

2. **无 `@retry` 注解**：
   - 直接调用 `m.next.Method()`，不做任何包装

### 3.2 Context 取消处理

重试过程中会检查 context 是否被取消：

```go
select {
case <-ctx.Done():
    return ctx.Err()
case <-time.After(delay):
}
```

---

## 四、使用示例

```go
// 使用指数退避
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithRetry(3, user.UserRepositoryExponentialBackoff(100*time.Millisecond, 2.0, 5*time.Second)).
    Build()

// 使用常量退避
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithRetry(5, user.UserRepositoryConstantBackoff(500*time.Millisecond)).
    Build()

// 自定义退避策略
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithRetry(3, func(attempt int) time.Duration {
        // 自定义逻辑
        return time.Duration(attempt) * 100 * time.Millisecond
    }).
    Build()
```

---

## 五、ToolConfig 配置

```go
{
    Name: "retry",
    Type: "field",
    Doc: `配置方法的重试行为。

用法：
  // delegatorgen:@retry              // 启用（默认 3 次）
  // delegatorgen:@retry(max=5)       // 最大重试次数
  // 无注解 = 跳过此方法`,
    Params: &genkit.AnnotationParams{
        Docs: map[string]string{
            "max": "最大重试次数（默认 3）",
        },
    },
},
```

---

## 六、Validate 诊断

| 错误码 | 说明 |
|--------|------|
| `E006` | 无效的重试次数（必须是正整数） |

### 6.1 验证示例

```go
// delegatorgen:@retry(max=-1)
func (r *repo) Save(ctx context.Context, user *User) error
// 错误：E006 - 无效的重试次数 "-1"，必须是正整数

// delegatorgen:@retry(max=abc)
func (r *repo) Save(ctx context.Context, user *User) error
// 错误：E006 - 无效的重试次数 "abc"，必须是正整数
```

---

## 七、高级用法

### 7.1 条件重试

如果需要根据错误类型决定是否重试，可以自定义中间件：

```go
// 自定义重试中间件，只重试特定错误
func WithConditionalRetry(maxRetries int, backoff func(int) time.Duration, shouldRetry func(error) bool) UserRepositoryMiddleware {
    return func(next UserRepository) UserRepository {
        return &conditionalRetryMiddleware{
            next:        next,
            maxRetries:  maxRetries,
            backoff:     backoff,
            shouldRetry: shouldRetry,
        }
    }
}

// 使用
repo := user.NewUserRepositoryDelegator(baseRepo).
    Use(WithConditionalRetry(3, backoff, func(err error) bool {
        // 只重试网络错误
        return errors.Is(err, ErrNetwork)
    })).
    Build()
```

### 7.2 与熔断器配合

重试中间件通常与熔断器配合使用，避免在服务不可用时持续重试：

```go
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithCircuitBreaker(cb).  // 先检查熔断器
    WithRetry(3, backoff).   // 再重试
    Build()
```
