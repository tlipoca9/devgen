# AI Rules 适配器系统

中文 | [English](rules-adapter_EN.md)

devgen 中的适配器系统允许你维护单一的 AI 规则源，并自动为不同的 AI 助手生成特定格式。

## 概述

适配器系统遵循"源 → 适配器 → 输出"模式：

```
源规则 (cmd/*/rules/*.md)
        ↓
    适配器 (Kiro/CodeBuddy/Cursor)
        ↓
生成的规则 (.kiro/steering/*.md 等)
```

### 为什么需要适配器？

不同的 AI 助手使用不同的 frontmatter 格式和约定。与其为每个助手维护单独的文档，不如编写一次规则，让适配器处理转换。

## 内置适配器

devgen 包含三个内置适配器：

| 适配器 | 输出目录 | 文件扩展名 | Frontmatter 格式 |
|--------|---------|-----------|-----------------|
| **Kiro** | `.kiro/steering/` | `.md` | YAML，包含 `inclusion` 和 `fileMatchPattern` |
| **CodeBuddy** | `.codebuddy/rules/` | `.mdc` | YAML，包含 `description`、`globs`、`alwaysApply` |
| **Cursor** | `.cursor/rules/` | `.mdc` | YAML，包含 `description`、`globs`、`alwaysApply` |

### Kiro 适配器

生成带有 Kiro 特定 frontmatter 的规则：

```markdown
---
inclusion: fileMatch
fileMatchPattern: ['**/*.go', '**/devgen.toml']
---

# 规则内容
...
```

**Frontmatter 字段**：
- `inclusion`: `always`（始终加载）或 `fileMatch`（文件匹配时加载）
- `fileMatchPattern`: 文件匹配的 glob 模式数组

### CodeBuddy 适配器

生成带有 CodeBuddy 特定 frontmatter 的规则：

```markdown
---
description: 用于上下文加载的简短描述
globs: **/*.go
alwaysApply: false
---

# 规则内容
...
```

**Frontmatter 字段**：
- `description`: AI 上下文加载的简短摘要
- `globs`: 文件模式（单个字符串）
- `alwaysApply`: 是否始终包含在上下文中（布尔值）

### Cursor 适配器

使用与 CodeBuddy 相同的格式：

```markdown
---
description: 用于上下文加载的简短描述
globs: **/*.go
alwaysApply: false
---

# 规则内容
...
```

## 创建自定义适配器

你可以通过实现 `AgentAdapter` 接口为专有或新的 AI 助手创建自定义适配器。

### AgentAdapter 接口

```go
package genkit

// AgentAdapter 为特定 AI 助手转换规则
type AgentAdapter interface {
    // Name 返回助手名称（例如 "kiro"、"codebuddy"）
    Name() string
    
    // OutputDir 返回规则应写入的目录
    OutputDir() string
    
    // Transform 将 genkit.Rule 转换为助手特定格式
    Transform(rule Rule) (filename string, content string, err error)
}
```

### 示例：自定义适配器实现

这是为假设的 AI 助手"MyAI"创建自定义适配器的完整示例：

```go
package main

import (
    "fmt"
    "strings"
    
    "github.com/tlipoca9/devgen/genkit"
)

// MyAIAdapter 为 MyAI 助手实现 AgentAdapter
type MyAIAdapter struct{}

func (a *MyAIAdapter) Name() string {
    return "myai"
}

func (a *MyAIAdapter) OutputDir() string {
    return ".myai/docs"
}

func (a *MyAIAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // 构建自定义 frontmatter
    var frontmatter strings.Builder
    frontmatter.WriteString("---\n")
    frontmatter.WriteString(fmt.Sprintf("title: %s\n", rule.Name))
    frontmatter.WriteString(fmt.Sprintf("description: %s\n", rule.Description))
    
    // 处理文件模式
    if len(rule.Globs) > 0 {
        frontmatter.WriteString("patterns:\n")
        for _, glob := range rule.Globs {
            frontmatter.WriteString(fmt.Sprintf("  - %s\n", glob))
        }
    }
    
    // 处理始终应用
    if rule.AlwaysApply {
        frontmatter.WriteString("autoLoad: true\n")
    }
    
    frontmatter.WriteString("---\n\n")
    
    // 组合 frontmatter 和内容
    content := frontmatter.String() + rule.Content
    filename := rule.Name + ".md"
    
    return filename, content, nil
}
```

### 注册自定义适配器

要使你的自定义适配器对 devgen 可用，请将其注册到适配器注册表：

```go
package main

import (
    "github.com/tlipoca9/devgen/genkit"
)

func init() {
    // 获取全局适配器注册表
    registry := genkit.GetAdapterRegistry()
    
    // 注册你的自定义适配器
    registry.Register(&MyAIAdapter{})
}
```

