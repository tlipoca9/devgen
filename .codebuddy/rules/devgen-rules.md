---
description: devgen AI Rules 系统说明。介绍如何查看、生成和编写 AI rules。
globs: **/*.go
alwaysApply: true
---

# devgen AI Rules 系统

devgen 提供了一套 AI Rules 系统，用于生成 AI 编程助手（如 CodeBuddy、Cursor 等）可以理解的文档。

## 什么是 AI Rules？

AI Rules 是一种结构化的文档格式，帮助 AI 编程助手理解如何使用特定的工具或框架。当你在编写代码时，AI 助手会根据这些 rules 提供更准确的建议。

## 查看可用的 Rules

### 列出支持的 AI Agents

```bash
devgen rules --list-agents
```

输出：
```
Supported AI agents:

  codebuddy     .codebuddy/rules/*.md

Usage: devgen rules --agent <name> [-w]
```

### 预览 Rules 内容

```bash
# 输出到终端（不写入文件）
devgen rules --agent codebuddy
```

这会显示所有工具的 rules 内容，方便你查看和确认。

### 生成 Rules 文件

```bash
# 写入文件
devgen rules --agent codebuddy -w
```

输出：
```
ℹ Generating rules for codebuddy...
✓ Generated 4 rule file(s) in .codebuddy/rules
  • .codebuddy/rules/enumgen.md
  • .codebuddy/rules/validategen.md
  • .codebuddy/rules/devgen.md
  • .codebuddy/rules/devgen-plugin.md
```

## Rules 文件结构

生成的 rules 文件包含 YAML frontmatter 和 Markdown 内容：

```markdown
---
description: Go 枚举代码生成工具 enumgen 的使用指南
globs: **/*.go
alwaysApply: false
---

# enumgen - Go 枚举代码生成工具

enumgen 是 devgen 工具集的一部分...
```

### Frontmatter 字段

| 字段 | 说明 |
|------|------|
| description | 简短描述，AI 用于判断是否加载此规则 |
| globs | 文件匹配模式，匹配时自动加载规则 |
| alwaysApply | 是否始终加载（true = 始终在上下文中） |

## 告诉 AI 查看其他工具的 Rules

当你需要使用某个 devgen 工具时，可以告诉 AI 助手查看相应的 rule：

### 示例对话

**你**：我想为订单状态定义一个枚举类型

**AI**：（查看 enumgen rule 后）好的，你可以这样定义：
```go
// OrderStatus 订单状态
// enumgen:@enum(string, json)
type OrderStatus int

const (
    OrderStatusPending OrderStatus = iota + 1
    OrderStatusPaid
    OrderStatusShipped
)
```

### 可用的 Rules

| Rule 名称 | 说明 |
|-----------|------|
| enumgen | 枚举代码生成工具使用指南 |
| validategen | 结构体验证代码生成工具使用指南 |
| devgen | devgen 综合使用指南 |
| devgen-plugin | 插件开发指南 |
| devgen-genkit | genkit API 参考 |
| devgen-rules | AI Rules 系统说明（本文档） |

## 为自定义插件实现 RuleTool

如果你开发了自定义插件，可以实现 `RuleTool` 接口来提供 AI rules：

### 第一步：实现 RuleTool 接口

```go
package main

import "github.com/tlipoca9/devgen/genkit"

type MyGenerator struct{}

func (m *MyGenerator) Name() string { return "mygen" }

func (m *MyGenerator) Run(gen *genkit.Generator, log *genkit.Logger) error {
    // ... 生成代码
    return nil
}

// Rules 返回 AI rules
func (m *MyGenerator) Rules() []genkit.Rule {
    return []genkit.Rule{
        {
            Name:        "mygen",
            Description: "mygen 代码生成工具使用指南",
            Globs:       []string{"**/*.go"},
            AlwaysApply: false,
            Content:     mygenRuleContent,
        },
    }
}

var Tool genkit.Tool = &MyGenerator{}
```

### 第二步：编写 Rule Content

推荐将 rule content 放在单独的 .md 文件中，使用 go:embed 嵌入：

```go
// rules/embed.go
package rules

