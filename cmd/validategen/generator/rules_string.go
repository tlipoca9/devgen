// Package generator provides validation code generation functionality.
package generator

import "github.com/tlipoca9/devgen/genkit"

func init() {
	DefaultRegistry.Register("email", PriorityFormat, func() Rule { return &EmailRule{stringFormatRule{name: "email"}} })
	DefaultRegistry.Register("url", PriorityFormat+1, func() Rule { return &URLRule{stringFormatRule{name: "url"}} })
	DefaultRegistry.Register("uuid", PriorityFormat+2, func() Rule { return &UUIDRule{stringFormatRule{name: "uuid"}} })
	DefaultRegistry.Register("ip", PriorityFormat+3, func() Rule { return &IPRule{stringFormatRule{name: "ip"}} })
	DefaultRegistry.Register("ipv4", PriorityFormat+4, func() Rule { return &IPv4Rule{stringFormatRule{name: "ipv4"}} })
	DefaultRegistry.Register("ipv6", PriorityFormat+5, func() Rule { return &IPv6Rule{stringFormatRule{name: "ipv6"}} })
	DefaultRegistry.Register("alpha", PriorityFormat+9, func() Rule { return &AlphaRule{stringFormatRule{name: "alpha"}} })
	DefaultRegistry.Register("alphanum", PriorityFormat+10, func() Rule { return &AlphanumRule{stringFormatRule{name: "alphanum"}} })
	DefaultRegistry.Register("numeric", PriorityFormat+11, func() Rule { return &NumericStringRule{stringFormatRule{name: "numeric"}} })
	DefaultRegistry.Register("dns1123_label", PriorityFormat+6, func() Rule { return &DNS1123Rule{stringFormatRule{name: "dns1123_label"}} })
	DefaultRegistry.Register("contains", PriorityString, func() Rule { return &ContainsRule{stringFormatRule{name: "contains"}} })
	DefaultRegistry.Register("excludes", PriorityString+1, func() Rule { return &ExcludesRule{stringFormatRule{name: "excludes"}} })
	DefaultRegistry.Register("startswith", PriorityString+2, func() Rule { return &StartsWithRule{stringFormatRule{name: "startswith"}} })
	DefaultRegistry.Register("endswith", PriorityString+3, func() Rule { return &EndsWithRule{stringFormatRule{name: "endswith"}} })
	DefaultRegistry.Register("regex", PriorityFormat+12, func() Rule { return &RegexRule{stringFormatRule{name: "regex"}} })
}

// stringFormatRule is a base for string format validation rules.
type stringFormatRule struct {
	name string
}

func (r *stringFormatRule) validateStringType(ctx *ValidateContext) {
	if !IsStringType(ctx.UnderlyingType) {
		ctx.Collector.Errorf(
			ErrCodeInvalidFieldType,
			ctx.Field.Pos,
			"@%s annotation requires string underlying type, got %s",
			r.name,
			ctx.UnderlyingType,
		)
	}
}

// RequiredRegex returns nil by default (no predefined regex needed).
func (r *stringFormatRule) RequiredRegex() []string { return nil }

// EmailRule validates email format.
type EmailRule struct{ stringFormatRule }

func (r *EmailRule) Name() string { return "email" }

func (r *EmailRule) RequiredRegex() []string { return []string{RegexEmail} }

func (r *EmailRule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" && !", RegexVarNames[RegexEmail], ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid email address, got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *EmailRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// URLRule validates URL format.
type URLRule struct{ stringFormatRule }

func (r *URLRule) Name() string { return "url" }

func (r *URLRule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" {")
	g.P("if _, err := ", genkit.GoImportPath("net/url").Ident("ParseRequestURI"), "(x.", fieldName, "); err != nil {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid URL, got %q\", x.", fieldName, "))")
	g.P("}")
	g.P("}")
}