### 在插件中使用自定义适配器

如果你正在开发插件，可以在插件加载时注册适配器：

```go
package main

import (
    "github.com/tlipoca9/devgen/genkit"
)

type MyGenerator struct{}

func (m *MyGenerator) Name() string { return "mygen" }

func (m *MyGenerator) Run(gen *genkit.Generator, log *genkit.Logger) error {
    // 注册自定义适配器
    registry := genkit.GetAdapterRegistry()
    registry.Register(&MyAIAdapter{})
    
    // ... 其余生成器逻辑
    return nil
}

var Tool genkit.Tool = &MyGenerator{}

func main() {}
```

## 适配器最佳实践

### 1. 保持内容完整性

始终保留原始规则内容而不进行修改：

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    frontmatter := buildFrontmatter(rule)
    
    // ✅ 好：保留原始内容
    content := frontmatter + rule.Content
    
    // ❌ 坏：修改内容
    // content := frontmatter + strings.ToUpper(rule.Content)
    
    return rule.Name + ".md", content, nil
}
```

### 2. 优雅地处理空字段

并非所有规则都会填充所有字段：

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    var frontmatter strings.Builder
    frontmatter.WriteString("---\n")
    
    // ✅ 好：使用前检查
    if rule.Description != "" {
        frontmatter.WriteString(fmt.Sprintf("description: %s\n", rule.Description))
    }
    
    if len(rule.Globs) > 0 {
        // 处理 globs
    }
    
    frontmatter.WriteString("---\n\n")
    return rule.Name + ".md", frontmatter.String() + rule.Content, nil
}
```

### 3. 使用一致的文件名模式

生成可预测的文件名：

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // ✅ 好：简单、可预测
    filename := rule.Name + ".md"
    
    // ❌ 坏：复杂、不可预测
    // filename := fmt.Sprintf("%s_%d.markdown", rule.Name, time.Now().Unix())
    
    content := buildContent(rule)
    return filename, content, nil
}
```

### 4. 验证 YAML Frontmatter

确保你的 frontmatter 是有效的 YAML：

```go
import "gopkg.in/yaml.v3"

func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // 将 frontmatter 构建为结构体
    fm := map[string]interface{}{
        "title":       rule.Name,
        "description": rule.Description,
        "patterns":    rule.Globs,
    }
    
    // 编组为 YAML
    yamlBytes, err := yaml.Marshal(fm)
    if err != nil {
        return "", "", fmt.Errorf("marshal frontmatter: %w", err)
    }
    
    // 构建最终内容
    content := "---\n" + string(yamlBytes) + "---\n\n" + rule.Content
    return rule.Name + ".md", content, nil
}
```

### 5. 记录你的适配器

为适配器的用户提供清晰的文档：

```go
// MyAIAdapter 为 MyAI 助手转换规则。
//
// Frontmatter 格式：
//   title: 规则名称
//   description: 简短描述
//   patterns: 文件模式列表
//   autoLoad: 是否始终加载（可选）
//
// 输出目录：.myai/docs/
type MyAIAdapter struct{}
```

## 测试自定义适配器

### 单元测试示例

```go
package main

import (
    "strings"
    "testing"
    
    "github.com/tlipoca9/devgen/genkit"
)

func TestMyAIAdapter_Transform(t *testing.T) {
    adapter := &MyAIAdapter{}
    
    rule := genkit.Rule{
        Name:        "test-rule",
        Description: "测试规则描述",
        Globs:       []string{"**/*.go", "**/*.md"},
        AlwaysApply: true,
        Content:     "# 测试规则\n\n这是测试内容。",
    }
    
    filename, content, err := adapter.Transform(rule)
    
    // 检查无错误
    if err != nil {
        t.Fatalf("Transform 失败: %v", err)
    }
    
    // 检查文件名
    if filename != "test-rule.md" {
        t.Errorf("期望文件名 'test-rule.md'，得到 '%s'", filename)
    }
    
    // 检查 frontmatter
    if !strings.Contains(content, "title: test-rule") {
        t.Error("Frontmatter 缺少 title")
    }
    
    if !strings.Contains(content, "description: 测试规则描述") {
        t.Error("Frontmatter 缺少 description")
    }
    
    if !strings.Contains(content, "autoLoad: true") {
        t.Error("Frontmatter 缺少 autoLoad")
    }
    
    // 检查内容保留
    if !strings.Contains(content, "# 测试规则") {
        t.Error("原始内容未保留")
    }
}

func TestMyAIAdapter_Name(t *testing.T) {
    adapter := &MyAIAdapter{}
    if adapter.Name() != "myai" {
        t.Errorf("期望名称 'myai'，得到 '%s'", adapter.Name())
    }
}

