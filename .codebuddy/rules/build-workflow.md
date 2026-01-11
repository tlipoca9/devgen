---
description: 构建和发布流程 - Makefile 使用、版本管理、发布规范
globs: ["Makefile", "**/*.go"]
alwaysApply: false
---

# 构建和发布流程

本规则定义了 devgen 项目的构建、测试和发布流程。

## Makefile 目标

### 常用命令

```bash
# 完整构建流程
make all           # tidy + generate + lint + test + build

# 单独步骤
make tidy          # go mod tidy
make generate      # 运行代码生成
make lint          # 运行 golangci-lint --fix
make test          # 运行测试
make build         # 构建所有工具

# 安装
make install       # 安装到 $GOPATH/bin
make tools         # 安装开发依赖

# 清理
make clean         # 清理构建产物
```

### VSCode 扩展

```bash
make vscode         # 构建 VSCode 扩展
make vscode-install # 构建并安装（自动检测 IDE）
```

## 构建产物

```
_output/
└── bin/
    ├── devgen          # 主 CLI 工具
    ├── enumgen         # 枚举生成器
    ├── validategen     # 验证生成器
    └── golangcilint    # lint 集成
```

## 版本管理

### 版本号格式

遵循语义化版本（SemVer）：`MAJOR.MINOR.PATCH`

- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能新增
- **PATCH**: 向后兼容的问题修复

### Git Tag

```bash
# 创建版本标签
git tag v0.4.0
git push origin v0.4.0
```

### 版本注入

构建时自动注入版本信息：

```makefile
VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.version=$(VERSION) \
           -X main.commit=$(COMMIT) \
           -X main.date=$(DATE)
```

## 代码生成

### 运行生成

```bash
# 生成所有代码
make generate

# 或直接运行
go generate ./...
```

### AI 规则生成

```bash
# 生成 AI 规则到各 IDE 目录
devgen rules

# 指定适配器
devgen rules --adapter codebuddy
devgen rules --adapter cursor
devgen rules --adapter kiro
```

## 测试流程

### 运行测试

```bash
# 使用 make
make test

# 直接运行
go test ./...

# 使用 ginkgo（如果安装）
ginkgo -r -v
```

### 测试覆盖率

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Lint 检查

### 运行 lint

```bash
# 自动修复
make lint

# 或直接运行
golangci-lint run --fix
```

### 配置文件

`.golangci.yaml` 定义了 lint 规则：

```yaml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  default: standard

formatters:
  enable:
    - gci        # import 排序
    - gofmt      # 代码格式化
    - golines    # 行长度 (120)
```

## 发布流程

**推荐使用 `/publish` 命令**进行版本发布，它提供了完整的引导式发布流程。

### 使用 /publish 命令

```bash
# 在 AI 助手中运行
/publish 0.4.0
```

`/publish` 命令会自动执行以下步骤：
1. 确定版本号
2. 检查 Config 接口变更（影响 VSCode 语法高亮）
3. 检查 Rules 接口变更（影响 AI 使用）
4. 检查文档更新
5. 运行代码质量检查（`make`）
6. 提交代码
7. 创建版本标签和构建产物
8. 创建发布说明
9. 推送到远程
10. 创建 GitHub Release
11. 发布到 VSCode Marketplace

### 手动发布（备选）

如果需要手动发布，按以下步骤操作：

#### 1. 准备发布

```bash
# 确保代码是最新的
git pull origin main

# 运行完整检查
make all
```

#### 2. 创建版本说明

```bash
# 创建版本说明文件
vim docs/release/v0.4.0.md
vim docs/release/v0.4.0_EN.md
```

#### 3. 提交并标签

```bash
git add .
git commit -m "release: v0.4.0"
git tag -a v0.4.0 -m "Release v0.4.0"
```

#### 4. 推送

```bash
git push origin main --tags
```

#### 5. 创建 GitHub Release

```bash
gh release create v0.4.0 --notes-file docs/release/v0.4.0.md vscode-devgen/devgen-0.4.0.vsix
```

#### 6. 发布 VSCode 扩展

```bash
cd vscode-devgen
VSCE_PAT="$AZURE_PAT" vsce publish
```

## 开发工作流

### 日常开发

```bash
# 1. 拉取最新代码
git pull

# 2. 创建功能分支
git checkout -b feature/my-feature

# 3. 开发...

# 4. 运行检查
make lint test

# 5. 提交
git add .
git commit -m "feat: add new feature"

# 6. 推送
git push origin feature/my-feature
```

### 提交信息规范

使用 Conventional Commits 格式：

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

类型：
- `feat`: 新功能
- `fix`: 修复
- `docs`: 文档
- `style`: 格式
- `refactor`: 重构
- `test`: 测试
- `chore`: 杂项

示例：
```
feat(enumgen): add JSON marshaling support
fix(genkit): handle nil pointer in ParseAnnotations
docs(readme): update installation instructions
```

## 依赖管理

### 添加依赖

```bash
go get github.com/example/package@v1.2.3
make tidy
```

### 更新依赖

```bash
go get -u ./...
make tidy
```

### 检查依赖

```bash
go mod verify
go mod graph
```

## CI/CD 集成

### 推荐的 CI 步骤

```yaml
steps:
  - name: Setup Go
    uses: actions/setup-go@v4
    with:
      go-version: '1.24'

  - name: Install dependencies
    run: make tools

  - name: Lint
    run: make lint

  - name: Test
    run: make test

  - name: Build
    run: make build
```

## 故障排除

### 常见问题

**Q: `make lint` 失败**
```bash
# 安装 golangci-lint
make tools
```

**Q: `make generate` 失败**
```bash
# 确保 devgen 已安装
make install
```

**Q: 测试失败**
```bash
# 查看详细输出
go test -v ./...
```
