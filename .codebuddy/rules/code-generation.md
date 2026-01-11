---
description: 代码生成器开发模式 - 注解设计、模板编写、生成器实现
globs: ["cmd/**/generator/*.go", "**/*_gen.go", "**/*_enum.go", "**/*_validate.go"]
alwaysApply: false
---

# 代码生成器开发模式

本规则定义了开发新代码生成器的最佳实践和模式。

## 注解设计

### 注解格式

```
tool:@name
tool:@name(arg1, arg2)
tool:@name(key=value, flag)
tool:@name.subname(args)
```

### 注解示例

```go
// enumgen:@enum - 基础枚举
type Status int

// enumgen:@enum(string, json) - 带标志
type Priority int

// validategen:@validate(required, min=1, max=100) - 带参数
type Config struct {
    Count int // validategen:@validate(required, min=1)
}

// enumgen:@enum.value(name="Active", value=1) - 子注解
```

### 设计原则

1. **简洁性**：常用场景使用简短语法
2. **可扩展**：支持参数和标志扩展
3. **一致性**：所有工具使用相同的注解格式
4. **自文档**：注解名称应自解释

## 生成器实现模式

### 基础结构

```go
package generator

import (
    "github.com/tlipoca9/devgen/genkit"
)

const ToolName = "mytool"

// 错误码
const (
    ErrCodeMissingAnnotation = "E001"
    ErrCodeInvalidType       = "E002"
    ErrCodeUnsupportedField  = "E003"
)

type Generator struct {
    // 配置选项
    OutputSuffix string
    Verbose      bool
}

func (g *Generator) Name() string {
    return ToolName
}
```

### Run 方法模式

```go
func (g *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error {
    for _, pkg := range gen.Packages() {
        if err := g.processPackage(pkg, gen, log); err != nil {
            return fmt.Errorf("process package %s: %w", pkg.Path(), err)
        }
    }
    return nil
}

func (g *Generator) processPackage(
    pkg *genkit.Package,
    gen *genkit.Generator,
    log *genkit.Logger,
) error {
    for _, typ := range pkg.Types() {
        ann := genkit.GetAnnotation(typ.Doc(), ToolName, "generate")
        if ann == nil {
            continue
        }
        
        if err := g.processType(typ, ann, pkg, gen, log); err != nil {
            return fmt.Errorf("process type %s: %w", typ.Name(), err)
        }
    }
    return nil
}
```

### Validate 方法模式

```go
func (g *Generator) Validate(gen *genkit.Generator, log *genkit.Logger) []genkit.Diagnostic {
    collector := genkit.NewDiagnosticCollector(ToolName)
    
    for _, pkg := range gen.Packages() {
        g.validatePackage(pkg, collector, log)
    }
    
    return collector.Collect()
}

func (g *Generator) validatePackage(
    pkg *genkit.Package,
    collector *genkit.DiagnosticCollector,
    log *genkit.Logger,
) {
    for _, typ := range pkg.Types() {
        ann := genkit.GetAnnotation(typ.Doc(), ToolName, "generate")
        if ann == nil {
            continue
        }
        
        // 验证类型
        if !g.isSupportedType(typ) {
            collector.Errorf(
                ErrCodeInvalidType,
                typ.Pos(),
                "type %s is not supported for generation",
                typ.Name(),
            )
        }
        
        // 验证字段
        for _, field := range typ.Fields() {
            if !g.isSupportedField(field) {
                collector.Warningf(
                    ErrCodeUnsupportedField,
                    field.Pos(),
                    "field %s may not generate correctly",
                    field.Name(),
                )
            }
        }
    }
}
```

## 代码生成模式

### 使用 GeneratedFile

```go
func (g *Generator) generateCode(
    typ *genkit.Type,
    pkg *genkit.Package,
    gen *genkit.Generator,
) error {
    filename := strings.ToLower(typ.Name()) + g.OutputSuffix + ".go"
    f := gen.NewGeneratedFile(filename, pkg.Path())
    
    // 包声明
    f.P("package ", pkg.Name())
    f.P()
    
    // 导入
    f.P("import (")
    f.P(`    "fmt"`)
    f.P(`    "encoding/json"`)
    f.P(")")
    f.P()
    
    // 生成代码
    g.generateType(f, typ)
    
    return nil
}

func (g *Generator) generateType(f *genkit.GeneratedFile, typ *genkit.Type) {
    name := typ.Name()
    
    // 类型别名
    f.P("type ", name, "String string")
    f.P()
    
    // String 方法
    f.P("func (", strings.ToLower(name[:1]), " ", name, ") String() string {")
    f.P("    return string(", strings.ToLower(name[:1]), ")")
    f.P("}")
    f.P()
}
```

### 模板模式

对于复杂生成，使用 text/template：

```go
import "text/template"

var typeTemplate = template.Must(template.New("type").Parse(`
// {{ .Name }}String is the string representation of {{ .Name }}.
type {{ .Name }}String string

