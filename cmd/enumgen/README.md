# enumgen

中文 | [English](README_EN.md)

`enumgen` 是一个 Go 枚举代码生成工具，为带有 `enumgen:@enum` 注解的类型生成辅助方法。

## 安装

```bash
go install github.com/tlipoca9/devgen/cmd/enumgen@latest
```

## 使用

```bash
enumgen ./...              # 所有包
enumgen ./pkg/status       # 指定包
```

## 注解

### @enum - 标记枚举类型

在类型定义上方添加注解，指定要生成的方法：

```go
// Status 表示状态
// enumgen:@enum(string, json, text, sql)
type Status int

const (
    StatusPending Status = iota
    StatusActive
    StatusInactive
)
```

支持的选项：
- `string` - 生成 `String()` 方法
- `json` - 生成 `MarshalJSON()` / `UnmarshalJSON()` 方法
- `text` - 生成 `MarshalText()` / `UnmarshalText()` 方法
- `sql` - 生成 `Value()` (driver.Valuer) / `Scan()` (sql.Scanner) 方法

### @name - 自定义值名称

默认情况下，枚举值的字符串名称会自动去除类型名前缀（如 `StatusPending` → `Pending`）。

使用 `@name` 可以自定义名称：

```go
// Level 表示日志级别
// enumgen:@enum(string, json)
type Level int

const (
    // enumgen:@name(DEBUG)
    LevelDebug Level = iota + 1
    // enumgen:@name(INFO)
    LevelInfo
    // enumgen:@name(WARN)
    LevelWarn
    // enumgen:@name(ERROR)
    LevelError
)
```

**注意**：`@name` 的值不能重复，否则会报错。

## 生成的代码

对于带有 `enumgen:@enum(string, json, text, sql)` 注解的 `Status` 类型，会生成以下代码：

### 类型方法

无论选择哪些选项，都会生成 `IsValid()` 方法：

```go
// IsValid reports whether x is a valid Status.
func (x Status) IsValid() bool {
    return StatusEnums.Contains(x)
}
```

根据注解选项生成对应的方法：

**string 选项：**
```go
// String returns the string representation of Status.
func (x Status) String() string {
    return StatusEnums.Name(x)
}
```

**json 选项：**
```go
// MarshalJSON implements json.Marshaler.
func (x Status) MarshalJSON() ([]byte, error) {
    return json.Marshal(StatusEnums.Name(x))
}

// UnmarshalJSON implements json.Unmarshaler.
func (x *Status) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return err
    }
    v, err := StatusEnums.Parse(s)
    if err != nil {
        return err
    }
    *x = v
    return nil
}
```

**text 选项：**
```go
// MarshalText implements encoding.TextMarshaler.
func (x Status) MarshalText() ([]byte, error) {
    return []byte(StatusEnums.Name(x)), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (x *Status) UnmarshalText(data []byte) error {
    v, err := StatusEnums.Parse(string(data))
    if err != nil {
        return err
    }
    *x = v
    return nil
}
```

**sql 选项：**
```go
// Value implements driver.Valuer.
func (x Status) Value() (driver.Value, error) {
    return StatusEnums.Name(x), nil
}

// Scan implements sql.Scanner.
func (x *Status) Scan(src any) error {
    if src == nil {
        return nil
    }
    var s string
    switch v := src.(type) {
    case string:
        s = v
    case []byte:
        s = string(v)
    default:
        return fmt.Errorf("cannot scan %T into Status", src)
    }
    v, err := StatusEnums.Parse(s)
    if err != nil {
        return err
    }
    *x = v
    return nil
}
```

### 辅助变量 StatusEnums

无论选择哪些选项，都会生成辅助变量和类型：

