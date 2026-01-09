# validategen

中文 | [English](README_EN.md)

`validategen` 是一个 Go 结构体验证代码生成工具，为带有 `validategen:@validate` 注解的结构体生成 `Validate()` 方法。

## 安装

```bash
go install github.com/tlipoca9/devgen/cmd/validategen@latest
```

## 使用

```bash
validategen ./...              # 所有包
validategen ./pkg/models       # 指定包
```

## 注解

### @validate - 标记需要验证的结构体

在结构体定义上方添加注解，表示需要生成 `Validate()` 方法：

```go
// User 用户模型
// validategen:@validate
type User struct {
    // validategen:@required
    Name string
}
```

### 字段级验证注解

在字段注释中添加验证规则，支持多个规则组合使用。

---

## 验证规则详解

### 0. @default(value) - 默认值

为零值字段设置默认值。生成 `SetDefaults()` 方法，在验证前调用。

| 字段类型 | 说明 |
|---------|------|
| `string` | 设置默认字符串值 |
| `int/float/...` | 设置默认数值 |
| `bool` | 设置默认布尔值 (true/false) |

```go
// validategen:@validate
type Config struct {
    // validategen:@default(localhost)
    Host string  // 默认 "localhost"

    // validategen:@default(8080)
    Port int  // 默认 8080

    // validategen:@default(true)
    Enabled bool  // 默认 true

    // validategen:@default(1.0)
    Version float64  // 默认 1.0
}

// 使用方式：
cfg := &Config{}
cfg.SetDefaults()  // 设置默认值
if err := cfg.Validate(); err != nil {
    // 处理错误
}
```

**注意**：`SetDefaults()` 使用指针接收器，必须在指针上调用：
```go
cfg := &Config{}
cfg.SetDefaults()  // ✓ 正确

cfg2 := Config{}
cfg2.SetDefaults()  // ✗ 不会修改 cfg2
```

---

### 1. @required - 必填验证

验证字段不能为空/零值。

| 字段类型 | 验证逻辑 |
|---------|---------|
| `string` | 不能为空字符串 `""` |
| `int/float/...` | 不能为 `0` |
| `bool` | 必须为 `true` |
| `slice/map` | 长度不能为 `0` |
| `pointer` | 不能为 `nil` |

```go
// validategen:@validate
type User struct {
    // validategen:@required
    Name string  // 必填字符串

    // validategen:@required
    Age int  // 必填数字（不能为0）

    // validategen:@required
    IsActive bool  // 必须为 true

    // validategen:@required
    Tags []string  // 切片不能为空

    // validategen:@required
    Profile *Profile  // 指针不能为 nil
}
```

---

### 2. @min(n) - 最小值/最小长度

| 字段类型 | 验证逻辑 |
|---------|---------|
| `string` | 字符串长度 >= n |
| `slice/map` | 元素数量 >= n |
| `int/float/...` | 数值 >= n |

```go
// validategen:@validate
type Config struct {
    // validategen:@min(2)
    Name string  // 至少 2 个字符

    // validategen:@min(1)
    Tags []string  // 至少 1 个元素

    // validategen:@min(1)
    Port int  // 最小值为 1
}
```

---

### 3. @max(n) - 最大值/最大长度

| 字段类型 | 验证逻辑 |
|---------|---------|
| `string` | 字符串长度 <= n |
| `slice/map` | 元素数量 <= n |
| `int/float/...` | 数值 <= n |

```go
// validategen:@validate
type Config struct {
    // validategen:@max(50)
    Name string  // 最多 50 个字符

    // validategen:@max(10)
    Tags []string  // 最多 10 个元素

    // validategen:@max(65535)
    Port int  // 最大值为 65535
}
```

---

### 4. @len(n) - 精确长度

| 字段类型 | 验证逻辑 |
|---------|---------|
| `string` | 字符串长度 == n |
| `slice/map` | 元素数量 == n |