// String returns the string representation.
func ({{ .Receiver }} {{ .Name }}) String() string {
    return string({{ .Receiver }})
}

// MarshalJSON implements json.Marshaler.
func ({{ .Receiver }} {{ .Name }}) MarshalJSON() ([]byte, error) {
    return json.Marshal({{ .Receiver }}.String())
}
`))

type templateData struct {
    Name     string
    Receiver string
}

func (g *Generator) generateWithTemplate(f *genkit.GeneratedFile, typ *genkit.Type) error {
    data := templateData{
        Name:     typ.Name(),
        Receiver: strings.ToLower(typ.Name()[:1]),
    }
    
    var buf bytes.Buffer
    if err := typeTemplate.Execute(&buf, data); err != nil {
        return fmt.Errorf("execute template: %w", err)
    }
    
    f.P(buf.String())
    return nil
}
```

## 枚举生成模式

### 枚举值处理

```go
func (g *Generator) processEnum(enum *genkit.Enum, f *genkit.GeneratedFile) {
    name := enum.Name()
    values := enum.Values()
    
    // 生成常量
    f.P("const (")
    for _, v := range values {
        f.P("    ", name, v.Name(), " ", name, " = ", v.Value())
    }
    f.P(")")
    f.P()
    
    // 生成字符串映射
    f.P("var _", name, "Names = map[", name, "]string{")
    for _, v := range values {
        f.P("    ", name, v.Name(), `: "`, v.Name(), `",`)
    }
    f.P("}")
    f.P()
    
    // 生成 String 方法
    f.P("func (", strings.ToLower(name[:1]), " ", name, ") String() string {")
    f.P("    if s, ok := _", name, "Names[", strings.ToLower(name[:1]), "]; ok {")
    f.P("        return s")
    f.P("    }")
    f.P(`    return fmt.Sprintf("`, name, `(%d)", `, strings.ToLower(name[:1]), ")")
    f.P("}")
}
```

## 验证生成模式

### 字段验证

```go
func (g *Generator) generateFieldValidation(
    f *genkit.GeneratedFile,
    field *genkit.Field,
    ann *genkit.Annotation,
) {
    name := field.Name()
    
    // required 验证
    if ann.Has("required") {
        f.P("    if s.", name, " == ", g.zeroValue(field.Type()), " {")
        f.P(`        errs = append(errs, "`, name, ` is required")`)
        f.P("    }")
    }
    
    // min/max 验证
    if min := ann.Get("min"); min != "" {
        f.P("    if s.", name, " < ", min, " {")
        f.P(`        errs = append(errs, "`, name, ` must be >= `, min, `")`)
        f.P("    }")
    }
    
    if max := ann.Get("max"); max != "" {
        f.P("    if s.", name, " > ", max, " {")
        f.P(`        errs = append(errs, "`, name, ` must be <= `, max, `")`)
        f.P("    }")
    }
}
```

## AI 规则集成

### 嵌入规则文件

```go
// cmd/mytool/rules/embed.go
package rules

import "embed"

//go:embed *.md
var FS embed.FS
```

### 实现 RuleTool

```go
func (g *Generator) Rules() []genkit.Rule {
    content, _ := rules.FS.ReadFile("mytool.md")
    
    return []genkit.Rule{
        {
            Name:        "mytool",
            Description: "MyTool code generation guide",
            Globs:       []string{"**/*.go"},
            AlwaysApply: false,
            Content:     string(content),
        },
    }
}
```

### 规则文件结构

```markdown
# MyTool 使用指南

## 概述
MyTool 用于生成 XXX 代码。

## 注解

### @generate
标记类型进行代码生成。

```go
// mytool:@generate
type MyType struct { ... }
```

## 示例

### 基础用法
...

### 高级用法
...

## 常见问题
...
```

## 测试模式

### 生成器测试

```go
func TestGenerator_Run(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name: "basic enum",
            input: `
package test

// mytool:@generate
type Status int
`,
            want: `
package test

type StatusString string
`,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 设置测试环境
            // 运行生成器
            // 比较输出
        })
    }
}
```

### 验证测试

```go
func TestGenerator_Validate(t *testing.T) {
    tests := []struct {
        name       string
        input      string
        wantErrors int
        wantCodes  []string
    }{
        {
            name: "invalid type",
            input: `
package test

// mytool:@generate
type Invalid func()
`,
            wantErrors: 1,
            wantCodes:  []string{"E002"},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 运行验证
            // 检查诊断数量和错误码
        })
    }
}
```

## 最佳实践清单

- [ ] 使用常量定义工具名和错误码
- [ ] 实现 ValidatableTool 提供 IDE 集成
- [ ] 使用 DiagnosticCollector 收集错误
- [ ] 包装错误并添加上下文
- [ ] 提供详细的 AI 规则文档
- [ ] 编写表驱动测试
- [ ] 支持 dry-run 模式
- [ ] 生成的代码包含来源注释
