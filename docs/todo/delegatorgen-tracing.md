# delegatorgen Tracing 中间件设计

> 本文档详细描述 Tracing 中间件的设计和实现。

## 一、注解规范

```go
// delegatorgen:@trace                      // 启用（默认 span 名 = 接口名.方法名）
// delegatorgen:@trace(span=CustomName)     // 自定义 span 名
// delegatorgen:@trace(attrs=id,name)       // 记录参数为属性
// 无注解 = 跳过（直接透传）
```

### 1.1 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `span` | 自定义 span 名称 | `{接口名}.{方法名}` |
| `attrs` | 记录为属性的参数名（逗号分隔） | 无 |

---

## 二、生成的接口

```go
// UserRepositoryTracer defines the tracing interface.
// Implement this interface to integrate with your tracing library (e.g., OpenTelemetry).
type UserRepositoryTracer interface {
	// Start begins a new span. Returns a new context and a Span.
	// attrs are key-value pairs: Start(ctx, "name", "key1", val1, "key2", val2)
	Start(ctx context.Context, spanName string, attrs ...any) (context.Context, UserRepositorySpan)
}

// UserRepositorySpan represents an active span.
type UserRepositorySpan interface {
	// End completes the span.
	End()
	// RecordError records an error on the span.
	RecordError(err error)
}
```

### 2.1 接口设计说明

- **零依赖**：接口内联生成，不依赖任何外部包
- **简单**：只有 2 个方法需要实现
- **灵活**：`attrs ...any` 支持任意类型的属性

---

## 三、生成的 Middleware 实现

```go
type userRepositoryTracingMiddleware struct {
	next   UserRepository
	tracer UserRepositoryTracer
}

// GetByID: @trace(attrs=id)
func (m *userRepositoryTracingMiddleware) GetByID(ctx context.Context, id string) (*User, error) {
	ctx, span := m.tracer.Start(ctx, "UserRepository.GetByID", "id", id)
	defer span.End()

	result, err := m.next.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

// Save: @trace (无 attrs)
func (m *userRepositoryTracingMiddleware) Save(ctx context.Context, user *User) error {
	ctx, span := m.tracer.Start(ctx, "UserRepository.Save")
	defer span.End()

	err := m.next.Save(ctx, user)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Count: 无 @trace 注解 - 直接透传
func (m *userRepositoryTracingMiddleware) Count(ctx context.Context) (int, error) {
	return m.next.Count(ctx)
}
```

### 3.1 生成逻辑

1. **有 `@trace` 注解**：
   - 调用 `tracer.Start()` 创建 span
   - 使用 `defer span.End()` 确保 span 结束
   - 如果有错误，调用 `span.RecordError()`
   - 如果指定了 `attrs`，将对应参数作为属性传入

2. **无 `@trace` 注解**：
   - 直接调用 `m.next.Method()`，不做任何包装

---

## 四、用户适配示例

### 4.1 OpenTelemetry 适配

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

### 4.2 使用示例

```go
import "go.opentelemetry.io/otel"

// 创建 tracer
tracer := adapters.NewOTelTracer(otel.Tracer("user-repo"))

// 组装委托器
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithTracing(tracer).
    Build()
```

---

## 五、ToolConfig 配置

```go
{
    Name: "trace",
    Type: "field",
    Doc: `配置方法的链路追踪行为。

用法：
  // delegatorgen:@trace                    // 启用，默认 span 名
  // delegatorgen:@trace(span=CustomName)   // 自定义 span 名
  // delegatorgen:@trace(attrs=id,name)     // 记录参数为属性
  // 无注解 = 跳过此方法`,
    Params: &genkit.AnnotationParams{
        Docs: map[string]string{
            "span":  "自定义 span 名称（默认：接口名.方法名）",
            "attrs": "记录为属性的参数名（逗号分隔）",
        },
    },
},
```

---

## 六、Validate 诊断

| 错误码 | 说明 |
|--------|------|
| `E007` | `@trace(attrs=)` 引用了不存在的参数 |

### 6.1 验证示例

```go
// delegatorgen:@trace(attrs=userId)
func (r *repo) GetByID(ctx context.Context, id string) (*User, error)
// 错误：E007 - attrs 引用了不存在的参数 "userId"，可用参数：id
```