```go
// validategen:@validate
type VerifyCode struct {
    // validategen:@len(6)
    Code string  // 必须是 6 个字符

    // validategen:@len(3)
    Items []int  // 必须有 3 个元素
}
```

---

### 5. @gt(n) - 大于

| 字段类型 | 验证逻辑 |
|---------|---------|
| `string` | 字符串长度 > n |
| `slice/map` | 元素数量 > n |
| `int/float/...` | 数值 > n |

```go
// validategen:@validate
type Product struct {
    // validategen:@gt(0)
    ID int64  // 必须大于 0

    // validategen:@gt(5)
    Description string  // 长度必须大于 5

    // validategen:@gt(0)
    Items []string  // 元素数量必须大于 0
}
```

---

### 6. @gte(n) - 大于等于

| 字段类型 | 验证逻辑 |
|---------|---------|
| `string` | 字符串长度 >= n |
| `slice/map` | 元素数量 >= n |
| `int/float/...` | 数值 >= n |

```go
// validategen:@validate
type User struct {
    // validategen:@gte(0)
    Age int  // 年龄 >= 0

    // validategen:@gte(2)
    Name string  // 名字长度 >= 2

    // validategen:@gte(1)
    Roles []string  // 角色数量 >= 1
}
```

---

### 7. @lt(n) - 小于

| 字段类型 | 验证逻辑 |
|---------|---------|
| `string` | 字符串长度 < n |
| `slice/map` | 元素数量 < n |
| `int/float/...` | 数值 < n |

```go
// validategen:@validate
type Product struct {
    // validategen:@lt(100)
    Discount float32  // 折扣必须小于 100

    // validategen:@lt(256)
    ShortName string  // 长度必须小于 256

    // validategen:@lt(100)
    Tags []string  // 标签数量必须小于 100
}
```

---

### 8. @lte(n) - 小于等于

| 字段类型 | 验证逻辑 |
|---------|---------|
| `string` | 字符串长度 <= n |
| `slice/map` | 元素数量 <= n |
| `int/float/...` | 数值 <= n |

```go
// validategen:@validate
type User struct {
    // validategen:@lte(150)
    Age int  // 年龄 <= 150

    // validategen:@lte(1000)
    Weight uint  // 重量 <= 1000

    // validategen:@lte(50)
    Bio string  // 简介长度 <= 50
}
```

---

### 9. @eq(value) - 等于

验证字段值必须等于指定值。支持 `string`、`int/float`、`bool` 类型。

```go
// validategen:@validate
type Config struct {
    // validategen:@eq(1)
    Version int  // 版本必须等于 1

    // validategen:@eq(active)
    Status string  // 状态必须等于 "active"

    // validategen:@eq(true)
    Enabled bool  // 必须为 true
}
```

---

### 10. @ne(value) - 不等于

验证字段值不能等于指定值。支持 `string`、`int/float`、`bool` 类型。

```go
// validategen:@validate
type Product struct {
    // validategen:@ne(0)
    Stock int  // 库存不能为 0

    // validategen:@ne(deleted)
    Status string  // 状态不能为 "deleted"

    // validategen:@ne(false)
    Available bool  // 不能为 false
}
```

---

### 11. @oneof(a, b, c) - 枚举值

验证字段值必须是指定值之一。支持 `string` 和数字类型。

```go
// validategen:@validate
type User struct {
    // validategen:@oneof(admin, user, guest)
    Role string  // 角色必须是 admin、user 或 guest

    // validategen:@oneof(1, 2, 3)
    Level int  // 等级必须是 1、2 或 3
}

// validategen:@validate
type Config struct {
    // validategen:@oneof(debug, info, warn, error)
    LogLevel string  // 日志级别
}
```

---

### 12. @oneof_enum(EnumType) - 枚举类型验证

验证字段值必须是指定枚举类型的有效值。与 `@oneof` 不同，`@oneof_enum` 自动从 enumgen 生成的 `EnumTypeEnums.Contains()` 方法获取有效值，避免在 enum 新增值时需要同时修改 `@oneof` 注解。

