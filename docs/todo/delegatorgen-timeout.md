# delegatorgen Timeout Delegator 设计

> 本文档详细描述 Timeout Delegator 的设计和实现。

## 一、注解规范

```go
// delegatorgen:@timeout(5s)                // 设置 5 秒超时
// delegatorgen:@timeout(1m)                // 设置 1 分钟超时
// 无注解 = 使用默认超时（通过 WithTimeout 设置）
```

### 1.1 参数说明

注解值为 Go duration 格式：

| 格式 | 说明 |
|------|------|
| `5s` | 5 秒 |
| `100ms` | 100 毫秒 |
| `1m` | 1 分钟 |
| `1h30m` | 1 小时 30 分钟 |

---

## 二、Builder 方法

```go
// WithTimeout adds timeout delegator.
// timeout: default timeout for all methods.
func (d *UserRepositoryDelegator) WithTimeout(timeout time.Duration) *UserRepositoryDelegator {
	return d.Use(func(next UserRepository) UserRepository {
		return &userRepositoryTimeoutDelegator{next: next, timeout: timeout}
	})
}
```

---

## 三、生成的 Delegator 实现

```go
type userRepositoryTimeoutDelegator struct {
	next    UserRepository
	timeout time.Duration
}

// GetByID: 无 @timeout 注解 - 使用默认超时
func (m *userRepositoryTimeoutDelegator) GetByID(ctx context.Context, id string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	return m.next.GetByID(ctx, id)
}

// List: @timeout(10s) - 使用注解指定的超时
func (m *userRepositoryTimeoutDelegator) List(ctx context.Context, limit, offset int) ([]*User, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return m.next.List(ctx, limit, offset)
}
```

### 3.1 生成逻辑

1. **有 `@timeout` 注解**：
   - 使用注解中指定的超时时间
   - 创建带超时的 context：`context.WithTimeout(ctx, duration)`
   - 使用 `defer cancel()` 确保资源释放

2. **无 `@timeout` 注解**：
   - 使用 `WithTimeout()` 设置的默认超时
   - 如果没有调用 `WithTimeout()`，则不添加超时 Delegator

### 3.2 超时优先级

```
注解指定的超时 > WithTimeout() 默认超时 > 原始 context 超时
```

---

## 四、使用示例

```go
// 设置默认超时 10 秒
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithTimeout(10 * time.Second).
    Build()

// 方法级别可以通过注解覆盖默认超时
// @timeout(30s) 的方法会使用 30 秒超时
// 无注解的方法会使用默认的 10 秒超时
```

---

## 五、ToolConfig 配置

```go
{
    Name: "timeout",
    Type: "field",
    Doc: `配置方法的超时时间。

用法：
  // delegatorgen:@timeout(5s)        // 5 秒超时
  // 无注解 = 使用默认超时`,
    Params: &genkit.AnnotationParams{
        Type:        "string",
        Placeholder: "duration (e.g., 5s, 1m)",
    },
},
```

---

## 六、Validate 诊断

| 错误码 | 说明 |
|--------|------|
| `E005` | 无效的超时格式 |

### 6.1 验证示例

```go
// delegatorgen:@timeout(5)
func (r *repo) GetByID(ctx context.Context, id string) (*User, error)
// 错误：E005 - 无效的超时格式 "5"，应为 duration 格式（如 5s, 1m）

// delegatorgen:@timeout(abc)
func (r *repo) GetByID(ctx context.Context, id string) (*User, error)
// 错误：E005 - 无效的超时格式 "abc"，应为 duration 格式（如 5s, 1m）
```

---

## 七、注意事项

### 7.1 Context 传播

超时 Delegator 创建的 context 会传递给下游，下游可以通过 `ctx.Done()` 检测超时：

```go
func (r *repo) GetByID(ctx context.Context, id string) (*User, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err() // context.DeadlineExceeded
    case result := <-r.doQuery(ctx, id):
        return result, nil
    }
}
```

### 7.2 与其他 Delegator 的顺序

超时 Delegator 通常放在靠近底层的位置，这样超时会包含所有下游操作：

```go
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithTracing(tracer).      // 最外层
    WithMetrics(metrics).
    WithRetry(3, backoff).
    WithTimeout(10*time.Second). // 超时包含重试
    Build()
```

### 7.3 超时与重试

如果同时使用超时和重试，需要注意：

- **超时在重试外层**：整个重试过程受超时限制
- **超时在重试内层**：每次重试有独立的超时

```go
// 整个重试过程 10 秒超时
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithTimeout(10*time.Second).
    WithRetry(3, backoff).
    Build()

// 每次重试 3 秒超时
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithRetry(3, backoff).
    WithTimeout(3*time.Second).
    Build()
```
