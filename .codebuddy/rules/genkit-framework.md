---
description: genkit 框架使用指南 - Tool 接口实现、Generator API、诊断系统
globs: "**/*.go"
alwaysApply: false
---

# genkit 框架使用指南

genkit 是 devgen 的核心框架，提供代码生成工具的基础设施。

## Tool 接口层次

### 基础接口

```go
// Tool - 所有代码生成工具必须实现
type Tool interface {
    Name() string                           // 工具名称
    Run(gen *Generator, log *Logger) error  // 执行生成
}

// ConfigurableTool - 提供配置元数据
type ConfigurableTool interface {
    Tool
    Config() ToolConfig  // 返回工具配置（注解定义等）
}

// ValidatableTool - 支持验证模式
type ValidatableTool interface {
    Tool
    Validate(gen *Generator, log *Logger) []Diagnostic  // 验证而不生成
}

// RuleTool - 提供 AI 规则
type RuleTool interface {
    Tool
    Rules() []Rule  // 返回 AI 规则
}
```

### 实现新工具

```go
package mytool

import "github.com/tlipoca9/devgen/genkit"

const ToolName = "mytool"

type MyTool struct{}

func (t *MyTool) Name() string {
    return ToolName
}

func (t *MyTool) Run(gen *genkit.Generator, log *genkit.Logger) error {
    // 1. 遍历包
    for _, pkg := range gen.Packages() {
        // 2. 遍历类型
        for _, typ := range pkg.Types() {
            // 3. 检查注解
            if !genkit.HasAnnotation(typ.Doc(), ToolName, "generate") {
                continue
            }
            
            // 4. 生成代码
            f := gen.NewGeneratedFile(typ.Name()+"_gen.go", pkg.Path())
            f.P("package ", pkg.Name())
            f.P()
            f.P("// Generated code for ", typ.Name())
        }
    }
    return nil
}

// 可选：实现 ConfigurableTool
func (t *MyTool) Config() genkit.ToolConfig {
    return genkit.ToolConfig{
        Name: ToolName,
        Annotations: []genkit.AnnotationConfig{
            {
                Name:        "generate",
                Description: "Mark type for code generation",
                Args: []genkit.ArgConfig{
                    {Name: "output", Description: "Output format"},
                },
            },
        },
    }
}

// 可选：实现 ValidatableTool
func (t *MyTool) Validate(gen *genkit.Generator, log *genkit.Logger) []genkit.Diagnostic {
    collector := genkit.NewDiagnosticCollector(ToolName)
    // 验证逻辑...
    return collector.Collect()
}
```

## Generator API

### 包操作

```go
// 获取所有加载的包
packages := gen.Packages()

// 遍历包
for _, pkg := range packages {
    // 包信息
    name := pkg.Name()     // 包名
    path := pkg.Path()     // 导入路径
    dir := pkg.Dir()       // 目录路径
    
    // 类型信息
    types := pkg.Types()   // 所有类型
    enums := pkg.Enums()   // 枚举类型
    structs := pkg.Structs() // 结构体
}
```

### 类型操作

```go
// Type 表示一个类型定义
type Type struct {
    name    string
    doc     string
    pos     token.Position
    // ...
}

// 常用方法
typ.Name()      // 类型名
typ.Doc()       // 文档注释
typ.Pos()       // 源码位置
typ.Fields()    // 字段列表（结构体）
typ.Values()    // 值列表（枚举）
```

### 注解解析

```go
// 检查是否有注解
if genkit.HasAnnotation(doc, "enumgen", "enum") {
    // 处理枚举
}

// 获取注解
ann := genkit.GetAnnotation(doc, "enumgen", "enum")
if ann != nil {
    // 检查标志
    if ann.Has("json") {
        // 生成 JSON 方法
    }
    
    // 获取参数
    format := ann.Get("format")
    defaultVal := ann.GetOr("default", "unknown")
}

// 解析所有注解
annotations := genkit.ParseDoc(doc)
for _, ann := range annotations {
    fmt.Printf("Tool: %s, Name: %s\n", ann.Tool, ann.Name)
}
```

### 代码生成

