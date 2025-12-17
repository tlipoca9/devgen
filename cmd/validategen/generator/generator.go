// Package generator provides validation code generation functionality.
package generator

import (
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tlipoca9/devgen/cmd/validategen/rules"
	"github.com/tlipoca9/devgen/genkit"
)

// ToolName is the name of this tool, used in annotations.
const ToolName = "validategen"

// Predefined regex patterns
const (
	regexEmail    = "email"
	regexUUID     = "uuid"
	regexAlpha    = "alpha"
	regexAlphanum = "alphanum"
	regexNumeric  = "numeric"
	regexDNS1123  = "dns1123_label"
)

var regexPatterns = map[string]string{
	regexEmail:    `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
	regexUUID:     `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	regexAlpha:    `^[a-zA-Z]+$`,
	regexAlphanum: `^[a-zA-Z0-9]+$`,
	regexNumeric:  `^[0-9]+$`,
	regexDNS1123:  `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`,
}

var regexVarNames = map[string]string{
	regexEmail:    "_validateRegexEmail",
	regexUUID:     "_validateRegexUUID",
	regexAlpha:    "_validateRegexAlpha",
	regexAlphanum: "_validateRegexAlphanum",
	regexNumeric:  "_validateRegexNumeric",
	regexDNS1123:  "_validateRegexDNS1123Label",
}

// Generator generates Validate() methods for structs.
type Generator struct {
	// pkgIndex maps import path to package for cross-package enum lookup
	pkgIndex map[genkit.GoImportPath]*genkit.Package
}

// New creates a new Generator.
func New() *Generator {
	return &Generator{
		pkgIndex: make(map[genkit.GoImportPath]*genkit.Package),
	}
}

// buildPkgIndex builds the package index from all loaded packages.
func (vg *Generator) buildPkgIndex(gen *genkit.Generator) {
	for _, pkg := range gen.Packages {
		vg.pkgIndex[pkg.GoImportPath()] = pkg
	}
}

// findEnum looks up an enum by import path and type name.
// Returns nil if not found.
func (vg *Generator) findEnum(importPath genkit.GoImportPath, typeName string) *genkit.Enum {
	pkg, ok := vg.pkgIndex[importPath]
	if !ok {
		return nil
	}
	for _, e := range pkg.Enums {
		if e.Name == typeName {
			return e
		}
	}
	return nil
}

// regexTracker tracks custom regex patterns and assigns variable names.
type regexTracker struct {
	patterns map[string]string // pattern -> variable name
	counter  int
}

func newRegexTracker() *regexTracker {
	return &regexTracker{
		patterns: make(map[string]string),
	}
}

// getVarName returns the variable name for a pattern, creating one if needed.
func (rt *regexTracker) getVarName(pattern string) string {
	if varName, ok := rt.patterns[pattern]; ok {
		return varName
	}
	rt.counter++
	varName := fmt.Sprintf("_validateRegex%d", rt.counter)
	rt.patterns[pattern] = varName
	return varName
}

// Name returns the tool name.
func (vg *Generator) Name() string {
	return ToolName
}