**同包 enum：**
```go
// enumgen:@enum(string)
type Role int

const (
    RoleAdmin Role = iota
    RoleUser
    RoleGuest
)

// validategen:@validate
type User struct {
    // validategen:@oneof_enum(Role)
    Role Role  // 自动使用 RoleEnums.Contains() 验证
}

// 生成的验证代码:
// if !RoleEnums.Contains(x.Role) {
//     errs = append(errs, fmt.Sprintf("Role must be a valid Role, got %v", x.Role))
// }
```

**跨包 enum（完整 import 路径，自动添加 import）：**
```go
// validategen:@validate
type Request struct {
    // validategen:@oneof_enum(github.com/myorg/pkg/types.Status)
    Status types.Status  // 自动 import "github.com/myorg/pkg/types"
}

// 生成的验证代码会自动添加 import:
// import "github.com/myorg/pkg/types"
// ...
// if !types.StatusEnums.Contains(x.Status) { ... }
```

**跨包 enum（指定 import alias）：**
```go
// validategen:@validate
type Request struct {
    // validategen:@oneof_enum(mytypes:github.com/myorg/pkg/types.Status)
    Status mytypes.Status  // 使用指定的 alias
}

// 生成的验证代码：
// import mytypes "github.com/myorg/pkg/types"
// ...
// if !mytypes.StatusEnums.Contains(x.Status) { ... }
```

**优势**：当 enum 新增值时（如添加 `RoleModerator`），只需修改 enum 定义，无需修改 `@oneof_enum` 注解，符合单一修改原则。

---

### 13. @email - 邮箱格式

验证字符串是有效的邮箱地址。空字符串会跳过验证。

```go
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@email
    Email string  // 必填且格式正确的邮箱
}
```

---

### 14. @url - URL 格式

验证字符串是有效的 URL。空字符串会跳过验证。

```go
// validategen:@validate
type Profile struct {
    // validategen:@url
    Website string  // 可选的网站 URL

    // validategen:@required
    // validategen:@url
    Homepage string  // 必填的主页 URL
}
```

---

### 15. @uuid - UUID 格式

验证字符串是有效的 UUID（8-4-4-4-12 格式）。空字符串会跳过验证。

```go
// validategen:@validate
type Resource struct {
    // validategen:@uuid
    ID string  // UUID 格式的 ID

    // validategen:@required
    // validategen:@uuid
    TraceID string  // 必填的追踪 ID
}
```

---

### 16. @ip - IP 地址

验证字符串是有效的 IP 地址（IPv4 或 IPv6）。空字符串会跳过验证。

```go
// validategen:@validate
type Server struct {
    // validategen:@ip
    Address string  // 任意 IP 地址
}
```

---

### 17. @ipv4 - IPv4 地址

验证字符串是有效的 IPv4 地址。空字符串会跳过验证。

```go
// validategen:@validate
type NetworkConfig struct {
    // validategen:@ipv4
    IPv4Address string  // 必须是 IPv4 地址
}
```

---

### 18. @ipv6 - IPv6 地址

验证字符串是有效的 IPv6 地址。空字符串会跳过验证。

```go
// validategen:@validate
type NetworkConfig struct {
    // validategen:@ipv6
    IPv6Address string  // 必须是 IPv6 地址
}
```

---

### 19. @duration - 时间间隔格式

验证字符串是有效的 Go duration 格式（如 `1h30m`、`500ms`）。空字符串会跳过验证。

```go
// validategen:@validate
type Config struct {
    // validategen:@duration
    Timeout string  // 必须是有效的 duration 格式
}
```

---

### 20. @duration_min(duration) - 最小时间间隔

验证 duration 字符串的值不小于指定值。空字符串会跳过验证。

```go
// validategen:@validate
type Config struct {
    // validategen:@duration_min(1s)
    Timeout string  // 超时时间至少 1 秒

    // validategen:@duration_min(100ms)
    RetryInterval string  // 重试间隔至少 100 毫秒
}
```

