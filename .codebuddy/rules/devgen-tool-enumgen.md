---
description: Go 枚举代码生成工具 enumgen 的使用指南。当用户需要定义类型安全的枚举、生成枚举辅助方法（String、JSON、SQL等）时使用此规则。
globs: *.go
alwaysApply: false
---

# enumgen - Go 枚举代码生成工具

enumgen 是 devgen 工具集的一部分，用于为 Go 枚举类型自动生成辅助方法。

## 什么时候使用 enumgen？

当你需要：
- 定义类型安全的枚举（而不是裸的 int/string 常量）
- 枚举值与字符串之间的转换（String() 方法）
- 枚举的 JSON 序列化/反序列化
- 枚举的数据库存储（SQL driver 接口）
- 验证一个值是否是有效的枚举值

## 第一步：定义枚举类型

### 基本语法

在类型定义的注释中添加 `enumgen:@enum(...)` 注解：

```go
// Status 表示订单状态
// enumgen:@enum(string, json)
type Status int

const (
    StatusPending   Status = iota + 1  // 待处理
    StatusConfirmed                     // 已确认
    StatusShipped                       // 已发货
    StatusDelivered                     // 已送达
)
```

### 注解参数说明

| 参数 | 作用 | 生成的方法 |
|------|------|-----------|
| string | 实现 fmt.Stringer 接口 | String() string |
| json | 实现 JSON 序列化接口 | MarshalJSON() / UnmarshalJSON() |
| text | 实现文本序列化接口 | MarshalText() / UnmarshalText() |
| sql | 实现数据库接口 | Value() / Scan() |

**常用组合**：
- `enumgen:@enum(string)` - 只需要打印输出
- `enumgen:@enum(string, json)` - API 接口常用
- `enumgen:@enum(string, json, sql)` - 需要存数据库

## 第二步：运行代码生成

```bash
# 在项目根目录运行
devgen ./...

# 或者只处理特定包
devgen ./pkg/models
```

生成的文件命名为 `{package}_enum.go`，例如 `models_enum.go`。

## 第三步：使用生成的代码

### 生成的辅助变量

对于 `Status` 类型，会生成 `StatusEnums` 辅助变量：

```go
// 获取所有有效值
allStatuses := StatusEnums.List()
// 返回: []Status{StatusPending, StatusConfirmed, StatusShipped, StatusDelivered}

// 检查值是否有效
isValid := StatusEnums.Contains(StatusPending)  // true
isValid = StatusEnums.Contains(Status(999))     // false

// 从字符串解析
status, err := StatusEnums.Parse("Pending")
if err != nil {
    // 处理无效字符串
}

// 获取字符串名称
name := StatusEnums.Name(StatusPending)  // "Pending"

// 获取所有名称
names := StatusEnums.Names()  // []string{"Pending", "Confirmed", "Shipped", "Delivered"}
```

### 类型方法

```go
status := StatusPending

// 检查是否有效（始终生成）
if status.IsValid() {
    // ...
}

// String() 方法（需要 string 参数）
fmt.Println(status)  // 输出: Pending

// JSON 序列化（需要 json 参数）
data, _ := json.Marshal(status)  // "Pending"

var s Status
json.Unmarshal([]byte(`"Confirmed"`), &s)  // s = StatusConfirmed
```

## 自定义枚举值名称

默认情况下，枚举值的字符串名称会自动去除类型名前缀：
- `StatusPending` → `"Pending"`
- `StatusConfirmed` → `"Confirmed"`

如果需要自定义名称，使用 `@name` 注解：

```go
// ErrorCode 错误码
// enumgen:@enum(string, json)
type ErrorCode int

const (
    // enumgen:@name(ERR_NOT_FOUND)
    ErrorCodeNotFound ErrorCode = 404

    // enumgen:@name(ERR_INTERNAL)
    ErrorCodeInternal ErrorCode = 500

    // enumgen:@name(ERR_BAD_REQUEST)
    ErrorCodeBadRequest ErrorCode = 400
)
```

使用效果：
```go
fmt.Println(ErrorCodeNotFound.String())  // "ERR_NOT_FOUND"
code, _ := ErrorCodeEnums.Parse("ERR_INTERNAL")  // ErrorCodeInternal
```

