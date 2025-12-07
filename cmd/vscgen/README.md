# vscgen

VSCode 扩展配置生成器，从 `devgen.toml` 文件生成 VSCode 扩展所需的配置。

## 概述

`vscgen` 是 devgen 工具链的核心组件，它读取各代码生成器（如 enumgen、validategen）的 `devgen.toml` 配置文件，并生成统一的 `tools-config.json` 供 VSCode 扩展使用。

这种设计实现了**配置即文档**的理念：
- 生成器开发者只需维护一份 `devgen.toml` 配置
- VSCode 扩展自动获得注解补全、参数验证、文档提示等功能
- 新增生成器时无需修改扩展代码

## 安装

```bash
go install github.com/tlipoca9/devgen/cmd/vscgen@latest
```

## 使用

```bash
# 默认：扫描 cmd 目录，输出到 vscode-devgen/src
vscgen

# 自定义输入输出目录
vscgen -input ./generators -output ./extension/src
```

### 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-input` | `cmd` | 包含生成器子目录的目录（每个子目录应有 `devgen.toml`） |
| `-output` | `vscode-devgen/src` | 输出 `tools-config.json` 的目录 |

## devgen.toml 配置详解

`devgen.toml` 是 devgen 生态的核心配置文件，定义了代码生成器支持的注解、参数类型和文档。

### 完整配置结构

```toml
# DevGen tool configuration
# 此文件用于生成 VSCode 扩展配置

[tool]
name = "toolname"           # 工具名称，用于注解前缀 (toolname:@annotation)
output_suffix = "_gen.go"   # 生成文件的后缀

[[annotations]]
name = "annotationName"     # 注解名称
type = "type"               # 注解类型: "type"(类型级) 或 "field"(字段级)
doc = "注解说明文档"         # 注解文档，显示在补全和悬停提示中

[annotations.params]        # 参数配置（可选）
type = "string"             # 参数类型
placeholder = "value"       # 代码片段占位符
```

### 参数类型详解

#### 1. 无参数注解

适用于不需要参数的注解：

```toml
[[annotations]]
name = "required"
type = "field"
doc = "Field must not be empty/zero"
# 无 [annotations.params] 部分
```

使用：`// validategen:@required`

#### 2. 单类型参数

```toml
[[annotations]]
name = "min"
type = "field"
doc = "Minimum value or length"

[annotations.params]
type = "string"         # 可选: "string", "number", "bool", "list"
placeholder = "value"   # 代码片段中显示的占位符
```

支持的参数类型：
- `"string"` - 字符串值
- `"number"` - 数字值（整数或浮点数）
- `"bool"` - 布尔值（true/false）
- `"list"` - 逗号分隔的值列表

使用：`// validategen:@min(10)`

#### 3. 多类型参数

支持多种类型的参数：

```toml
[[annotations]]
name = "eq"
type = "field"
doc = "Must equal specified value (supports string, number, bool)"

[annotations.params]
type = ["string", "number", "bool"]  # 数组形式指定多种类型
placeholder = "value"
```

使用：
- `// validategen:@eq(hello)` - 字符串
- `// validategen:@eq(42)` - 数字
- `// validategen:@eq(true)` - 布尔值

#### 4. 枚举参数

预定义的选项列表：

```toml
[[annotations]]
name = "enum"
type = "type"
doc = "Generate enum helper methods (options: string, json, text, sql)"

[annotations.params]
values = ["string", "json", "text", "sql"]  # 可选值列表

[annotations.params.docs]                    # 每个选项的文档（可选）
string = "Generate String() method"
json = "Generate MarshalJSON/UnmarshalJSON methods"
text = "Generate MarshalText/UnmarshalText methods"
sql = "Generate Value/Scan methods for database/sql"
```

使用：`// enumgen:@enum(string, json, sql)`

#### 5. 限制参数数量的枚举