---

### 21. @duration_max(duration) - 最大时间间隔

验证 duration 字符串的值不大于指定值。空字符串会跳过验证。

```go
// validategen:@validate
type Config struct {
    // validategen:@duration_max(1h)
    Timeout string  // 超时时间最多 1 小时

    // validategen:@duration_max(30s)
    RetryInterval string  // 重试间隔最多 30 秒
}
```

---

### 22. @duration + @duration_min + @duration_max 组合

可以组合使用这三个注解，生成的代码会合并为一个代码块，只解析一次：

```go
// validategen:@validate
type Config struct {
    // validategen:@duration
    // validategen:@duration_min(1s)
    // validategen:@duration_max(1h)
    RetryInterval string  // 有效 duration，范围 1s ~ 1h
}
```

---

### 23. @alpha - 纯字母

验证字符串只包含字母（a-zA-Z）。空字符串会跳过验证。

```go
// validategen:@validate
type Person struct {
    // validategen:@alpha
    FirstName string  // 只能包含字母
}
```

---

### 24. @alphanum - 字母数字

验证字符串只包含字母和数字（a-zA-Z0-9）。空字符串会跳过验证。

```go
// validategen:@validate
type User struct {
    // validategen:@alphanum
    Username string  // 只能包含字母和数字

    // validategen:@alphanum
    // validategen:@len(6)
    Code string  // 6位字母数字验证码
}
```

---

### 25. @numeric - 纯数字

验证字符串只包含数字（0-9）。空字符串会跳过验证。

```go
// validategen:@validate
type Contact struct {
    // validategen:@numeric
    PhoneNumber string  // 只能包含数字
}
```

---

### 26. @contains(substring) - 包含子串

验证字符串包含指定的子串。

```go
// validategen:@validate
type Email struct {
    // validategen:@contains(@)
    Address string  // 必须包含 @

    // validategen:@contains(example)
    TestEmail string  // 必须包含 "example"
}
```

---

### 27. @excludes(substring) - 不包含子串

验证字符串不包含指定的子串。

```go
// validategen:@validate
type User struct {
    // validategen:@excludes(admin)
    DisplayName string  // 不能包含 "admin"

    // validategen:@excludes(<script>)
    Bio string  // 不能包含 "<script>"
}
```

---

### 28. @startswith(prefix) - 前缀匹配

验证字符串以指定前缀开头。

```go
// validategen:@validate
type URL struct {
    // validategen:@startswith(https://)
    SecureURL string  // 必须以 https:// 开头

    // validategen:@startswith(+86)
    ChinaPhone string  // 必须以 +86 开头
}
```

---

### 29. @endswith(suffix) - 后缀匹配

验证字符串以指定后缀结尾。

```go
// validategen:@validate
type Domain struct {
    // validategen:@endswith(.com)
    Website string  // 必须以 .com 结尾

    // validategen:@endswith(.go)
    GoFile string  // 必须以 .go 结尾
}
```

---

### 30. @regex(pattern) - 正则表达式

验证字符串匹配指定的正则表达式。空字符串会跳过验证。

```go
// validategen:@validate
type Product struct {
    // validategen:@regex(^[A-Z]{2}-\d{4}$)
    ProductCode string  // 格式：XX-0000（两个大写字母-四位数字）

    // validategen:@regex(^\d{4}-\d{2}-\d{2}$)
    Date string  // 格式：YYYY-MM-DD
}
```

---

### 31. @format(type) - 格式验证

验证字符串是有效的指定格式。支持 `json`、`yaml`、`toml`、`csv` 四种格式。空字符串会跳过验证。

| 格式 | 说明 |
|------|------|
| `json` | 使用 `encoding/json.Valid` 验证 |
| `yaml` | 使用 `gopkg.in/yaml.v3` 解析验证 |
| `toml` | 使用 `github.com/BurntSushi/toml` 解析验证 |
| `csv` | 使用 `encoding/csv` 解析验证 |