// Config returns the tool configuration for VSCode extension integration.
func (vg *Generator) Config() genkit.ToolConfig {
	return genkit.ToolConfig{
		OutputSuffix: "_validate.go",
		Annotations: []genkit.AnnotationConfig{
			{
				Name: "validate",
				Type: "type",
				Doc: `为结构体生成 Validate() 方法。

用法：在结构体定义上方添加注解
  // validategen:@validate
  type User struct {
      // validategen:@required
      // validategen:@min(2)
      // validategen:@max(50)
      Name string

      // validategen:@required
      // validategen:@email
      Email string

      // validategen:@min(0)
      // validategen:@max(150)
      Age int
  }

生成的方法：
  func (x User) Validate() error

自定义后置验证（postValidate 钩子）：
  定义 postValidate 方法添加自定义验证逻辑：
  func (x User) postValidate(errs []string) error {
      if x.Age < 18 && x.Role == "admin" {
          errs = append(errs, "admin must be 18+")
      }
      if len(errs) > 0 {
          return fmt.Errorf("%s", strings.Join(errs, "; "))
      }
      return nil
  }`,
			},
			{
				Name: "required",
				Type: "field",
				Doc: `字段不能为空/零值。

用法：在字段上方添加注解
  // validategen:@required
  Name string

不同类型的验证逻辑：
  string:    不能为空字符串 ""
  slice/map: 长度不能为 0
  pointer:   不能为 nil
  bool:      必须为 true
  numeric:   不能为 0

示例：
  // validategen:@validate
  type Config struct {
      // validategen:@required
      Name string  // Name != ""

      // validategen:@required
      Items []string  // len(Items) > 0

      // validategen:@required
      Metadata map[string]string  // len(Metadata) > 0

      // validategen:@required
      Handler *Handler  // Handler != nil

      // validategen:@required
      Enabled bool  // Enabled == true

      // validategen:@required
      Port int  // Port != 0
  }`,
			},
			{
				Name: "min",
				Type: "field",
				Doc: `最小值（数字）或最小长度（字符串/切片/map）。

用法：在字段上方添加注解
  // validategen:@min(0)
  Age int

  // validategen:@min(2)
  Name string

  // validategen:@min(1)
  Items []int

不同类型的验证逻辑：
  string:    len(field) >= min
  slice/map: len(field) >= min
  numeric:   field >= min
  *string:   非 nil 时，len(*field) >= min
  *numeric:  非 nil 时，*field >= min

示例：
  // validategen:@validate
  type Product struct {
      // validategen:@min(1)
      Name string  // 至少 1 个字符

      // validategen:@min(0)
      Price float64  // 非负价格

      // validategen:@min(1)
      Quantity int  // 至少 1 件

      // validategen:@min(1)
      Tags []string  // 至少 1 个标签
  }`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "max",
				Type: "field",
				Doc: `最大值（数字）或最大长度（字符串/切片/map）。

用法：在字段上方添加注解
  // validategen:@max(150)
  Age int

  // validategen:@max(100)
  Name string

  // validategen:@max(10)
  Items []int

不同类型的验证逻辑：
  string:    len(field) <= max
  slice/map: len(field) <= max
  numeric:   field <= max
  *string:   非 nil 时，len(*field) <= max
  *numeric:  非 nil 时，*field <= max

示例：
  // validategen:@validate
  type Comment struct {
      // validategen:@max(1000)
      Content string  // 最多 1000 个字符

      // validategen:@max(5)
      Rating int  // 评分 1-5

      // validategen:@max(5)
      Tags []string  // 最多 5 个标签
  }`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "len",
				Type: "field",
				Doc: `精确长度（字符串/切片/map）。

用法：在字段上方添加注解
  // validategen:@len(6)
  Code string

  // validategen:@len(2)
  Pair [2]int

示例：
  // validategen:@validate
  type VerificationCode struct {
      // validategen:@len(6)
      Code string  // 必须是 6 个字符

      // validategen:@len(4)
      Digits []int  // 必须有 4 个元素
  }`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "eq",
				Type: "field",
				Doc: `字段值必须等于指定值。

用法：在字段上方添加注解
  // validategen:@eq(v1)
  Version string

  // validategen:@eq(1)
  Status int

  // validategen:@eq(true)
  Active bool

支持类型：string, numeric, bool（及其指针类型）

示例：
  // validategen:@validate
  type APIRequest struct {
      // validategen:@eq(v2)
      Version string  // 必须是 "v2"

      // validategen:@eq(1)
      Type int  // 必须是 1
  }`,
				Params: &genkit.AnnotationParams{Type: []string{"string", "number", "bool"}, Placeholder: "value"},
			},
			{
				Name: "ne",
				Type: "field",
				Doc: `字段值不能等于指定值。

用法：在字段上方添加注解
  // validategen:@ne(deleted)
  Status string

  // validategen:@ne(0)
  Code int

支持类型：string, numeric, bool（及其指针类型）

示例：
  // validategen:@validate
  type User struct {
      // validategen:@ne(banned)
      Status string  // 不能是 "banned"

      // validategen:@ne(123456)
      Password string  // 不能是 "123456"
  }`,
				Params: &genkit.AnnotationParams{Type: []string{"string", "number", "bool"}, Placeholder: "value"},
			},
			{
				Name: "gt",
				Type: "field",
				Doc: `大于（数字）或长度大于（字符串/切片）。

用法：在字段上方添加注解
  // validategen:@gt(0)
  Age int

  // validategen:@gt(0)
  Name string

示例：
  // validategen:@validate
  type Order struct {
      // validategen:@gt(0)
      Amount float64  // 必须为正数

      // validategen:@gt(0)
      Quantity int  // 必须大于 0
  }`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "gte",
				Type: "field",
				Doc: `大于等于（数字）或长度大于等于（字符串/切片）。

用法：在字段上方添加注解
  // validategen:@gte(18)
  Age int

示例：
  // validategen:@validate
  type Adult struct {
      // validategen:@gte(18)
      Age int  // 必须 >= 18 岁
  }`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "lt",
				Type: "field",
				Doc: `小于（数字）或长度小于（字符串/切片）。

用法：在字段上方添加注解
  // validategen:@lt(100)
  Discount float64

示例：
  // validategen:@validate
  type Discount struct {
      // validategen:@lt(100)
      Percent float64  // 必须小于 100%
  }`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "lte",
				Type: "field",
				Doc: `小于等于（数字）或长度小于等于（字符串/切片）。

用法：在字段上方添加注解
  // validategen:@lte(5)
  Rating int

示例：
  // validategen:@validate
  type Review struct {
      // validategen:@lte(5)
      Rating int  // 评分必须 <= 5
  }`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "oneof",
				Type: "field",
				Doc: `字段值必须是指定值之一（空格分隔）。

用法：在字段上方添加注解
  // validategen:@oneof(pending active completed)
  Status string

  // validategen:@oneof(1 2 3)
  Level int

支持类型：string, numeric

示例：
  // validategen:@validate
  type Task struct {
      // validategen:@oneof(todo doing done)
      Status string

      // validategen:@oneof(1 2 3 4 5)
      Priority int
  }`,
				Params: &genkit.AnnotationParams{Type: "list", Placeholder: "value1 value2 ..."},
			},
			{
				Name: "oneof_enum",
				Type: "field",
				Doc: `字段值必须是有效的枚举值（使用 EnumTypeEnums.Contains）。

用法：在字段上方添加注解
  // validategen:@oneof_enum(OrderStatus)
  Status OrderStatus

跨包枚举（自动添加 import）：
  // validategen:@oneof_enum(github.com/you/pkg/common.Status)
  Status common.Status

跨包枚举（指定 import alias）：
  // validategen:@oneof_enum(mytypes:github.com/you/pkg/types.Status)
  Status mytypes.Status
  // 生成代码：import mytypes "github.com/you/pkg/types"
  //          if !mytypes.StatusEnums.Contains(x.Status) { ... }

要求：枚举类型必须有 enumgen:@enum 注解。

示例：
  // 同一个包内：
  // enumgen:@enum(string)
  type Status int
  const (StatusActive Status = iota + 1; StatusInactive)

  // validategen:@validate
  type User struct {
      // validategen:@oneof_enum(Status)
      Status Status
  }
  // 生成代码：if !StatusEnums.Contains(x.Status) { ... }`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "[alias:]import/path.EnumType"},
			},
			{
				Name: "email",
				Type: "field",
				Doc: `字段必须是有效的邮箱地址格式。

用法：在字段上方添加注解
  // validategen:@email
  Email string

正则：^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$

示例：
  // validategen:@validate
  type Contact struct {
      // validategen:@email
      Email string  // "user@example.com" ✓, "invalid" ✗
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "url",
				Type: "field",
				Doc: `字段必须是有效的 URL（使用 net/url.ParseRequestURI）。

用法：在字段上方添加注解
  // validategen:@url
  Website string

示例：
  // validategen:@validate
  type Company struct {
      // validategen:@url
      Website string  // "https://example.com" ✓
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "uuid",
				Type: "field",
				Doc: `字段必须是有效的 UUID 格式。

用法：在字段上方添加注解
  // validategen:@uuid
  ID string

正则：^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$

示例：
  // validategen:@validate
  type Entity struct {
      // validategen:@uuid
      ID string  // "550e8400-e29b-41d4-a716-446655440000" ✓
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "ip",
				Type: "field",
				Doc: `字段必须是有效的 IP 地址（IPv4 或 IPv6）。

用法：在字段上方添加注解
  // validategen:@ip
  Address string

示例：
  // validategen:@validate
  type Server struct {
      // validategen:@ip
      IP string  // "192.168.1.1" ✓, "::1" ✓
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "ipv4",
				Type: "field",
				Doc: `字段必须是有效的 IPv4 地址。

用法：在字段上方添加注解
  // validategen:@ipv4
  Address string

示例：
  // validategen:@validate
  type Server struct {
      // validategen:@ipv4
      IPv4 string  // "192.168.1.1" ✓, "::1" ✗
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "ipv6",
				Type: "field",
				Doc: `字段必须是有效的 IPv6 地址。

用法：在字段上方添加注解
  // validategen:@ipv6
  Address string

示例：
  // validategen:@validate
  type Server struct {
      // validategen:@ipv6
      IPv6 string  // "::1" ✓, "192.168.1.1" ✗
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "dns1123_label",
				Type: "field",
				Doc: `字段必须符合 DNS label 标准（RFC 1123 兼容的 DNS 名称）。

用法：在字段上方添加注解
  // validategen:@dns1123_label
  Hostname string

DNS label 规则：
  - 只能包含小写字母、数字和连字符
  - 必须以字母数字开头
  - 必须以字母数字结尾
  - 可以用点号分隔标签
  - 总长度 1-253 个字符
  - 每个标签最多 63 个字符

示例：
  // validategen:@validate
  type KubernetesObject struct {
      // validategen:@dns1123
      Pod string  // "my-pod-123" ✓, "Pod" ✗, "-invalid" ✗
      
      // validategen:@dns1123
      Service string  // "api-service" ✓
      
      // validategen:@dns1123
      Namespace string  // "default" ✓, "kube-system" ✓
  }

应用场景：
  - Kubernetes 对象命名（Pod、Service、Namespace）
  - DNS 主机名验证
  - 微服务实例命名
  - 容器镜像仓库域名验证

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "duration",
				Type: "field",
				Doc: `字段必须是有效的 Go duration 字符串（time.ParseDuration）。

用法：在字段上方添加注解
  // validategen:@duration
  Timeout string

有效格式：1h, 30m, 500ms, 1h30m, 2h45m30s 等

示例：
  // validategen:@validate
  type Config struct {
      // validategen:@duration
      Timeout string  // "30s" ✓, "invalid" ✗

      // validategen:@duration
      RetryDelay string  // "500ms" ✓
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "duration_min",
				Type: "field",
				Doc: `最小时间间隔（字段必须是有效的 duration 字符串）。

用法：在字段上方添加注解
  // validategen:@duration_min(1s)
  Timeout string

示例：
  // validategen:@validate
  type Config struct {
      // validategen:@duration_min(100ms)
      Timeout string  // 至少 100ms

      // validategen:@duration_min(1h)
      TTL string  // 至少 1 小时
  }

注意：同时会验证字符串是否为有效的 duration 格式。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "1s, 5m, 1h, etc."},
			},
			{
				Name: "duration_max",
				Type: "field",
				Doc: `最大时间间隔（字段必须是有效的 duration 字符串）。

用法：在字段上方添加注解
  // validategen:@duration_max(1h)
  Timeout string

示例：
  // validategen:@validate
  type Config struct {
      // validategen:@duration_max(30s)
      Timeout string  // 最多 30 秒

      // validategen:@duration_max(24h)
      TTL string  // 最多 24 小时
  }

注意：同时会验证字符串是否为有效的 duration 格式。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "1s, 5m, 1h, etc."},
			},
			{
				Name: "alpha",
				Type: "field",
				Doc: `字段只能包含 ASCII 字母（a-zA-Z）。

用法：在字段上方添加注解
  // validategen:@alpha
  Name string

正则：^[a-zA-Z]+$

示例：
  // validategen:@validate
  type Person struct {
      // validategen:@alpha
      FirstName string  // "John" ✓, "John123" ✗
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "alphanum",
				Type: "field",
				Doc: `字段只能包含 ASCII 字母和数字（a-zA-Z0-9）。

用法：在字段上方添加注解
  // validategen:@alphanum
  Code string

正则：^[a-zA-Z0-9]+$

示例：
  // validategen:@validate
  type Product struct {
      // validategen:@alphanum
      SKU string  // "ABC123" ✓, "ABC-123" ✗
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "numeric",
				Type: "field",
				Doc: `字段只能包含数字字符（0-9）。

用法：在字段上方添加注解
  // validategen:@numeric
  Phone string

正则：^[0-9]+$

示例：
  // validategen:@validate
  type Contact struct {
      // validategen:@numeric
      Phone string  // "1234567890" ✓, "123-456" ✗
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
			},
			{
				Name: "contains",
				Type: "field",
				Doc: `字段必须包含指定的子串。

用法：在字段上方添加注解
  // validategen:@contains(@)
  Email string

示例：
  // validategen:@validate
  type User struct {
      // validategen:@contains(@)
      Email string  // 必须包含 "@"

      // validategen:@contains(://)
      URL string  // 必须包含 "://"
  }`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "substring"},
			},
			{
				Name: "excludes",
				Type: "field",
				Doc: `字段不能包含指定的子串。

用法：在字段上方添加注解
  // validategen:@excludes(admin)
  Name string

示例：
  // validategen:@validate
  type User struct {
      // validategen:@excludes(admin)
      Username string  // 不能包含 "admin"

      // validategen:@excludes(password)
      Password string  // 不能包含 "password"
  }`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "substring"},
			},
			{
				Name: "startswith",
				Type: "field",
				Doc: `字段必须以指定前缀开头。

用法：在字段上方添加注解
  // validategen:@startswith(usr_)
  ID string

示例：
  // validategen:@validate
  type Entity struct {
      // validategen:@startswith(usr_)
      UserID string  // "usr_123" ✓

      // validategen:@startswith(prod_)
      ProductID string  // "prod_abc" ✓
  }`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "prefix"},
			},
			{
				Name: "endswith",
				Type: "field",
				Doc: `字段必须以指定后缀结尾。

用法：在字段上方添加注解
  // validategen:@endswith(.json)
  Filename string

示例：
  // validategen:@validate
  type Config struct {
      // validategen:@endswith(.yaml)
      ConfigFile string  // "config.yaml" ✓

      // validategen:@endswith(.png)
      ImageFile string  // "logo.png" ✓
  }`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "suffix"},
			},
			{
				Name: "method",
				Type: "field",
				Doc: `调用字段的方法进行嵌套验证。

用法：在字段上方添加注解
  // validategen:@method(Validate)
  Address Address

  // validategen:@method(Validate)
  Items []Item  // 验证每个元素

  // validategen:@method(Validate)
  Users map[int]User  // 验证每个值

方法签名：func() error

行为：
  - 指针字段：非 nil 时才调用
  - 切片：对每个元素调用，错误信息包含索引
  - map：对每个值调用，错误信息包含键

示例：
  // validategen:@validate
  type Order struct {
      // validategen:@method(Validate)
      Customer Customer

      // validategen:@method(Validate)
      Items []OrderItem

      // validategen:@method(Validate)
      Discounts map[string]Discount
  }

  // Items 生成的代码：
  for i, v := range x.Items {
      if err := v.Validate(); err != nil {
          errs = append(errs, fmt.Sprintf("Items[%d]: %v", i, err))
      }
  }`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "MethodName"},
				LSP: &genkit.LSPConfig{
					Enabled:     true,
					Provider:    "gopls",
					Feature:     "method",
					Signature:   "func() error",
					ResolveFrom: "fieldType",
				},
			},
			{
				Name: "regex",
				Type: "field",
				Doc: `字段必须匹配指定的正则表达式。

用法：在字段上方添加注解
  // validategen:@regex(^\+?[0-9]{10,14}$)
  Phone string

示例：
  // validategen:@validate
  type Contact struct {
      // 国际电话号码格式
      // validategen:@regex(^\+?[0-9]{10,14}$)
      Phone string

      // Slug 格式（小写字母、数字、连字符）
      // validategen:@regex(^[a-z0-9]+(-[a-z0-9]+)*$)
      Slug string

      // 版本号格式（semver）
      // validategen:@regex(^v?[0-9]+\.[0-9]+\.[0-9]+$)
      Version string
  }

注意：正则表达式在包初始化时编译一次，性能优化。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "pattern"},
			},
			{
				Name: "format",
				Type: "field",
				Doc: `字段必须是有效的指定格式（json, yaml, toml, csv）。

用法：在字段上方添加注解
  // validategen:@format(json)
  Config string

支持的格式：
  - json: 使用 encoding/json.Valid 验证
  - yaml: 使用 gopkg.in/yaml.v3.Unmarshal 验证
  - toml: 使用 github.com/BurntSushi/toml.Unmarshal 验证
  - csv:  使用 encoding/csv.Reader.ReadAll 验证

示例：
  // validategen:@validate
  type Template struct {
      // validategen:@format(json)
      JSONConfig string  // '{"key":"value"}' ✓

      // validategen:@format(yaml)
      YAMLConfig string  // 'key: value' ✓

      // validategen:@format(csv)
      CSVData string  // 'a,b,c\n1,2,3' ✓
  }

注意：空字符串会跳过验证。如果字段必填，请配合 @required 使用。`,
				Params: &genkit.AnnotationParams{
					Values:  []string{"json", "yaml", "toml", "csv"},
					MaxArgs: 1,
					Docs: map[string]string{
						"json": "使用 encoding/json.Valid 验证 JSON 格式",
						"yaml": "使用 gopkg.in/yaml.v3 验证 YAML 格式",
						"toml": "使用 github.com/BurntSushi/toml 验证 TOML 格式",
						"csv":  "使用 encoding/csv 验证 CSV 格式",
					},
				},
			},
			{
				Name: "default",
				Type: "field",
				Doc: `为字段设置默认值（在验证前应用）。

用法：在字段上方添加注解
  // validategen:@default(unknown)
  Name string

  // validategen:@default(0)
  Count int

  // validategen:@default(true)
  Enabled bool

支持类型：
  - string: 设置默认字符串值
  - numeric (int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64): 设置默认数值
  - bool: 设置默认布尔值 (true/false)

生成逻辑：
  在 Validate() 方法开头，检查字段是否为零值，如果是则设置默认值。

示例：
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

注意：
  - 默认值在所有验证规则之前应用
  - 对于指针类型，只有当指针为 nil 时才会设置默认值
  - 字符串值不需要引号包裹`,
				Params: &genkit.AnnotationParams{Type: []string{"string", "number", "bool"}, Placeholder: "value"},
			},
			{
				Name: "cpu",
				Type: "field",
				Doc: `验证 Kubernetes CPU 资源数量格式（有效单位：m, 核数）。

用法：在字段上方添加注解
  // validategen:@cpu
  CPURequest string

支持的格式：
  - 毫核（m）: "500m", "100m"
  - 核数: "1", "2", "0.5"
  - 科学记数法: "1e3m" (1000m)

验证逻辑：
  使用 k8s.io/apimachinery/pkg/api/resource 的 ParseQuantity 函数解析。
  检查数量是否为有效的格式和非负数。

示例：
  // validategen:@validate
  type PodSpec struct {
      // validategen:@cpu
      CPURequest string  // "500m" ✓
  }

生成的错误消息：
  - 格式错误: "CPURequest invalid quantity: ..."
  - 负数: "CPURequest must be non-negative"`,
			},
			{
				Name: "memory",
				Type: "field",
				Doc: `验证 Kubernetes 内存资源数量格式（有效单位：Ki, Mi, Gi, Ti, Pi, Ei）。

用法：在字段上方添加注解
  // validategen:@memory
  MemoryRequest string

支持的格式：
  - 二进制单位: "512Mi", "2Gi", "1Ti"
  - 十进制单位: "512M", "2G" (等同于 Mi, Gi)
  - 字节: "1000000", "1Gi"

验证逻辑：
  使用 k8s.io/apimachinery/pkg/api/resource 的 ParseQuantity 函数解析。
  检查数量是否为有效的格式和非负数。

示例：
  // validategen:@validate
  type ContainerSpec struct {
      // validategen:@memory
      MemoryRequest string  // "512Mi" ✓
      
      // validategen:@memory
      MemoryLimit string  // "2Gi" ✓
  }

生成的错误消息：
  - 格式错误: "MemoryRequest invalid quantity: ..."
  - 负数: "MemoryRequest must be non-negative"`,
			},
		},
	}
}

// Rules implements genkit.RuleTool.
// Returns AI-friendly documentation for validategen.
func (vg *Generator) Rules() []genkit.Rule {
	return []genkit.Rule{
		{
			Name:        "devgen-tool-validategen",
			Description: "Go 结构体验证代码生成工具 validategen 的使用指南。当用户需要为结构体生成 Validate() 方法、添加字段验证规则时使用此规则。",
			Globs:       []string{"*.go"},
			AlwaysApply: false,
			Content:     rules.ValidategenRule,
		},
	}
}

// Run processes all packages and generates validation methods.
func (vg *Generator) Run(gen *genkit.Generator, log *genkit.Logger) error {
	// Build package index for cross-package enum lookup
	vg.buildPkgIndex(gen)

	var totalCount int
	for _, pkg := range gen.Packages {
		types := vg.FindTypes(pkg)
		if len(types) == 0 {
			continue
		}
		log.Find("Found %v type(s) with validation in %v", len(types), pkg.GoImportPath())
		for _, t := range types {
			log.Item("%v", t.Name)
		}
		totalCount += len(types)
		if err := vg.ProcessPackage(gen, pkg); err != nil {
			return fmt.Errorf("process %s: %w", pkg.Name, err)
		}
	}

	if totalCount == 0 {
		return nil
	}

	return nil
}

// ProcessPackage processes a package and generates validation methods.
func (vg *Generator) ProcessPackage(gen *genkit.Generator, pkg *genkit.Package) error {
	types := vg.FindTypes(pkg)
	if len(types) == 0 {
		return nil
	}

	outPath := genkit.OutputPath(pkg.Dir, pkg.Name+"_validate.go")
	g := gen.NewGeneratedFile(outPath, pkg.GoImportPath())

	// Track which regex patterns are used
	usedRegex := make(map[string]bool)
	// Track custom regex patterns
	customRegex := newRegexTracker()

	// First pass: collect all used regex patterns
	for _, typ := range types {
		for _, field := range typ.Fields {
			rules := vg.parseFieldAnnotations(field)
			for _, rule := range rules {
				switch rule.Name {
				case "email":
					usedRegex[regexEmail] = true
				case "uuid":
					usedRegex[regexUUID] = true
				case "alpha":
					usedRegex[regexAlpha] = true
				case "alphanum":
					usedRegex[regexAlphanum] = true
				case "numeric":
					usedRegex[regexNumeric] = true
				case "dns1123_label":
					usedRegex[regexDNS1123] = true
				case "regex":
					if rule.Param != "" {
						customRegex.getVarName(rule.Param)
					}
				}
			}
		}
	}

	vg.WriteHeader(g, pkg.Name, usedRegex, customRegex)
	for _, typ := range types {
		if err := vg.GenerateValidate(g, typ, customRegex); err != nil {
			return err
		}
	}

	// Generate test file if requested
	if gen.IncludeTests() {
		testPath := genkit.OutputPath(pkg.Dir, pkg.Name+"_validate_test.go")
		tg := gen.NewGeneratedFile(testPath, pkg.GoImportPath())
		vg.WriteTestHeader(tg, pkg.Name)
		for _, typ := range types {
			vg.GenerateValidateTest(tg, typ)
			vg.GenerateSetDefaultsTest(tg, typ)
		}
	}

	return nil
}

// FindTypes finds all types with validategen:@validate annotation.
func (vg *Generator) FindTypes(pkg *genkit.Package) []*genkit.Type {
	var types []*genkit.Type
	for _, t := range pkg.Types {
		if genkit.HasAnnotation(t.Doc, ToolName, "validate") {
			types = append(types, t)
		}
	}
	return types
}

// WriteHeader writes the file header and global regex variables.
func (vg *Generator) WriteHeader(
	g *genkit.GeneratedFile,
	pkgName string,
	usedRegex map[string]bool,
	customRegex *regexTracker,
) {
	g.P("// Code generated by ", ToolName, ". DO NOT EDIT.")
	g.P()
	g.P("package ", pkgName)

	// Generate global regex variables if any are used
	hasBuiltin := len(usedRegex) > 0
	hasCustom := len(customRegex.patterns) > 0
	if hasBuiltin || hasCustom {
		g.P()
		g.P("// Precompiled regex patterns for validation.")
		g.P("var (")
		// Built-in patterns (sorted for deterministic output)
		var builtinNames []string
		for name := range usedRegex {
			builtinNames = append(builtinNames, name)
		}
		sort.Strings(builtinNames)
		for _, name := range builtinNames {
			varName := regexVarNames[name]
			pattern := regexPatterns[name]
			g.P(varName, " = ", genkit.GoImportPath("regexp").Ident("MustCompile"), "(`", pattern, "`)")
		}
		// Custom patterns (sorted for deterministic output)
		var customPatterns []string
		for pattern := range customRegex.patterns {
			customPatterns = append(customPatterns, pattern)
		}
		sort.Strings(customPatterns)
		for _, pattern := range customPatterns {
			varName := customRegex.patterns[pattern]
			g.P(varName, " = ", genkit.GoImportPath("regexp").Ident("MustCompile"), "(", genkit.RawString(pattern), ")")
		}
		g.P(")")
	}
}

// GenerateValidate generates Validate method for a single type.
// It always generates:
//   - _validate(): field-level validations (excluding @method)
//   - _validateMethod(): @method validations only (if any)
//   - Validate(): calls _validate() + _validateMethod() + postValidate() (if exists)
//
// This separation allows testing _validate() independently without needing to
// set up valid nested types for @method validation.
func (vg *Generator) GenerateValidate(g *genkit.GeneratedFile, typ *genkit.Type, customRegex *regexTracker) error {
	typeName := typ.Name
	pkg := typ.Pkg

	// Collect fields with validation annotations
	var validatedFields []*fieldValidation
	for _, field := range typ.Fields {
		rules := vg.parseFieldAnnotations(field)
		if len(rules) > 0 {
			validatedFields = append(validatedFields, &fieldValidation{
				Field: field,
				Rules: rules,
			})
		}
	}

	if len(validatedFields) == 0 {
		return nil
	}

	// Separate fields: those with @method and those without
	var nonMethodFields []*fieldValidation
	var methodFields []*fieldValidation
	for _, fv := range validatedFields {
		hasMethod := false
		var nonMethodRules []*validateRule
		var methodRules []*validateRule
		for _, rule := range fv.Rules {
			if rule.Name == "method" {
				hasMethod = true
				methodRules = append(methodRules, rule)
			} else {
				nonMethodRules = append(nonMethodRules, rule)
			}
		}
		if len(nonMethodRules) > 0 {
			nonMethodFields = append(nonMethodFields, &fieldValidation{
				Field: fv.Field,
				Rules: nonMethodRules,
			})
		}
		if hasMethod {
			methodFields = append(methodFields, &fieldValidation{
				Field: fv.Field,
				Rules: methodRules,
			})
		}
	}

	hasMethodValidation := len(methodFields) > 0
	hasPostValidate := vg.hasPostValidateMethod(typ)

	// Collect fields with @default annotation
	var defaultFields []*fieldValidation
	for _, fv := range nonMethodFields {
		for _, rule := range fv.Rules {
			if rule.Name == "default" {
				defaultFields = append(defaultFields, &fieldValidation{
					Field: fv.Field,
					Rules: []*validateRule{rule},
				})
				break
			}
		}
	}

	// Generate SetDefaults() method if there are @default annotations
	if len(defaultFields) > 0 {
		g.P()
		g.P("// SetDefaults sets default values for zero-value fields.")
		g.P("func (x *", typeName, ") SetDefaults() {")
		for _, fv := range defaultFields {
			vg.generateSetDefault(g, fv)
		}
		g.P("}")
	}

	// Generate _validate() method - field-level validations (excluding @method)
	g.P()
	g.P("// _validate performs field-level validation for ", typeName, ".")
	g.P("// This method excludes @method validations for easier testing.")
	g.P("func (x ", typeName, ") _validate() []string {")
	g.P("var errs []string")
	if len(nonMethodFields) > 0 {
		g.P()
		for _, fv := range nonMethodFields {
			vg.generateFieldValidation(g, fv, customRegex, pkg)
		}
	}
	g.P()
	g.P("return errs")
	g.P("}")

	// Generate _validateMethod() if there are @method validations
	if hasMethodValidation {
		g.P()
		g.P("// _validateMethod performs nested validation via @method annotations.")
		g.P("func (x ", typeName, ") _validateMethod() []string {")
		g.P("var errs []string")
		g.P()
		for _, fv := range methodFields {
			vg.generateFieldValidation(g, fv, customRegex, pkg)
		}
		g.P()
		g.P("return errs")
		g.P("}")
	}

	// Generate Validate() method
	g.P()
	g.P(genkit.GoMethod{
		Doc:     genkit.GoDoc("Validate validates the " + typeName + " fields."),
		Recv:    genkit.GoReceiver{Name: "x", Type: typeName},
		Name:    "Validate",
		Results: genkit.GoResults{{Type: "error"}},
	}, " {")
	g.P("errs := x._validate()")

	if hasMethodValidation {
		g.P("errs = append(errs, x._validateMethod()...)")
	}

	if hasPostValidate {
		g.P("return x.postValidate(errs)")
	} else {
		g.P("if len(errs) > 0 {")
		g.P(
			"return ",
			genkit.GoImportPath("fmt").Ident("Errorf"),
			"(\"%s\", ",
			genkit.GoImportPath("strings").Ident("Join"),
			"(errs, \"; \"))",
		)
		g.P("}")
		g.P("return nil")
	}
	g.P("}")

	return nil
}

// hasPostValidateMethod checks if the type has a postValidate(errs []string) error method.
func (vg *Generator) hasPostValidateMethod(typ *genkit.Type) bool {
	if typ.Pkg == nil || typ.Pkg.TypesPkg == nil {
		return false
	}

	// Look up the type in the package scope
	obj := typ.Pkg.TypesPkg.Scope().Lookup(typ.Name)
	if obj == nil {
		return false
	}

	// Get the named type
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}

	// Check methods on the type (including pointer receiver)
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if method.Name() == "postValidate" {
			// Verify signature: func(errs []string) error
			sig, ok := method.Type().(*types.Signature)
			if !ok {
				continue
			}
			// One parameter of type []string
			if sig.Params().Len() != 1 {
				continue
			}
			param := sig.Params().At(0)
			slice, ok := param.Type().(*types.Slice)
			if !ok {
				continue
			}
			if basic, ok := slice.Elem().(*types.Basic); !ok || basic.Kind() != types.String {
				continue
			}
			// One result of type error
			if sig.Results().Len() != 1 {
				continue
			}
			if sig.Results().At(0).Type().String() == "error" {
				return true
			}
		}
	}

	return false
}

