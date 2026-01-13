# delegatorgen 设计文档

> 状态：待实现  
> 创建时间：2026-01-11  
> 优先级：P1

## 一、概述

delegatorgen 是一个为 Go 接口生成委托器（装饰器模式）的代码生成工具。它支持生成多种 Delegator，包括链路追踪、指标收集、缓存、重试、超时、日志和熔断等。

### 设计目标

| 目标 | 实现方式 |
|------|----------|
| **零依赖** | 所有接口内联生成，不引用任何外部包 |
| **减少心智负担** | 最小注解、合理默认值、一致的 API |
| **可扩展性** | 用户可自定义 Delegator、适配任意库 |
| **可维护性** | 生成代码清晰可读、完整的 IDE 诊断 |

### 相关文档

- [生成代码示例](./delegatorgen-generated.md)
- [用户适配示例](./delegatorgen-adapters.md)

#### Delegator 设计文档

| Delegator | 文档 | 复杂度 |
|--------|------|--------|
| Tracing | [delegatorgen-tracing.md](./delegatorgen-tracing.md) | 低 |
| Metrics | [delegatorgen-metrics.md](./delegatorgen-metrics.md) | 低 |
| Cache | [delegatorgen-cache.md](./delegatorgen-cache.md) | 高 |
| Retry | [delegatorgen-retry.md](./delegatorgen-retry.md) | 低 |
| Timeout | [delegatorgen-timeout.md](./delegatorgen-timeout.md) | 低 |
| Logging | [delegatorgen-logging.md](./delegatorgen-logging.md) | 低 |
| CircuitBreaker | [delegatorgen-circuitbreaker.md](./delegatorgen-circuitbreaker.md) | 低 |

---

## 二、注解规范

### 2.1 类型级注解

```go
// delegatorgen:@delegator
type UserRepository interface { ... }
```

唯一必需注解，标记接口需要生成委托器。

### 2.2 方法级注解

所有方法级注解都是**可选的**，用于定制特定方法的行为。详细设计请参考各 Delegator 的独立文档。

| 注解 | 说明 | 详细文档 |
|------|------|----------|
| `@trace` | 链路追踪 | [delegatorgen-tracing.md](./delegatorgen-tracing.md) |
| `@metrics` | 指标收集 | [delegatorgen-metrics.md](./delegatorgen-metrics.md) |
| `@cache` | 缓存 | [delegatorgen-cache.md](./delegatorgen-cache.md) |
| `@cache_invalidate` | 缓存失效 | [delegatorgen-cache.md](./delegatorgen-cache.md) |
| `@retry` | 重试 | [delegatorgen-retry.md](./delegatorgen-retry.md) |
| `@timeout` | 超时 | [delegatorgen-timeout.md](./delegatorgen-timeout.md) |
| `@log` | 日志 | [delegatorgen-logging.md](./delegatorgen-logging.md) |
| `@circuitbreaker` | 熔断器 | [delegatorgen-circuitbreaker.md](./delegatorgen-circuitbreaker.md) |

#### 注解速览

```go
// Tracing
// delegatorgen:@trace                      // 启用（默认 span 名 = 接口名.方法名）
// delegatorgen:@trace(span=CustomName)     // 自定义 span 名
// delegatorgen:@trace(attrs=id,name)       // 记录参数为属性

// Metrics
// delegatorgen:@metrics                    // 启用（默认标签 = method）
// delegatorgen:@metrics(labels=type)       // 额外标签（从参数提取）

// Cache
// delegatorgen:@cache                      // 启用缓存
// delegatorgen:@cache(ttl=5m)              // 自定义 TTL
// delegatorgen:@cache(key=user:{id})       // 自定义 key 模板
// delegatorgen:@cache_invalidate(key=user:{id})  // 失效缓存

// Retry
// delegatorgen:@retry                      // 启用（默认 3 次）
// delegatorgen:@retry(max=5)               // 最大重试次数

// Timeout
// delegatorgen:@timeout(5s)                // 设置超时

// Logging
// delegatorgen:@log                        // 启用（默认 debug 级别）
// delegatorgen:@log(level=info)            // 指定级别

// CircuitBreaker
// delegatorgen:@circuitbreaker             // 启用
```

---

## 三、Validate 诊断

### 3.1 错误码定义