```go
// validategen:@validate
type Config struct {
    // validategen:@format(json)
    JSONConfig string  // 必须是有效的 JSON 格式

    // validategen:@format(yaml)
    YAMLConfig string  // 必须是有效的 YAML 格式

    // validategen:@format(toml)
    TOMLConfig string  // 必须是有效的 TOML 格式

    // validategen:@format(csv)
    CSVData string  // 必须是有效的 CSV 格式
}
```

---

### 32. @dns1123_label - DNS 标签格式

验证字符串符合 RFC 1123 DNS 标签规范。空字符串会跳过验证。

**DNS 标签规则**：
- 只能包含小写字母、数字和连字符
- 必须以字母或数字开头
- 必须以字母或数字结尾
- 每个标签最多 63 个字符

```go
// validategen:@validate
type KubernetesObject struct {
    // validategen:@required
    // validategen:@dns1123_label
    Namespace string  // "default", "kube-system" ✓

    // validategen:@required
    // validategen:@dns1123_label
    PodName string  // "my-pod-123" ✓, "Pod" ✗, "-invalid" ✗

    // validategen:@dns1123_label
    ServiceName string  // "api-service" ✓
}
```

**适用场景**：
- Kubernetes 对象命名（Pod、Service、Namespace、ConfigMap）
- DNS 主机名验证
- 微服务实例命名
- 容器镜像仓库域名验证

---

### 33. @cpu - Kubernetes CPU 资源格式

验证字符串是有效的 Kubernetes CPU 资源数量。空字符串会跳过验证。

**支持的格式**：
- 毫核：`500m`, `100m`
- 核数：`1`, `2`, `0.5`
- 科学记数法：`1e3m` (1000m)

```go
// validategen:@validate
type PodSpec struct {
    // validategen:@required
    // validategen:@cpu
    CPURequest string  // "500m", "1" ✓

    // validategen:@cpu
    CPULimit string  // 可选的 CPU 限制
}
```

---

### 34. @memory - Kubernetes 内存资源格式

验证字符串是有效的 Kubernetes 内存资源数量。空字符串会跳过验证。

**支持的格式**：
- 二进制单位：`128Mi`, `1Gi`, `512Ki`
- 十进制单位：`128M`, `1G`
- 字节数：`134217728`

```go
// validategen:@validate
type PodSpec struct {
    // validategen:@required
    // validategen:@memory
    MemoryRequest string  // "128Mi", "1Gi" ✓

    // validategen:@memory
    MemoryLimit string  // 可选的内存限制
}
```

---

### 35. @disk - Kubernetes 磁盘资源格式

验证字符串是有效的 Kubernetes 存储资源数量。空字符串会跳过验证。

**支持的格式**：
- 二进制单位：`10Gi`, `100Gi`, `1Ti`
- 十进制单位：`10G`, `100G`

```go
// validategen:@validate
type PersistentVolume struct {
    // validategen:@required
    // validategen:@disk
    StorageRequest string  // "10Gi", "100Gi" ✓

    // validategen:@disk
    StorageLimit string  // 可选的存储限制
}
```

---

### 36. @method(MethodName) - 调用验证方法

调用嵌套结构体或自定义类型的验证方法。对于指针类型，会先检查 nil。

```go
// Address 地址
type Address struct {
    Street string
    City   string
}

// Validate 验证地址
func (a Address) Validate() error {
    if a.Street == "" {
        return fmt.Errorf("street is required")
    }
    if a.City == "" {
        return fmt.Errorf("city is required")
    }
    return nil
}

// Status 自定义状态类型
type Status int

// Validate 验证状态
func (s Status) Validate() error {
    if s < 0 || s > 10 {
        return fmt.Errorf("status must be between 0 and 10")
    }
    return nil
}

// validategen:@validate
type User struct {
    // validategen:@method(Validate)
    Address Address  // 调用 Address.Validate()

    // validategen:@method(Validate)
    OptionalAddress *Address  // 非 nil 时调用 Validate()

    // validategen:@method(Validate)
    Status Status  // 调用 Status.Validate()
}
```

