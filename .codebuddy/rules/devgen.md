---
description: devgen 代码生成工具集使用指南。包含安装、命令行用法、配置文件、故障排查等。
globs: **/devgen.toml, **/*.go
alwaysApply: false
---

# devgen - Go 代码生成工具集

devgen 是一个 Go 代码生成工具集，通过注解自动生成样板代码。

## 什么时候使用 devgen？

当你需要：
- 为枚举类型生成 String()、JSON、SQL 等方法 → 使用 enumgen
- 为结构体生成 Validate() 验证方法 → 使用 validategen
- 运行多个代码生成器 → 使用 devgen 统一运行
- 开发自定义代码生成工具 → 使用 genkit 框架

## 安装

```bash
# 安装 devgen（包含所有内置工具）
go install github.com/tlipoca9/devgen/cmd/devgen@latest

# 或单独安装某个工具
go install github.com/tlipoca9/devgen/cmd/enumgen@latest
go install github.com/tlipoca9/devgen/cmd/validategen@latest
```

## 命令行用法

### 基本用法

```bash
# 运行所有生成器，处理当前目录及子目录
devgen ./...

# 处理特定包
devgen ./pkg/models

# 处理特定目录下所有包
devgen ./internal/...
```

### Dry-run 模式

在不写入文件的情况下验证注解：

```bash
# 验证注解，显示将要生成的文件
devgen --dry-run ./...

# JSON 格式输出（用于 IDE 集成）
devgen --dry-run --json ./...
```

JSON 输出示例：
```json
{
  "success": true,
  "files": {
    "/path/to/models_enum.go": "// Code generated...",
    "/path/to/models_validate.go": "// Code generated..."
  },
  "stats": {
    "packagesLoaded": 5,
    "filesGenerated": 2,
    "errorCount": 0,
    "warningCount": 0
  }
}
```

### 查看工具配置

```bash
# TOML 格式（人类可读）
devgen config

# JSON 格式（IDE/工具集成）
devgen config --json
```

### 生成 AI Rules

```bash
# 列出支持的 AI agents
devgen rules --list-agents

# 预览 rules（输出到 stdout）
devgen rules --agent codebuddy

# 写入 rules 文件
devgen rules --agent codebuddy -w
```

## 配置文件

devgen 使用 `devgen.toml` 配置文件来加载插件。配置文件会从当前目录向上查找。

### 基本配置

```toml
# 插件配置
[[plugins]]
name = "myplugin"        # 插件名称
path = "./plugins/mygen" # 插件路径（相对或绝对）
type = "source"          # source（源码）或 plugin（.so 文件）
```

### 多插件配置

```toml
[[plugins]]
name = "jsongen"
path = "./tools/jsongen"
type = "source"

[[plugins]]
name = "mockgen"
path = "./tools/mockgen.so"
type = "plugin"
```

## 内置工具

### enumgen - 枚举代码生成器

为 Go 枚举类型生成辅助方法：

```go
// Status 订单状态
// enumgen:@enum(string, json, sql)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
    StatusCompleted
)
```

生成：String()、MarshalJSON()、UnmarshalJSON()、Value()、Scan()、IsValid() 等方法。

**查看详细用法**：AI 助手可以通过 `enumgen` rule 获取完整文档。

### validategen - 验证代码生成器

为 Go 结构体生成 Validate() 方法：

```go
// User 用户模型
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@email
    Email string

    // validategen:@min(0)
    // validategen:@max(150)
    Age int
}
```

生成 `func (x User) Validate() error` 方法。

**查看详细用法**：AI 助手可以通过 `validategen` rule 获取完整文档。

## 注解语法

devgen 工具使用注解来标记需要处理的类型和字段。

### 基本格式

```
tool:@annotation
tool:@annotation(arg1, arg2)
tool:@annotation(key=value)
```

### 类型注解

放在类型定义的注释中：

```go
// MyType 类型说明
// enumgen:@enum(string, json)
type MyType int
```

### 字段注解

放在字段的注释中：

```go
type User struct {
    // validategen:@required
    // validategen:@email
    Email string
}
```

## VSCode 扩展

安装 [vscode-devgen](https://marketplace.visualstudio.com/items?itemName=tlipoca9.devgen) 扩展获得：

- 注解语法高亮
- 自动补全
- 参数验证提示
- 错误诊断

扩展会自动从 `devgen config --json` 获取所有工具（包括插件）的注解配置。

## 故障排查

### 1. "undefined: XxxEnums" 错误

**原因**：忘记运行 devgen 生成代码。

**解决**：
```bash
devgen ./...
```

### 2. "package errors" 加载失败

**原因**：代码有语法错误，或依赖未安装。

**解决**：
```bash
# 先确保代码能编译
go build ./...

# 再运行 devgen
devgen ./...
```

### 3. 生成的代码有冲突

**原因**：手动修改了生成的文件。

**解决**：生成的文件以 `// Code generated` 开头，不要手动修改。删除后重新生成：
```bash
rm *_enum.go *_validate.go
devgen ./...
```

### 4. 插件加载失败

**原因**：插件路径错误或代码有问题。

**解决**：
```bash
# 检查插件路径
cat devgen.toml

# 单独编译插件确认无误
go build ./plugins/myplugin
```

### 5. 注解不生效

**原因**：注解格式错误。

**检查**：
- 格式必须是 `tool:@name` 或 `tool:@name(args)`
- 冒号和 @ 符号不能少
- 注解必须在正确的位置（类型注解在类型上，字段注解在字段上）

```go
// ❌ 错误
// enumgen@enum(string)     // 缺少冒号
// enumgen:enum(string)     // 缺少 @
// @enum(string)            // 缺少工具名

// ✅ 正确
// enumgen:@enum(string)
```

## 工作流程示例

### 典型开发流程

1. **定义类型并添加注解**：
```go
// models/order.go

// OrderStatus 订单状态
// enumgen:@enum(string, json)
type OrderStatus int

const (
    OrderStatusPending OrderStatus = iota + 1
    OrderStatusPaid
    OrderStatusShipped
)

// Order 订单
// validategen:@validate
type Order struct {
    // validategen:@required
    // validategen:@gt(0)
    ID int64

    // validategen:@required
    Status OrderStatus
}
```

2. **运行代码生成**：
```bash
devgen ./...
```

3. **使用生成的代码**：
```go
order := Order{ID: 1, Status: OrderStatusPending}
if err := order.Validate(); err != nil {
    log.Fatal(err)
}
fmt.Println(order.Status.String()) // "Pending"
```

### 添加到 go generate

在包的任意 .go 文件中添加：

```go
//go:generate devgen ./...
```

然后运行：
```bash
go generate ./...
```