// hasMethodOnFieldType checks if a method exists on the field's type.
// It handles qualified types (e.g., "common.NetworkConfiguration"), pointers, slices, and maps.
func (vg *Generator) hasMethodOnFieldType(pkg *genkit.Package, fieldType, methodName string) bool {
	if pkg == nil || pkg.TypesInfo == nil {
		return false
	}

	// Strip slice prefix
	baseType := strings.TrimPrefix(fieldType, "[]")
	// Strip map prefix (extract value type)
	if strings.HasPrefix(baseType, "map[") {
		// Find the value type after ]
		idx := strings.Index(baseType, "]")
		if idx != -1 && idx+1 < len(baseType) {
			baseType = baseType[idx+1:]
		}
	}
	// Strip pointer prefix
	baseType = strings.TrimPrefix(baseType, "*")

	// Handle qualified types (e.g., "common.NetworkConfiguration")
	var typeName string
	var lookupPkg *types.Package
	if strings.Contains(baseType, ".") {
		parts := strings.SplitN(baseType, ".", 2)
		pkgAlias := parts[0]
		typeName = parts[1]
		// Find the imported package by alias
		lookupPkg = vg.findImportedPackage(pkg, pkgAlias)
		if lookupPkg == nil {
			return false
		}
	} else {
		typeName = baseType
		lookupPkg = pkg.TypesPkg
	}

	if lookupPkg == nil {
		return false
	}

	// Look up the type in the package scope
	obj := lookupPkg.Scope().Lookup(typeName)
	if obj == nil {
		return false
	}

	// Get the named type
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}

	// Check methods on the type (value receiver)
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if method.Name() == methodName {
			return true
		}
	}

	// Also check methods on pointer receiver
	ptrType := types.NewPointer(named)
	methodSet := types.NewMethodSet(ptrType)
	for i := 0; i < methodSet.Len(); i++ {
		sel := methodSet.At(i)
		if sel.Obj().Name() == methodName {
			return true
		}
	}

	return false
}

// findImportedPackage finds an imported package by its alias name.
func (vg *Generator) findImportedPackage(pkg *genkit.Package, alias string) *types.Package {
	if pkg.TypesPkg == nil {
		return nil
	}

	// Check all imports
	for _, imp := range pkg.TypesPkg.Imports() {
		// Check if the import name matches the alias
		if imp.Name() == alias {
			return imp
		}
		// Also check the last part of the path (default import name)
		path := imp.Path()
		parts := strings.Split(path, "/")
		if len(parts) > 0 && parts[len(parts)-1] == alias {
			return imp
		}
	}

	return nil
}

// parseFieldAnnotations parses validation annotations from field doc/comment.
// Supported annotations:
//   - validategen:@default(v) - set default value (applied before validation)
//   - validategen:@required
//   - validategen:@min(n)
//   - validategen:@max(n)
//   - validategen:@len(n)
//   - validategen:@gt(n)
//   - validategen:@gte(n)
//   - validategen:@lt(n)
//   - validategen:@lte(n)
//   - validategen:@eq(v)
//   - validategen:@ne(v)
//   - validategen:@oneof(a, b, c)
//   - validategen:@email
//   - validategen:@url
//   - validategen:@uuid
//   - validategen:@ip
//   - validategen:@ipv4
//   - validategen:@ipv6
//   - validategen:@duration
//   - validategen:@alpha
//   - validategen:@alphanum
//   - validategen:@numeric
//   - validategen:@contains(s)
//   - validategen:@excludes(s)
//   - validategen:@startswith(s)
//   - validategen:@endswith(s)
//   - validategen:@regex(pattern)
func (vg *Generator) parseFieldAnnotations(field *genkit.Field) []*validateRule {
	var rules []*validateRule

	// Parse from both Doc and Comment
	doc := field.Doc + "\n" + field.Comment
	annotations := genkit.ParseAnnotations(doc)

	for _, ann := range annotations {
		if ann.Tool != ToolName {
			continue
		}

		rule := &validateRule{Name: ann.Name}

		// Get parameter from Flags (positional args)
		if len(ann.Flags) > 0 {
			rule.Param = strings.Join(ann.Flags, " ")
		}

		rules = append(rules, rule)
	}

	return rules
}

type fieldValidation struct {
	Field *genkit.Field
	Rules []*validateRule
}

type validateRule struct {
	Name  string
	Param string
}

// rulePriority defines the execution order for validation rules.
// Lower numbers execute first.
var rulePriority = map[string]int{
	// 1. Required check - must come first
	"required": 10,

	// 2. Range/length checks
	"min": 20,
	"max": 21,
	"len": 22,
	"gt":  23,
	"gte": 24,
	"lt":  25,
	"lte": 26,

	// 3. Equality checks
	"eq":         30,
	"ne":         31,
	"oneof":      32,
	"oneof_enum": 33,

	// 4. Format checks
	"email":         40,
	"url":           41,
	"uuid":          42,
	"ip":            43,
	"ipv4":          44,
	"ipv6":          45,
	"dns1123_label": 46,
	"duration_min":  47,
	"duration_max":  48,
	"alpha":         49,
	"alphanum":      50,
	"numeric":       51,
	"regex":         52,
	"format":        53,
	"cpu":           54,
	"memory":        55,

	// 5. String content checks
	"contains":   60,
	"excludes":   61,
	"startswith": 62,
	"endswith":   63,

	// 6. Nested validation - should come last
	"method": 70,
}

func (vg *Generator) generateFieldValidation(
	g *genkit.GeneratedFile,
	fv *fieldValidation,
	customRegex *regexTracker,
	pkg *genkit.Package,
) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type

	// Sort rules by priority for deterministic output
	rules := make([]*validateRule, len(fv.Rules))
	copy(rules, fv.Rules)
	sort.SliceStable(rules, func(i, j int) bool {
		pi := rulePriority[rules[i].Name]
		pj := rulePriority[rules[j].Name]
		if pi != pj {
			return pi < pj
		}
		// Same priority: maintain original order (stable sort)
		return false
	})

	// Collect duration-related rules to generate them together
	var hasDuration, hasDurationMin, hasDurationMax bool
	var durationMinParam, durationMaxParam string
	for _, rule := range rules {
		switch rule.Name {
		case "duration":
			hasDuration = true
		case "duration_min":
			hasDurationMin = true
			durationMinParam = rule.Param
		case "duration_max":
			hasDurationMax = true
			durationMaxParam = rule.Param
		}
	}

	// Track if duration block has been generated
	durationGenerated := false

	for _, rule := range rules {
		switch rule.Name {
		case "required":
			vg.genRequired(g, fieldName, fieldType)
		case "min":
			vg.genMin(g, fieldName, fieldType, rule.Param)
		case "max":
			vg.genMax(g, fieldName, fieldType, rule.Param)
		case "len":
			vg.genLen(g, fieldName, fieldType, rule.Param)
		case "eq":
			vg.genEq(g, fieldName, fieldType, rule.Param)
		case "ne":
			vg.genNe(g, fieldName, fieldType, rule.Param)
		case "gt":
			vg.genGt(g, fieldName, fieldType, rule.Param)
		case "gte":
			vg.genGte(g, fieldName, fieldType, rule.Param)
		case "lt":
			vg.genLt(g, fieldName, fieldType, rule.Param)
		case "lte":
			vg.genLte(g, fieldName, fieldType, rule.Param)
		case "oneof":
			vg.genOneof(g, fv.Field, rule.Param)
		case "oneof_enum":
			vg.genOneofEnum(g, fv.Field, rule.Param, pkg)
		case "email":
			vg.genEmail(g, fieldName)
		case "url":
			vg.genURL(g, fieldName)
		case "uuid":
			vg.genUUID(g, fieldName)
		case "alpha":
			vg.genAlpha(g, fieldName)
		case "alphanum":
			vg.genAlphanum(g, fieldName)
		case "numeric":
			vg.genNumeric(g, fieldName)
		case "contains":
			vg.genContains(g, fieldName, rule.Param)
		case "excludes":
			vg.genExcludes(g, fieldName, rule.Param)
		case "startswith":
			vg.genStartsWith(g, fieldName, rule.Param)
		case "endswith":
			vg.genEndsWith(g, fieldName, rule.Param)
		case "ip":
			vg.genIP(g, fieldName)
		case "ipv4":
			vg.genIPv4(g, fieldName)
		case "ipv6":
			vg.genIPv6(g, fieldName)
		case "dns1123_label":
			vg.genDNS1123(g, fieldName)
		case "duration", "duration_min", "duration_max":
			// Generate all duration validations together, only once
			if !durationGenerated {
				vg.genDurationCombined(
					g,
					fieldName,
					hasDuration,
					hasDurationMin,
					durationMinParam,
					hasDurationMax,
					durationMaxParam,
				)
				durationGenerated = true
			}
		case "method":
			vg.genMethod(g, fv.Field, rule.Param)
		case "regex":
			vg.genRegex(g, fv.Field, rule.Param, customRegex)
		case "format":
			vg.genFormat(g, fv.Field, rule.Param)
		case "cpu":
			vg.genCPU(g, fieldName)
		case "memory":
			vg.genMemory(g, fieldName)
		}
	}
}

