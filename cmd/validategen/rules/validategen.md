# validategen - Go 结构体验证代码生成工具

validategen 是 devgen 工具集的一部分，用于为 Go 结构体自动生成 Validate() 方法。

## 什么时候使用 validategen？

当你需要：
- 验证 API 请求参数
- 验证配置结构体
- 验证用户输入
- 在数据入库前进行校验

## 快速开始

### 第一步：标记结构体

在结构体定义上方添加 `validategen:@validate` 注解：

```go
// User 用户模型
// validategen:@validate
type User struct {
    Name  string
    Email string
    Age   int
}
```

### 第二步：添加字段验证规则

在字段注释中添加验证注解：

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
}
```

### 第三步：运行代码生成

```bash
devgen ./...
```

生成的文件命名为 `{package}_validate.go`。

### 第四步：使用生成的方法

```go
user := User{
    Name:  "J",      // 太短，min(2)
    Email: "invalid", // 格式错误
    Age:   -1,       // 小于 0
}

if err := user.Validate(); err != nil {
    fmt.Println(err)
    // 输出: Name must have at least 2 characters, got 1; Email must be a valid email address; Age must be >= 0, got -1
}
```

## 验证注解速查表

### 必填验证

| 注解 | 说明 | 示例 |
|------|------|------|
| @required | 字段不能为空/零值 | `// validategen:@required` |

**不同类型的 @required 行为**：
- string: 不能为 ""
- int/float: 不能为 0
- bool: 必须为 true
- slice/map: 长度不能为 0
- pointer: 不能为 nil

### 数值/长度验证

| 注解 | 说明 | 适用类型 |
|------|------|---------|
| @min(n) | 最小值/最小长度 | 数字、字符串、切片、map |
| @max(n) | 最大值/最大长度 | 数字、字符串、切片、map |
| @len(n) | 精确长度 | 字符串、切片、map |
| @gt(n) | 大于 | 数字、字符串长度、切片长度 |
| @gte(n) | 大于等于 | 数字、字符串长度、切片长度 |
| @lt(n) | 小于 | 数字、字符串长度、切片长度 |
| @lte(n) | 小于等于 | 数字、字符串长度、切片长度 |

```go
// validategen:@validate
type Config struct {
    // validategen:@min(1)
    // validategen:@max(65535)
    Port int  // 1 <= Port <= 65535

    // validategen:@min(2)
    // validategen:@max(50)
    Name string  // 2 <= len(Name) <= 50

    // validategen:@len(6)
    Code string  // len(Code) == 6

    // validategen:@gt(0)
    ID int64  // ID > 0

    // validategen:@gte(1)
    // validategen:@lte(10)
    Items []string  // 1 <= len(Items) <= 10
}
```

### 等值验证

| 注解 | 说明 | 示例 |
|------|------|------|
| @eq(value) | 必须等于指定值 | `@eq(1)`, `@eq(active)` |
| @ne(value) | 不能等于指定值 | `@ne(0)`, `@ne(deleted)` |
| @oneof(a,b,c) | 必须是指定值之一 | `@oneof(admin, user, guest)` |

```go
// validategen:@validate
type Request struct {
    // validategen:@eq(2)
    Version int  // Version == 2

    // validategen:@ne(deleted)
    Status string  // Status != "deleted"

    // validategen:@oneof(GET, POST, PUT, DELETE)
    Method string  // Method 必须是 HTTP 方法之一
}
```

### 格式验证

| 注解 | 说明 | 示例 |
|------|------|------|
| @email | 邮箱格式 | user@example.com |
| @url | URL 格式 | https://example.com |
| @uuid | UUID 格式 | 550e8400-e29b-41d4-a716-446655440000 |
| @ip | IP 地址 (v4 或 v6) | 192.168.1.1 |
| @ipv4 | IPv4 地址 | 192.168.1.1 |
| @ipv6 | IPv6 地址 | ::1 |
| @alpha | 纯字母 | abc |
| @alphanum | 字母数字 | abc123 |
| @numeric | 纯数字字符串 | 12345 |

