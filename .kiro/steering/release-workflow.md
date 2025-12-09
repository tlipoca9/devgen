---
inclusion: manual
---

# devgen Release Workflow

## Important: AI Execution Guidelines

**When the user requests to release a new version, AI must follow this process:**

### 1. Create Task Checklist

Before starting the release, **must** create a task checklist to track release progress:

```
Task Checklist Template:
1. [pending] Check Config interface
2. [pending] Check Rules interface
3. [pending] Check documentation updates
4. [pending] Code quality check (make)
5. [pending] Commit code
6. [pending] Publish version (make publish)
7. [pending] Create Release Notes
8. [pending] Merge documentation to release commit
9. [pending] Push to remote
10. [pending] Create GitHub Release
11. [pending] Publish to VSCode Marketplace
```

### 2. Execute Step by Step

**Before executing each step**, must:
1. Mark current task as `in_progress`
2. **Re-read** the detailed instructions for that step in this rule
3. Execute according to instructions
4. Mark as `completed` after completion

### 3. Confirmation Between Steps

After completing each major step, briefly report progress to the user, then continue to the next step.

---

## Prerequisites (First-time Setup)

```bash
# Install GitHub CLI
brew install gh
gh auth login  # Select GitHub.com → SSH → Login with web browser

# Install vsce
npm install -g @vscode/vsce

# Create Azure DevOps PAT (for VSCode Marketplace)
# 1. Visit https://dev.azure.com/ → Profile icon → Personal access tokens
# 2. New Token → Scopes: Marketplace → Manage → Save to .env file
```

---

## Release Steps

### Step 1: Check Config Interface

The `Config()` method returns `genkit.ToolConfig`, which defines the tool's annotation configuration for VSCode plugin syntax highlighting and auto-completion. **Only needs updating when annotation syntax changes.**

**When Config needs updating:**
- Adding/modifying/removing annotation names (e.g., adding `@newannotation`)
- Modifying annotation parameters (e.g., `@enum(string)` adding parameter `@enum(string, json)`)
- Modifying annotation scope (type/field)

**When Config doesn't need updating:**
- Internal generation logic changes (not affecting annotation syntax)
- Bug fixes
- Performance optimizations

**Check steps:**

```bash
# 1. View generator file changes
git diff --name-only HEAD~5 | grep -E 'cmd/.*/generator/.*\.go'

# 2. If there are changes, check if Config() method has modifications
git diff HEAD~5 -- cmd/enumgen/generator/generator.go cmd/validategen/generator/generator.go | grep -A 20 'func.*Config()'

# 3. Determine if it affects annotation configuration (Annotations array content)
```

**If updates are needed, modify these files:**

1. **Generator code**
   - `Config()` method in `cmd/<tool>/generator/generator.go`
   - Update `Annotations` field in `genkit.ToolConfig`

2. **Rules documentation** (synchronize annotation descriptions)
   - `cmd/<tool>/rules/*.md`
   - After modification, run `devgen rules --agent kiro -w` to regenerate

3. **README documentation**
   - `README.md` / `README_EN.md`
   - `cmd/<tool>/README.md`

**Note**: The `ToolConfig.Annotations` returned by `Config()` will be read by the VSCode plugin for annotation syntax highlighting and auto-completion.

### Step 2: Check Rules Interface

Rules are tool usage documentation for AI, stored in the `cmd/<tool>/rules/` directory. **Only needs updating when the tool's "user interface" changes.**

**When Rules need updating:**
- Adding/modifying/removing annotation syntax (e.g., `@enum(string)` → `@enum(string, json)`)
- Adding/modifying/removing configuration options or command-line parameters
- Modifying generated code usage (e.g., generated method signature changes)
- Adding usage examples or best practices

**When Rules don't need updating:**
- Internal implementation refactoring (not affecting user usage)
- Bug fixes (not changing interface)
- Performance optimizations
- Test code changes
- Generated code internal implementation changes (as long as interface remains unchanged)

