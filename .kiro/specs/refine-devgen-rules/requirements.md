# Requirements Document

## Introduction

This document outlines the requirements for refining and standardizing the devgen AI rules system to support multiple AI coding assistants (Kiro, CodeBuddy, Cursor, etc.) with a consistent adapter pattern.

## Glossary

- **devgen**: Go code generation toolkit with annotation-based code generation
- **AI Rules**: Structured markdown documentation with YAML frontmatter for AI assistants
- **Kiro**: AI coding assistant that uses YAML frontmatter with `inclusion` and `fileMatchPattern` fields
- **RuleTool**: genkit interface that tools implement to provide AI documentation
- **Adapter**: Code that converts between different AI rule formats
- **Source Rules**: Canonical English rules stored in `cmd/*/rules/` directories
- **Generated Rules**: AI-specific rules generated from source rules (e.g., `.kiro/steering/`, `.codebuddy/rules/`)

## Requirements

### Requirement 1: Standardize Source Rules Format

**User Story:** As a devgen maintainer, I want all source rules to be in English with consistent structure, so that they can be easily translated and adapted for different AI assistants.

#### Acceptance Criteria

1. WHEN source rules are stored in `cmd/*/rules/` directories, THE System SHALL use English language for all content
2. WHEN a source rule is created, THE System SHALL include comprehensive sections: "When to Use", "Quick Start", "Complete Example", and "Common Errors"
3. WHEN source rules reference `go generate`, THE System SHALL limit mentions to single-file use cases only
4. THE System SHALL NOT include YAML frontmatter in source rules files
5. THE System SHALL maintain source rules as canonical documentation that adapters transform

### Requirement 2: Create Kiro Rules Adapter

**User Story:** As a Kiro user, I want devgen rules to automatically work with Kiro's steering system, so that I get context-aware code generation assistance.

#### Acceptance Criteria

1. WHEN the `devgen rules --agent kiro -w` command is executed, THE System SHALL generate rules files in `.kiro/steering/` directory
2. WHEN generating Kiro rules, THE System SHALL add YAML frontmatter with `inclusion` and `fileMatchPattern` fields
3. WHEN a rule applies to Go files, THE System SHALL set `fileMatchPattern: ['**/*.go']` in the frontmatter
4. WHEN a rule applies to configuration files, THE System SHALL set `fileMatchPattern: ['**/devgen.toml', '**/*.go']` in the frontmatter
5. WHEN a rule should always be available, THE System SHALL set `inclusion: always` in the frontmatter
6. WHEN a rule should load based on file patterns, THE System SHALL set `inclusion: fileMatch` in the frontmatter
7. THE System SHALL preserve all markdown content from source rules in generated Kiro rules

### Requirement 3: Support Multiple AI Agents

**User Story:** As a developer using different AI assistants, I want devgen to generate appropriate rule formats for each assistant, so that I get consistent help regardless of which tool I use.

#### Acceptance Criteria

1. WHEN listing agents with `devgen rules --list-agents`, THE System SHALL display all supported AI assistants including Kiro
2. WHEN generating rules for an agent, THE System SHALL apply agent-specific formatting and frontmatter
3. THE System SHALL support at minimum: Kiro, CodeBuddy, and Cursor agents
4. WHEN an agent requires specific frontmatter fields, THE System SHALL include those fields in generated rules
5. THE System SHALL maintain a registry of agent adapters that can be extended

### Requirement 4: Refine Existing Rules Content

**User Story:** As an AI assistant, I want devgen rules to be comprehensive and well-structured, so that I can provide accurate code generation guidance.

#### Acceptance Criteria

1. WHEN a rule document is created, THE System SHALL include step-by-step instructions with code examples
2. WHEN documenting annotations, THE System SHALL use ❌/✅ comparisons for common errors
3. WHEN providing examples, THE System SHALL include complete, runnable code
4. THE System SHALL NOT recommend `go generate` broadly in workflow sections
5. WHERE `go generate` is mentioned, THE System SHALL limit it to single-file integration examples
6. WHEN documenting tool usage, THE System SHALL prioritize direct `devgen` command usage

### Requirement 5: Implement RuleTool Adapter Interface

**User Story:** As a plugin developer, I want to implement RuleTool once and have it work with all AI assistants, so that I don't need to maintain multiple rule formats.

#### Acceptance Criteria

1. WHEN a tool implements RuleTool interface, THE System SHALL provide adapter functions to convert rules to agent-specific formats
2. WHEN converting rules, THE System SHALL preserve the `Name`, `Description`, `Globs`, and `Content` fields
3. WHEN generating agent-specific rules, THE System SHALL map `Globs` to agent-specific file pattern fields
4. WHEN generating agent-specific rules, THE System SHALL map `AlwaysApply` to agent-specific inclusion settings
5. THE System SHALL provide a `ConvertRuleToKiro(rule genkit.Rule) string` function that adds appropriate YAML frontmatter

### Requirement 6: Update Rules Command Implementation

**User Story:** As a devgen user, I want the `devgen rules` command to generate properly formatted rules for my AI assistant, so that I can immediately use them without manual editing.

#### Acceptance Criteria

1. WHEN executing `devgen rules --agent kiro -w`, THE System SHALL create `.kiro/steering/` directory if it does not exist
2. WHEN writing rules files, THE System SHALL use the pattern `{rule-name}.md` for filenames
3. WHEN a rule file already exists, THE System SHALL overwrite it with updated content
4. WHEN rules generation completes, THE System SHALL log the number of files generated and their paths
5. THE System SHALL validate that generated files have correct YAML frontmatter syntax