---

## 高级特性

### postValidate 钩子

如果结构体定义了 `postValidate(errs []string) error` 方法，生成的 `Validate()` 方法会在所有字段验证后调用它，并传入已收集的错误列表。

```go
// validategen:@validate
type User struct {
    // validategen:@required
    Role string

    // validategen:@gte(0)
    Age int
}

// postValidate 自定义验证逻辑
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

### 多规则组合

一个字段可以同时使用多个验证规则：

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

---

## 完整示例

### 定义

```go
package models

import "fmt"

// Address 地址
type Address struct {
    Street string
    City   string
}

// Validate 验证地址
func (a Address) Validate() error {
    if a.Street == "" {
        return fmt.Errorf("street is required")
    }
    if a.City == "" {
        return fmt.Errorf("city is required")
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

    // validategen:@uuid
    UUID string

    // validategen:@ip
    IP string

    // validategen:@alphanum
    // validategen:@len(6)
    Code string

    // validategen:@method(Validate)
    Address Address

    // validategen:@method(Validate)
    OptionalAddress *Address
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

运行代码生成：

```bash
validategen ./...
```

### 使用

```go
package main

import (
    "fmt"
    
    "example.com/models"
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
        Code:     "ABC123",
        Address:  models.Address{Street: "123 Main St", City: "New York"},
    }
    
    if err := user.Validate(); err != nil {
        fmt.Println("Validation failed:", err)
    } else {
        fmt.Println("Validation passed!")
    }
    
    // 无效用户
    invalidUser := models.User{
        ID:       0,  // 无效：required 且 gt(0)
        Name:     "",  // 无效：required
        Email:    "invalid-email",  // 无效：email 格式
        Password: "short",  // 无效：min(8)
        Role:     "invalid",  // 无效：oneof
    }
    
    if err := invalidUser.Validate(); err != nil {
        fmt.Println("Validation failed:", err)
        // Output: ID is required; ID must be greater than 0, got 0; Name is required; ...
    }
}
```

---

## 测试与基准测试

### 运行单元测试

```bash
# 运行所有测试
go test -v ./cmd/validategen/generator/...

# 运行测试并查看覆盖率
go test -v ./cmd/validategen/generator/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### 运行基准测试

基准测试使用 [Ginkgo gmeasure](https://onsi.github.io/ginkgo/#benchmarking-code) 实现，可以获得统计学意义上的性能数据。

#### Step 1: 运行基准测试

```bash
# 运行测试（包含基准测试）
cd /path/to/devgen
go test -v ./cmd/validategen/generator/... -count=1
```

#### Step 2: 生成报告文件

```bash
# 生成 JSON 格式报告
ginkgo --json-report=benchmark_report.json ./cmd/validategen/generator/...

# 生成 JUnit XML 格式报告
ginkgo --junit-report=benchmark_report.xml ./cmd/validategen/generator/...

# 同时生成两种格式
ginkgo --json-report=benchmark_report.json --junit-report=benchmark_report.xml ./cmd/validategen/generator/...
```

### 基准测试结果摘要

**测试环境**：
- OS: darwin (macOS)
- Arch: arm64
- CPU: Apple M4 Pro
- 每个注解执行 1000 次迭代，采样 100 次

#### 简单验证注解（无正则）

| 注解 | Mean (1000次) | 单次约 | 说明 |
|------|---------------|--------|------|
| `@required` (string) | 272ns | 0.27ns | 字符串非空检查 |
| `@required` (int) | 273ns | 0.27ns | 数值非零检查 |
| `@required` (slice) | 269ns | 0.27ns | 切片长度检查 |
| `@required` (pointer) | 288ns | 0.29ns | 指针非 nil 检查 |
| `@min` / `@max` (int) | 270ns | 0.27ns | 数值比较 |
| `@min` / `@max` (string len) | 271ns | 0.27ns | 字符串长度比较 |
| `@len` | 272ns | 0.27ns | 精确长度检查 |
| `@gt` / `@gte` / `@lt` / `@lte` | 271ns | 0.27ns | 数值比较 |
| `@eq` / `@ne` | 270ns | 0.27ns | 等值比较 |
| `@oneof` (string, 4 values) | 531ns | 0.53ns | 枚举值检查 |
| `@oneof` (int) | 269ns | 0.27ns | 整数枚举检查 |

#### 字符串操作注解

| 注解 | Mean (1000次) | 单次约 | 说明 |
|------|---------------|--------|------|
| `@contains` | 4.8µs | 4.8ns | `strings.Contains` |
| `@excludes` | 4.5µs | 4.5ns | `!strings.Contains` |
| `@startswith` | 299ns | 0.30ns | `strings.HasPrefix` |
| `@endswith` | 1.6µs | 1.6ns | `strings.HasSuffix` |

#### 正则验证注解

| 注解 | Mean (1000次) | 单次约 | 说明 |
|------|---------------|--------|------|
| `@email` (valid) | 180µs | 180ns | 邮箱正则匹配 |
| `@email` (invalid) | 156µs | 156ns | 正则快速失败 |
| `@url` (valid) | 253µs | 253ns | URL 正则匹配 |
| `@url` (invalid) | 16µs | 16ns | 正则快速失败 |
| `@uuid` (valid) | 152µs | 152ns | UUID 正则匹配 |
| `@uuid` (invalid) | 1.3µs | 1.3ns | 正则快速失败 |
| `@alpha` | 89µs | 89ns | 纯字母正则 |
| `@alphanum` | 78µs | 78ns | 字母数字正则 |
| `@numeric` | 89µs | 89ns | 纯数字正则 |
| `@regex` (simple) | 45µs | 45ns | 简单自定义正则 |
| `@regex` (complex) | 246µs | 246ns | 复杂正则表达式 |

#### 格式验证注解

| 注解 | Mean (1000次) | 单次约 | 说明 |
|------|---------------|--------|------|
| `@format(json)` (valid) | 69µs | 69ns | `json.Valid` |
| `@format(json)` (invalid) | 131µs | 131ns | JSON 解析失败 |
| `@format(yaml)` (valid) | 3.85ms | 3.85µs | YAML 解析 |
| `@format(yaml)` (invalid) | 3.37ms | 3.37µs | YAML 解析失败 |
| `@format(toml)` (valid) | 1.72ms | 1.72µs | TOML 解析 |
| `@format(toml)` (invalid) | 1.31ms | 1.31µs | TOML 解析失败 |
| `@format(csv)` (valid) | 885µs | 885ns | CSV 解析 |
| `@format(csv)` (invalid) | 854µs | 854ns | CSV 解析失败 |

#### IP 地址验证注解

| 注解 | Mean (1000次) | 单次约 | 说明 |
|------|---------------|--------|------|
| `@ip` (ipv4) | 12.6µs | 12.6ns | `net.ParseIP` |
| `@ip` (ipv6) | 42µs | 42ns | IPv6 解析较慢 |
| `@ipv4` | 15.6µs | 15.6ns | IPv4 + To4() 检查 |
| `@ipv6` | 43.6µs | 43.6ns | IPv6 + To4() 检查 |

#### Duration 验证注解

| 注解 | Mean (1000次) | 单次约 | 说明 |
|------|---------------|--------|------|
| `@duration` (valid) | 25.45µs | 25.45ns | `time.ParseDuration` |
| `@duration` (invalid) | 67.56µs | 67.56ns | 解析失败 |
| `@duration_min` | 11.78µs | 11.78ns | 解析后比较纳秒值 |
| `@duration_max` | ~11µs | ~11ns | 解析后比较纳秒值 |
| 组合使用 | ~11µs | ~11ns | 只解析一次，多次比较 |

### 性能分析

**性能亮点**：
- 简单验证注解性能极佳（< 1ns/op）：`@required`、`@min`、`@max`、`@eq`、`@ne` 等
- 预编译正则表达式，避免重复编译开销
- 无效输入通常比有效输入快，因为正则可以快速失败

**性能差异原因**：
- 格式验证（`@format`）需要完整解析，YAML/TOML 较慢（µs 级别）
- 正则验证（`@email`、`@uuid`、`@regex`）比简单比较慢 100-500 倍
- IP 地址解析需要调用 `net.ParseIP`，性能中等
- 字符串操作（`@contains`、`@startswith`）性能介于两者之间
- JSON 验证最快（使用 `json.Valid`），CSV 次之

---

## 注解速查表

| 注解 | 参数 | 适用类型 | 说明 |
|------|------|---------|------|
| `@validate` | - | struct | 标记需要生成 Validate 方法的结构体 |
| `@default(v)` | string/number/bool | string, number, bool | 设置默认值（生成 SetDefaults 方法） |
| `@required` | - | string, number, bool, slice, map, pointer | 必填验证 |
| `@min(n)` | number | string, slice, map, number | 最小值/最小长度 |
| `@max(n)` | number | string, slice, map, number | 最大值/最大长度 |
| `@len(n)` | number | string, slice, map | 精确长度 |
| `@gt(n)` | number | string, slice, map, number | 大于 |
| `@gte(n)` | number | string, slice, map, number | 大于等于 |
| `@lt(n)` | number | string, slice, map, number | 小于 |
| `@lte(n)` | number | string, slice, map, number | 小于等于 |
| `@eq(v)` | string/number/bool | string, number, bool | 等于 |
| `@ne(v)` | string/number/bool | string, number, bool | 不等于 |
| `@oneof(a, b, c)` | 逗号分隔的值列表 | string, number | 枚举值 |
| `@oneof_enum(EnumType)` | 枚举类型名 | enum type | 枚举类型验证 |
| `@email` | - | string | 邮箱格式 |
| `@url` | - | string | URL 格式 |
| `@uuid` | - | string | UUID 格式 |
| `@ip` | - | string | IP 地址（v4 或 v6） |
| `@ipv4` | - | string | IPv4 地址 |
| `@ipv6` | - | string | IPv6 地址 |
| `@duration` | - | string | 时间间隔格式（如 1h30m, 500ms） |
| `@duration_min(d)` | duration | string | 最小时间间隔 |
| `@duration_max(d)` | duration | string | 最大时间间隔 |
| `@alpha` | - | string | 纯字母 |
| `@alphanum` | - | string | 字母数字 |
| `@numeric` | - | string | 纯数字 |
| `@contains(s)` | string | string | 包含子串 |
| `@excludes(s)` | string | string | 不包含子串 |
| `@startswith(s)` | string | string | 前缀匹配 |
| `@endswith(s)` | string | string | 后缀匹配 |
| `@regex(pattern)` | regex | string | 正则匹配 |
| `@format(type)` | json, yaml, toml, csv | string | 格式验证 |
| `@dns1123_label` | - | string | DNS 标签格式（RFC 1123） |
| `@cpu` | - | string | Kubernetes CPU 资源格式 |
| `@memory` | - | string | Kubernetes 内存资源格式 |
| `@disk` | - | string | Kubernetes 磁盘资源格式 |
| `@method(name)` | method name | struct, pointer, custom type | 调用验证方法 |

---

## 支持的数值类型

validategen 支持所有 Go 内置数值类型：

- 有符号整数：`int`, `int8`, `int16`, `int32`, `int64`
- 无符号整数：`uint`, `uint8`, `uint16`, `uint32`, `uint64`
- 浮点数：`float32`, `float64`
- 别名类型：`byte` (uint8), `rune` (int32), `uintptr`
