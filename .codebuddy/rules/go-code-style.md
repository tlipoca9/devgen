---
description: Go 代码风格规范 - 命名约定、错误处理、测试模式
globs: "**/*.go"
alwaysApply: false
---

# Go 代码风格规范

本规则定义了 devgen 项目的 Go 代码风格约定，确保代码一致性和可维护性。

## 命名约定

### 文件命名

| 类型 | 格式 | 示例 |
|------|------|------|
| 普通文件 | 小写 + 下划线 | `rule_loader.go`, `adapter_kiro.go` |
| 生成文件 | `*_enum.go`, `*_validate.go` | `status_enum.go` |
| 测试文件 | `*_test.go` | `generator_test.go` |

### 标识符命名

```go
// 包名：小写单词，无下划线
package genkit
package generator

// 接口：名词或动词，描述行为
type Tool interface { ... }
type ConfigurableTool interface { ... }
type ValidatableTool interface { ... }

// 结构体：名词，描述实体
type Generator struct { ... }
type Diagnostic struct { ... }
type Annotation struct { ... }

// 常量：驼峰命名，可带前缀
const ToolName = "enumgen"
const ErrCodeUnsupportedType = "E001"

// 错误码：ErrCode + 描述
const (
    ErrCodeMissingParam    = "E001"
    ErrCodeInvalidType     = "E002"
    ErrCodeUnsupportedType = "E003"
)
```

### 类型命名模式

```go
// 选项类型：*Option 或 *Options
type GeneratorOption func(*Generator)
type LoadOptions struct { ... }

// 结果类型：*Result
type DryRunResult struct { ... }

// 配置类型：*Config
type ToolConfig struct { ... }

// 收集器类型：*Collector
type DiagnosticCollector struct { ... }
```

## 错误处理模式

### 1. 错误包装

始终使用 `%w` 包装错误，保留错误链：

```go
// 正确：包装错误
if err := gen.Load(args...); err != nil {
    return fmt.Errorf("load: %w", err)
}

// 正确：带上下文的错误
if err := parseConfig(path); err != nil {
    return fmt.Errorf("parse config %s: %w", path, err)
}

// 错误：丢失原始错误
if err := gen.Load(args...); err != nil {
    return errors.New("load failed")  // 不要这样做
}
```

### 2. 诊断收集器模式

用于收集多个验证错误，而不是遇到第一个错误就返回：

```go
func (g *Generator) Validate() []Diagnostic {
    collector := genkit.NewDiagnosticCollector(ToolName)
    
    // 收集多个错误
    if field.Type == nil {
        collector.Errorf(ErrCodeMissingType, pos, "missing type for field %s", field.Name)
    }
    
    if !isValidName(field.Name) {
        collector.Warningf(ErrCodeInvalidName, pos, "field name %s should be exported", field.Name)
    }
    
    return collector.Collect()
}
```

### 3. 结构化诊断

使用 `Diagnostic` 类型报告带位置信息的错误：

```go
// 创建诊断
diag := genkit.NewDiagnostic(
    genkit.DiagnosticError,  // 严重程度
    "enumgen",               // 工具名
    "E001",                  // 错误码
    "unsupported type",      // 消息
    pos,                     // token.Position
)

// 使用收集器
collector.Error("E001", "unsupported type", pos)
collector.Errorf("E002", pos, "invalid value: %s", value)
collector.Warning("W001", "deprecated annotation", pos)
```

## 测试模式

### 表驱动测试

```go
func TestFormatPatterns(t *testing.T) {
    tests := []struct {
        name     string
        patterns []string
        want     string
    }{
        {name: "empty", patterns: []string{}, want: "[]"},
        {name: "single", patterns: []string{"*.go"}, want: "['*.go']"},
        {name: "multiple", patterns: []string{"*.go", "*.md"}, want: "['*.go', '*.md']"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := formatPatterns(tt.patterns)
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### 子测试分组

```go
func TestGenerator(t *testing.T) {
    gen := NewGenerator()

    t.Run("Load", func(t *testing.T) {
        t.Run("valid package", func(t *testing.T) { ... })
        t.Run("invalid path", func(t *testing.T) { ... })
    })

    t.Run("Generate", func(t *testing.T) {
        t.Run("enum type", func(t *testing.T) { ... })
        t.Run("struct type", func(t *testing.T) { ... })
    })
}
```

### 测试辅助函数

```go
// 使用 t.Helper() 标记辅助函数
func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func assertEqual[T comparable](t *testing.T, got, want T) {
    t.Helper()
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}
```

## 代码格式化

项目使用 golangci-lint v2，配置如下：

```yaml
formatters:
  enable:
    - gci        # import 排序
    - gofmt      # 代码格式化
    - golines    # 行长度限制 (120)
  settings:
    gofmt:
      rewrite-rules:
        - pattern: 'interface{}'
          replacement: 'any'    # 使用 any 替代 interface{}
    golines:
      max-len: 120              # 最大行长度
```

### Import 顺序

使用 gci 自动排序，顺序为：

1. 标准库
2. 第三方包
3. 空行
4. 本地模块

```go
import (
    "fmt"
    "go/token"
    "strings"

    "github.com/spf13/cobra"

    "github.com/tlipoca9/devgen/genkit"
)
```

## 注释规范

### 包注释

```go
// Package genkit provides the core framework for code generation tools.
// It includes the Generator, Tool interfaces, and diagnostic utilities.
package genkit
```

### 导出符号注释

```go
// Tool is the interface that code generation tools must implement.
// It provides a unified way to run code generators.
type Tool interface {
    // Name returns the tool name (e.g., "enumgen", "validategen").
    Name() string

    // Run processes all packages and generates code.
    // It should handle logging internally.
    Run(gen *Generator, log *Logger) error
}
```

### TODO 注释

```go
// TODO(username): description of what needs to be done
// FIXME(username): description of the bug to fix
```