```go
// StatusEnums is the enum helper for Status.
var StatusEnums = _StatusEnums{
    values: []Status{
        StatusPending,
        StatusActive,
        StatusInactive,
    },
    names: map[Status]string{
        StatusPending:  "Pending",
        StatusActive:   "Active",
        StatusInactive: "Inactive",
    },
    byName: map[string]Status{
        "Pending":  StatusPending,
        "Active":   StatusActive,
        "Inactive": StatusInactive,
    },
}

// _StatusEnums provides enum metadata and validation for Status.
type _StatusEnums struct {
    values []Status
    names  map[Status]string
    byName map[string]Status
}
```

### 辅助方法

| 方法 | 说明 |
|------|------|
| `IsValid() bool` | 检查当前值是否有效（类型方法，始终生成） |
| `List() []Status` | 返回所有有效的枚举值 |
| `Contains(v Status) bool` | 检查值是否有效 |
| `ContainsName(name string) bool` | 检查名称是否有效 |
| `Parse(s string) (Status, error)` | 从字符串解析枚举值 |
| `Name(v Status) string` | 获取枚举值的字符串名称 |
| `Names() []string` | 返回所有有效的名称列表 |

## 完整示例

### 定义

```go
package order

// OrderStatus 订单状态
// enumgen:@enum(string, json, sql)
type OrderStatus int

const (
    OrderStatusPending    OrderStatus = iota + 1 // 待处理
    OrderStatusProcessing                        // 处理中
    OrderStatusCompleted                         // 已完成
    // enumgen:@name(Cancelled)
    OrderStatusCanceled                          // 已取消（自定义名称）
)
```

运行代码生成：

```bash
enumgen ./...
```

### 使用

```go
package main

import (
    "encoding/json"
    "fmt"
    
    "example.com/order"
)

func main() {
    status := order.OrderStatusPending
    
    // String
    fmt.Println(status.String()) // Output: Pending
    
    // JSON 序列化
    data, _ := json.Marshal(status)
    fmt.Println(string(data)) // Output: "Pending"
    
    // JSON 反序列化
    var s order.OrderStatus
    json.Unmarshal([]byte(`"Completed"`), &s)
    fmt.Println(s) // Output: Completed
    
    // 解析字符串
    parsed, err := order.OrderStatusEnums.Parse("Processing")
    if err == nil {
        fmt.Println(parsed) // Output: Processing
    }
    
    // 列出所有值
    for _, v := range order.OrderStatusEnums.List() {
        fmt.Printf("%d: %s\n", v, order.OrderStatusEnums.Name(v))
    }
    // Output:
    // 1: Pending
    // 2: Processing
    // 3: Completed
    // 4: Cancelled
    
    // 验证
    fmt.Println(order.OrderStatusEnums.Contains(order.OrderStatusPending)) // true
    fmt.Println(order.OrderStatusEnums.ContainsName("Invalid"))            // false
}
```

## 测试与基准测试

### 运行单元测试

```bash
# 运行所有测试
go test -v ./cmd/enumgen/generator/...

# 运行测试并查看覆盖率
go test -v ./cmd/enumgen/generator/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

当前测试覆盖率：**99.6%**（46 个测试用例）

### 运行基准测试

基准测试使用 [Ginkgo gmeasure](https://onsi.github.io/ginkgo/#benchmarking-code) 实现，可以获得统计学意义上的性能数据。

#### Step 1: 运行基准测试

```bash
# 运行测试（包含基准测试）
cd /path/to/devgen
go test -v ./cmd/enumgen/generator/... -count=1
```

#### Step 2: 生成报告文件

```bash
# 生成 JSON 格式报告
ginkgo --json-report=benchmark_report.json ./cmd/enumgen/generator/...

# 生成 JUnit XML 格式报告
ginkgo --junit-report=benchmark_report.xml ./cmd/enumgen/generator/...