## 字符串底层类型

enumgen 也支持 `string` 作为底层类型：

```go
// Color 颜色枚举
// enumgen:@enum(string, json)
type Color string

const (
    ColorRed   Color = "red"
    ColorGreen Color = "green"
    ColorBlue  Color = "blue"
)
```

**注意**：字符串类型枚举：
- ✅ 支持 IsValid()、String()、JSON、SQL 等方法
- ❌ 不支持 `@name` 注解（字符串值本身就是名称）
- ❌ 不生成 Name()、Names()、ContainsName() 方法

## 支持的底层类型

| 类型 | 支持 |
|------|------|
| int, int8, int16, int32, int64 | ✅ |
| uint, uint8, uint16, uint32, uint64 | ✅ |
| string | ✅ |
| float32, float64 | ❌ |
| bool | ❌ |

## 完整示例

### 定义文件 (models/order.go)

```go
package models

// OrderStatus 订单状态
// enumgen:@enum(string, json, sql)
type OrderStatus int

const (
    OrderStatusPending    OrderStatus = iota + 1  // 待处理
    OrderStatusProcessing                          // 处理中
    OrderStatusCompleted                           // 已完成
    // enumgen:@name(Cancelled)
    OrderStatusCanceled                            // 已取消（注意：使用英式拼写）
)

// PaymentMethod 支付方式
// enumgen:@enum(string, json)
type PaymentMethod string

const (
    PaymentMethodCreditCard PaymentMethod = "credit_card"
    PaymentMethodDebitCard  PaymentMethod = "debit_card"
    PaymentMethodPayPal     PaymentMethod = "paypal"
    PaymentMethodCrypto     PaymentMethod = "crypto"
)
```

### 使用示例

```go
package main

import (
    "encoding/json"
    "fmt"
    "myapp/models"
)

func main() {
    // 创建订单
    order := Order{
        Status:  models.OrderStatusPending,
        Payment: models.PaymentMethodCreditCard,
    }

    // JSON 序列化
    data, _ := json.Marshal(order)
    fmt.Println(string(data))
    // {"status":"Pending","payment":"credit_card"}

    // 从 API 请求解析
    var req struct {
        Status models.OrderStatus `json:"status"`
    }
    json.Unmarshal([]byte(`{"status":"Processing"}`), &req)
    fmt.Println(req.Status)  // Processing

    // 验证用户输入
    userInput := "InvalidStatus"
    if _, err := models.OrderStatusEnums.Parse(userInput); err != nil {
        fmt.Println("无效的订单状态:", userInput)
    }

    // 在下拉框中显示所有选项
    for _, status := range models.OrderStatusEnums.List() {
        fmt.Printf("值: %d, 名称: %s\n", status, status.String())
    }
}
```

## 常见错误

### 1. 忘记运行 devgen

```
错误: undefined: StatusEnums
解决: 运行 devgen ./...
```

### 2. 底层类型不支持

```go
// ❌ 错误：float 不支持
// enumgen:@enum(string)
type Score float64

// ✅ 正确：使用 int
// enumgen:@enum(string)
type Score int
```

### 3. 字符串类型使用 @name

```go
// ❌ 错误：string 类型不支持 @name
// enumgen:@enum(string)
type Color string

const (
    // enumgen:@name(RED)  // 这会报错！
    ColorRed Color = "red"
)

// ✅ 正确：直接使用想要的字符串值
const (
    ColorRed Color = "RED"
)
```

### 4. @name 值重复

```go
// ❌ 错误：重复的 @name 值
const (
    // enumgen:@name(Active)
    StatusActive Status = 1
    // enumgen:@name(Active)  // 重复！
    StatusEnabled Status = 2
)
```

## 与 validategen 配合使用

enumgen 生成的枚举可以与 validategen 的 `@oneof_enum` 配合：

```go
// enumgen:@enum(string, json)
type Role int

const (
    RoleAdmin Role = iota + 1
    RoleUser
    RoleGuest
)

// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@oneof_enum(Role)
    Role Role
}
```

这样当 Role 新增值时，验证逻辑会自动包含新值，无需手动更新。