```go
// 创建生成文件
f := gen.NewGeneratedFile("output_gen.go", pkgPath)

// 写入代码
f.P("package ", pkgName)
f.P()
f.P("import (")
f.P(`    "fmt"`)
f.P(`    "encoding/json"`)
f.P(")")
f.P()

// 使用模板
f.P("type ", typeName, "String string")
f.P()
f.P("func (", receiver, " ", typeName, ") String() string {")
f.P("    return string(", receiver, ")")
f.P("}")

// 格式化输出（自动处理）
// 生成的文件会自动经过 gofmt 格式化
```

## 诊断系统

### DiagnosticCollector

```go
// 创建收集器
collector := genkit.NewDiagnosticCollector(ToolName)

// 添加错误
collector.Error("E001", "missing type annotation", pos)
collector.Errorf("E002", pos, "unsupported type: %s", typeName)

// 添加警告
collector.Warning("W001", "deprecated annotation", pos)
collector.Warningf("W002", pos, "field %s should be exported", fieldName)

// 合并其他收集器
collector.Merge(otherCollector)
collector.MergeSlice(diagnostics)

// 检查是否有错误
if collector.HasErrors() {
    return collector.Collect()
}

// 获取所有诊断
return collector.Collect()
```

### 错误码规范

```go
const (
    // 错误码格式：E + 3位数字
    ErrCodeMissingAnnotation = "E001"
    ErrCodeInvalidType       = "E002"
    ErrCodeUnsupportedType   = "E003"
    ErrCodeMissingField      = "E004"
    
    // 警告码格式：W + 3位数字
    WarnCodeDeprecated       = "W001"
    WarnCodeNamingConvention = "W002"
)
```

## AI 规则系统

### Rule 结构

```go
type Rule struct {
    Name        string   // 规则文件名（无扩展名）
    Description string   // 简短描述
    Globs       []string // 触发文件模式
    AlwaysApply bool     // 是否始终应用
    Content     string   // Markdown 内容
}
```

### 实现 RuleTool

```go
//go:embed rules/*.md
var rulesFS embed.FS

func (t *MyTool) Rules() []genkit.Rule {
    return []genkit.Rule{
        {
            Name:        "mytool",
            Description: "MyTool code generation guide",
            Globs:       []string{"**/*.go"},
            AlwaysApply: false,
            Content:     mustReadFile(rulesFS, "rules/mytool.md"),
        },
    }
}
```

## 插件系统

### 源码插件

```go
// examples/plugin/plugin-source/main.go
package main

import "github.com/tlipoca9/devgen/genkit"

type Plugin struct{}

func (p *Plugin) Name() string { return "myplugin" }

func (p *Plugin) Run(gen *genkit.Generator, log *genkit.Logger) error {
    // 生成逻辑
    return nil
}

// 导出符号
var Tool genkit.Tool = &Plugin{}
```

### 配置插件

```toml
# devgen.toml
[[plugins]]
name = "myplugin"
path = "./examples/plugin/plugin-source"
type = "source"
```

## 最佳实践

### 1. 工具组织

```
cmd/mytool/
├── main.go              # 入口点（最小化）
├── generator/
│   ├── generator.go     # 主生成器
│   ├── generator_*.go   # 功能分离
│   └── *_test.go        # 测试
└── rules/
    ├── embed.go         # //go:embed
    └── mytool.md        # AI 规则
```

### 2. 错误处理

```go
func (g *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error {
    for _, pkg := range gen.Packages() {
        if err := g.processPackage(pkg, gen, log); err != nil {
            return fmt.Errorf("process package %s: %w", pkg.Path(), err)
        }
    }
    return nil
}
```

### 3. 日志使用

```go
func (g *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error {
    log.Info("Processing packages...")
    
    for _, pkg := range gen.Packages() {
        log.Debug("Processing package: %s", pkg.Path())
        
        for _, typ := range pkg.Types() {
            if shouldSkip(typ) {
                log.Debug("Skipping type: %s", typ.Name())
                continue
            }
            
            log.Info("Generating code for: %s", typ.Name())
        }
    }
    
    return nil
}
```
