# delegatorgen Logging Delegator 设计

> 本文档详细描述 Logging Delegator 的设计和实现。

## 一、注解规范

```go
// delegatorgen:@log                        // 启用（默认 debug 级别）
// delegatorgen:@log(level=info)            // 指定级别
// delegatorgen:@log(fields=id,name)        // 记录参数
// 无注解 = 跳过（直接透传）
```

### 1.1 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `level` | 日志级别（debug, info, warn, error） | debug |
| `fields` | 记录的参数（逗号分隔） | 无 |

---

## 二、生成的接口

```go
// UserRepositoryLogger defines the logging interface.
// Implement this interface to integrate with your logging library (e.g., slog, zap).
type UserRepositoryLogger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}
```

### 2.1 接口设计说明

- **零依赖**：接口内联生成，不依赖任何外部包
- **简单**：4 个方法对应 4 个日志级别
- **兼容**：`keysAndValues ...any` 兼容 slog、zap 等主流日志库

---

## 三、生成的 Delegator 实现

```go
type userRepositoryLoggingDelegator struct {
	next   UserRepository
	logger UserRepositoryLogger
}

// GetByID: @log(fields=id)
func (m *userRepositoryLoggingDelegator) GetByID(ctx context.Context, id string) (*User, error) {
	m.logger.Debug("UserRepository.GetByID called", "id", id)
	result, err := m.next.GetByID(ctx, id)
	if err != nil {
		m.logger.Error("UserRepository.GetByID failed", "id", id, "error", err)
	}
	return result, err
}

// Save: @log(level=info)
func (m *userRepositoryLoggingDelegator) Save(ctx context.Context, user *User) error {
	m.logger.Info("UserRepository.Save called")
	err := m.next.Save(ctx, user)
	if err != nil {
		m.logger.Error("UserRepository.Save failed", "error", err)
	}
	return err
}

// Count: 无 @log 注解 - 直接透传
func (m *userRepositoryLoggingDelegator) Count(ctx context.Context) (int, error) {
	return m.next.Count(ctx)
}
```

### 3.1 生成逻辑

1. **有 `@log` 注解**：
   - 方法调用前记录日志（使用指定级别）
   - 如果有错误，使用 `Error` 级别记录
   - 如果指定了 `fields`，将对应参数作为日志字段

2. **无 `@log` 注解**：
   - 直接调用 `m.next.Method()`，不做任何包装

---

## 四、用户适配示例

### 4.1 slog 适配

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

### 4.2 zap 适配

```go
package adapters

import "go.uber.org/zap"

// ZapLogger adapts zap to UserRepositoryLogger.
type ZapLogger struct {
    logger *zap.SugaredLogger
}

func NewZapLogger(logger *zap.Logger) *ZapLogger {
    return &ZapLogger{logger: logger.Sugar()}
}

func (l *ZapLogger) Debug(msg string, keysAndValues ...any) {
    l.logger.Debugw(msg, keysAndValues...)
}

func (l *ZapLogger) Info(msg string, keysAndValues ...any) {
    l.logger.Infow(msg, keysAndValues...)
}

func (l *ZapLogger) Warn(msg string, keysAndValues ...any) {
    l.logger.Warnw(msg, keysAndValues...)
}

func (l *ZapLogger) Error(msg string, keysAndValues ...any) {
    l.logger.Errorw(msg, keysAndValues...)
}
```

### 4.3 使用示例

```go
import "log/slog"

// 创建 logger
logger := adapters.NewSlogLogger(slog.Default())

// 组装委托器
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithLogging(logger).
    Build()
```

---

## 五、ToolConfig 配置

```go
{
    Name: "log",
    Type: "field",
    Doc: `配置方法的日志行为。

用法：
  // delegatorgen:@log                    // 启用（默认 debug）
  // delegatorgen:@log(level=info)        // 指定级别
  // delegatorgen:@log(fields=id,name)    // 记录参数
  // 无注解 = 跳过此方法`,
    Params: &genkit.AnnotationParams{
        Docs: map[string]string{
            "level":  "日志级别（debug, info, warn, error）",
            "fields": "记录的参数（逗号分隔）",
        },
    },
},
```

---

## 六、Validate 诊断

| 错误码 | 说明 |
|--------|------|
| `E009` | `@log(fields=)` 引用了不存在的参数 |

### 6.1 验证示例

```go
// delegatorgen:@log(fields=userId)
func (r *repo) GetByID(ctx context.Context, id string) (*User, error)
// 错误：E009 - fields 引用了不存在的参数 "userId"，可用参数：id
```

---

## 七、日志格式

### 7.1 调用日志

```
level=DEBUG msg="UserRepository.GetByID called" id=123
level=DEBUG msg="UserRepository.Save called"
```

### 7.2 错误日志

```
level=ERROR msg="UserRepository.GetByID failed" id=123 error="user not found"
level=ERROR msg="UserRepository.Save failed" error="database connection failed"
```

---

## 八、高级用法

### 8.1 带 Context 的日志

如果需要从 context 中提取字段（如 trace ID），可以自定义 delegator：

```go
func WithContextLogging(logger UserRepositoryLogger, extractFields func(context.Context) []any) UserRepositoryDelegatorFunc {
    return func(next UserRepository) UserRepository {
        return &contextLoggingDelegator{
            next:          next,
            logger:        logger,
            extractFields: extractFields,
        }
    }
}

// 使用
repo := user.NewUserRepositoryDelegator(baseRepo).
    Use(WithContextLogging(logger, func(ctx context.Context) []any {
        return []any{"trace_id", ctx.Value("trace_id")}
    })).
    Build()
```

### 8.2 与 Tracing 配合

日志 Delegator 通常与 Tracing 配合使用，在日志中记录 trace ID：

```go
repo := user.NewUserRepositoryDelegator(baseRepo).
    WithTracing(tracer).   // 先创建 span，注入 trace ID 到 context
    WithLogging(logger).   // 从 context 提取 trace ID 记录到日志
    Build()
```