func (r *URLRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// UUIDRule validates UUID format.
type UUIDRule struct{ stringFormatRule }

func (r *UUIDRule) Name() string { return "uuid" }

func (r *UUIDRule) RequiredRegex() []string { return []string{RegexUUID} }

func (r *UUIDRule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" && !", RegexVarNames[RegexUUID], ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid UUID, got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *UUIDRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// IPRule validates IP address format.
type IPRule struct{ stringFormatRule }

func (r *IPRule) Name() string { return "ip" }

func (r *IPRule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" && ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ") == nil {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid IP address, got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *IPRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// IPv4Rule validates IPv4 address format.
type IPv4Rule struct{ stringFormatRule }

func (r *IPv4Rule) Name() string { return "ipv4" }

func (r *IPv4Rule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" {")
	g.P("ip := ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ")")
	g.P("if ip == nil || ip.To4() == nil {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid IPv4 address, got %q\", x.", fieldName, "))")
	g.P("}")
	g.P("}")
}

func (r *IPv4Rule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// IPv6Rule validates IPv6 address format.
type IPv6Rule struct{ stringFormatRule }

func (r *IPv6Rule) Name() string { return "ipv6" }

func (r *IPv6Rule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" {")
	g.P("ip := ", genkit.GoImportPath("net").Ident("ParseIP"), "(x.", fieldName, ")")
	g.P("if ip == nil || ip.To4() != nil {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must be a valid IPv6 address, got %q\", x.", fieldName, "))")
	g.P("}")
	g.P("}")
}

func (r *IPv6Rule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// AlphaRule validates alphabetic characters only.
type AlphaRule struct{ stringFormatRule }

func (r *AlphaRule) Name() string { return "alpha" }

func (r *AlphaRule) RequiredRegex() []string { return []string{RegexAlpha} }

func (r *AlphaRule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" && !", RegexVarNames[RegexAlpha], ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must contain only letters, got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *AlphaRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// AlphanumRule validates alphanumeric characters only.
type AlphanumRule struct{ stringFormatRule }

func (r *AlphanumRule) Name() string { return "alphanum" }

func (r *AlphanumRule) RequiredRegex() []string { return []string{RegexAlphanum} }

func (r *AlphanumRule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" && !", RegexVarNames[RegexAlphanum], ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must contain only letters and numbers, got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *AlphanumRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// NumericStringRule validates numeric string characters only.
type NumericStringRule struct{ stringFormatRule }

func (r *NumericStringRule) Name() string { return "numeric" }

func (r *NumericStringRule) RequiredRegex() []string { return []string{RegexNumeric} }

func (r *NumericStringRule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" && !", RegexVarNames[RegexNumeric], ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must contain only numbers, got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *NumericStringRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// DNS1123Rule validates DNS label format.
type DNS1123Rule struct{ stringFormatRule }

func (r *DNS1123Rule) Name() string { return "dns1123_label" }

func (r *DNS1123Rule) RequiredRegex() []string { return []string{RegexDNS1123} }

func (r *DNS1123Rule) Generate(ctx *GenerateContext) {
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	g.P("if x.", fieldName, " != \"\" {")
	g.P("if len(x.", fieldName, ") > 63 {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must follow DNS label format (RFC 1123, not exceed 63 characters), got %d characters\", len(x.", fieldName, ")))")
	g.P("}")
	g.P("if !", RegexVarNames[RegexDNS1123], ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must follow DNS label format (RFC 1123, lowercase alphanumeric and '-', start/end with alphanumeric), got %q\", x.", fieldName, "))")
	g.P("}")
	g.P("}")
}

func (r *DNS1123Rule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
}

// ContainsRule validates string contains substring.
type ContainsRule struct{ stringFormatRule }

func (r *ContainsRule) Name() string { return "contains" }

func (r *ContainsRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName

	// Escape string parameters for safe embedding in generated code
	escapedParam := escapeString(ctx.Param)

	g.P("if !", genkit.GoImportPath("strings").Ident("Contains"), "(x.", fieldName, ", \"", escapedParam, "\") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must contain '", escapedParam, "', got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *ContainsRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@contains annotation requires a string parameter")
	}
}

// ExcludesRule validates string does not contain substring.
type ExcludesRule struct{ stringFormatRule }

func (r *ExcludesRule) Name() string { return "excludes" }

func (r *ExcludesRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName

	// Escape string parameters for safe embedding in generated code
	escapedParam := escapeString(ctx.Param)

	g.P("if ", genkit.GoImportPath("strings").Ident("Contains"), "(x.", fieldName, ", \"", escapedParam, "\") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must not contain '", escapedParam, "', got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *ExcludesRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@excludes annotation requires a string parameter")
	}
}

// StartsWithRule validates string starts with prefix.
type StartsWithRule struct{ stringFormatRule }

func (r *StartsWithRule) Name() string { return "startswith" }

func (r *StartsWithRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName

	// Escape string parameters for safe embedding in generated code
	escapedParam := escapeString(ctx.Param)

	g.P("if !", genkit.GoImportPath("strings").Ident("HasPrefix"), "(x.", fieldName, ", \"", escapedParam, "\") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must start with '", escapedParam, "', got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *StartsWithRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@startswith annotation requires a string parameter")
	}
}

// EndsWithRule validates string ends with suffix.
type EndsWithRule struct{ stringFormatRule }

func (r *EndsWithRule) Name() string { return "endswith" }

func (r *EndsWithRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName

	// Escape string parameters for safe embedding in generated code
	escapedParam := escapeString(ctx.Param)

	g.P("if !", genkit.GoImportPath("strings").Ident("HasSuffix"), "(x.", fieldName, ", \"", escapedParam, "\") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must end with '", escapedParam, "', got %q\", x.", fieldName, "))")
	g.P("}")
}

func (r *EndsWithRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
	if ctx.Param == "" {
		ctx.Collector.Errorf(ErrCodeMissingParam, ctx.Field.Pos, "@endswith annotation requires a string parameter")
	}
}

// RegexRule validates string matches regex pattern.
type RegexRule struct{ stringFormatRule }

func (r *RegexRule) Name() string { return "regex" }

func (r *RegexRule) Generate(ctx *GenerateContext) {
	if ctx.Param == "" {
		return
	}
	fmtSprintf := fmtSprintf()
	g := ctx.G
	fieldName := ctx.FieldName
	varName := ctx.CustomRegex.GetVarName(ctx.Param)
	g.P("if x.", fieldName, " != \"\" && !", varName, ".MatchString(x.", fieldName, ") {")
	g.P("errs = append(errs, ", fmtSprintf, "(\"", fieldName, " must match pattern %s, got %q\", ", genkit.RawString(ctx.Param), ", x.", fieldName, "))")
	g.P("}")
}

func (r *RegexRule) Validate(ctx *ValidateContext) {
	r.stringFormatRule.validateStringType(ctx)
	if ctx.Param == "" {
		ctx.Collector.Error(ErrCodeRegexMissingPattern, "@regex annotation requires a pattern parameter", ctx.Field.Pos)
	} else if !isValidRegex(ctx.Param) {
		ctx.Collector.Errorf(ErrCodeInvalidRegex, ctx.Field.Pos, "@regex pattern is invalid: %q", ctx.Param)
	}
}