# 同时生成两种格式
ginkgo --json-report=benchmark_report.json --junit-report=benchmark_report.xml ./cmd/enumgen/generator/...
```

#### Step 3: 查看详细输出

运行测试时会在控制台输出详细的基准测试结果，包括：
- Mean（平均值）
- StdDev（标准差）
- Min/Max（最小/最大值）
- 采样次数和迭代次数

### 基准测试结果摘要 (2025-12-07)

**测试环境**：
- OS: darwin (macOS)
- Arch: arm64
- CPU: Apple M4 Pro
- 每个方法执行 1000 次迭代，采样 100 次

**测试配置**：
- RandomSeed: 1765086644
- TotalSpecs: 70
- SuiteSucceeded: true
- RunTime: ~290ms

| 方法 | Mean (1000次) | 单次约 | 说明 |
|------|---------------|--------|------|
| IsValid/valid | 1.54µs | 1.5ns | 验证有效枚举值 |
| IsValid/invalid | 2.77µs | 2.8ns | 验证无效枚举值 |
| String/valid | 2.26µs | 2.3ns | 有效值转字符串 |
| String/invalid | 69.87µs | 70ns | 无效值转字符串（需格式化） |
| MarshalJSON/direct | 53.66µs | 54ns | 直接调用 MarshalJSON |
| MarshalJSON/via_json_Marshal | 124.24µs | 124ns | 通过 json.Marshal 调用 |
| UnmarshalJSON/direct | 95.93µs | 96ns | 直接调用 UnmarshalJSON |
| UnmarshalJSON/via_json_Unmarshal | 157.38µs | 157ns | 通过 json.Unmarshal 调用 |
| MarshalText | 15.32µs | 15ns | 文本序列化 |
| UnmarshalText | 15.15µs | 15ns | 文本反序列化 |
| Value | 11.61µs | 12ns | SQL driver.Valuer |
| Scan/string | 7.61µs | 7.6ns | SQL Scanner (string) |
| Scan/bytes | 30.09µs | 30ns | SQL Scanner ([]byte) |
| Scan/nil | 1.11µs | 1.1ns | SQL Scanner (nil) |
| Parse/valid | 5.29µs | 5.3ns | 解析有效字符串 |
| Parse/invalid | 95.01µs | 95ns | 解析无效字符串（需创建 error） |
| Contains/valid | 1.80µs | 1.8ns | 检查有效值 |
| Contains/invalid | 2.57µs | 2.6ns | 检查无效值 |
| ContainsName/valid | 5.36µs | 5.4ns | 检查有效名称 |
| ContainsName/invalid | 6.16µs | 6.2ns | 检查无效名称 |
| Name/valid | 2.31µs | 2.3ns | 获取有效值名称 |
| Name/invalid | 55.12µs | 55ns | 获取无效值名称（需格式化） |
| List | 301ns | 0.3ns | 返回所有枚举值 |
| Names | 24.36µs | 24ns | 返回所有名称 |

### 性能分析

**性能亮点**：
- 核心方法（`IsValid`、`String`、`Contains`、`Parse`）在有效输入时性能极佳（< 10ns/op）
- `List()` 方法仅需约 0.3ns，因为直接返回预分配的切片
- 直接调用 `MarshalJSON` 比通过 `json.Marshal` 快约 **2.3 倍**
- 直接调用 `UnmarshalJSON` 比通过 `json.Unmarshal` 快约 **1.6 倍**

**性能差异原因**：
- 无效值处理较慢是因为需要格式化错误信息或生成默认字符串
- `Scan/bytes` 比 `Scan/string` 慢是因为需要进行类型转换
- 通过标准库调用比直接调用慢是因为有反射开销

### 基准测试代码位置

基准测试代码位于 `cmd/enumgen/generator/generator_benchmark_test.go`，使用 Ginkgo gmeasure 包实现：

```go
experiment := gmeasure.NewExperiment("EnumGen Benchmark")
AddReportEntry(experiment.Name, experiment)

experiment.SampleDuration("IsValid/valid", func(idx int) {
    for i := 0; i < iterations; i++ {
        _ = gen.GenerateOptionString.IsValid()
    }
}, gmeasure.SamplingConfig{N: samples, Duration: time.Second})