func (vg *Generator) generateSetDefault(g *genkit.GeneratedFile, fv *fieldValidation) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type
	param := fv.Rules[0].Param

	if param == "" {
		return
	}

	if isStringType(fieldType) {
		g.P("if x.", fieldName, " == \"\" {")
		g.P("x.", fieldName, " = \"", param, "\"")
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " == nil {")
		g.P("_default", fieldName, " := \"", param, "\"")
		g.P("x.", fieldName, " = &_default", fieldName)
		g.P("}")
	} else if isBoolType(fieldType) {
		// Parse bool value
		boolVal := param == "true" || param == "1"
		if boolVal {
			g.P("if !x.", fieldName, " {")
			g.P("x.", fieldName, " = true")
			g.P("}")
		}
		// For false default, no action needed since zero value is already false
	} else if isPointerToBoolType(fieldType) {
		boolVal := param == "true" || param == "1"
		g.P("if x.", fieldName, " == nil {")
		g.P("_default", fieldName, " := ", boolVal)
		g.P("x.", fieldName, " = &_default", fieldName)
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " == 0 {")
		g.P("x.", fieldName, " = ", param)
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " == nil {")
		// Extract base type from pointer type (e.g., "*int" -> "int")
		baseType := strings.TrimPrefix(fieldType, "*")
		g.P("_default", fieldName, " := ", baseType, "(", param, ")")
		g.P("x.", fieldName, " = &_default", fieldName)
		g.P("}")
	}
}

func (vg *Generator) genRequired(g *genkit.GeneratedFile, fieldName, fieldType string) {
	if isStringType(fieldType) {
		g.P("if x.", fieldName, " == \"\" {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") == 0 {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if isPointerType(fieldType) {
		g.P("if x.", fieldName, " == nil {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if isBoolType(fieldType) {
		// For bool, required means must be true
		g.P("if !x.", fieldName, " {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	} else if isNumericType(fieldType) {
		// For numeric types, check zero value
		g.P("if x.", fieldName, " == 0 {")
		g.P("errs = append(errs, \"", fieldName, " is required\")")
		g.P("}")
	}
	// Other types (struct, interface, etc.) are not supported for required check
}

func (vg *Generator) genMin(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") < ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") < ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have at least ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genMax(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") > ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") > ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have at most ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genLen(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") != ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be exactly ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") != ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have exactly ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	}
}

func (vg *Generator) genEq(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if x.", fieldName, " != \"", param, "\" {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must equal ",
			param,
			", got %q\", x.",
			fieldName,
			"))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " != \"", param, "\" {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must equal ",
			param,
			", got %q\", *x.",
			fieldName,
			"))",
		)
		g.P("}")
	} else if isNumericType(fieldType) || isBoolType(fieldType) {
		g.P("if x.", fieldName, " != ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must equal ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) || isPointerToBoolType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " != ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must equal ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genNe(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	if isStringType(fieldType) {
		g.P("if x.", fieldName, " == \"", param, "\" {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", param, "\")")
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " == \"", param, "\" {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", param, "\")")
		g.P("}")
	} else if isNumericType(fieldType) || isBoolType(fieldType) {
		g.P("if x.", fieldName, " == ", param, " {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", param, "\")")
		g.P("}")
	} else if isPointerToNumericType(fieldType) || isPointerToBoolType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " == ", param, " {")
		g.P("errs = append(errs, \"", fieldName, " must not equal ", param, "\")")
		g.P("}")
	}
}

func (vg *Generator) genGt(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") <= ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be more than ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") <= ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be more than ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") <= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have more than ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " <= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be greater than ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " <= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be greater than ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genGte(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") < ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") < ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have at least ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " < ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at least ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genLt(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") >= ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be less than ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") >= ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be less than ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") >= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have less than ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " >= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be less than ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " >= ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be less than ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genLte(g *genkit.GeneratedFile, fieldName, fieldType, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		g.P("if len(x.", fieldName, ") > ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			param,
			" characters, got %d\", len(x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isPointerToStringType(fieldType) {
		g.P("if x.", fieldName, " != nil && len(*x.", fieldName, ") > ", param, " {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			param,
			" characters, got %d\", len(*x.",
			fieldName,
			")))",
		)
		g.P("}")
	} else if isSliceOrMapType(fieldType) {
		g.P("if len(x.", fieldName, ") > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must have at most ", param, " elements, got %d\", len(x.", fieldName, ")))")
		g.P("}")
	} else if isNumericType(fieldType) {
		g.P("if x.", fieldName, " > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", param, ", got %v\", x.", fieldName, "))")
		g.P("}")
	} else if isPointerToNumericType(fieldType) {
		g.P("if x.", fieldName, " != nil && *x.", fieldName, " > ", param, " {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be at most ", param, ", got %v\", *x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genOneof(g *genkit.GeneratedFile, field *genkit.Field, param string) {
	// Validation already done in validateRule, skip if invalid
	if param == "" {
		return
	}

	values := strings.Split(param, " ")
	// Clean up values
	var cleanValues []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			cleanValues = append(cleanValues, v)
		}
	}
	if len(cleanValues) == 0 {
		return // Validation already done in validateRule
	}

	fieldName := field.Name
	fieldType := field.Type
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	if isStringType(fieldType) {
		var quoted []string
		for _, v := range cleanValues {
			quoted = append(quoted, fmt.Sprintf("%q", v))
		}
		g.P("if !func() bool {")
		g.P("for _, v := range []string{", strings.Join(quoted, ", "), "} {")
		g.P("if x.", fieldName, " == v { return true }")
		g.P("}")
		g.P("return false")
		g.P("}() {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be one of [",
			strings.Join(cleanValues, ", "),
			"], got %q\", x.",
			fieldName,
			"))",
		)
		g.P("}")
	} else {
		g.P("if !func() bool {")
		g.P("for _, v := range []", fieldType, "{", strings.Join(cleanValues, ", "), "} {")
		g.P("if x.", fieldName, " == v { return true }")
		g.P("}")
		g.P("return false")
		g.P("}() {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be one of [", strings.Join(cleanValues, ", "), "], got %v\", x.", fieldName, "))")
		g.P("}")
	}
}

func (vg *Generator) genOneofEnum(g *genkit.GeneratedFile, field *genkit.Field, param string, pkg *genkit.Package) {
	// Validation already done in validateRule, skip if invalid
	if param == "" {
		return
	}

	fieldName := field.Name
	enumType := strings.TrimSpace(param)
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")

	// Parse enum type parameter
	// Formats supported:
	// - Same package: "Status" -> StatusEnums.Contains
	// - Cross-package: "github.com/user/pkg/common.Status" -> common.StatusEnums.Contains
	//   (automatically adds import for "github.com/user/pkg/common")
	// - Cross-package with alias: "alias:github.com/user/pkg/common.Status" -> alias.StatusEnums.Contains
	//   (imports with alias: import alias "github.com/user/pkg/common")

	var enumsVar string     // Variable name for the enums instance (e.g., "StatusEnums")
	var enumValues []string // For generating comment with enum values
	var isStringEnum bool   // Whether the enum's underlying type is string

	// Check for alias format: "alias:import/path.Type"
	var importAlias string
	if colonIdx := strings.Index(enumType, ":"); colonIdx != -1 {
		importAlias = enumType[:colonIdx]
		enumType = enumType[colonIdx+1:]
	}

	if lastDot := strings.LastIndex(enumType, "."); lastDot != -1 {
		// Cross-package with full import path: "github.com/user/pkg/common.Status"
		beforeDot := enumType[:lastDot]
		typeName := enumType[lastDot+1:]

		importPath := genkit.GoImportPath(beforeDot)

		// Determine package name for generated code and ensure import is added
		var pkgName string
		if importAlias != "" {
			// Use specified alias
			g.ImportAs(importPath, genkit.GoPackageName(importAlias))
			pkgName = importAlias
		} else {
			// Use default package name and add import
			pkgName = string(g.Import(importPath))
		}

		enumsVar = pkgName + "." + typeName + "Enums"

		// Look up cross-package enum from package index
		if enum := vg.findEnum(importPath, typeName); enum != nil {
			isStringEnum = isStringType(enum.UnderlyingType)
			for _, v := range enum.Values {
				enumValues = append(enumValues, pkgName+"."+v.Name)
			}
		}
	} else {
		// Same package: "Status" -> StatusEnums
		enumsVar = enumType + "Enums"

		// Find enum in the same package
		for _, e := range pkg.Enums {
			if e.Name == enumType {
				isStringEnum = isStringType(e.UnderlyingType)
				for _, v := range e.Values {
					enumValues = append(enumValues, v.Name)
				}
				break
			}
		}
	}

	// Generate comment with enum values for code review
	if len(enumValues) > 0 {
		g.P("// Valid values:")
		for _, v := range enumValues {
			g.P("//   - ", v)
		}
	}

	// Generate validation code based on enum's underlying type
	// - String enums: use Contains() and List()
	// - Non-string enums: use ContainsName() and Names() for better error messages
	if isStringEnum {
		// String enum: use Contains and List
		g.P("if !", enumsVar, ".Contains(x.", fieldName, ") {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be one of %v, got %v\", ",
			enumsVar,
			".List(), x.",
			fieldName,
			"))",
		)
		g.P("}")
	} else {
		// Non-string enum: use ContainsName and Names for string representation
		g.P("if !", enumsVar, ".ContainsName(", fmtSprintf, "(\"%v\", x.", fieldName, ")) {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be one of %v, got %v\", ",
			enumsVar,
			".Names(), x.",
			fieldName,
			"))",
		)
		g.P("}")
	}
}

func (vg *Generator) genEmail(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexEmail], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be a valid email address, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genURL(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" {")
	g.P("if _, err := ", genkit.GoImportPath("net/url").Ident("ParseRequestURI"), "(x.", fieldName, "); err != nil {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid URL, got %q\", x.", fieldName, "))")
	g.P("}")
	g.P("}")
}

func (vg *Generator) genUUID(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexUUID], ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid UUID, got %q\", x.", fieldName, "))")
	g.P("}")
}

func (vg *Generator) genAlpha(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexAlpha], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must contain only letters, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genAlphanum(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexAlphanum], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must contain only letters and numbers, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genNumeric(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && !", regexVarNames[regexNumeric], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must contain only numbers, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genContains(g *genkit.GeneratedFile, fieldName, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if !", genkit.GoImportPath("strings").Ident("Contains"), "(x.", fieldName, ", \"", param, "\") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must contain '",
		param,
		"', got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genExcludes(g *genkit.GeneratedFile, fieldName, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if ", genkit.GoImportPath("strings").Ident("Contains"), "(x.", fieldName, ", \"", param, "\") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must not contain '",
		param,
		"', got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genStartsWith(g *genkit.GeneratedFile, fieldName, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if !", genkit.GoImportPath("strings").Ident("HasPrefix"), "(x.", fieldName, ", \"", param, "\") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must start with '",
		param,
		"', got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genEndsWith(g *genkit.GeneratedFile, fieldName, param string) {
	if param == "" {
		return
	}
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if !", genkit.GoImportPath("strings").Ident("HasSuffix"), "(x.", fieldName, ", \"", param, "\") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must end with '",
		param,
		"', got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genIP(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" && ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ") == nil {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be a valid IP address, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

func (vg *Generator) genIPv4(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" {")
	g.P("ip := ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ")")
	g.P("if ip == nil || ip.To4() == nil {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be a valid IPv4 address, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
	g.P("}")
}

func (vg *Generator) genIPv6(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" {")
	g.P("ip := ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ")")
	g.P("if ip == nil || ip.To4() != nil {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be a valid IPv6 address, got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
	g.P("}")
}

func (vg *Generator) genDNS1123(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	g.P("if x.", fieldName, " != \"\" {")
	// Check length (max 63 characters)
	g.P("if len(x.", fieldName, ") > 63 {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must follow DNS label format (RFC 1123, not exceed 63 characters), got %d characters\", len(x.",
		fieldName,
		")))",
	)
	g.P("}")
	// Check format (pattern matching)
	g.P("if !", regexVarNames[regexDNS1123], ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must follow DNS label format (RFC 1123, lowercase alphanumeric and '-', start/end with alphanumeric), got %q\", x.",
		fieldName,
		"))",
	)
	g.P("}")
	g.P("}")
}

func (vg *Generator) genDurationCombined(
	g *genkit.GeneratedFile,
	fieldName string,
	checkFormat, hasMin bool,
	minParam string,
	hasMax bool,
	maxParam string,
) {
	// Parse min/max durations at generation time
	var minDur, maxDur time.Duration
	if hasMin && minParam != "" {
		if dur, err := time.ParseDuration(minParam); err == nil {
			minDur = dur
		} else {
			hasMin = false // Invalid duration, skip
		}
	}
	if hasMax && maxParam != "" {
		if dur, err := time.ParseDuration(maxParam); err == nil {
			maxDur = dur
		} else {
			hasMax = false // Invalid duration, skip
		}
	}

	// If only format check, use simple validation
	if checkFormat && !hasMin && !hasMax {
		fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
		g.P("if x.", fieldName, " != \"\" {")
		g.P("if _, err := ", genkit.GoImportPath("time").Ident("ParseDuration"), "(x.", fieldName, "); err != nil {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be a valid duration (e.g., 1h30m, 500ms), got %q\", x.",
			fieldName,
			"))",
		)
		g.P("}")
		g.P("}")
		return
	}

	// Combined validation with min/max
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	timePkg := genkit.GoImportPath("time")

	g.P("if x.", fieldName, " != \"\" {")
	g.P("if _dur, _err := ", timePkg.Ident("ParseDuration"), "(x.", fieldName, "); _err != nil {")
	if checkFormat {
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be a valid duration (e.g., 1h30m, 500ms), got %q\", x.",
			fieldName,
			"))",
		)
	}
	g.P("} else {")
	if hasMin {
		// Build the condition line with duration expression
		minArgs := []any{"if _dur < "}
		minArgs = append(minArgs, durationToExpr(minDur)...)
		minArgs = append(minArgs, " {")
		g.P(minArgs...)
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at least ",
			minParam,
			", got %s\", x.",
			fieldName,
			"))",
		)
		g.P("}")
	}
	if hasMax {
		// Build the condition line with duration expression
		maxArgs := []any{"if _dur > "}
		maxArgs = append(maxArgs, durationToExpr(maxDur)...)
		maxArgs = append(maxArgs, " {")
		g.P(maxArgs...)
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be at most ",
			maxParam,
			", got %s\", x.",
			fieldName,
			"))",
		)
		g.P("}")
	}
	g.P("}")
	g.P("}")
}

