# vscode-devgen

[![VS Marketplace](https://img.shields.io/visual-studio-marketplace/v/tlipoca9.devgen)](https://marketplace.visualstudio.com/items?itemName=tlipoca9.devgen)

VSCode 扩展，为 devgen 注解提供编辑器支持：语法高亮、自动补全、参数验证和诊断提示。

## 功能特性

- **语法高亮** - 注解关键字、参数、工具名称等语法着色
- **智能补全** - 工具名称、注解名称、枚举参数的自动补全
- **参数验证** - 实时检测无效参数、缺失参数、未知注解
- **诊断提示** - 检测缺失的生成文件、字段注解缺少类型注解等问题
- **悬停文档** - 鼠标悬停显示注解文档和可用选项
- **LSP 集成** - 与 gopls 联动，支持跨包类型方法查找和验证
- **Dry-run 验证** - 保存时自动运行 `devgen --dry-run` 进行静态验证

## 安装

从 [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=tlipoca9.devgen) 安装。

或在 VSCode 中搜索 `devgen`。

## 配置项

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `devgen.enableDiagnostics` | boolean | `true` | 启用/禁用诊断功能 |
| `devgen.executablePath` | string | `devgen` | devgen 可执行文件路径 |
| `devgen.validateOnSave` | boolean | `true` | 保存和打开文件时运行 dry-run 验证 |

## 命令

| 命令 | 说明 |
|------|------|
| `DevGen: Validate Annotations` | 手动触发 dry-run 验证 |

## 支持的工具

扩展通过运行 `devgen config --json` 命令自动获取工具和注解配置。如果 devgen 未安装，扩展会自动尝试通过 `go install` 安装。

当前内置支持：

### enumgen

类型级注解：
- `@enum(options...)` - 生成枚举辅助方法

字段级注解：
- `@name(value)` - 自定义枚举值名称

### validategen

类型级注解：
- `@validate` - 生成 Validate() 方法

字段级注解：
- `@required` - 必填验证
- `@min(n)` / `@max(n)` / `@len(n)` - 长度/值范围
- `@gt(n)` / `@gte(n)` / `@lt(n)` / `@lte(n)` - 数值比较
- `@eq(v)` / `@ne(v)` - 等值/不等值
- `@oneof(a, b, c)` - 枚举值
- `@email` / `@url` / `@uuid` / `@ip` / `@ipv4` / `@ipv6` - 格式验证
- `@alpha` / `@alphanum` / `@numeric` - 字符类型
- `@contains(s)` / `@excludes(s)` / `@startswith(s)` / `@endswith(s)` - 字符串匹配
- `@regex(pattern)` - 正则匹配
- `@format(json|yaml|toml|csv)` - 数据格式验证
- `@method(MethodName)` - 调用自定义方法（支持 LSP 补全和验证）

## LSP 集成

扩展与 gopls 深度集成，为 `@method` 注解提供智能支持：

### 方法补全

输入 `// validategen:@method(` 时，自动补全字段类型上符合 `func() error` 签名的方法：

```go
type Address struct {
    Street string
    City   string
}

func (a Address) Validate() error { ... }

// validategen:@validate
type User struct {
    // validategen:@method(|)  // <- 光标位置，自动补全 "Validate"
    Address Address
}
```

### 方法验证

实时检测：
- ❌ 方法不存在
- ⚠️ 方法签名不匹配（要求 `func() error`）
- ✅ 方法存在且签名正确

### 跨包查找

通过 gopls workspace symbol 查找其他包定义的类型方法：

```go
import "github.com/example/models"

// validategen:@validate
type Request struct {
    // validategen:@method(Validate)  // 查找 models.User 的 Validate 方法
    User models.User
}
```

### 悬停提示

鼠标悬停在 `@method(Validate)` 上显示：
- 方法是否存在
- 实际方法签名
- 签名是否匹配要求

## 使用示例

```go
// enumgen:@enum(string, json, sql)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
    // enumgen:@name(Cancelled)
    StatusCanceled
)

// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@min(2)
    // validategen:@max(50)
    Name string

    // validategen:@required
    // validategen:@email
    Email string

    // validategen:@gte(0)
    // validategen:@lte(150)
    Age int
}
```

## 工作原理

扩展通过以下方式获取工具配置：

1. **devgen CLI** - 运行 `devgen config --json` 获取所有工具（内置 + 插件）的注解配置
2. **devgen.toml** - 读取项目配置文件中的 `[tools.xxx]` 手动覆盖配置
3. **自动安装** - 如果 devgen 未安装，扩展会自动通过 `go install` 安装

插件可以实现 `ConfigurableTool` 接口来自描述配置，无需手动编写 `[tools.xxx]` 配置。

### Dry-run 验证

扩展在保存和打开 Go 文件时自动运行 `devgen --dry-run --json` 进行静态验证：

- 检测注解参数错误（如 `@min` 参数不是数字）
- 检测字段类型与注解不兼容（如 `@method` 用于内置类型）
- 检测 `@oneof` 缺少值等问题

验证结果显示在 Problems 面板中，可通过 `devgen.validateOnSave` 配置项禁用。

详见 [插件开发文档](../docs/plugin.md)。

## 开发

```bash
# 安装依赖
npm install

# 编译
npm run compile

# 打包
npm run package

# 本地安装
npm run install-local
```

### 注意事项

**避免使用外部运行时依赖**

VSCode 扩展打包使用 `--no-dependencies` 参数，不会包含 `node_modules`。因此：

1. 尽量避免使用外部运行时依赖（如 `smol-toml`）
2. 如果必须使用，需要用 webpack/esbuild 将依赖打包到 bundle 中
3. 当前扩展已移除所有外部依赖，配置完全通过 `devgen config --json` 获取

## License

MIT
