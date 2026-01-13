# delegatorgen Tracing Delegator 设计

> 本文档详细描述 Tracing Delegator 的设计和实现。

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

## 二、使用 OpenTelemetry 接口

直接使用 OTel 的 `trace.Tracer` 和 `trace.Span` 接口：

```go
import "go.opentelemetry.io/otel/trace"
```

### 2.1 OTel 接口

```go
// go.opentelemetry.io/otel/trace

type Tracer interface {
    Start(ctx context.Context, spanName string, opts ...SpanStartOption) (context.Context, Span)
}

type Span interface {
    End(options ...SpanEndOption)
    RecordError(err error, options ...EventOption)
    SetStatus(code codes.Code, description string)
    // ... 其他方法
}
```

### 2.2 设计说明

- **标准接口**：直接使用 OTel 标准，无需适配器
- **生态兼容**：与 OTel 生态无缝集成
- **零适配代码**：用户直接传入 `otel.Tracer`

---

## 三、生成的 Tracing Delegator 实现

```go
import (
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

type userRepositoryTracingDelegator struct {
	next   UserRepository
	tracer trace.Tracer
}

// GetByID: @trace(attrs=id)
func (d *userRepositoryTracingDelegator) GetByID(ctx context.Context, id string) (*User, error) {
	ctx, span := d.tracer.Start(ctx, "UserRepository.GetByID",
		trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	result, err := d.next.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return result, err
}

// Save: @trace (无 attrs)
func (d *userRepositoryTracingDelegator) Save(ctx context.Context, user *User) error {
	ctx, span := d.tracer.Start(ctx, "UserRepository.Save")
	defer span.End()

	err := d.next.Save(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

// Count: 无 @trace 注解 - 直接透传
func (d *userRepositoryTracingDelegator) Count(ctx context.Context) (int, error) {
	return d.next.Count(ctx)
}
```

### 3.1 生成逻辑

1. **有 `@trace` 注解**：
   - 调用 `tracer.Start()` 创建 span
   - 使用 `defer span.End()` 确保 span 结束
   - 如果有错误，调用 `span.RecordError()`
   - 如果指定了 `attrs`，将对应参数作为属性传入

2. **无 `@trace` 注解**：
   - 直接调用 `d.next.Method()`，不做任何包装

---

## 四、使用示例

```go
import "go.opentelemetry.io/otel"

// 获取 tracer
tracer := otel.Tracer("user-repo")

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