// durationToExpr converts a duration to a readable Go expression using time package constants.
// Returns a slice of values that can be passed to g.P().
// Examples:
//   - 1s -> time.Second
//   - 100ms -> 100*time.Millisecond
//   - 1h30m -> 90*time.Minute
func durationToExpr(d time.Duration) []any {
	timePkg := genkit.GoImportPath("time")

	// Try to express in the most readable unit
	switch {
	case d%time.Hour == 0:
		hours := int64(d / time.Hour)
		if hours == 1 {
			return []any{timePkg.Ident("Hour")}
		}
		return []any{hours, "*", timePkg.Ident("Hour")}
	case d%time.Minute == 0:
		minutes := int64(d / time.Minute)
		if minutes == 1 {
			return []any{timePkg.Ident("Minute")}
		}
		return []any{minutes, "*", timePkg.Ident("Minute")}
	case d%time.Second == 0:
		seconds := int64(d / time.Second)
		if seconds == 1 {
			return []any{timePkg.Ident("Second")}
		}
		return []any{seconds, "*", timePkg.Ident("Second")}
	case d%time.Millisecond == 0:
		ms := int64(d / time.Millisecond)
		if ms == 1 {
			return []any{timePkg.Ident("Millisecond")}
		}
		return []any{ms, "*", timePkg.Ident("Millisecond")}
	case d%time.Microsecond == 0:
		us := int64(d / time.Microsecond)
		if us == 1 {
			return []any{timePkg.Ident("Microsecond")}
		}
		return []any{us, "*", timePkg.Ident("Microsecond")}
	default:
		// Fall back to nanoseconds
		ns := d.Nanoseconds()
		if ns == 1 {
			return []any{timePkg.Ident("Nanosecond")}
		}
		return []any{ns, "*", timePkg.Ident("Nanosecond")}
	}
}

func (vg *Generator) genMethod(g *genkit.GeneratedFile, field *genkit.Field, methodName string) {
	// Validation already done in validateRule, skip if invalid
	if methodName == "" {
		return
	}
	fieldName := field.Name
	fieldType := field.Type
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")

	if isSliceType(fieldType) {
		// For slice types, iterate over elements and call method on each
		g.P("for _i, _v := range x.", fieldName, " {")
		elemType := strings.TrimPrefix(fieldType, "[]")
		if isPointerType(elemType) {
			g.P("if _v != nil {")
			g.P("if err := _v.", methodName, "(); err != nil {")
			g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, "[%d]: %v\", _i, err))")
			g.P("}")
			g.P("}")
		} else {
			g.P("if err := _v.", methodName, "(); err != nil {")
			g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, "[%d]: %v\", _i, err))")
			g.P("}")
		}
		g.P("}")
	} else if isMapType(fieldType) {
		// For map types, iterate over values and call method on each
		g.P("for _k, _v := range x.", fieldName, " {")
		valueType := extractMapValueType(fieldType)
		if isPointerType(valueType) {
			g.P("if _v != nil {")
			g.P("if err := _v.", methodName, "(); err != nil {")
			g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, "[%v]: %v\", _k, err))")
			g.P("}")
			g.P("}")
		} else {
			g.P("if err := _v.", methodName, "(); err != nil {")
			g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, "[%v]: %v\", _k, err))")
			g.P("}")
		}
		g.P("}")
	} else if isPointerType(fieldType) {
		// For pointer types, check nil first
		g.P("if x.", fieldName, " != nil {")
		g.P("if err := x.", fieldName, ".", methodName, "(); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, ": %v\", err))")
		g.P("}")
		g.P("}")
	} else {
		// For value types (struct, etc.), call directly
		g.P("if err := x.", fieldName, ".", methodName, "(); err != nil {")
		g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, ": %v\", err))")
		g.P("}")
	}
}

func (vg *Generator) genRegex(g *genkit.GeneratedFile, field *genkit.Field, pattern string, customRegex *regexTracker) {
	// Validation already done in validateRule, skip if invalid
	if pattern == "" {
		return
	}
	fieldName := field.Name
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	varName := customRegex.getVarName(pattern)
	g.P("if x.", fieldName, " != \"\" && !", varName, ".MatchString(x.", fieldName, ") {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must match pattern %s, got %q\", ",
		genkit.RawString(pattern),
		", x.",
		fieldName,
		"))",
	)
	g.P("}")
}

// Supported format types for @format annotation.
var supportedFormats = map[string]bool{
	"json": true,
	"yaml": true,
	"toml": true,
	"csv":  true,
}

func (vg *Generator) genFormat(g *genkit.GeneratedFile, field *genkit.Field, format string) {
	// Validation already done in validateRule, skip if invalid
	if format == "" || strings.Contains(format, " ") {
		return
	}
	format = strings.ToLower(format)
	if !supportedFormats[format] {
		return
	}

	fieldName := field.Name
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")

	g.P("if x.", fieldName, " != \"\" {")
	switch format {
	case "json":
		g.P("if !", genkit.GoImportPath("encoding/json").Ident("Valid"), "([]byte(x.", fieldName, ")) {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be valid JSON format\"))",
		)
		g.P("}")
	case "yaml":
		// Import gopkg.in/yaml.v3 with alias "yaml" since the path base is "v3"
		yamlImport := genkit.GoImportPath("gopkg.in/yaml.v3")
		g.ImportAs(yamlImport, "yaml")
		g.P("var _yamlVal", fieldName, " interface{}")
		g.P(
			"if err := ",
			yamlImport.Ident("Unmarshal"),
			"([]byte(x.",
			fieldName,
			"), &_yamlVal",
			fieldName,
			"); err != nil {",
		)
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be valid YAML format: %v\", err))",
		)
		g.P("}")
	case "toml":
		g.P("var _tomlVal", fieldName, " interface{}")
		g.P(
			"if err := ",
			genkit.GoImportPath("github.com/BurntSushi/toml").Ident("Unmarshal"),
			"([]byte(x.",
			fieldName,
			"), &_tomlVal",
			fieldName,
			"); err != nil {",
		)
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be valid TOML format: %v\", err))",
		)
		g.P("}")
	case "csv":
		g.P(
			"_csvReader",
			fieldName,
			" := ",
			genkit.GoImportPath("encoding/csv").Ident("NewReader"),
			"(",
			genkit.GoImportPath("strings").Ident("NewReader"),
			"(x.",
			fieldName,
			"))",
		)
		g.P("if _, err := _csvReader", fieldName, ".ReadAll(); err != nil {")
		g.P(
			"errs = append(errs, ",
			fmtSprintf,
			"(\"",
			fieldName,
			" must be valid CSV format: %v\", err))",
		)
		g.P("}")
	}
	g.P("}")
}

func (vg *Generator) genCPU(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	stringsHasSuffix := genkit.GoImportPath("strings").Ident("HasSuffix")
	stringsToLower := genkit.GoImportPath("strings").Ident("ToLower")
	strconvParseInt := genkit.GoImportPath("strconv").Ident("ParseInt")

	g.P("if x.", fieldName, " != \"\" {")
	g.P(
		"_qty, err := ",
		genkit.GoImportPath("k8s.io/apimachinery/pkg/api/resource").Ident("ParseQuantity"),
		"(x.",
		fieldName,
		")",
	)
	g.P("if err != nil {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" invalid quantity: %v\", err))",
	)
	g.P("} else if _qty.Sign() == -1 {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be non-negative, got %s\", x.",
		fieldName,
		"))",
	)
	g.P("} else {")
	// Format validation: CPU must be pure digits or end with 'm'
	g.P("_lower := ", stringsToLower, "(x.", fieldName, ")")
	g.P("_base := _lower")
	g.P("if ", stringsHasSuffix, "(_lower, \"m\") {")
	g.P("_base = _lower[:len(_lower)-1]")
	g.P("}")
	g.P("if _, err := ", strconvParseInt, "(_base, 10, 64); err != nil || _base == \"\" {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" invalid CPU format: must be pure digits or end with 'm', got %s\", x.",
		fieldName,
		"))",
	)
	g.P("}")
	g.P("}")
	g.P("}")
}

func (vg *Generator) genMemory(g *genkit.GeneratedFile, fieldName string) {
	fmtSprintf := genkit.GoImportPath("fmt").Ident("Sprintf")
	stringsHasSuffix := genkit.GoImportPath("strings").Ident("HasSuffix")
	stringsToLower := genkit.GoImportPath("strings").Ident("ToLower")
	strconvParseInt := genkit.GoImportPath("strconv").Ident("ParseInt")

	g.P("if x.", fieldName, " != \"\" {")
	g.P(
		"_qty, err := ",
		genkit.GoImportPath("k8s.io/apimachinery/pkg/api/resource").Ident("ParseQuantity"),
		"(x.",
		fieldName,
		")",
	)
	g.P("if err != nil {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" invalid quantity: %v\", err))",
	)
	g.P("} else if _qty.Sign() == -1 {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" must be non-negative, got %s\", x.",
		fieldName,
		"))",
	)
	g.P("} else {")
	// Format validation: Memory must be pure digits or end with Ki/Gi/Ti/etc
	g.P("_lower := ", stringsToLower, "(x.", fieldName, ")")
	g.P("_base := _lower")
	g.P(
		"if ",
		stringsHasSuffix,
		"(_lower, \"ki\") || ",
		stringsHasSuffix,
		"(_lower, \"mi\") || ",
		stringsHasSuffix,
		"(_lower, \"gi\") || ",
		stringsHasSuffix,
		"(_lower, \"ti\") || ",
		stringsHasSuffix,
		"(_lower, \"pi\") || ",
		stringsHasSuffix,
		"(_lower, \"ei\") {",
	)
	g.P("_base = _lower[:len(_lower)-2]")
	g.P("}")
	g.P("if _, err := ", strconvParseInt, "(_base, 10, 64); err != nil || _base == \"\" {")
	g.P(
		"errs = append(errs, ",
		fmtSprintf,
		"(\"",
		fieldName,
		" invalid memory format: must be pure digits or end with Ki/Mi/Gi/Ti/Pi/Ei, got %s\", x.",
		fieldName,
		"))",
	)
	g.P("}")
	g.P("}")
	g.P("}")
}

// Helper functions

func isStringType(t string) bool {
	return t == "string"
}

// isPointerToStringType checks if t is a pointer to string type (e.g., "*string").
func isPointerToStringType(t string) bool {
	return t == "*string"
}

func isSliceOrMapType(t string) bool {
	return strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[")
}

func isSliceType(t string) bool {
	return strings.HasPrefix(t, "[]")
}

func isMapType(t string) bool {
	return strings.HasPrefix(t, "map[")
}

// ensureTypeImport checks if a type string contains a cross-package reference (e.g., "common.Priority")
// and adds the necessary import. It returns the type string with proper package reference.
// For types like "[]common.Priority" or "map[string]common.Level", it extracts the package
// and ensures it's imported.
func ensureTypeImport(g *genkit.GeneratedFile, fieldType string, pkg *genkit.Package) {
	// Extract the element type from slice/map
	elemType := fieldType
	if strings.HasPrefix(elemType, "[]") {
		elemType = strings.TrimPrefix(elemType, "[]")
	} else if strings.HasPrefix(elemType, "map[") {
		elemType = extractMapValueType(elemType)
	}
	// Strip pointer
	elemType = strings.TrimPrefix(elemType, "*")

	// Check if it's a cross-package type (contains ".")
	if dotIdx := strings.Index(elemType, "."); dotIdx != -1 {
		pkgAlias := elemType[:dotIdx]
		// Find the import path for this alias
		if pkg != nil && pkg.TypesPkg != nil {
			for _, imp := range pkg.TypesPkg.Imports() {
				// Check if the import name matches the alias
				if imp.Name() == pkgAlias {
					g.Import(genkit.GoImportPath(imp.Path()))
					return
				}
				// Also check the last part of the path (default import name)
				path := imp.Path()
				parts := strings.Split(path, "/")
				if len(parts) > 0 && parts[len(parts)-1] == pkgAlias {
					g.Import(genkit.GoImportPath(imp.Path()))
					return
				}
			}
		}
	}
}

// extractMapValueType extracts the value type from a map type string.
// e.g., "map[string]Address" -> "Address", "map[int]*User" -> "*User"
func extractMapValueType(t string) string {
	// Find the closing bracket of the key type
	if !strings.HasPrefix(t, "map[") {
		return ""
	}
	depth := 0
	for i := 4; i < len(t); i++ {
		switch t[i] {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				// Found the closing bracket, value type starts after it
				return t[i+1:]
			}
			depth--
		}
	}
	return ""
}

func isPointerType(t string) bool {
	return strings.HasPrefix(t, "*")
}

func isBoolType(t string) bool {
	return t == "bool"
}

// isPointerToBoolType checks if t is a pointer to bool type (e.g., "*bool").
func isPointerToBoolType(t string) bool {
	return t == "*bool"
}

func isNumericType(t string) bool {
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"byte", "rune", "uintptr":
		return true
	default:
		return false
	}
}

// isPointerToNumericType checks if t is a pointer to a numeric type (e.g., "*int", "*float64").
func isPointerToNumericType(t string) bool {
	if !strings.HasPrefix(t, "*") {
		return false
	}
	return isNumericType(strings.TrimPrefix(t, "*"))
}

