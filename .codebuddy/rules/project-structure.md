---
description: devgen 项目结构规范 - 目录组织、文件命名、模块划分
globs: "**/*"
alwaysApply: true
---

# devgen 项目结构规范

本规则定义了 devgen 项目的目录结构和组织约定。

## 顶层目录结构

```
devgen/
├── cmd/                    # 命令行工具
│   ├── devgen/            # 主 CLI 工具
│   ├── enumgen/           # 枚举生成器
│   ├── validategen/       # 验证生成器
│   └── golangcilint/      # lint 集成
│
├── genkit/                 # 核心框架库
│   ├── genkit.go          # Generator 核心
│   ├── types.go           # 类型定义
│   ├── tool.go            # Tool 接口
│   ├── config.go          # 配置系统
│   ├── plugin.go          # 插件加载
│   ├── adapter_*.go       # AI 规则适配器
│   └── *_test.go          # 测试文件
│
├── docs/                   # 文档
│   ├── *.md               # 中文文档
│   ├── *_EN.md            # 英文文档
│   └── release/           # 版本说明
│
├── examples/               # 示例代码
│   └── plugin/            # 插件示例
│
├── vscode-devgen/         # VSCode 扩展
│
├── .codebuddy/            # AI 助手配置
│   ├── agents/            # Agent 定义
│   ├── commands/          # 命令定义
│   ├── skills/            # 技能定义
│   └── rules/             # AI 规则
│
├── devgen.toml            # 项目配置
├── go.mod / go.sum        # Go 模块
├── Makefile               # 构建脚本
├── .golangci.yaml         # Lint 配置
└── README.md              # 项目说明
```

## cmd/ 目录规范

每个命令行工具遵循统一结构：

```
cmd/<tool>/
├── main.go              # 入口点（最小化逻辑）
├── generator/           # 核心生成器
│   ├── generator.go     # 主实现
│   ├── generator_*.go   # 功能分离文件
│   ├── constants.go     # 常量定义
│   └── *_test.go        # 测试文件
├── rules/               # AI 规则（嵌入式）
│   ├── embed.go         # //go:embed 声明
│   └── *.md             # 规则文件
└── examples/            # 示例（可选）
```

### main.go 模板

```go
package main

import (
    "os"

    "github.com/tlipoca9/devgen/cmd/<tool>/generator"
    "github.com/tlipoca9/devgen/genkit"
)

func main() {
    gen := genkit.NewGenerator()
    if err := gen.Run(&generator.Generator{}); err != nil {
        os.Exit(1)
    }
}
```

### generator/ 组织

```go
// generator.go - 主入口
package generator

const ToolName = "mytool"

type Generator struct {
    // 配置字段
}

func (g *Generator) Name() string { return ToolName }

func (g *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error {
    // 主逻辑
}

// generator_enum.go - 枚举处理
func (g *Generator) processEnum(e *genkit.Enum) error { ... }

// generator_struct.go - 结构体处理
func (g *Generator) processStruct(s *genkit.Struct) error { ... }

// generator_validate.go - 验证逻辑
func (g *Generator) Validate(gen *genkit.Generator, log *genkit.Logger) []genkit.Diagnostic { ... }
```

## genkit/ 目录规范

核心框架的文件组织：

| 文件 | 职责 |
|------|------|
| `genkit.go` | Generator 核心实现 |
| `types.go` | 类型定义（Diagnostic, Annotation 等） |
| `tool.go` | Tool 接口定义 |
| `config.go` | 配置加载（devgen.toml） |
| `plugin.go` | 插件加载系统 |
| `rule_loader.go` | 规则加载器 |
| `adapter_*.go` | AI 规则适配器 |
| `log.go` | 日志工具 |
| `*_test.go` | 对应的测试文件 |

### 新增文件规则

1. **单一职责**：每个文件只负责一个功能领域
2. **命名清晰**：文件名反映内容（`adapter_kiro.go` 而非 `kiro.go`）
3. **测试配套**：每个 `.go` 文件应有对应的 `_test.go`

## docs/ 目录规范

```
docs/
├── plugin.md           # 插件开发（中文）
├── plugin_EN.md        # 插件开发（英文）
├── rules-adapter.md    # 规则适配器（中文）
├── rules-adapter_EN.md # 规则适配器（英文）
└── release/            # 版本说明
    ├── v0.1.0.md
    ├── v0.1.0_EN.md
    └── ...
```

### 文档约定

1. **双语文档**：主要文档提供中英双语版本
2. **版本说明**：按版本号命名，包含变更日志
3. **格式统一**：使用 Markdown，代码块使用 fenced blocks

## .codebuddy/ 目录规范

AI 助手配置结构：

```
.codebuddy/
├── agents/              # Agent 定义
│   └── <name>.md       # 每个 Agent 一个文件
├── commands/            # 命令定义
│   └── <name>.md       # 每个命令一个文件
├── skills/              # 技能定义
│   └── <skill-name>/   # 每个技能一个目录
│       ├── SKILL.md    # 技能主文件
│       ├── references/ # 参考文档
│       ├── examples/   # 示例
│       └── scripts/    # 辅助脚本
└── rules/               # AI 规则
    └── <name>.md       # 规则文件
```

## 文件命名约定

### Go 文件

| 类型 | 格式 | 示例 |
|------|------|------|
| 普通文件 | 小写 + 下划线 | `rule_loader.go` |
| 适配器 | `adapter_<name>.go` | `adapter_kiro.go` |
| 生成文件 | `*_enum.go`, `*_validate.go` | `status_enum.go` |
| 测试文件 | `*_test.go` | `adapter_test.go` |

### Markdown 文件

| 类型 | 格式 | 示例 |
|------|------|------|
| 中文文档 | `<name>.md` | `plugin.md` |
| 英文文档 | `<name>_EN.md` | `plugin_EN.md` |
| 版本说明 | `v<version>.md` | `v0.3.6.md` |
| Agent/Command | `<name>.md` | `code-reviewer.md` |
| 规则 | `<name>.md` | `go-code-style.md` |

## 模块边界

### genkit（核心框架）

- 不依赖具体工具实现
- 提供通用接口和工具
- 保持向后兼容

### cmd/<tool>（具体工具）

- 依赖 genkit
- 实现特定功能
- 可以有自己的子包

### 依赖方向

```
cmd/devgen ──────┐
cmd/enumgen ─────┼──▶ genkit
cmd/validategen ─┘
```

## 新增模块指南

### 添加新的代码生成器

1. 创建目录结构：
   ```bash
   mkdir -p cmd/newtool/generator cmd/newtool/rules
   ```

2. 实现 Tool 接口：
   ```go
   // cmd/newtool/generator/generator.go
   package generator
   
   type Generator struct{}
   
   func (g *Generator) Name() string { return "newtool" }
   func (g *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error { ... }
   ```

3. 创建入口点：
   ```go
   // cmd/newtool/main.go
   package main
   
   func main() {
       gen := genkit.NewGenerator()
       gen.Run(&generator.Generator{})
   }
   ```

4. 添加 AI 规则：
   ```go
   // cmd/newtool/rules/embed.go
   package rules
   
   import "embed"
   
   //go:embed *.md
   var FS embed.FS
   ```

5. 更新 Makefile：
   ```makefile
   TOOLS := devgen enumgen validategen newtool
   ```

### 添加新的 genkit 功能

1. 在适当的文件中添加（或创建新文件）
2. 添加对应的测试
3. 更新文档
4. 保持向后兼容
