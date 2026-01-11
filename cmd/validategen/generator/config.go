// Package generator provides validation code generation functionality.
package generator

import "github.com/tlipoca9/devgen/genkit"

// ToolName is the name of this tool, used in annotations.
const ToolName = "validategen"

// Config returns the tool configuration for VSCode extension integration.
func Config() genkit.ToolConfig {
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
  numeric:   不能为 0`,
			},
			{
				Name: "min",
				Type: "field",
				Doc: `最小值（数字）或最小长度（字符串/切片/map）。

用法：在字段上方添加注解
  // validategen:@min(0)
  Age int

  // validategen:@min(2)
  Name string`,
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
  Name string`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "len",
				Type: "field",
				Doc: `精确长度（字符串/切片/map）。

用法：在字段上方添加注解
  // validategen:@len(6)
  Code string`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "eq",
				Type: "field",
				Doc: `字段值必须等于指定值。

用法：在字段上方添加注解
  // validategen:@eq(v1)
  Version string`,
				Params: &genkit.AnnotationParams{Type: []string{"string", "number", "bool"}, Placeholder: "value"},
			},
			{
				Name: "ne",
				Type: "field",
				Doc: `字段值不能等于指定值。

用法：在字段上方添加注解
  // validategen:@ne(deleted)
  Status string`,
				Params: &genkit.AnnotationParams{Type: []string{"string", "number", "bool"}, Placeholder: "value"},
			},
			{
				Name: "gt",
				Type: "field",
				Doc:  `大于（数字）或长度大于（字符串/切片）。`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "gte",
				Type: "field",
				Doc:  `大于等于（数字）或长度大于等于（字符串/切片）。`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "lt",
				Type: "field",
				Doc:  `小于（数字）或长度小于（字符串/切片）。`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "lte",
				Type: "field",
				Doc:  `小于等于（数字）或长度小于等于（字符串/切片）。`,
				Params: &genkit.AnnotationParams{Type: "number", Placeholder: "value"},
			},
			{
				Name: "oneof",
				Type: "field",
				Doc: `字段值必须是指定值之一（空格分隔）。

用法：在字段上方添加注解
  // validategen:@oneof(pending active completed)
  Status string`,
				Params: &genkit.AnnotationParams{Type: "list", Placeholder: "value1 value2 ..."},
			},
			{
				Name: "oneof_enum",
				Type: "field",
				Doc: `字段值必须是有效的枚举值（使用 EnumTypeEnums.Contains）。

用法：在字段上方添加注解
  // validategen:@oneof_enum(OrderStatus)
  Status OrderStatus

跨包枚举：
  // validategen:@oneof_enum(github.com/you/pkg/common.Status)
  Status common.Status`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "[alias:]import/path.EnumType"},
			},
			{
				Name: "email",
				Type: "field",
				Doc:  `字段必须是有效的邮箱地址格式。`,
			},
			{
				Name: "url",
				Type: "field",
				Doc:  `字段必须是有效的 URL。`,
			},
			{
				Name: "uuid",
				Type: "field",
				Doc:  `字段必须是有效的 UUID 格式。`,
			},
			{
				Name: "ip",
				Type: "field",
				Doc:  `字段必须是有效的 IP 地址（IPv4 或 IPv6）。`,
			},
			{
				Name: "ipv4",
				Type: "field",
				Doc:  `字段必须是有效的 IPv4 地址。`,
			},
			{
				Name: "ipv6",
				Type: "field",
				Doc:  `字段必须是有效的 IPv6 地址。`,
			},
			{
				Name: "dns1123_label",
				Type: "field",
				Doc:  `字段必须符合 DNS label 标准（RFC 1123）。`,
			},
			{
				Name: "duration",
				Type: "field",
				Doc:  `字段必须是有效的 Go duration 字符串。`,
			},
			{
				Name:   "duration_min",
				Type:   "field",
				Doc:    `最小时间间隔。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "1s, 5m, 1h, etc."},
			},
			{
				Name:   "duration_max",
				Type:   "field",
				Doc:    `最大时间间隔。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "1s, 5m, 1h, etc."},
			},
			{
				Name: "alpha",
				Type: "field",
				Doc:  `字段只能包含 ASCII 字母。`,
			},
			{
				Name: "alphanum",
				Type: "field",
				Doc:  `字段只能包含 ASCII 字母和数字。`,
			},
			{
				Name: "numeric",
				Type: "field",
				Doc:  `字段只能包含数字字符。`,
			},
			{
				Name:   "contains",
				Type:   "field",
				Doc:    `字段必须包含指定的子串。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "substring"},
			},
			{
				Name:   "excludes",
				Type:   "field",
				Doc:    `字段不能包含指定的子串。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "substring"},
			},
			{
				Name:   "startswith",
				Type:   "field",
				Doc:    `字段必须以指定前缀开头。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "prefix"},
			},
			{
				Name:   "endswith",
				Type:   "field",
				Doc:    `字段必须以指定后缀结尾。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "suffix"},
			},
			{
				Name: "method",
				Type: "field",
				Doc: `调用字段的方法进行嵌套验证。

用法：在字段上方添加注解
  // validategen:@method(Validate)
  Address Address`,
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
				Name:   "regex",
				Type:   "field",
				Doc:    `字段必须匹配指定的正则表达式。`,
				Params: &genkit.AnnotationParams{Type: "string", Placeholder: "pattern"},
			},
			{
				Name: "format",
				Type: "field",
				Doc:  `字段必须是有效的指定格式（json, yaml, toml, csv）。`,
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
				Name:   "default",
				Type:   "field",
				Doc:    `为字段设置默认值（在验证前应用）。`,
				Params: &genkit.AnnotationParams{Type: []string{"string", "number", "bool"}, Placeholder: "value"},
			},
			{
				Name: "cpu",
				Type: "field",
				Doc:  `验证 Kubernetes CPU 资源数量格式。`,
			},
			{
				Name: "memory",
				Type: "field",
				Doc:  `验证 Kubernetes 内存资源数量格式。`,
			},
			{
				Name: "disk",
				Type: "field",
				Doc:  `验证 Kubernetes 磁盘资源数量格式。`,
			},
		},
	}
}