// isBuiltinType checks if the type is a Go builtin type that cannot have methods.
func isBuiltinType(t string) bool {
	// Check pointer to builtin
	if strings.HasPrefix(t, "*") {
		return isBuiltinType(strings.TrimPrefix(t, "*"))
	}
	// Check slice - recurse to check element type
	if strings.HasPrefix(t, "[]") {
		return isBuiltinType(strings.TrimPrefix(t, "[]"))
	}
	// Check map - recurse to check value type
	if strings.HasPrefix(t, "map[") {
		valueType := extractMapValueType(t)
		if valueType == "" {
			return true // malformed map type
		}
		return isBuiltinType(valueType)
	}
	// Builtin primitive types
	switch t {
	case "string", "bool", "error",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128",
		"byte", "rune", "uintptr", "any":
		return true
	}
	// interface{} is builtin
	if strings.HasPrefix(t, "interface") {
		return true
	}
	return false
}

// isValidNumber checks if a string is a valid number (integer or float).
func isValidNumber(s string) bool {
	if s == "" {
		return false
	}
	// Try parsing as integer first
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	// Try parsing as float
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// isValidDuration checks if a string is a valid Go duration (e.g., "1h", "30m", "500ms").
func isValidDuration(s string) bool {
	if s == "" {
		return false
	}
	_, err := time.ParseDuration(s)
	return err == nil
}

// Error codes for diagnostics.
const (
	ErrCodeMethodMissingParam  = "E001"
	ErrCodeRegexMissingPattern = "E002"
	ErrCodeFormatMissingType   = "E003"
	ErrCodeFormatMultipleArgs  = "E004"
	ErrCodeFormatUnsupported   = "E005"
	ErrCodeOneofMissingValues  = "E006"
	ErrCodeMissingParam        = "E007" // Generic missing parameter error
	ErrCodeInvalidParamType    = "E008" // Invalid parameter type
	ErrCodeInvalidFieldType    = "E009" // Annotation not applicable to field type
	ErrCodeMethodNotFound      = "E010" // Method not found on type
)

// Validate implements genkit.ValidatableTool.
// It checks for errors without generating files, returning diagnostics for IDE integration.
func (vg *Generator) Validate(gen *genkit.Generator, _ *genkit.Logger) []genkit.Diagnostic {
	// Build package index for cross-package enum lookup
	vg.buildPkgIndex(gen)

	c := genkit.NewDiagnosticCollector(ToolName)

	for _, pkg := range gen.Packages {
		for _, typ := range pkg.Types {
			if !genkit.HasAnnotation(typ.Doc, ToolName, "validate") {
				continue
			}
			vg.validateType(c, typ)
		}
	}

	return c.Collect()
}

// validateType validates a single type and collects diagnostics.
func (vg *Generator) validateType(c *genkit.DiagnosticCollector, typ *genkit.Type) {
	for _, field := range typ.Fields {
		rules := vg.parseFieldAnnotations(field)
		for _, rule := range rules {
			vg.validateRule(c, typ, field, rule)
		}
	}
}

// validateRule validates a single rule and collects diagnostics.
func (vg *Generator) validateRule(
	c *genkit.DiagnosticCollector,
	typ *genkit.Type,
	field *genkit.Field,
	rule *validateRule,
) {
	// Use UnderlyingType for validation (supports custom types like `type Email string`)
	underlyingType := field.UnderlyingType

	switch rule.Name {
	// Annotations that require string underlying type
	case "email", "url", "uuid", "alpha", "alphanum", "numeric", "regex", "format", "dns1123_label":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		// Additional validation for specific annotations
		switch rule.Name {
		case "regex":
			if rule.Param == "" {
				c.Error(ErrCodeRegexMissingPattern, "@regex annotation requires a pattern parameter", field.Pos)
			}
		case "format":
			if rule.Param == "" {
				c.Error(ErrCodeFormatMissingType, "@format annotation requires a format type parameter", field.Pos)
			} else if strings.Contains(rule.Param, " ") {
				c.Error(ErrCodeFormatMultipleArgs, "@format annotation only accepts one parameter", field.Pos)
			} else if !supportedFormats[strings.ToLower(rule.Param)] {
				c.Errorf(ErrCodeFormatUnsupported, field.Pos,
					"unsupported format %q, supported: json, yaml, toml, csv", rule.Param)
			}
		}

	// Annotations that require string underlying type with parameter
	case "contains", "excludes", "startswith", "endswith":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@%s annotation requires a string parameter", rule.Name)
		}

	// IP validation annotations - require string underlying type
	case "ip", "ipv4", "ipv6", "duration":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}

	// Duration range validation - require string underlying type and valid duration parameter
	case "duration_min", "duration_max":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@%s annotation requires a duration parameter", rule.Name)
		} else if !isValidDuration(rule.Param) {
			c.Errorf(ErrCodeInvalidParamType, field.Pos, "@%s parameter must be a valid duration (e.g., 1h, 30m, 500ms), got %q", rule.Name, rule.Param)
		}

	// Annotations that work on string/slice/map (length) or numeric (value)
	case "min", "max", "gt", "gte", "lt", "lte":
		if !isStringType(underlyingType) && !isPointerToStringType(underlyingType) &&
			!isSliceOrMapType(underlyingType) &&
			!isNumericType(underlyingType) && !isPointerToNumericType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string, slice, map, or numeric underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@%s annotation requires a value parameter", rule.Name)
		} else if !isValidNumber(rule.Param) {
			c.Errorf(ErrCodeInvalidParamType, field.Pos, "@%s parameter must be a number, got %q", rule.Name, rule.Param)
		}

	// len annotation - only for string/slice/map
	case "len":
		if !isStringType(underlyingType) && !isSliceOrMapType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@len annotation requires string, slice, or map underlying type, got %s",
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@len annotation requires a value parameter")
		} else if !isValidNumber(rule.Param) {
			c.Errorf(ErrCodeInvalidParamType, field.Pos, "@len parameter must be a number, got %q", rule.Param)
		}

	// eq/ne - string, numeric, or bool
	case "eq", "ne":
		if !isStringType(underlyingType) && !isPointerToStringType(underlyingType) &&
			!isNumericType(underlyingType) && !isPointerToNumericType(underlyingType) &&
			!isBoolType(underlyingType) && !isPointerToBoolType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@%s annotation requires string, numeric, or bool underlying type, got %s",
				rule.Name,
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@%s annotation requires a value parameter", rule.Name)
		}

	// required - string, slice, map, pointer, bool, numeric
	case "required":
		if !isStringType(underlyingType) && !isSliceOrMapType(underlyingType) && !isPointerType(underlyingType) &&
			!isBoolType(underlyingType) && !isNumericType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@required annotation requires string, slice, map, pointer, bool, or numeric underlying type, got %s",
				underlyingType,
			)
		}

	// default - string, numeric, or bool
	case "default":
		if !isStringType(underlyingType) && !isPointerToStringType(underlyingType) &&
			!isNumericType(underlyingType) && !isPointerToNumericType(underlyingType) &&
			!isBoolType(underlyingType) && !isPointerToBoolType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@default annotation requires string, numeric, or bool underlying type, got %s",
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos, "@default annotation requires a value parameter")
		} else if isNumericType(underlyingType) || isPointerToNumericType(underlyingType) {
			if !isValidNumber(rule.Param) {
				c.Errorf(ErrCodeInvalidParamType, field.Pos, "@default parameter must be a number for numeric field, got %q", rule.Param)
			}
		} else if isBoolType(underlyingType) || isPointerToBoolType(underlyingType) {
			if rule.Param != "true" && rule.Param != "false" && rule.Param != "1" && rule.Param != "0" {
				c.Errorf(ErrCodeInvalidParamType, field.Pos, "@default parameter must be true/false for bool field, got %q", rule.Param)
			}
		}

	// cpu - Kubernetes CPU resource validation
	case "cpu":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@cpu annotation requires string underlying type, got %s",
				underlyingType,
			)
		}

	// memory - Kubernetes memory resource validation
	case "memory":
		if !isStringType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@memory annotation requires string underlying type, got %s",
				underlyingType,
			)
		}

	// method - must be a custom type (not builtin types like string, int, bool, etc.)
	case "method":
		if rule.Param == "" {
			c.Error(ErrCodeMethodMissingParam, "@method annotation requires a method name parameter", field.Pos)
			return
		}
		// For method, check declared type (not underlying) - custom types can have methods
		if isBuiltinType(field.Type) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@method annotation can only be applied to custom types, got builtin type %s",
				field.Type,
			)
			return
		}
		// Check if the method exists on the field type
		if typ.Pkg != nil && typ.Pkg.TypesInfo != nil {
			if !vg.hasMethodOnFieldType(typ.Pkg, field.Type, rule.Param) {
				c.Errorf(
					ErrCodeMethodNotFound,
					field.Pos,
					"method '%s' not found on type '%s'",
					rule.Param,
					field.Type,
				)
			}
		}

	// oneof - string or numeric
	case "oneof":
		if !isStringType(underlyingType) && !isNumericType(underlyingType) {
			c.Errorf(
				ErrCodeInvalidFieldType,
				field.Pos,
				"@oneof annotation requires string or numeric underlying type, got %s",
				underlyingType,
			)
		}
		if rule.Param == "" {
			c.Errorf(ErrCodeOneofMissingValues, field.Pos,
				"@oneof annotation requires at least one value")
		} else {
			// Check if all values are empty after cleanup
			hasValue := false
			for _, v := range strings.Split(rule.Param, " ") {
				if strings.TrimSpace(v) != "" {
					hasValue = true
					break
				}
			}
			if !hasValue {
				c.Errorf(ErrCodeOneofMissingValues, field.Pos,
					"@oneof annotation requires at least one value")
			}
		}

	// oneof_enum - field type must be the enum type itself or string underlying type
	case "oneof_enum":
		if rule.Param == "" {
			c.Errorf(ErrCodeMissingParam, field.Pos,
				"@oneof_enum annotation requires an enum type parameter")
			return
		}

		enumType := strings.TrimSpace(rule.Param)

		// Strip alias prefix if present: "alias:import/path.Type" -> "import/path.Type"
		if colonIdx := strings.Index(enumType, ":"); colonIdx != -1 {
			enumType = enumType[colonIdx+1:]
		}

		// Check if it's a cross-package enum (has full import path)
		if lastDot := strings.LastIndex(enumType, "."); lastDot != -1 && strings.Contains(enumType[:lastDot], "/") {
			// Cross-package enum: "github.com/user/pkg/common.Status"
			importPath := genkit.GoImportPath(enumType[:lastDot])
			typeName := enumType[lastDot+1:]

			// Look up the enum from package index
			// Note: If package is not loaded (not in ./... scope), skip validation
			// The generated code will still work as long as the import path is correct
			enum := vg.findEnum(importPath, typeName)
			if enum != nil {
				// Enum found: field type must match enum type or underlying type must match
				if field.Type != typeName && !isStringType(underlyingType) && underlyingType != enum.UnderlyingType {
					c.Errorf(
						ErrCodeInvalidFieldType,
						field.Pos,
						"@oneof_enum(%s) requires field underlying type to be %s or string, got %s",
						rule.Param,
						enum.UnderlyingType,
						underlyingType,
					)
				}
			}
			// If enum not found, skip validation - package may not be loaded
		} else {
			// Same package enum: look up enum first
			var enum *genkit.Enum
			for _, e := range typ.Pkg.Enums {
				if e.Name == enumType {
					enum = e
					break
				}
			}

			if enum == nil {
				c.Errorf(
					ErrCodeInvalidFieldType,
					field.Pos,
					"@oneof_enum(%s): enum type %s not found in current package",
					enumType,
					enumType,
				)
				return
			}

			// Enum found: field type must be the enum type itself, or underlying type must match
			if field.Type != enumType && !isStringType(underlyingType) && underlyingType != enum.UnderlyingType {
				c.Errorf(
					ErrCodeInvalidFieldType,
					field.Pos,
					"@oneof_enum(%s) requires field type to be %s or have underlying type %s, got %s (underlying: %s)",
					enumType,
					enumType,
					enum.UnderlyingType,
					field.Type,
					underlyingType,
				)
			}
		}
	}
}

// WriteTestHeader writes the test file header.
func (vg *Generator) WriteTestHeader(g *genkit.GeneratedFile, pkgName string) {
	g.P("// Code generated by ", ToolName, ". DO NOT EDIT.")
	g.P()
	g.P("package ", pkgName)
}

// GenerateValidateTest generates table-driven tests for a single type's _validate method.
// It always tests _validate() which excludes @method validations, making tests simpler
// and not requiring valid nested types to be set up.
func (vg *Generator) GenerateValidateTest(g *genkit.GeneratedFile, typ *genkit.Type) {
	typeName := typ.Name

	// Collect fields with validation annotations, excluding @method
	var validatedFields []*fieldValidation
	for _, field := range typ.Fields {
		rules := vg.parseFieldAnnotations(field)
		// Filter out @method rules for test generation
		var nonMethodRules []*validateRule
		for _, rule := range rules {
			if rule.Name != "method" {
				nonMethodRules = append(nonMethodRules, rule)
			}
		}
		if len(nonMethodRules) > 0 {
			validatedFields = append(validatedFields, &fieldValidation{
				Field: field,
				Rules: nonMethodRules,
			})
		}
	}

	if len(validatedFields) == 0 {
		return
	}

	// Ensure imports for all cross-package field types used in tests
	for _, fv := range validatedFields {
		ensureTypeImport(g, fv.Field.Type, typ.Pkg)
	}

	// Always test _validate() method
	methodName := "_validate"
	testFuncName := "Test" + typeName + "__validate"

	g.P()
	g.P("func ", testFuncName, "(t *", genkit.GoImportPath("testing").Ident("T"), ") {")

	// _validate() returns []string
	g.P("tests := []struct {")
	g.P("name    string")
	g.P("input   ", typeName)
	g.P("wantErr bool")
	g.P("}{")

	// Generate valid case - create a struct with all valid values
	g.P("{")
	g.P("name: \"valid\",")
	g.P("input: ", typeName, "{")
	for _, fv := range validatedFields {
		vg.generateValidFieldValue(g, fv, typ.Pkg)
	}
	g.P("},")
	g.P("wantErr: false,")
	g.P("},")

	// Generate invalid cases for each field with validation rules
	for _, fv := range validatedFields {
		vg.generateInvalidTestCases(g, typeName, fv, validatedFields, typ.Pkg)
	}

	g.P("}")
	g.P("for _, tt := range tests {")
	g.P("t.Run(tt.name, func(t *testing.T) {")

	// _validate() returns []string, check if non-empty
	g.P("errs := tt.input.", methodName, "()")
	g.P("hasErr := len(errs) > 0")
	g.P("if hasErr != tt.wantErr {")
	g.P("t.Errorf(\"", methodName, "() errors = %v, wantErr %v\", errs, tt.wantErr)")
	g.P("}")

	g.P("})")
	g.P("}")
	g.P("}")
}

