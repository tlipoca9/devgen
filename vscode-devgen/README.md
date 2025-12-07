# vscode-devgen

[![VS Marketplace](https://img.shields.io/visual-studio-marketplace/v/tlipoca9.devgen)](https://marketplace.visualstudio.com/items?itemName=tlipoca9.devgen)

VSCode 扩展，为 devgen 注解提供编辑器支持：语法高亮、自动补全、参数验证和诊断提示。

## 功能特性

- **语法高亮** - 注解关键字、参数、工具名称等语法着色
- **智能补全** - 工具名称、注解名称、枚举参数的自动补全
- **参数验证** - 实时检测无效参数、缺失参数、未知注解
- **诊断提示** - 检测缺失的生成文件、字段注解缺少类型注解等问题
- **悬停文档** - 鼠标悬停显示注解文档和可用选项

## 安装

从 [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=tlipoca9.devgen) 安装。

或在 VSCode 中搜索 `devgen`。

## 配置项

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `devgen.enableDiagnostics` | boolean | `true` | 启用/禁用诊断功能 |

## 支持的工具

扩展通过读取 `devgen.toml` 配置文件自动识别支持的工具和注解。当前支持：

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
- `@method(MethodName)` - 调用自定义方法

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

扩展的配置来源于 `tools-config.json`，该文件由 `vscgen` 工具从各生成器的 `devgen.toml` 文件自动生成。

详见 [vscgen README](../cmd/vscgen/README.md) 了解配置生成机制。

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

## License

MIT
