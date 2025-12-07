# devgen

中文 | [English](README_EN.md)

Go 代码生成工具集，通过注解自动生成样板代码，减少手写重复代码的工作量。

## 安装

```bash
# 安装 devgen（包含所有工具）
go install github.com/tlipoca9/devgen/cmd/devgen@latest

# 或单独安装
go install github.com/tlipoca9/devgen/cmd/enumgen@latest
go install github.com/tlipoca9/devgen/cmd/validategen@latest
```

## 使用

```bash
devgen ./...        # 运行所有生成器
enumgen ./...       # 仅运行枚举生成器
validategen ./...   # 仅运行验证生成器
```

## 工具

### enumgen - 枚举代码生成器

为 Go 枚举类型自动生成序列化、反序列化和验证方法。

```go
// Status 表示状态
// enumgen:@enum(string, json, sql)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
    // enumgen:@enum.name(Cancelled)
    StatusCanceled  // 自定义名称
)
```

**支持的选项**：
- `string` - 生成 `String()` 方法
- `json` - 生成 `MarshalJSON()` / `UnmarshalJSON()`
- `text` - 生成 `MarshalText()` / `UnmarshalText()`
- `sql` - 生成 `Value()` / `Scan()` 用于数据库操作

**生成的辅助方法**：
- `IsValid()` - 验证枚举值是否有效
- `{Type}Enums.List()` - 返回所有有效枚举值
- `{Type}Enums.Parse(s)` - 从字符串解析枚举
- `{Type}Enums.Name(v)` - 获取枚举值的字符串名称

详见 [enumgen README](cmd/enumgen/README.md)

---

### validategen - 验证代码生成器

为 Go 结构体自动生成 `Validate()` 方法。

```go
// User 用户模型
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

    // validategen:@oneof(admin, user, guest)
    Role string
}
```

**验证注解**：

| 类别 | 注解 |
|------|------|
| 必填 | `@required` |
| 范围 | `@min(n)` `@max(n)` `@len(n)` `@gt(n)` `@gte(n)` `@lt(n)` `@lte(n)` |
| 等值 | `@eq(v)` `@ne(v)` `@oneof(a, b, c)` |
| 格式 | `@email` `@url` `@uuid` `@ip` `@ipv4` `@ipv6` |
| 字符 | `@alpha` `@alphanum` `@numeric` |
| 字符串 | `@contains(s)` `@excludes(s)` `@startswith(s)` `@endswith(s)` |
| 正则 | `@regex(pattern)` |
| 数据格式 | `@format(json\|yaml\|toml\|csv)` |
| 嵌套 | `@method(MethodName)` |

**高级特性**：
- `postValidate(errs []string) error` 钩子实现自定义验证逻辑

详见 [validategen README](cmd/validategen/README.md)

---

### vscode-devgen - VSCode 扩展

提供 devgen 注解的编辑器支持：语法高亮、自动补全、参数验证提示。

## 构建

```bash
make build    # 构建所有工具
make test     # 运行测试
make vscode   # 构建 VSCode 扩展
```

## Release Notes

- [v0.1.0](docs/release/v0.1.0.md) - 2025-12-07

## License

MIT