// GenerateSetDefaultsTest generates table-driven tests for a single type's SetDefaults method.
// It only generates tests if the type has @default annotations.
func (vg *Generator) GenerateSetDefaultsTest(g *genkit.GeneratedFile, typ *genkit.Type) {
	typeName := typ.Name

	// Collect fields with @default annotation
	var defaultFields []*fieldValidation
	for _, field := range typ.Fields {
		rules := vg.parseFieldAnnotations(field)
		for _, rule := range rules {
			if rule.Name == "default" {
				defaultFields = append(defaultFields, &fieldValidation{
					Field: field,
					Rules: []*validateRule{rule},
				})
				break
			}
		}
	}

	if len(defaultFields) == 0 {
		return
	}

	testFuncName := "Test" + typeName + "_SetDefaults"

	g.P()
	g.P("func ", testFuncName, "(t *", genkit.GoImportPath("testing").Ident("T"), ") {")

	g.P("tests := []struct {")
	g.P("name   string")
	g.P("input  ", typeName)
	g.P("expect ", typeName)
	g.P("}{")

	// Test case 1: all zero values should be set to defaults
	g.P("{")
	g.P("name: \"all_zero_values\",")
	g.P("input: ", typeName, "{},")
	g.P("expect: ", typeName, "{")
	for _, fv := range defaultFields {
		vg.generateDefaultExpectValue(g, fv)
	}
	g.P("},")
	g.P("},")

	// Test case 2: non-zero values should not be overwritten
	g.P("{")
	g.P("name: \"non_zero_values_preserved\",")
	g.P("input: ", typeName, "{")
	for _, fv := range defaultFields {
		vg.generateNonZeroValue(g, fv)
	}
	g.P("},")
	g.P("expect: ", typeName, "{")
	for _, fv := range defaultFields {
		vg.generateNonZeroValue(g, fv)
	}
	g.P("},")
	g.P("},")

	g.P("}")

	g.P("for _, tt := range tests {")
	g.P("t.Run(tt.name, func(t *testing.T) {")
	g.P("got := tt.input")
	g.P("got.SetDefaults()")

	// Generate field-by-field comparison for each default field
	for _, fv := range defaultFields {
		fieldName := fv.Field.Name
		fieldType := fv.Field.Type
		if isPointerType(fieldType) {
			// For pointer types, compare dereferenced values
			g.P("if got.", fieldName, " == nil || tt.expect.", fieldName, " == nil {")
			g.P("if got.", fieldName, " != tt.expect.", fieldName, " {")
			g.P("t.Errorf(\"", fieldName, " = %v, want %v\", got.", fieldName, ", tt.expect.", fieldName, ")")
			g.P("}")
			g.P("} else if *got.", fieldName, " != *tt.expect.", fieldName, " {")
			g.P("t.Errorf(\"", fieldName, " = %v, want %v\", *got.", fieldName, ", *tt.expect.", fieldName, ")")
			g.P("}")
		} else {
			g.P("if got.", fieldName, " != tt.expect.", fieldName, " {")
			g.P("t.Errorf(\"", fieldName, " = %v, want %v\", got.", fieldName, ", tt.expect.", fieldName, ")")
			g.P("}")
		}
	}

	g.P("})")
	g.P("}")
	g.P("}")
}

// generateDefaultExpectValue generates the expected default value for a field.
func (vg *Generator) generateDefaultExpectValue(g *genkit.GeneratedFile, fv *fieldValidation) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type
	param := fv.Rules[0].Param

	if isStringType(fieldType) {
		g.P(fieldName, ": \"", param, "\",")
	} else if isPointerToStringType(fieldType) {
		g.P(fieldName, ": func() *string { v := \"", param, "\"; return &v }(),")
	} else if isBoolType(fieldType) {
		boolVal := param == "true" || param == "1"
		g.P(fieldName, ": ", boolVal, ",")
	} else if isPointerToBoolType(fieldType) {
		boolVal := param == "true" || param == "1"
		g.P(fieldName, ": func() *bool { v := ", boolVal, "; return &v }(),")
	} else if isNumericType(fieldType) {
		g.P(fieldName, ": ", param, ",")
	} else if isPointerToNumericType(fieldType) {
		baseType := strings.TrimPrefix(fieldType, "*")
		g.P(fieldName, ": func() *", baseType, " { v := ", baseType, "(", param, "); return &v }(),")
	}
}

// generateNonZeroValue generates a non-zero value for a field (different from default).
func (vg *Generator) generateNonZeroValue(g *genkit.GeneratedFile, fv *fieldValidation) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type

	if isStringType(fieldType) {
		g.P(fieldName, ": \"custom\",")
	} else if isPointerToStringType(fieldType) {
		g.P(fieldName, ": func() *string { v := \"custom\"; return &v }(),")
	} else if isBoolType(fieldType) {
		g.P(fieldName, ": true,")
	} else if isPointerToBoolType(fieldType) {
		g.P(fieldName, ": func() *bool { v := true; return &v }(),")
	} else if isNumericType(fieldType) {
		g.P(fieldName, ": 999,")
	} else if isPointerToNumericType(fieldType) {
		baseType := strings.TrimPrefix(fieldType, "*")
		g.P(fieldName, ": func() *", baseType, " { v := ", baseType, "(999); return &v }(),")
	}
}

// generateValidFieldValue generates a valid value for a field in test cases.
// Note: @method fields are already filtered out before calling this function.
func (vg *Generator) generateValidFieldValue(g *genkit.GeneratedFile, fv *fieldValidation, pkg *genkit.Package) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type
	underlyingType := fv.Field.UnderlyingType

	// Determine a valid value based on the rules
	var hasRequired, hasMin, hasMax, hasLen, hasGt, hasGte, hasLt, hasLte bool
	var hasEmail, hasURL, hasUUID, hasIP, hasIPv4, hasIPv6, hasDuration bool
	var hasAlpha, hasAlphanum, hasNumeric, hasOneof bool
	var hasContains, hasStartsWith, hasEndsWith, hasRegex bool
	var hasEq, hasNe, hasFormat, hasDNS1123 bool
	var hasDurationMin, hasDurationMax bool
	var hasExcludes, hasOneofEnum bool
	var hasCPU, hasMemory bool
	var minVal, maxVal, lenVal, gtVal, gteVal, ltVal, lteVal string
	var oneofValues, containsVal, startsWithVal, endsWithVal string
	var eqVal, neVal, regexVal, formatVal string
	var durationMinVal, durationMaxVal string
	var oneofEnumParam string

	for _, rule := range fv.Rules {
		switch rule.Name {
		case "required":
			hasRequired = true
		case "min":
			hasMin = true
			minVal = rule.Param
		case "max":
			hasMax = true
			maxVal = rule.Param
		case "len":
			hasLen = true
			lenVal = rule.Param
		case "gt":
			hasGt = true
			gtVal = rule.Param
		case "gte":
			hasGte = true
			gteVal = rule.Param
		case "lt":
			hasLt = true
			ltVal = rule.Param
		case "lte":
			hasLte = true
			lteVal = rule.Param
		case "email":
			hasEmail = true
		case "url":
			hasURL = true
		case "uuid":
			hasUUID = true
		case "ip":
			hasIP = true
		case "ipv4":
			hasIPv4 = true
		case "ipv6":
			hasIPv6 = true
		case "duration":
			hasDuration = true
		case "duration_min":
			hasDurationMin = true
			durationMinVal = rule.Param
		case "duration_max":
			hasDurationMax = true
			durationMaxVal = rule.Param
		case "dns1123_label":
			hasDNS1123 = true
		case "alpha":
			hasAlpha = true
		case "alphanum":
			hasAlphanum = true
		case "numeric":
			hasNumeric = true
		case "oneof":
			hasOneof = true
			oneofValues = rule.Param
		case "oneof_enum":
			hasOneofEnum = true
			oneofEnumParam = rule.Param
		case "contains":
			hasContains = true
			containsVal = rule.Param
		case "excludes":
			hasExcludes = true
		case "startswith":
			hasStartsWith = true
			startsWithVal = rule.Param
		case "endswith":
			hasEndsWith = true
			endsWithVal = rule.Param
		case "regex":
			hasRegex = true
			regexVal = rule.Param
		case "eq":
			hasEq = true
			eqVal = rule.Param
		case "ne":
			hasNe = true
			neVal = rule.Param
		case "format":
			hasFormat = true
			formatVal = rule.Param
		case "cpu":
			hasCPU = true
		case "memory":
			hasMemory = true
		}
	}

	// Generate value based on type and rules
	if isStringType(underlyingType) {
		var value string
		if hasOneofEnum {
			// For oneof_enum with string underlying type, use proper type conversion
			vg.generateEnumTestValue(g, fieldName, fieldType, oneofEnumParam, 1, pkg)
			return
		} else if hasEmail {
			value = "test@example.com"
		} else if hasURL {
			value = "https://example.com"
		} else if hasUUID {
			value = "550e8400-e29b-41d4-a716-446655440000"
		} else if hasIP || hasIPv4 {
			value = "192.168.1.1"
		} else if hasIPv6 {
			value = "::1"
		} else if hasDuration || hasDurationMin || hasDurationMax {
			// Generate a duration that satisfies min/max constraints
			if hasDurationMin && hasDurationMax {
				// Use min value
				value = durationMinVal
			} else if hasDurationMin {
				value = durationMinVal
			} else if hasDurationMax {
				value = durationMaxVal
			} else {
				value = "1h"
			}
		} else if hasCPU {
			value = "100m"
		} else if hasMemory {
			value = "128Mi"
		} else if hasDNS1123 {
			value = "example-name"
		} else if hasAlpha {
			value = "abcdef"
		} else if hasAlphanum {
			value = "abc123"
		} else if hasNumeric {
			value = "123456"
		} else if hasOneof {
			// Use first value from oneof
			parts := strings.Split(oneofValues, " ")
			if len(parts) > 0 {
				value = strings.TrimSpace(parts[0])
			}
		} else if hasRegex {
			// Generate a value that matches the regex pattern
			value = vg.generateValidRegexValue(regexVal)
		} else if hasFormat {
			// Generate valid format content
			value = vg.generateValidFormatValue(formatVal)
		} else if hasEq {
			value = eqVal
		} else if hasStartsWith && hasEndsWith {
			value = startsWithVal + "middle" + endsWithVal
		} else if hasStartsWith {
			value = startsWithVal + "test"
		} else if hasEndsWith {
			value = "test" + endsWithVal
		} else if hasContains {
			value = "test" + containsVal + "test"
		} else if hasExcludes {
			// Generate a value that doesn't contain the excluded string
			value = "validvalue"
		} else if hasLen {
			// Generate string of exact length
			n, _ := strconv.Atoi(lenVal)
			value = strings.Repeat("a", n)
		} else if hasMin {
			n, _ := strconv.Atoi(minVal)
			value = strings.Repeat("a", n)
		} else if hasRequired {
			value = "test"
		} else {
			value = "test"
		}
		g.P(fieldName, ": \"", value, "\",")
	} else if isNumericType(underlyingType) {
		var value string
		if hasOneofEnum {
			// For oneof_enum with numeric underlying type, use proper type conversion
			vg.generateEnumTestValue(g, fieldName, fieldType, oneofEnumParam, 1, pkg)
			return
		} else if hasOneof {
			// Use first value from oneof
			parts := strings.Split(oneofValues, " ")
			if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
				value = strings.TrimSpace(parts[0])
			} else {
				value = "1"
			}
		} else if hasEq {
			value = eqVal
		} else if hasGt {
			// Value must be > gtVal
			n, _ := strconv.ParseFloat(gtVal, 64)
			value = fmt.Sprintf("%v", int(n)+1)
		} else if hasGte {
			value = gteVal
		} else if hasLt {
			// Value must be < ltVal
			n, _ := strconv.ParseFloat(ltVal, 64)
			value = fmt.Sprintf("%v", int(n)-1)
		} else if hasLte {
			value = lteVal
		} else if hasMin {
			value = minVal
		} else if hasMax {
			value = maxVal
		} else if hasNe {
			// Generate a value that's not the excluded value
			value = "1"
		} else if hasRequired {
			value = "1"
		} else {
			value = "1"
		}
		g.P(fieldName, ": ", value, ",")
	} else if isBoolType(underlyingType) {
		if hasEq {
			g.P(fieldName, ": ", eqVal, ",")
		} else if hasNe {
			// Generate the opposite of the excluded value
			if neVal == "true" {
				g.P(fieldName, ": false,")
			} else {
				g.P(fieldName, ": true,")
			}
		} else if hasRequired {
			g.P(fieldName, ": true,")
		} else {
			g.P(fieldName, ": true,")
		}
	} else if isSliceType(fieldType) {
		if hasLen {
			n, _ := strconv.Atoi(lenVal)
			g.P(fieldName, ": make(", fieldType, ", ", n, "),")
		} else if hasMin {
			n, _ := strconv.Atoi(minVal)
			g.P(fieldName, ": make(", fieldType, ", ", n, "),")
		} else if hasRequired {
			g.P(fieldName, ": make(", fieldType, ", 1),")
		}
	} else if isMapType(fieldType) {
		if hasLen {
			// For map with exact length, we need to create with actual entries
			// This is tricky, so just create an empty map for now
			g.P(fieldName, ": make(", fieldType, "),")
		} else if hasMin {
			// For map with min, we need to create with actual entries
			g.P(fieldName, ": make(", fieldType, "),")
		} else if hasRequired {
			// Create a map with one entry to satisfy @required
			g.P(fieldName, ": ", fieldType, "{\"key\": 0},")
		}
	} else if isPointerType(fieldType) {
		if hasRequired {
			// Create a non-nil pointer with valid value
			elemType := strings.TrimPrefix(fieldType, "*")
			g.P(fieldName, ": &", elemType, "{},")
		}
	} else if hasOneofEnum {
		// For enum types, use the field type with a valid value
		// Handle cross-package enum types by adding import
		vg.generateEnumTestValue(g, fieldName, fieldType, oneofEnumParam, 1, pkg)
	}
}

