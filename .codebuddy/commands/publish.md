---
description: Guided version release workflow with checks, build, publish, and GitHub Release
argument-hint: Optional version number (e.g., 0.3.0)
allowed-tools: ["Read", "Write", "Bash", "TodoWrite", "AskUserQuestion", "Grep"]
---

# Publish Workflow

You are helping a developer release a new version of devgen. Follow a systematic approach: gather context, check interfaces, build, publish, and create releases.

## Core Principles

- **Step-by-step execution**: Complete each phase before moving to the next
- **Conditional logic**: Skip unnecessary steps based on actual changes
- **Use TodoWrite**: Track all progress throughout
- **Report progress**: Briefly report after each major step

---

## Context

- Current version tags: !`git tag --sort=-v:refname | head -5`
- Recent commits: !`git log --oneline -5`
- Current branch: !`git branch --show-current`
- Uncommitted changes: !`git status --short`

---

## Phase 1: Initialize

**Goal**: Determine version number and create task checklist

Initial request: $ARGUMENTS

**Actions**:
1. If `$ARGUMENTS` contains a version number (e.g., `0.3.0`), use it
2. Otherwise, ask user with AskUserQuestion:
   - Suggest patch version (bug fix): `current` → `next patch`
   - Suggest minor version (new feature): `current` → `next minor`
   - Allow custom version input
3. Create todo list with all phases using TodoWrite

---

## Phase 2: Check Config Interface

**Goal**: Determine if `Config()` method needs updating (affects VSCode plugin syntax highlighting)

**Actions**:
1. Check for generator file changes:
   ```bash
   git diff --name-only HEAD~5 | grep -E 'cmd/.*/generator/.*\.go' || echo "NO_CHANGES"
   ```

2. **If NO_CHANGES**: Mark complete, proceed to Phase 3

3. **If changes exist**, check Config() modifications:
   ```bash
   git diff HEAD~5 -- cmd/enumgen/generator/generator.go cmd/validategen/generator/generator.go | grep -A 20 'func.*Config()'
   ```

4. **Decision logic**:
   - If `Annotations` array changed → Update needed:
     - Modify `cmd/<tool>/generator/generator.go`
     - Update `cmd/<tool>/rules/*.md`
     - Run `devgen rules --agent kiro -w`
   - If only internal logic changed → No update needed

---

## Phase 3: Check Rules Interface

**Goal**: Determine if Rules documentation needs updating (affects AI usage)

**Actions**:
1. Check for tool-related changes:
   ```bash
   git diff --name-only HEAD~5 | grep -E 'cmd/.*/generator|genkit/' || echo "NO_CHANGES"
   ```

2. **If NO_CHANGES**: Mark complete, proceed to Phase 4

3. **If changes exist**, analyze impact:
   ```bash
   git diff HEAD~5 -- cmd/enumgen/generator/ cmd/validategen/generator/ genkit/
   ```

4. **Decision logic**:
   - Annotation syntax changed → Update `cmd/<tool>/rules/*.md`, run `devgen rules --agent kiro -w`
   - Config options changed → Update rules
   - Generated code interface changed → Update rules
   - Internal refactoring only → No update needed

---

## Phase 4: Check Documentation

**Goal**: Ensure documentation is up-to-date

**Actions**:
1. Review changed files:
   ```bash
   git diff --name-only HEAD~5
   ```

2. **Checklist** (check each based on changes):
   - [ ] `README.md` / `README_EN.md` - Feature descriptions, usage examples
   - [ ] `cmd/<tool>/README.md` - Tool-specific documentation
   - [ ] `docs/` - Detailed documentation

3. If updates needed, make changes before proceeding

---

## Phase 5: Code Quality Check

**Goal**: Ensure all tests pass and code builds

**CRITICAL**: Do not proceed if this fails.

**Actions**:
1. Run full check:
   ```bash
   make
   ```

2. **If successful**: Mark complete, proceed to Phase 6
3. **If failed**: Report errors, wait for user to fix, then retry