import _ "embed"

//go:embed mygen.md
var MygenRule string
```

```markdown
<!-- rules/mygen.md -->
# mygen - 自定义代码生成工具

## 什么时候使用 mygen？

当你需要：
- 功能点 1
- 功能点 2

## 快速开始

### 第一步：添加注解

```go
// MyType 示例类型
// mygen:@gen
type MyType struct {
    Name string
}
```

### 第二步：运行生成

```bash
devgen ./...
```

## 注解参考

| 注解 | 说明 | 示例 |
|------|------|------|
| @gen | 生成代码 | `mygen:@gen` |
| @gen(option) | 带选项生成 | `mygen:@gen(json)` |

## 完整示例

...

## 常见错误

### 1. 错误名称

**原因**：...
**解决**：...
```

### 第三步：在生成器中引用

```go
package main

import (
    "github.com/tlipoca9/devgen/genkit"
    "myapp/plugins/mygen/rules"
)

func (m *MyGenerator) Rules() []genkit.Rule {
    return []genkit.Rule{
        {
            Name:        "mygen",
            Description: "mygen 代码生成工具使用指南",
            Globs:       []string{"**/*.go"},
            Content:     rules.MygenRule,  // 使用嵌入的内容
        },
    }
}
```

## Rule 结构体字段说明

```go
type Rule struct {
    // Name 是规则文件名（不含扩展名）
    // 例如: "enumgen" 会生成 "enumgen.md"
    Name string

    // Description 是简短描述
    // AI 助手用这个来判断是否需要加载此规则
    // 写清楚这个规则是干什么的
    Description string

    // Globs 是文件匹配模式
    // 当用户打开匹配的文件时，AI 可能会自动加载此规则
    // 例如: []string{"**/*.go", "**/*_enum.go"}
    Globs []string

    // AlwaysApply 表示是否始终加载
    // true: 规则始终在 AI 的上下文中
    // false: 根据 Globs 或用户请求加载
    AlwaysApply bool

    // Content 是规则的实际内容（Markdown 格式）
    // 这是 AI 会阅读的文档，要写得详细、清晰
    Content string
}
```

## 编写高质量 Rule Content 的建议

### 1. 假设读者什么都不知道

不要假设 AI 知道你的工具是干什么的。从头解释：

```markdown
## 什么时候使用 XXX？

当你需要：
- 场景 1
- 场景 2
- 场景 3
```

### 2. Step by Step 指导

分步骤说明，每一步都要清晰：

```markdown
## 快速开始

### 第一步：定义类型

```go
// 代码示例
```

### 第二步：添加注解

```go
// 代码示例
```

### 第三步：运行生成

```bash
命令
```

### 第四步：使用生成的代码

```go
// 代码示例
```
```

### 3. 充足的代码示例

每个功能点都要有代码示例：

```markdown
### @required 注解

标记字段为必填：

```go
// validategen:@validate
type User struct {
    // validategen:@required
    Name string
}
```

验证效果：
```go
user := User{Name: ""}
err := user.Validate()
// err: "Name is required"
```
```

### 4. 列出常见错误

帮助 AI 避免常见错误：

```markdown
## 常见错误

### 1. 忘记运行 devgen

```
错误: undefined: StatusEnums
解决: 运行 devgen ./...
```

### 2. 注解格式错误

```go
// ❌ 错误
// enumgen@enum(string)  // 缺少冒号

// ✅ 正确
// enumgen:@enum(string)
```
```

### 5. 提供完整的工作示例

在文档末尾提供一个完整的、可运行的示例：

```markdown
## 完整示例

### 定义文件 (models/user.go)

```go
package models

// User 用户模型
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@email
    Email string
}
```

### 使用示例

```go
package main

import "myapp/models"

func main() {
    user := models.User{Email: "test@example.com"}
    if err := user.Validate(); err != nil {
        log.Fatal(err)
    }
}
```
```