func TestMyAIAdapter_OutputDir(t *testing.T) {
    adapter := &MyAIAdapter{}
    if adapter.OutputDir() != ".myai/docs" {
        t.Errorf("期望输出目录 '.myai/docs'，得到 '%s'", adapter.OutputDir())
    }
}
```

### 集成测试示例

```go
func TestMyAIAdapter_Integration(t *testing.T) {
    // 创建临时目录
    tmpDir := t.TempDir()
    
    // 创建适配器
    adapter := &MyAIAdapter{}
    
    // 创建测试规则
    rule := genkit.Rule{
        Name:    "integration-test",
        Content: "# 集成测试\n\n测试内容。",
    }
    
    // 转换
    filename, content, err := adapter.Transform(rule)
    if err != nil {
        t.Fatalf("Transform 失败: %v", err)
    }
    
    // 写入文件
    filepath := filepath.Join(tmpDir, adapter.OutputDir(), filename)
    if err := os.MkdirAll(filepath.Dir(filepath), 0755); err != nil {
        t.Fatalf("创建目录失败: %v", err)
    }
    
    if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
        t.Fatalf("写入文件失败: %v", err)
    }
    
    // 验证文件存在且可读
    readContent, err := os.ReadFile(filepath)
    if err != nil {
        t.Fatalf("读取文件失败: %v", err)
    }
    
    if string(readContent) != content {
        t.Error("文件内容不匹配")
    }
}
```

## 高级主题

### 动态 Frontmatter 字段

某些适配器可能需要根据规则内容添加字段：

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    fm := map[string]interface{}{
        "title": rule.Name,
    }
    
    // 根据规则名称添加类别
    if strings.HasPrefix(rule.Name, "enum") {
        fm["category"] = "code-generation"
    } else if strings.HasPrefix(rule.Name, "validate") {
        fm["category"] = "validation"
    }
    
    // 根据内容添加标签
    if strings.Contains(rule.Content, "JSON") {
        fm["tags"] = []string{"json", "serialization"}
    }
    
    yamlBytes, _ := yaml.Marshal(fm)
    content := "---\n" + string(yamlBytes) + "---\n\n" + rule.Content
    
    return rule.Name + ".md", content, nil
}
```

### 内容转换

虽然通常不鼓励，但某些适配器可能需要转换内容：

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // 构建 frontmatter
    frontmatter := buildFrontmatter(rule)
    
    // 转换内容（谨慎使用！）
    transformedContent := rule.Content
    
    // 示例：添加助手特定注释
    if strings.Contains(rule.Content, "## 快速开始") {
        note := "\n> **MyAI 注意**: 此功能需要 MyAI v2.0+\n\n"
        transformedContent = strings.Replace(
            transformedContent,
            "## 快速开始",
            "## 快速开始"+note,
            1,
        )
    }
    
    content := frontmatter + transformedContent
    return rule.Name + ".md", content, nil
}
```

### 多文件输出

某些适配器可能为每个规则生成多个文件：

```go
type MultiFileAdapter struct{}

func (a *MultiFileAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // 此接口仅支持单文件输出
    // 对于多文件，你需要扩展接口
    // 或在后处理步骤中处理
    
    // 现在，返回主文件
    return rule.Name + ".md", buildContent(rule), nil
}
```

## 故障排除

### 找不到适配器

**问题**：`devgen rules --agent myai -w` 返回 "unknown agent: myai"

**解决方案**：确保在规则命令运行前注册了适配器：
1. 检查 `Register()` 是否在 `init()` 函数中调用
2. 验证适配器已导入（如需要使用空白导入）
3. 检查适配器名称是否完全匹配

### 无效的 YAML Frontmatter

**问题**：生成的规则在 frontmatter 中有语法错误

**解决方案**：
1. 使用 YAML 库生成 frontmatter
2. 使用 YAML 验证器测试 frontmatter
3. 转义字符串值中的特殊字符
4. 对嵌套结构使用正确的缩进

### 内容未保留

**问题**：原始规则内容丢失或损坏

**解决方案**：
1. 始终追加 rule.Content 而不进行修改
2. 不要对内容使用字符串替换
3. 保留换行符和格式
4. 使用包含特殊字符的规则进行测试

## 参考实现

研究内置适配器作为参考：

- [Kiro 适配器](../genkit/adapter_kiro.go) - 带数组的复杂 frontmatter
- [CodeBuddy 适配器](../genkit/adapter_codebuddy.go) - 简单 frontmatter
- [Cursor 适配器](../genkit/adapter_cursor.go) - 与 CodeBuddy 相同

## 下一步

- 了解 [RuleTool 接口](plugin.md#ai-rules-集成可选) 以提供规则
- 学习 [AI Rules 系统](../cmd/devgen/rules/devgen-rules.md) 的规则内容指南
- 探索 [内置适配器](../genkit/) 的实现示例