```toml
[[annotations]]
name = "format"
type = "field"
doc = "Must be valid format (json, yaml, toml, csv)"

[annotations.params]
values = ["json", "yaml", "toml", "csv"]
maxArgs = 1                                  # 最多允许 1 个参数

[annotations.params.docs]
json = "Validate JSON format"
yaml = "Validate YAML format"
toml = "Validate TOML format"
csv = "Validate CSV format"
```

使用：`// validategen:@format(json)` （只能选一个）

#### 6. LSP 集成参数（高级）

支持与 gopls 等语言服务器集成，实现跨包类型方法查找：

```toml
[[annotations]]
name = "method"
type = "field"
doc = "Call specified method for validation (for struct fields)"

[annotations.params]
type = "string"
placeholder = "MethodName"

[annotations.lsp]
enabled = true                    # 启用 LSP 集成
provider = "gopls"                # LSP 提供者
feature = "method"                # 功能类型: "method", "type", "symbol"
signature = "func() error"        # 要求的方法签名
resolveFrom = "fieldType"         # 从哪里解析类型: "fieldType", "receiverType"
```

**LSP 配置字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `enabled` | bool | 是否启用 LSP 集成 |
| `provider` | string | LSP 提供者，目前支持 `"gopls"` |
| `feature` | string | 功能类型：`"method"` 方法查找、`"type"` 类型查找、`"symbol"` 符号查找 |
| `signature` | string | 要求的方法签名模式，如 `"func() error"` |
| `resolveFrom` | string | 类型解析来源：`"fieldType"` 从字段类型、`"receiverType"` 从接收者类型 |

**LSP 集成功能：**

1. **方法补全** - 输入 `@method(` 时自动补全符合签名的方法
2. **方法验证** - 检测方法是否存在、签名是否匹配
3. **跨包查找** - 通过 gopls workspace symbol 查找其他包的类型方法
4. **悬停提示** - 显示方法是否存在及其实际签名

使用示例：

```go
// validategen:@validate
type User struct {
    // validategen:@method(Validate)  // 自动补全 Address 的方法
    Address Address                   // 验证 Address.Validate() error 是否存在
    
    // validategen:@method(Validate)
    Status Status                     // 验证 Status.Validate() error 是否存在
}
```

### 完整示例：enumgen

```toml
# DevGen tool configuration for enumgen
# This file is used to generate VSCode extension configuration

[tool]
name = "enumgen"
output_suffix = "_enum.go"

[[annotations]]
name = "enum"
type = "type"
doc = "Generate enum helper methods (options: string, json, text, sql)"

[annotations.params]
values = ["string", "json", "text", "sql"]

[annotations.params.docs]
string = "Generate String() method"
json = "Generate MarshalJSON/UnmarshalJSON methods"
text = "Generate MarshalText/UnmarshalText methods"
sql = "Generate Value/Scan methods for database/sql"

[[annotations]]
name = "name"
type = "field"
doc = "Custom name for enum value"

[annotations.params]
type = "string"
placeholder = "name"
```

### 完整示例：validategen