```go
// validategen:@validate
type Contact struct {
    // validategen:@required
    // validategen:@email
    Email string

    // validategen:@url
    Website string  // 可选，但如果填写必须是有效 URL

    // validategen:@uuid
    TraceID string

    // validategen:@ipv4
    ServerIP string

    // validategen:@alphanum
    // validategen:@len(8)
    InviteCode string  // 8位字母数字邀请码
}
```

### 字符串内容验证

| 注解 | 说明 | 示例 |
|------|------|------|
| @contains(s) | 必须包含子串 | `@contains(@)` |
| @excludes(s) | 不能包含子串 | `@excludes(<script>)` |
| @startswith(s) | 必须以指定前缀开头 | `@startswith(https://)` |
| @endswith(s) | 必须以指定后缀结尾 | `@endswith(.go)` |
| @regex(pattern) | 匹配正则表达式 | `@regex(^[A-Z]{2}-\\d{4}$)` |

```go
// validategen:@validate
type Input struct {
    // validategen:@startswith(https://)
    SecureURL string

    // validategen:@excludes(<script>)
    // validategen:@excludes(javascript:)
    UserContent string  // 防止 XSS

    // validategen:@regex(^1[3-9]\d{9}$)
    PhoneNumber string  // 中国手机号格式
}
```

### 时间间隔验证

| 注解 | 说明 | 示例 |
|------|------|------|
| @duration | 有效的 Go duration 格式 | 1h30m, 500ms |
| @duration_min(d) | 最小时间间隔 | `@duration_min(1s)` |
| @duration_max(d) | 最大时间间隔 | `@duration_max(24h)` |

```go
// validategen:@validate
type Config struct {
    // validategen:@duration
    // validategen:@duration_min(100ms)
    // validategen:@duration_max(30s)
    Timeout string  // 有效 duration，100ms <= timeout <= 30s
}
```

### 格式验证（JSON/YAML/TOML/CSV）

| 注解 | 说明 |
|------|------|
| @format(json) | 有效的 JSON 格式 |
| @format(yaml) | 有效的 YAML 格式 |
| @format(toml) | 有效的 TOML 格式 |
| @format(csv) | 有效的 CSV 格式 |

```go
// validategen:@validate
type Template struct {
    // validategen:@format(json)
    JSONTemplate string

    // validategen:@format(yaml)
    YAMLConfig string
}
```

### 枚举类型验证

| 注解 | 说明 |
|------|------|
| @oneof_enum(Type) | 必须是有效的枚举值 |

```go
// 配合 enumgen 使用
// enumgen:@enum(string, json)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
    StatusInactive
)

// validategen:@validate
type User struct {
    // validategen:@oneof_enum(Status)
    Status Status  // 自动使用 StatusEnums.Contains() 验证
}
```

**跨包枚举**：

```go
// validategen:@validate
type Request struct {
    // validategen:@oneof_enum(github.com/myorg/pkg/types.Status)
    Status types.Status  // 自动添加 import
}
```

### 嵌套结构体验证

| 注解 | 说明 |
|------|------|
| @method(MethodName) | 调用指定方法进行验证 |

```go
type Address struct {
    Street string
    City   string
}

func (a Address) Validate() error {
    if a.Street == "" {
        return fmt.Errorf("street is required")
    }
    return nil
}

// validategen:@validate
type User struct {
    // validategen:@method(Validate)
    Address Address  // 调用 Address.Validate()

    // validategen:@method(Validate)
    BillingAddress *Address  // 非 nil 时调用 Validate()
}
```

## 高级特性

### postValidate 钩子

如果需要添加自定义验证逻辑（比如跨字段验证），可以定义 postValidate 方法：

```go
// validategen:@validate
type User struct {
    // validategen:@required
    Role string

    // validategen:@gte(0)
    Age int
}

// postValidate 自定义验证逻辑
// 生成的 Validate() 方法会在所有字段验证后调用此方法
func (x User) postValidate(errs []string) error {
    // 跨字段验证：管理员必须年满 18 岁
    if x.Role == "admin" && x.Age < 18 {
        errs = append(errs, "admin must be at least 18 years old")
    }

    if len(errs) > 0 {
        return fmt.Errorf("%s", strings.Join(errs, "; "))
    }
    return nil
}
```