// generateEnumTestValue generates a test value for enum fields.
// It handles both same-package and cross-package enum types, adding imports as needed.
// For valid values (value > 0), it uses the first enum constant.
// For invalid values (value like 99999), it generates an invalid value with proper type conversion.
func (vg *Generator) generateEnumTestValue(
	g *genkit.GeneratedFile,
	fieldName, fieldType, enumParam string,
	value int,
	pkg *genkit.Package,
) {
	// Parse enum type parameter to determine if import is needed
	enumType := strings.TrimSpace(enumParam)

	// Strip alias prefix if present: "alias:import/path.Type" -> "import/path.Type"
	var importAlias string
	if colonIdx := strings.Index(enumType, ":"); colonIdx != -1 {
		importAlias = enumType[:colonIdx]
		enumType = enumType[colonIdx+1:]
	}

	// Check if field type is a basic type (string, int, etc.) rather than the enum type itself
	isBasicFieldType := isStringType(fieldType) || isNumericType(fieldType)

	if lastDot := strings.LastIndex(enumType, "."); lastDot != -1 && strings.Contains(enumType[:lastDot], "/") {
		// Cross-package enum: "github.com/user/pkg/common.Status"
		beforeDot := enumType[:lastDot]
		typeName := enumType[lastDot+1:]

		importPath := genkit.GoImportPath(beforeDot)

		// Determine package name and ensure import is added
		var pkgName string
		if importAlias != "" {
			// Use specified alias
			g.ImportAs(importPath, genkit.GoPackageName(importAlias))
			pkgName = importAlias
		} else {
			// Use default package name and add import
			pkgName = string(g.Import(importPath))
		}

		// Look up enum to get first valid value
		enum := vg.findEnum(importPath, typeName)

		if isBasicFieldType {
			// Field type is string/int, not the enum type itself
			if isStringType(fieldType) {
				if value > 0 && value < 100 && enum != nil && len(enum.Values) > 0 {
					// Use first enum value's String() method (generated by enumgen)
					g.P(fieldName, ": ", pkgName, ".", enum.Values[0].Name, ".String(),")
				} else {
					g.P(fieldName, ": \"__invalid__\",")
				}
			} else {
				// Numeric field type
				if value > 0 && value < 100 && enum != nil && len(enum.Values) > 0 {
					g.P(fieldName, ": int(", pkgName, ".", enum.Values[0].Name, "),")
				} else {
					g.P(fieldName, ": ", value, ",")
				}
			}
		} else {
			// Field type is the enum type itself
			if enum != nil && len(enum.Values) > 0 && value > 0 && value < 100 {
				// Use first enum constant for valid value
				g.P(fieldName, ": ", pkgName, ".", enum.Values[0].Name, ",")
			} else {
				// Generate invalid value with proper type conversion
				if enum != nil && isStringType(enum.UnderlyingType) {
					g.P(fieldName, ": ", pkgName, ".", typeName, "(\"__invalid__\"),")
				} else {
					g.P(fieldName, ": ", pkgName, ".", typeName, "(", value, "),")
				}
			}
		}
	} else {
		// Same package enum - look up enum to get first valid value
		var enum *genkit.Enum
		for _, e := range pkg.Enums {
			if e.Name == enumType {
				enum = e
				break
			}
		}

		if isBasicFieldType {
			// Field type is string/int, not the enum type itself
			if isStringType(fieldType) {
				if value > 0 && value < 100 && enum != nil && len(enum.Values) > 0 {
					// Use first enum value's String() method (generated by enumgen)
					g.P(fieldName, ": ", enum.Values[0].Name, ".String(),")
				} else {
					g.P(fieldName, ": \"__invalid__\",")
				}
			} else {
				// Numeric field type
				if value > 0 && value < 100 && enum != nil && len(enum.Values) > 0 {
					g.P(fieldName, ": int(", enum.Values[0].Name, "),")
				} else {
					g.P(fieldName, ": ", value, ",")
				}
			}
		} else {
			// Field type is the enum type itself
			if enum != nil && len(enum.Values) > 0 && value > 0 && value < 100 {
				// Use first enum constant for valid value
				g.P(fieldName, ": ", enum.Values[0].Name, ",")
			} else {
				// Generate invalid value with proper type conversion
				if enum != nil && isStringType(enum.UnderlyingType) {
					g.P(fieldName, ": ", fieldType, "(\"__invalid__\"),")
				} else {
					g.P(fieldName, ": ", fieldType, "(", value, "),")
				}
			}
		}
	}
}

// generateValidRegexValue generates a value that matches common regex patterns.
func (vg *Generator) generateValidRegexValue(pattern string) string {
	// Handle common patterns
	switch {
	case strings.Contains(pattern, "[A-Z]") && strings.Contains(pattern, "\\d"):
		// Pattern like ^[A-Z]{2}-\d{4}$ -> "AB-1234"
		return "AB-1234"
	case strings.Contains(pattern, "[a-z]") && strings.Contains(pattern, "[0-9]"):
		return "abc123"
	case strings.Contains(pattern, "[0-9]") || strings.Contains(pattern, "\\d"):
		return "12345"
	case strings.Contains(pattern, "[a-zA-Z]"):
		return "abcdef"
	default:
		// Default: return a simple alphanumeric string
		return "test123"
	}
}

// generateValidFormatValue generates a valid value for format validation.
func (vg *Generator) generateValidFormatValue(format string) string {
	switch strings.ToLower(format) {
	case "json":
		return `{\"key\":\"value\"}`
	case "yaml":
		return "key: value"
	case "toml":
		return "key = \\\"value\\\""
	case "csv":
		return "a,b,c"
	case "cpu":
		return "100m"
	case "memory":
		return "128Mi"
	default:
		return "test"
	}
}

// generateInvalidTestCases generates test cases for invalid field values.
func (vg *Generator) generateInvalidTestCases(
	g *genkit.GeneratedFile,
	typeName string,
	fv *fieldValidation,
	allFields []*fieldValidation,
	pkg *genkit.Package,
) {
	fieldName := fv.Field.Name
	fieldType := fv.Field.Type
	underlyingType := fv.Field.UnderlyingType

	for _, rule := range fv.Rules {
		switch rule.Name {
		case "required":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_required\",")
			g.P("input: ", typeName, "{")
			// Fill other fields with valid values, leave this one empty
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				}
				// Skip the field being tested - it will be zero value
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "min":
			if isStringType(underlyingType) {
				n, _ := strconv.Atoi(rule.Param)
				if n > 0 {
					g.P("{")
					g.P("name: \"invalid_", fieldName, "_min\",")
					g.P("input: ", typeName, "{")
					for _, otherFv := range allFields {
						if otherFv.Field.Name != fieldName {
							vg.generateValidFieldValue(g, otherFv, pkg)
						} else {
							// Generate string shorter than min
							g.P(fieldName, ": \"", strings.Repeat("a", n-1), "\",")
						}
					}
					g.P("},")
					g.P("wantErr: true,")
					g.P("},")
				}
			} else if isNumericType(underlyingType) {
				n, _ := strconv.ParseFloat(rule.Param, 64)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_min\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)-1), ",")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if isSliceOrMapType(fieldType) {
				n, _ := strconv.Atoi(rule.Param)
				if n > 0 {
					g.P("{")
					g.P("name: \"invalid_", fieldName, "_min\",")
					g.P("input: ", typeName, "{")
					for _, otherFv := range allFields {
						if otherFv.Field.Name != fieldName {
							vg.generateValidFieldValue(g, otherFv, pkg)
						} else {
							// Generate slice/map with fewer elements
							g.P(fieldName, ": make(", fieldType, ", ", n-1, "),")
						}
					}
					g.P("},")
					g.P("wantErr: true,")
					g.P("},")
				}
			}

		case "max":
			if isStringType(underlyingType) {
				n, _ := strconv.Atoi(rule.Param)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_max\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						// Generate string longer than max
						g.P(fieldName, ": \"", strings.Repeat("a", n+1), "\",")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if isNumericType(underlyingType) {
				n, _ := strconv.ParseFloat(rule.Param, 64)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_max\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)+1), ",")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if isSliceOrMapType(fieldType) {
				n, _ := strconv.Atoi(rule.Param)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_max\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						// Generate slice/map with more elements
						g.P(fieldName, ": make(", fieldType, ", ", n+1, "),")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			}

		case "len":
			if isStringType(underlyingType) {
				n, _ := strconv.Atoi(rule.Param)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_len\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						// Generate string with wrong length
						g.P(fieldName, ": \"", strings.Repeat("a", n+1), "\",")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if isSliceOrMapType(fieldType) {
				n, _ := strconv.Atoi(rule.Param)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_len\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						// Generate slice/map with wrong length
						g.P(fieldName, ": make(", fieldType, ", ", n+1, "),")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			}

		case "eq":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_eq\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					if isStringType(underlyingType) {
						g.P(fieldName, ": \"__wrong_value__\",")
					} else if isBoolType(underlyingType) {
						// Invert the expected value
						if rule.Param == "true" {
							g.P(fieldName, ": false,")
						} else {
							g.P(fieldName, ": true,")
						}
					} else {
						// Numeric: use a different value
						n, _ := strconv.ParseFloat(rule.Param, 64)
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)+1), ",")
					}
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "ne":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_ne\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					if isStringType(underlyingType) {
						g.P(fieldName, ": \"", rule.Param, "\",")
					} else if isBoolType(underlyingType) {
						g.P(fieldName, ": ", rule.Param, ",")
					} else {
						g.P(fieldName, ": ", rule.Param, ",")
					}
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "email":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_email\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"invalid-email\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "url":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_url\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"not-a-url\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "uuid":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_uuid\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"not-a-uuid\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "ip", "ipv4", "ipv6":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_", rule.Name, "\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"not-an-ip\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "dns1123_label":
			// Test invalid format (contains uppercase letters)
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_dns1123_label\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"Invalid-DNS\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")
			// Test too long (> 63 characters)
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_dns1123_label_too_long\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					// Generate a string with 64 characters (exceeds 63-char limit)
					g.P(fieldName, ": \"", strings.Repeat("a", 64), "\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "duration":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_duration\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"invalid-duration\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "duration_min":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_duration_min\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					// Generate a duration smaller than min
					g.P(fieldName, ": \"1ns\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "duration_max":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_duration_max\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					// Generate a duration larger than max
					g.P(fieldName, ": \"1000h\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "alpha":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_alpha\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"abc123\",") // Contains numbers
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "alphanum":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_alphanum\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"abc-123\",") // Contains special char
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "numeric":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_numeric\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"12a34\",") // Contains letter
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "contains":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_contains\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"no_match_here\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "excludes":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_excludes\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"contains_", rule.Param, "_here\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "startswith":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_startswith\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"wrong_prefix\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "endswith":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_endswith\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"wrong_suffix\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "regex":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_regex\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					g.P(fieldName, ": \"!@#$%invalid\",")
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "format":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_format\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					// Generate invalid format content
					switch strings.ToLower(rule.Param) {
					case "json":
						g.P(fieldName, ": \"{invalid json\",")
					case "yaml":
						g.P(fieldName, ": \":\\n  - invalid: yaml: here\",")
					case "toml":
						g.P(fieldName, ": \"[invalid toml\",")
					case "csv":
						g.P(fieldName, ": \"\\\"unclosed quote\",")
					default:
						g.P(fieldName, ": \"invalid\",")
					}
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "oneof":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_oneof\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					if isStringType(underlyingType) {
						g.P(fieldName, ": \"__invalid_value__\",")
					} else {
						g.P(fieldName, ": -99999,")
					}
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "oneof_enum":
			g.P("{")
			g.P("name: \"invalid_", fieldName, "_oneof_enum\",")
			g.P("input: ", typeName, "{")
			for _, otherFv := range allFields {
				if otherFv.Field.Name != fieldName {
					vg.generateValidFieldValue(g, otherFv, pkg)
				} else {
					// Use an invalid enum value (typically a large number)
					// Handle cross-package enum types by adding import
					vg.generateEnumTestValue(g, fieldName, fieldType, rule.Param, 99999, pkg)
				}
			}
			g.P("},")
			g.P("wantErr: true,")
			g.P("},")

		case "gt":
			if isNumericType(underlyingType) {
				n, _ := strconv.ParseFloat(rule.Param, 64)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_gt\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)), ",") // Equal to, not greater than
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if isStringType(underlyingType) || isSliceOrMapType(fieldType) {
				n, _ := strconv.Atoi(rule.Param)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_gt\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						if isStringType(underlyingType) {
							g.P(fieldName, ": \"", strings.Repeat("a", n), "\",") // Equal length
						} else {
							g.P(fieldName, ": make(", fieldType, ", ", n, "),")
						}
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			}

		case "gte":
			if isNumericType(underlyingType) {
				n, _ := strconv.ParseFloat(rule.Param, 64)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_gte\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)-1), ",") // Less than
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if isStringType(underlyingType) {
				n, _ := strconv.Atoi(rule.Param)
				if n > 0 {
					g.P("{")
					g.P("name: \"invalid_", fieldName, "_gte\",")
					g.P("input: ", typeName, "{")
					for _, otherFv := range allFields {
						if otherFv.Field.Name != fieldName {
							vg.generateValidFieldValue(g, otherFv, pkg)
						} else {
							g.P(fieldName, ": \"", strings.Repeat("a", n-1), "\",")
						}
					}
					g.P("},")
					g.P("wantErr: true,")
					g.P("},")
				}
			}

		case "lt":
			if isNumericType(underlyingType) {
				n, _ := strconv.ParseFloat(rule.Param, 64)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_lt\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)), ",") // Equal to, not less than
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if isStringType(underlyingType) || isSliceOrMapType(fieldType) {
				n, _ := strconv.Atoi(rule.Param)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_lt\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						if isStringType(underlyingType) {
							g.P(fieldName, ": \"", strings.Repeat("a", n), "\",") // Equal length
						} else {
							g.P(fieldName, ": make(", fieldType, ", ", n, "),")
						}
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			}

		case "lte":
			if isNumericType(underlyingType) {
				n, _ := strconv.ParseFloat(rule.Param, 64)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_lte\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": ", fmt.Sprintf("%v", int(n)+1), ",") // Greater than
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			} else if isStringType(underlyingType) {
				n, _ := strconv.Atoi(rule.Param)
				g.P("{")
				g.P("name: \"invalid_", fieldName, "_lte\",")
				g.P("input: ", typeName, "{")
				for _, otherFv := range allFields {
					if otherFv.Field.Name != fieldName {
						vg.generateValidFieldValue(g, otherFv, pkg)
					} else {
						g.P(fieldName, ": \"", strings.Repeat("a", n+1), "\",")
					}
				}
				g.P("},")
				g.P("wantErr: true,")
				g.P("},")
			}

		case "method":
			// For method validation, we need to pass an invalid nested object
			// This is complex because we need to know what makes the nested object invalid
			// For now, we'll skip generating invalid cases for method validation
			// as it requires knowledge of the nested type's validation rules
		}
	}
}