```toml
# DevGen tool configuration for validategen
# This file is used to generate VSCode extension configuration

[tool]
name = "validategen"
output_suffix = "_validate.go"

[[annotations]]
name = "validate"
type = "type"
doc = "Generate Validate() method for struct"

# 无参数注解
[[annotations]]
name = "required"
type = "field"
doc = "Field must not be empty/zero"

# 数字参数
[[annotations]]
name = "min"
type = "field"
doc = "Minimum value or length"

[annotations.params]
type = "number"
placeholder = "value"

[[annotations]]
name = "max"
type = "field"
doc = "Maximum value or length"

[annotations.params]
type = "number"
placeholder = "value"

# 多类型参数
[[annotations]]
name = "eq"
type = "field"
doc = "Must equal specified value (supports string, number, bool)"

[annotations.params]
type = ["string", "number", "bool"]
placeholder = "value"

# 列表参数
[[annotations]]
name = "oneof"
type = "field"
doc = "Must be one of the specified values"

[annotations.params]
type = "list"
placeholder = "values"

# 字符串参数
[[annotations]]
name = "regex"
type = "field"
doc = "Must match the specified regular expression"

[annotations.params]
type = "string"
placeholder = "pattern"

# 限制数量的枚举参数
[[annotations]]
name = "format"
type = "field"
doc = "Must be valid format (json, yaml, toml, csv)"

[annotations.params]
values = ["json", "yaml", "toml", "csv"]
maxArgs = 1

[annotations.params.docs]
json = "Validate JSON format"
yaml = "Validate YAML format"
toml = "Validate TOML format"
csv = "Validate CSV format"

# LSP 集成参数（跨包方法查找）
[[annotations]]
name = "method"
type = "field"
doc = "Call specified method for validation (for struct fields)"

[annotations.params]
type = "string"
placeholder = "MethodName"

[annotations.lsp]
enabled = true
provider = "gopls"
feature = "method"
signature = "func() error"
resolveFrom = "fieldType"
```

## 生成的 tools-config.json

`vscgen` 将所有 `devgen.toml` 合并为一个 JSON 文件：

```json
{
  "enumgen": {
    "typeAnnotations": ["enum"],
    "fieldAnnotations": ["name"],
    "outputSuffix": "_enum.go",
    "annotations": {
      "enum": {
        "doc": "Generate enum helper methods (options: string, json, text, sql)",
        "paramType": "enum",
        "values": ["string", "json", "text", "sql"],
        "valueDocs": {
          "string": "Generate String() method",
          "json": "Generate MarshalJSON/UnmarshalJSON methods",
          "text": "Generate MarshalText/UnmarshalText methods",
          "sql": "Generate Value/Scan methods for database/sql"
        }
      },
      "name": {
        "doc": "Custom name for enum value",
        "paramType": "string",
        "placeholder": "name"
      }
    }
  },
  "validategen": {
    "typeAnnotations": ["validate"],
    "fieldAnnotations": ["required", "min", "max", ...],
    "outputSuffix": "_validate.go",
    "annotations": {
      "validate": {
        "doc": "Generate Validate() method for struct"
      },
      "required": {
        "doc": "Field must not be empty/zero"
      },
      "min": {
        "doc": "Minimum value or length",
        "paramType": "number",
        "placeholder": "value"
      }
      // ...
    }
  }
}
```

## 添加新生成器

1. 创建生成器目录：`cmd/mygengen/`

2. 创建 `devgen.toml`：
```toml
[tool]
name = "mygengen"
output_suffix = "_mygen.go"

[[annotations]]
name = "generate"
type = "type"
doc = "Generate custom code"

[[annotations]]
name = "option"
type = "field"
doc = "Custom option"

[annotations.params]
type = "string"
placeholder = "value"
```

3. 重新运行 `vscgen`：
```bash
vscgen
```

4. 重新构建 VSCode 扩展：
```bash
cd vscode-devgen
npm run compile
npm run package
```

VSCode 扩展将自动支持新的注解补全、参数验证和文档提示。

## 架构设计

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ cmd/enumgen/    │     │ cmd/validategen/│     │ cmd/mygengen/   │
│ devgen.toml     │     │ devgen.toml     │     │ devgen.toml     │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                                 ▼
                        ┌────────────────┐
                        │    vscgen      │
                        │  (此工具)      │
                        └────────┬───────┘
                                 │
                                 ▼
                    ┌────────────────────────┐
                    │ vscode-devgen/src/     │
                    │ tools-config.json      │
                    └────────────┬───────────┘
                                 │
                                 ▼
                    ┌────────────────────────┐
                    │ VSCode Extension       │
                    │ - 语法高亮             │
                    │ - 自动补全             │
                    │ - 参数验证             │
                    │ - 诊断提示             │
                    └────────────────────────┘
```

## License

MIT
