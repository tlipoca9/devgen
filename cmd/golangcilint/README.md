# golangcilint

中文 | [English](README_EN.md)

`golangcilint` 是一个将 golangci-lint 与 devgen 集成的工具，用于 IDE 诊断集成。

## 概述

该工具检查 golangci-lint 是否已配置和安装，然后运行它并将输出转换为 devgen 诊断格式，供 IDE 集成使用。

**特点：**
- 仅验证，不生成代码
- 自动检测 golangci-lint 配置文件
- 支持 golangci-lint v1 和 v2 版本
- 输出标准 devgen 诊断格式

## 安装

```bash
go install github.com/tlipoca9/devgen/cmd/golangcilint@latest
```

**前置条件：**
- 需要安装 [golangci-lint](https://golangci-lint.run/usage/install/)

## 使用

```bash
golangcilint ./...              # 所有包
golangcilint ./pkg/models       # 指定包
```

## 工作原理

### 自动启用条件

golangcilint 在以下条件满足时自动启用：

1. 项目根目录存在 golangci-lint 配置文件：
   - `.golangci.yml`
   - `.golangci.yaml`
   - `.golangci.toml`
   - `.golangci.json`

2. 系统已安装 `golangci-lint` 命令

### 执行流程

1. 从加载的包中查找项目根目录
2. 检查是否存在 golangci-lint 配置文件
3. 检查 golangci-lint 是否已安装
4. 运行 `golangci-lint run --output.json.path stdout ./...`（v2）或 `golangci-lint run --out-format json ./...`（v1）
5. 解析 JSON 输出并转换为 devgen 诊断格式

### 诊断输出

每个诊断包含：
- `Severity` - 严重程度（error/warning）
- `Message` - 问题描述
- `File` - 文件路径
- `Line` / `Column` - 位置信息
- `Tool` - 工具名称（golangcilint）
- `Code` - 来源 linter 名称（如 gofmt、govet 等）

## 与 devgen 集成

golangcilint 作为 devgen 的内置工具，会在 `devgen --dry-run` 时自动运行：

```bash
# 运行所有验证（包括 golangci-lint）
devgen --dry-run ./...

# JSON 格式输出，用于 IDE 集成
devgen --dry-run --json ./...
```

## VSCode 集成

VSCode 插件会在以下时机自动运行 golangcilint：

1. **启动时** - 全局 dry-run 验证
2. **保存文件时** - 单文件验证

诊断结果会显示在 VSCode 的 Problems 面板中。

## 示例输出

```bash
$ golangcilint ./...
⚠ Found 3 issue(s)
⚠ [gofmt] cmd/example/main.go:10:1: File is not `gofmt`-ed
⚠ [govet] cmd/example/main.go:15:2: printf: fmt.Printf format %d has arg str of wrong type string
⚠ [unused] cmd/example/main.go:20:6: func `unusedFunc` is unused
```

## 配置

golangcilint 本身没有配置选项，它直接使用项目的 golangci-lint 配置文件。

关于 golangci-lint 的配置，请参考 [golangci-lint 官方文档](https://golangci-lint.run/usage/configuration/)。
