# delegatorgen 用户适配示例

> 本文档展示如何实现 delegatorgen 生成的 Delegator 接口，适配常用库。
>
> 各 Delegator 的详细设计请参考：
> - [Tracing Delegator](./delegatorgen-tracing.md)
> - [Metrics Delegator](./delegatorgen-metrics.md)
> - [Cache Delegator](./delegatorgen-cache.md)
> - [Retry Delegator](./delegatorgen-retry.md)
> - [Timeout Delegator](./delegatorgen-timeout.md)
> - [Logging Delegator](./delegatorgen-logging.md)
> - [CircuitBreaker Delegator](./delegatorgen-circuitbreaker.md)

## 一、OpenTelemetry 适配

```go
package adapters

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

// OTelTracer adapts OpenTelemetry tracer to UserRepositoryTracer.
type OTelTracer struct {
    tracer trace.Tracer
}

func NewOTelTracer(tracer trace.Tracer) *OTelTracer {
    return &OTelTracer{tracer: tracer}
}

func (t *OTelTracer) Start(ctx context.Context, spanName string, attrs ...any) (context.Context, UserRepositorySpan) {
    opts := make([]trace.SpanStartOption, 0)
    
    // Convert attrs to OpenTelemetry attributes
    for i := 0; i < len(attrs); i += 2 {
        if i+1 < len(attrs) {
            key := fmt.Sprint(attrs[i])
            val := attrs[i+1]
            switch v := val.(type) {
            case string:
                opts = append(opts, trace.WithAttributes(attribute.String(key, v)))
            case int:
                opts = append(opts, trace.WithAttributes(attribute.Int(key, v)))
            case int64:
                opts = append(opts, trace.WithAttributes(attribute.Int64(key, v)))
            case bool:
                opts = append(opts, trace.WithAttributes(attribute.Bool(key, v)))
            default:
                opts = append(opts, trace.WithAttributes(attribute.String(key, fmt.Sprint(v))))
            }
        }
    }
    
    ctx, span := t.tracer.Start(ctx, spanName, opts...)
    return ctx, &otelSpan{span: span}
}

type otelSpan struct {
    span trace.Span
}

func (s *otelSpan) End() {
    s.span.End()
}

func (s *otelSpan) RecordError(err error) {
    s.span.RecordError(err)
    s.span.SetStatus(codes.Error, err.Error())
}
```

---

## 二、Prometheus 适配

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

---

## 三、slog 日志适配

```go
package adapters

import "log/slog"

// SlogLogger adapts slog to UserRepositoryLogger.
type SlogLogger struct {
    logger *slog.Logger
}

func NewSlogLogger(logger *slog.Logger) *SlogLogger {
    return &SlogLogger{logger: logger}
}

func (l *SlogLogger) Debug(msg string, keysAndValues ...any) {
    l.logger.Debug(msg, keysAndValues...)
}

func (l *SlogLogger) Info(msg string, keysAndValues ...any) {
    l.logger.Info(msg, keysAndValues...)
}

func (l *SlogLogger) Warn(msg string, keysAndValues ...any) {
    l.logger.Warn(msg, keysAndValues...)
}

func (l *SlogLogger) Error(msg string, keysAndValues ...any) {
    l.logger.Error(msg, keysAndValues...)
}
```

---

## 四、简单熔断器实现

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

---

## 五、完整使用示例

```go
package main

import (
    "context"
    "log/slog"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/redis/go-redis/v9"
    "go.opentelemetry.io/otel"

    "myapp/adapters"
    "myapp/user"
)

func main() {
    // 创建基础实现
    baseRepo := &PostgresUserRepository{db: db}

    // 创建 Redis 客户端
    redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

    // 创建适配器
    tracer := adapters.NewOTelTracer(otel.Tracer("user-repo"))
    metrics := adapters.NewPrometheusMetrics("myapp", "user_repo", prometheus.DefaultRegisterer)
    logger := adapters.NewSlogLogger(slog.Default())
    cb := adapters.NewSimpleCircuitBreaker(5, 30*time.Second)

    // Cache 相关适配器（详见 delegatorgen-cache.md）
    cache := adapters.NewRedisUserRepositoryCache(redisClient, "myapp:user:")
    locker := adapters.NewRedisLocker(redisClient, "myapp:lock:", 10*time.Second)
    executor := adapters.NewSimpleAsyncExecutor(10, 1000)

    // 组装委托器
    repo := user.NewUserRepositoryDelegator(baseRepo).
        WithTracing(tracer).
        WithMetrics(metrics).
        WithLogging(logger).
        WithCircuitBreaker(cb).
        WithRetry(3, user.UserRepositoryExponentialBackoff(100*time.Millisecond, 2.0, 5*time.Second)).
        WithTimeout(10 * time.Second).
        WithCache(cache, user.UserRepositoryCacheOptions{
            DefaultTTL:            5 * time.Minute,
            JitterPercent:         10,
            AsyncRefreshThreshold: 0.2,
            NullValueTTLRatio:     0.2,
            Locker:                locker,
            AsyncExecutor:         executor,
        }).
        Build()

    // 使用
    ctx := context.Background()
    u, err := repo.GetByID(ctx, "123")
    // 执行顺序: Tracing → Metrics → Logging → CircuitBreaker → Retry → Timeout → Cache → Base
}
```