---

## Phase 6: Commit Code

**Goal**: Commit any pending changes

**Actions**:
1. Check for uncommitted changes:
   ```bash
   git status --short
   ```

2. **If no changes**: Mark complete, proceed to Phase 7

3. **If changes exist**:
   ```bash
   git add -A && git commit -m "<type>: <description>"
   ```
   - Choose type based on changes: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

---

## Phase 7: Publish Version

**Goal**: Create version tag and build artifacts

**Actions**:
1. Publish new version:
   ```bash
   make publish RELEASE_VERSION=<version>
   ```
   Where `<version>` is from Phase 1.

---

## Phase 8: Create Release Notes

**Goal**: Document changes for this release

**Actions**:
1. Get previous version and changes:
   ```bash
   PREV=$(git tag --sort=-v:refname | head -2 | tail -1)
   git log ${PREV}..v<version> --oneline
   git diff ${PREV}..v<version> --stat
   ```

2. Create release notes files:
   - `docs/release/v<version>.md` (Chinese)
   - `docs/release/v<version>_EN.md` (English)

3. Update README files with version links if needed

**Template**:
```markdown
# v<version> Release Notes

## 新特性 / New Features
- 

## Bug 修复 / Bug Fixes
- 

## 改进 / Improvements
- 
```

---

## Phase 9: Merge to Release Commit

**Goal**: Consolidate documentation into release commit

**Actions**:
1. Stage and amend:
   ```bash
   git add README.md README_EN.md docs/release/ cmd/*/README*.md
   git commit --amend --no-edit
   ```

2. Re-tag:
   ```bash
   git tag -d v<version> && git tag -a v<version> -m "Release v<version>"
   ```

---

## Phase 10: Push to Remote

**Goal**: Push code and tags to GitHub

**Actions**:
```bash
git push origin main --tags --force-with-lease
```

---

## Phase 11: Create GitHub Release

**Goal**: Create GitHub Release with artifacts

**Actions**:
1. Prepare release notes:
   ```bash
   cat docs/release/v<version>.md > /tmp/release_notes.md
   echo -e "\n\n---\n\n[English](https://github.com/tlipoca9/devgen/blob/main/docs/release/v<version>_EN.md)" >> /tmp/release_notes.md
   ```

2. Create release:
   ```bash
   gh release create v<version> --notes-file /tmp/release_notes.md vscode-devgen/devgen-<version>.vsix
   ```

---

## Phase 12: Publish to VSCode Marketplace

**Goal**: Publish VSCode extension

**Actions**:
```bash
cd vscode-devgen
source ../.env && VSCE_PAT="$AZURE_PAT" vsce publish
```

**Prerequisites**:
- `.env` file with `AZURE_PAT` configured
- `vsce` installed (`npm install -g @vscode/vsce`)

---

## Phase 13: Summary

**Goal**: Document what was accomplished

**Actions**:
1. Mark all todos complete
2. Summarize:
   - Version released
   - Key changes included
   - Artifacts published (GitHub Release, VSCode Marketplace)
   - Any issues encountered

---

## Quick Reference

| Step | Command |
|------|---------|
| Build | `make` |
| Publish version | `make publish RELEASE_VERSION=x.y.z` |
| Amend commit | `git commit --amend --no-edit` |
| Re-tag | `git tag -d v<ver> && git tag -a v<ver> -m "Release v<ver>"` |
| Push | `git push origin main --tags --force-with-lease` |
| GitHub Release | `gh release create v<ver> --notes-file /tmp/release_notes.md <files>` |
| VSCode publish | `VSCE_PAT="$PAT" vsce publish` |

## Version Convention

- **patch** (bug fix): `0.2.2` → `0.2.3`
- **minor** (new feature): `0.2.3` → `0.3.0`
- **major** (breaking change): `0.3.0` → `1.0.0`

## Commit Types

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `style:` Formatting
- `refactor:` Refactoring
- `test:` Testing
- `chore:` Other
