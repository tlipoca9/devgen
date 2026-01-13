# delegatorgen CircuitBreaker Delegator 设计

> 本文档详细描述 CircuitBreaker（熔断器）Delegator 的设计和实现。

## 一、注解规范

```go
// delegatorgen:@circuitbreaker             // 启用（默认配置）
// 无注解 = 跳过（直接透传）
```

### 1.1 参数说明

熔断器注解不支持参数，所有配置通过 `WithCircuitBreaker()` 传入的实现来控制。

---

## 二、生成的接口

```go
// UserRepositoryCircuitBreaker defines the circuit breaker interface.
// Implement this interface to integrate with your circuit breaker library.
type UserRepositoryCircuitBreaker interface {
	// Allow checks if a request should be allowed.
	// Returns true if allowed, false if the circuit is open.
	Allow() bool
	// Success records a successful request.
	Success()
	// Failure records a failed request.
	Failure()
}
```

### 2.1 接口设计说明

- **零依赖**：接口内联生成，不依赖任何外部包
- **简单**：只有 3 个方法需要实现
- **状态管理**：由用户实现决定熔断策略

---

## 三、生成的 Delegator 实现

```go
type userRepositoryCircuitBreakerDelegator struct {
	next UserRepository
	cb   UserRepositoryCircuitBreaker
}

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrUserRepositoryCircuitOpen = fmt.Errorf("circuit breaker is open")

// GetByID: @circuitbreaker
func (m *userRepositoryCircuitBreakerDelegator) GetByID(ctx context.Context, id string) (*User, error) {
	if !m.cb.Allow() {
		return nil, ErrUserRepositoryCircuitOpen
	}
	result, err := m.next.GetByID(ctx, id)
	if err != nil {
		m.cb.Failure()
	} else {
		m.cb.Success()
	}
	return result, err
}

// Count: 无 @circuitbreaker 注解 - 直接透传
func (m *userRepositoryCircuitBreakerDelegator) Count(ctx context.Context) (int, error) {
	return m.next.Count(ctx)
}
```

### 3.1 生成逻辑

1. **有 `@circuitbreaker` 注解**：
   - 调用 `cb.Allow()` 检查是否允许请求
   - 如果熔断器打开，直接返回 `ErrCircuitOpen`
   - 调用下游方法后，根据结果调用 `Success()` 或 `Failure()`

2. **无 `@circuitbreaker` 注解**：
   - 直接调用 `m.next.Method()`，不做任何包装

---

## 四、熔断器状态

标准的熔断器有三种状态：

```
     ┌─────────────────────────────────────┐
     │                                     │
     ▼                                     │
┌─────────┐  failures >= threshold  ┌──────┴──┐
│ CLOSED  │ ──────────────────────► │  OPEN   │
└────┬────┘                         └────┬────┘
     │                                   │
     │ success                           │ timeout elapsed
     │                                   │
     │         ┌───────────────┐         │
     └──────── │  HALF-OPEN    │ ◄───────┘
               └───────┬───────┘
                       │
          success      │      failure
       ┌───────────────┴───────────────┐
       ▼                               ▼
   CLOSED                            OPEN
```

| 状态 | 说明 |
|------|------|
| **CLOSED** | 正常状态，允许所有请求 |
| **OPEN** | 熔断状态，拒绝所有请求 |
| **HALF-OPEN** | 半开状态，允许少量请求探测 |

---

## 五、用户适配示例

### 5.1 简单熔断器实现

```go
package adapters

import (
    "sync"
    "time"
)

// SimpleCircuitBreaker is a basic circuit breaker implementation.
type SimpleCircuitBreaker struct {
    mu           sync.Mutex
    failures     int
    threshold    int
    timeout      time.Duration
    lastFailure  time.Time
    state        string // "closed", "open", "half-open"
}

func NewSimpleCircuitBreaker(threshold int, timeout time.Duration) *SimpleCircuitBreaker {
    return &SimpleCircuitBreaker{
        threshold: threshold,
        timeout:   timeout,
        state:     "closed",
    }
}

func (cb *SimpleCircuitBreaker) Allow() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    switch cb.state {
    case "open":
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = "half-open"
            return true
        }
        return false
    case "half-open":
        return true
    default: // closed
        return true
    }
}

func (cb *SimpleCircuitBreaker) Success() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures = 0
    cb.state = "closed"
}

func (cb *SimpleCircuitBreaker) Failure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures++
    cb.lastFailure = time.Now()

    if cb.failures >= cb.threshold {
        cb.state = "open"
    }
}
```

### 5.2 使用 sony/gobreaker 适配

```go
package adapters

import "github.com/sony/gobreaker"

// GoBreakerAdapter adapts gobreaker to UserRepositoryCircuitBreaker.
type GoBreakerAdapter struct {
    cb *gobreaker.CircuitBreaker
}

func NewGoBreakerAdapter(name string, settings gobreaker.Settings) *GoBreakerAdapter {
    return &GoBreakerAdapter{
        cb: gobreaker.NewCircuitBreaker(settings),
    }
}

func (a *GoBreakerAdapter) Allow() bool {
    return a.cb.State() != gobreaker.StateOpen
}

func (a *GoBreakerAdapter) Success() {
    // gobreaker 内部管理状态
}

func (a *GoBreakerAdapter) Failure() {
    // gobreaker 内部管理状态
}

// 注意：gobreaker 的使用方式略有不同，通常通过 Execute 方法
// 这里的适配是简化版，完整适配需要自定义 delegator
```

### 5.3 使用示例

```go
// 创建熔断器：5 次失败后熔断，30 秒后尝试恢复
cb := adapters.NewSimpleCircuitBreaker(5, 30*time.Second)

// 组装委托器
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithCircuitBreaker(cb).
    Build()
```

---

## 六、ToolConfig 配置

```go
{
    Name: "circuitbreaker",
    Type: "field",
    Doc: `配置方法的熔断行为。

用法：
  // delegatorgen:@circuitbreaker         // 启用
  // 无注解 = 跳过此方法`,
},
```

---

## 七、与其他 Delegator 的配合

### 7.1 与重试配合

熔断器通常放在重试外层，避免在服务不可用时持续重试：

```go
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithCircuitBreaker(cb).  // 先检查熔断器
    WithRetry(3, backoff).   // 再重试
    Build()
```

执行顺序：
1. 熔断器检查 → 如果打开，直接返回错误
2. 如果允许，进入重试逻辑
3. 重试成功/失败后，更新熔断器状态

### 7.2 与超时配合

```go
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithCircuitBreaker(cb).
    WithTimeout(5*time.Second).
    Build()
```

超时也会触发 `Failure()`，可能导致熔断器打开。

---

## 八、高级用法

### 8.1 按方法分组熔断

如果需要不同方法使用不同的熔断器，可以自定义 delegator：

```go
func WithMethodCircuitBreakers(breakers map[string]UserRepositoryCircuitBreaker) UserRepositoryDelegatorFunc {
    return func(next UserRepository) UserRepository {
        return &methodCircuitBreakerDelegator{
            next:     next,
            breakers: breakers,
        }
    }
}

// 使用
repo := user.NewUserRepositoryDelegator(baseRepo).
    Use(WithMethodCircuitBreakers(map[string]UserRepositoryCircuitBreaker{
        "GetByID": cbRead,
        "Save":    cbWrite,
    })).
    Build()
```

### 8.2 熔断器监控

可以扩展接口添加状态查询：

```go
type CircuitBreakerWithStatus interface {
    UserRepositoryCircuitBreaker
    State() string // "closed", "open", "half-open"
    Failures() int
}
```