```go
const (
    // 错误
    ErrCodeMissingContext      = "E001" // 方法缺少 context.Context 参数
    ErrCodeCacheNoResult       = "E002" // @cache 方法没有返回值
    ErrCodeInvalidCacheKey     = "E003" // 缓存 key 模板中引用了不存在的参数
    ErrCodeInvalidTTL          = "E004" // 无效的 TTL 格式
    ErrCodeInvalidTimeout      = "E005" // 无效的超时格式
    ErrCodeInvalidRetryMax     = "E006" // 无效的重试次数
    ErrCodeInvalidAttr         = "E007" // @trace(attrs=) 引用了不存在的参数
    ErrCodeInvalidLabel        = "E008" // @metrics(labels=) 引用了不存在的参数
    ErrCodeInvalidLogField     = "E009" // @log(fields=) 引用了不存在的参数
    ErrCodeCacheInvalidateNoID = "E010" // @cache_invalidate 引用了不存在的参数/字段
    
    // 警告
    WarnCodeNoMethods          = "W001" // 接口没有方法
    WarnCodeSkipAll            = "W002" // 所有方法都被 skip
)
```

### 3.2 验证规则

1. **context.Context 参数**：所有方法的第一个参数必须是 `context.Context`
2. **@cache 返回值**：带 `@cache` 注解的方法必须有返回值
3. **Key 模板验证**：`@cache` 和 `@cache_invalidate` 的 key 模板中引用的参数必须存在
4. **TTL 格式**：`@cache(ttl=)` 必须是有效的 duration 格式
5. **超时格式**：`@timeout()` 必须是有效的 duration 格式
6. **属性/标签/字段引用**：`@trace(attrs=)`、`@metrics(labels=)`、`@log(fields=)` 引用的参数必须存在

---

## 四、目录结构

```
cmd/delegatorgen/
├── main.go                    # 入口
├── generator/
│   ├── generator.go           # 主生成器
│   ├── generator_parse.go     # 解析接口和注解
│   ├── generator_validate.go  # Validate 实现
│   ├── generator_codegen.go   # 代码生成
│   ├── config.go              # ToolConfig
│   ├── constants.go           # 常量定义
│   └── templates.go           # 代码模板（可选）
└── rules/
    ├── embed.go               # //go:embed
    └── delegatorgen.md        # AI Rules
```

---

## 五、实现路线图

| 阶段 | 内容 | 复杂度 | 状态 |
|------|------|--------|------|
| **P0** | 基础框架：解析接口、生成 Builder | 中 | 待实现 |
| **P0** | Tracing Delegator | 低 | 待实现 |
| **P0** | Metrics Delegator | 低 | 待实现 |
| **P1** | Cache Delegator（基础：key 模板、TTL） | 中 | 待实现 |
| **P1** | Cache Delegator（高级：Jitter、异步刷新、空值缓存、分布式锁） | 高 | 待实现 |
| **P1** | Retry Delegator | 低 | 待实现 |
| **P1** | Timeout Delegator | 低 | 待实现 |
| **P1** | Logging Delegator | 低 | 待实现 |
| **P2** | CircuitBreaker Delegator | 低 | 待实现 |
| **P2** | Validate 诊断 | 中 | 待实现 |
| **P3** | AI Rules | 低 | 待实现 |

---

## 六、设计亮点

| 特性 | 说明 |
|------|------|
| **零依赖** | 所有接口内联生成，用户代码不依赖 devgen |
| **强类型** | 缓存接口使用具体类型，编译时类型安全 |
| **简单接口** | 每个 Delegator 只需实现 2-4 个方法 |
| **可组合** | Builder 模式，Delegator 自由组合 |
| **可扩展** | `Use()` 方法支持自定义 Delegator |
| **IDE 友好** | 完整的 Validate 诊断 |
| **生产就绪** | 缓存支持 Jitter、异步刷新、空值缓存、分布式锁 |

---

## 七、与其他生成器的对比

| 特性 | delegatorgen | enumgen | validategen | convertgen |
|------|--------------|---------|-------------|------------|
| 目标类型 | interface | type (enum) | struct | interface |
| 生成内容 | Delegator + Builder | 辅助方法 | Validate() | 转换实现 |
| 外部依赖 | 无 | 无 | 无 | 无 |