### 多规则组合

一个字段可以同时使用多个验证规则，按顺序执行：

```go
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@min(2)
    // validategen:@max(50)
    // validategen:@alpha
    Name string  // 必填，2-50个字符，只能是字母

    // validategen:@required
    // validategen:@email
    Email string  // 必填且格式正确的邮箱

    // validategen:@required
    // validategen:@min(1)
    // validategen:@max(65535)
    Port int  // 必填，范围 1-65535
}
```

## 完整示例

### 定义文件 (models/user.go)

```go
package models

import (
    "fmt"
    "strings"
)

// Address 地址
type Address struct {
    // validategen:@required
    Street string
    // validategen:@required
    City string
    // validategen:@len(6)
    ZipCode string
}

func (a Address) Validate() error {
    var errs []string
    if a.Street == "" {
        errs = append(errs, "street is required")
    }
    if a.City == "" {
        errs = append(errs, "city is required")
    }
    if len(a.ZipCode) != 6 {
        errs = append(errs, "zipcode must be 6 characters")
    }
    if len(errs) > 0 {
        return fmt.Errorf("%s", strings.Join(errs, "; "))
    }
    return nil
}

// User 用户模型
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@gt(0)
    ID int64

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

    // validategen:@required
    // validategen:@min(8)
    Password string

    // validategen:@oneof(admin, user, guest)
    Role string

    // validategen:@url
    Website string

    // validategen:@method(Validate)
    Address Address

    // validategen:@method(Validate)
    BillingAddress *Address
}

// postValidate 自定义验证
func (x User) postValidate(errs []string) error {
    if x.Role == "admin" && x.Age < 18 {
        errs = append(errs, "admin must be at least 18 years old")
    }
    if len(errs) > 0 {
        return fmt.Errorf("%s", strings.Join(errs, "; "))
    }
    return nil
}
```

### 使用示例

```go
package main

import (
    "fmt"
    "myapp/models"
)

func main() {
    // 有效用户
    user := models.User{
        ID:       1,
        Name:     "John Doe",
        Email:    "john@example.com",
        Age:      25,
        Password: "password123",
        Role:     "user",
        Address:  models.Address{Street: "123 Main St", City: "NYC", ZipCode: "100001"},
    }

    if err := user.Validate(); err != nil {
        fmt.Println("验证失败:", err)
    } else {
        fmt.Println("验证通过!")
    }

    // 无效用户
    invalidUser := models.User{
        ID:       0,           // 无效：gt(0)
        Name:     "",          // 无效：required
        Email:    "invalid",   // 无效：email 格式
        Password: "short",     // 无效：min(8)
        Role:     "superuser", // 无效：oneof
    }

    if err := invalidUser.Validate(); err != nil {
        fmt.Println("验证失败:", err)
        // 输出所有错误，用分号分隔
    }
}
```

## 常见错误

### 1. 忘记添加 @validate 注解

```go
// ❌ 错误：没有 @validate，不会生成 Validate() 方法
type User struct {
    // validategen:@required
    Name string
}

// ✅ 正确：添加 @validate
// validategen:@validate
type User struct {
    // validategen:@required
    Name string
}
```

### 2. 注解格式错误

```go
// ❌ 错误：缺少冒号
// validategen@required
Name string

// ❌ 错误：缺少 @
// validategen:required
Name string

// ✅ 正确
// validategen:@required
Name string
```

### 3. 参数类型不匹配

```go
// ❌ 错误：@email 只能用于 string
// validategen:@email
Age int

// ❌ 错误：@min 的参数必须是数字
// validategen:@min(abc)
Count int

// ✅ 正确
// validategen:@min(0)
Count int
```

### 4. 空字符串跳过格式验证

注意：@email、@url、@uuid 等格式验证对空字符串会跳过验证。如果字段必填，需要同时添加 @required：

```go
// ❌ 空字符串会通过验证
// validategen:@email
Email string

// ✅ 正确：必填 + 格式验证
// validategen:@required
// validategen:@email
Email string
```
