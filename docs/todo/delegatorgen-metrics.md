# delegatorgen Metrics 中间件设计

> 本文档详细描述 Metrics 中间件的设计和实现。

## 一、注解规范

```go
// delegatorgen:@metrics                    // 启用（默认标签 = method）
// delegatorgen:@metrics(labels=type)       // 额外标签（从参数提取）
// 无注解 = 跳过（直接透传）
```

### 1.1 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `labels` | 额外的标签（从参数提取，逗号分隔） | 无 |

---

## 二、生成的接口

```go
// UserRepositoryMetrics defines the metrics interface.
// Implement this interface to integrate with your metrics library (e.g., Prometheus).
type UserRepositoryMetrics interface {
	// Observe records a method call.
	// method: the method name
	// duration: how long the call took
	// err: the error returned (nil if successful)
	// labels: additional key-value labels
	Observe(method string, duration time.Duration, err error, labels ...string)
}
```

### 2.1 接口设计说明

- **零依赖**：接口内联生成，不依赖任何外部包
- **简单**：只有 1 个方法需要实现
- **灵活**：`labels ...string` 支持动态标签

---

## 三、生成的 Middleware 实现

```go
type userRepositoryMetricsMiddleware struct {
	next    UserRepository
	metrics UserRepositoryMetrics
}

// GetByID: @metrics
func (m *userRepositoryMetricsMiddleware) GetByID(ctx context.Context, id string) (*User, error) {
	start := time.Now()
	result, err := m.next.GetByID(ctx, id)
	m.metrics.Observe("GetByID", time.Since(start), err)
	return result, err
}

// Save: @metrics(labels=type)
func (m *userRepositoryMetricsMiddleware) Save(ctx context.Context, user *User) error {
	start := time.Now()
	err := m.next.Save(ctx, user)
	m.metrics.Observe("Save", time.Since(start), err, "type", user.Type)
	return err
}

// Count: 无 @metrics 注解 - 直接透传
func (m *userRepositoryMetricsMiddleware) Count(ctx context.Context) (int, error) {
	return m.next.Count(ctx)
}
```

### 3.1 生成逻辑

1. **有 `@metrics` 注解**：
   - 记录开始时间 `start := time.Now()`
   - 调用下游方法
   - 调用 `metrics.Observe()` 记录指标
   - 如果指定了 `labels`，将对应参数值作为标签传入

2. **无 `@metrics` 注解**：
   - 直接调用 `m.next.Method()`，不做任何包装

---

## 四、用户适配示例

### 4.1 Prometheus 适配

```go
package adapters

import (
    "time"

    "github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics adapts Prometheus to UserRepositoryMetrics.
type PrometheusMetrics struct {
    requestsTotal   *prometheus.CounterVec
    requestDuration *prometheus.HistogramVec
}

func NewPrometheusMetrics(namespace, subsystem string, reg prometheus.Registerer) *PrometheusMetrics {
    m := &PrometheusMetrics{
        requestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "requests_total",
                Help:      "Total number of requests",
            },
            []string{"method", "status"},
        ),
        requestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: subsystem,
                Name:      "request_duration_seconds",
                Help:      "Request duration in seconds",
                Buckets:   prometheus.DefBuckets,
            },
            []string{"method"},
        ),
    }
    
    reg.MustRegister(m.requestsTotal, m.requestDuration)
    return m
}

func (m *PrometheusMetrics) Observe(method string, duration time.Duration, err error, labels ...string) {
    status := "success"
    if err != nil {
        status = "error"
    }
    m.requestsTotal.WithLabelValues(method, status).Inc()
    m.requestDuration.WithLabelValues(method).Observe(duration.Seconds())
}
```

### 4.2 使用示例

```go
import "github.com/prometheus/client_golang/prometheus"

// 创建 metrics
metrics := adapters.NewPrometheusMetrics("myapp", "user_repo", prometheus.DefaultRegisterer)

// 组装委托器
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithMetrics(metrics).
    Build()
```

---

## 五、ToolConfig 配置

```go
{
    Name: "metrics",
    Type: "field",
    Doc: `配置方法的指标收集行为。

用法：
  // delegatorgen:@metrics                  // 启用
  // delegatorgen:@metrics(labels=type)     // 额外标签
  // 无注解 = 跳过此方法`,
    Params: &genkit.AnnotationParams{
        Docs: map[string]string{
            "labels": "额外的标签（从参数提取，逗号分隔）",
        },
    },
},
```

---

## 六、Validate 诊断

| 错误码 | 说明 |
|--------|------|
| `E008` | `@metrics(labels=)` 引用了不存在的参数 |

### 6.1 验证示例

```go
// delegatorgen:@metrics(labels=userType)
func (r *repo) GetByID(ctx context.Context, id string) (*User, error)
// 错误：E008 - labels 引用了不存在的参数 "userType"，可用参数：id
```

---

## 七、常见指标

生成的 Metrics 中间件通常用于收集以下指标：

| 指标类型 | 说明 | 示例 |
|----------|------|------|
| **Counter** | 请求总数 | `user_repo_requests_total{method="GetByID",status="success"}` |
| **Histogram** | 请求耗时分布 | `user_repo_request_duration_seconds{method="GetByID"}` |
| **Gauge** | 当前活跃请求数（可选） | `user_repo_active_requests{method="GetByID"}` |