**Check steps:**

```bash
# 1. View tool changes in this release
git diff --name-only HEAD~5 | grep -E 'cmd/.*/generator|genkit/'

# 2. If there are changes, view specific modifications
git diff HEAD~5 -- cmd/enumgen/generator/ cmd/validategen/generator/ genkit/

# 3. Determine if it affects user interface (annotation syntax, config options, generated code usage)
# 4. If updates are needed, modify cmd/<tool>/rules/*.md files
# 5. Regenerate rules files
devgen rules --agent kiro -w
```

**Note**: The `devgen rules` command only extracts rules from source code and generates them to the `.kiro/steering/` directory. It doesn't automatically determine if updates are needed. You need to manually determine if `cmd/<tool>/rules/*.md` source files need modification.

### Step 3: Check Documentation Updates

Check if the following documentation needs updating:

- [ ] `README.md` / `README_EN.md` - Feature descriptions, usage examples
- [ ] `cmd/<tool>/README.md` - Tool-specific documentation
- [ ] `docs/` - Detailed documentation

```bash
# View files involved in this change
git diff --name-only HEAD~5
```

### Step 4: Code Quality Check

After completing the above checks and fixes, run full checks:

```bash
# Execute tidy → lint → test → build in sequence, all must pass
make
```

### Step 5: Commit Code

```bash
git status
git add -A && git commit -m "<type>: <description>"
```

### Step 6: Publish Version

```bash
# View current version
git tag --sort=-v:refname | head -3

# Publish new version (automatically updates package.json, creates tag)
make publish RELEASE_VERSION=x.y.z
```

### Step 7: Create Release Notes

```bash
# View changes
git log v<prev>..v<curr> --oneline
git diff v<prev>..v<curr> --stat
```

**Create/update files:**
- `docs/release/v<version>.md` - Chinese Release Notes
- `docs/release/v<version>_EN.md` - English Release Notes
- `README.md` / `README_EN.md` - Add version links
- `cmd/<tool>/README.md` / `README_EN.md` - If tool changes exist

### Step 8: Merge Documentation to Release Commit

```bash
git add README.md README_EN.md docs/release/ cmd/*/README*.md
git commit --amend --no-edit
git tag -d v<version> && git tag -a v<version> -m "Release v<version>"
```

### Step 9: Push to Remote

```bash
git push origin main --tags --force-with-lease
```

### Step 10: Create GitHub Release

```bash
cat docs/release/v<version>.md > /tmp/release_notes.md
echo -e "\n\n---\n\n[English](https://github.com/tlipoca9/devgen/blob/main/docs/release/v<version>_EN.md)" >> /tmp/release_notes.md
gh release create v<version> --notes-file /tmp/release_notes.md vscode-devgen/devgen-<version>.vsix
```

### Step 11: Publish to VSCode Marketplace

```bash
cd vscode-devgen
source ../.env && VSCE_PAT="$AZURE_PAT" vsce publish
```

---

## Quick Reference

| Step | Command |
|------|------|
| Build | `make` |
| Publish version | `make publish RELEASE_VERSION=x.y.z` |
| Merge commit | `git commit --amend --no-edit` |
| Re-tag | `git tag -d v<ver> && git tag -a v<ver> -m "Release v<ver>"` |
| Push | `git push origin main --tags --force-with-lease` |
| GitHub Release | `gh release create v<ver> --notes-file /tmp/release_notes.md <files>` |
| VSCode publish | `VSCE_PAT="$PAT" vsce publish` |

## Version Number Convention

- **patch** (bug fix): `0.2.2` → `0.2.3`
- **minor** (new feature): `0.2.3` → `0.3.0`
- **major** (breaking change): `0.3.0` → `1.0.0`

## Commit Message Convention

`<type>: <description>`

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `style:` Formatting
- `refactor:` Refactoring
- `test:` Testing
- `chore:` Other
