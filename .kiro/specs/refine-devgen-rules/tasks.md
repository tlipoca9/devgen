# Implementation Plan

- [x] 1. Refine source rules content in cmd/*/rules/
  - Update all rules to English with consistent structure
  - Remove broad go generate recommendations
  - Ensure comprehensive examples and error sections
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_

- [x] 1.1 Update cmd/devgen/rules/devgen.md
  - Translate from Chinese to English
  - Remove go generate workflow section
  - Add comprehensive "When to Use" section
  - Add step-by-step Quick Start
  - Add complete working example
  - Add common errors with ❌/✅ comparisons
  - _Requirements: 1.1, 1.2, 1.3, 4.1, 4.2, 4.3, 4.4, 4.5, 4.6_

- [x] 1.2 Update cmd/devgen/rules/devgen-plugin.md
  - Translate from Chinese to English
  - Ensure consistent structure with other rules
  - Add more code examples for each interface
  - Expand troubleshooting section
  - _Requirements: 1.1, 1.2, 4.1, 4.2, 4.3_

- [x] 1.3 Update cmd/devgen/rules/devgen-genkit.md
  - Translate from Chinese to English
  - Add more practical examples
  - Improve API reference organization
  - Add common usage patterns section
  - _Requirements: 1.1, 1.2, 4.1, 4.2, 4.3_

- [x] 1.4 Update cmd/devgen/rules/devgen-rules.md
  - Translate from Chinese to English
  - Update to reflect new adapter system
  - Add examples for each agent type
  - Document adapter interface for custom agents
  - _Requirements: 1.1, 1.2, 3.3, 4.1, 4.2, 4.3_

- [x] 1.5 Update cmd/enumgen/rules/enumgen.md
  - Translate from Chinese to English
  - Ensure consistent structure
  - Verify all examples are complete and runnable
  - _Requirements: 1.1, 1.2, 4.1, 4.2, 4.3_

- [x] 1.6 Update cmd/validategen/rules/validategen.md
  - Translate from Chinese to English
  - Ensure consistent structure
  - Verify all validation annotations are documented
  - Add more complete examples
  - _Requirements: 1.1, 1.2, 4.1, 4.2, 4.3_

- [x] 2. Implement adapter system in genkit package
  - Create adapter interface and registry
  - Implement adapters for Kiro, CodeBuddy, and Cursor
  - Add adapter tests
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 3.1, 3.2, 3.3, 3.4, 3.5, 5.1, 5.2, 5.3, 5.4, 5.5_

- [x] 2.1 Create genkit/adapter.go with AgentAdapter interface
  - Define AgentAdapter interface with Name(), OutputDir(), Transform() methods
  - Add documentation for interface usage
  - _Requirements: 5.1, 5.2_

- [x] 2.2 Create genkit/adapter_registry.go
  - Implement AdapterRegistry struct
  - Add Register(), Get(), List() methods
  - Initialize with built-in adapters
  - _Requirements: 3.5, 5.1_

- [x] 2.3 Create genkit/adapter_kiro.go
  - Implement KiroAdapter struct
  - Transform rules to Kiro format with YAML frontmatter
  - Handle inclusion types (always vs fileMatch)
  - Convert Globs to fileMatchPattern
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 5.3, 5.4, 5.5_

- [x] 2.4 Create genkit/adapter_codebuddy.go
  - Implement CodeBuddyAdapter struct
  - Maintain existing CodeBuddy frontmatter format
  - Map Rule fields to CodeBuddy format
  - _Requirements: 3.2, 3.3, 3.4, 5.3, 5.4_

- [x] 2.5 Create genkit/adapter_cursor.go
  - Implement CursorAdapter struct
  - Maintain existing Cursor frontmatter format
  - Map Rule fields to Cursor format
  - _Requirements: 3.2, 3.3, 3.4, 5.3, 5.4_

- [x] 2.6 Add unit tests for adapters
  - Test Kiro adapter frontmatter generation
  - Test pattern formatting
  - Test inclusion type selection
  - Test content preservation
  - Test CodeBuddy and Cursor adapters
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7_

- [x] 3. Update devgen rules command
  - Integrate adapter registry
  - Update --list-agents to show all adapters
  - Update rule generation to use adapters
  - Add proper error handling
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 3.1 Create cmd/devgen/rules_command.go
  - Implement RulesCommand struct
  - Add Execute() method with agent and write parameters
  - Implement collectRules() to gather rules from all tools
  - Implement preview() for stdout output
  - Implement writeRules() for file generation
  - _Requirements: 3.1, 3.2, 6.1, 6.2, 6.3, 6.4_

- [x] 3.2 Update cmd/devgen/main.go to use RulesCommand
  - Integrate RulesCommand into CLI
  - Update --list-agents flag handler
  - Update --agent flag handler
  - Update -w flag handler
  - Add error handling and user feedback
  - _Requirements: 3.1, 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 3.3 Add integration tests for rules command
  - Test rule collection from multiple tools
  - Test preview mode output
  - Test write mode file creation
  - Test error handling for unknown agents
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 6.1, 6.2, 6.3, 6.4_

- [x] 4. Generate and validate rules for all agents
  - Run devgen rules for each agent
  - Verify file creation and content
  - Validate frontmatter syntax
  - Test with actual AI assistants
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 4.1 Generate Kiro rules
  - Run `devgen rules --agent kiro -w`
  - Verify files created in .kiro/steering/
  - Validate YAML frontmatter syntax
  - Check inclusion and fileMatchPattern fields
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 4.2 Generate CodeBuddy rules
  - Run `devgen rules --agent codebuddy -w`
  - Verify files created in .codebuddy/rules/
  - Validate frontmatter matches existing format
  - _Requirements: 3.2, 3.3, 3.4, 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 4.3 Generate Cursor rules
  - Run `devgen rules --agent cursor -w`
  - Verify files created in .cursor/rules/
  - Validate frontmatter matches existing format
  - _Requirements: 3.2, 3.3, 3.4, 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 4.4 Validate all generated rules
  - Check all files have correct YAML frontmatter
  - Verify content is complete and properly formatted
  - Ensure no content loss during transformation
  - _Requirements: 6.5_

- [x] 5. Update documentation
  - Update README with new rules system
  - Add examples for each agent
  - Document adapter interface for custom agents
  - Update plugin development guide
  - _Requirements: 1.1, 1.2, 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 5.1 Update README.md
  - Add section on AI rules system
  - Document `devgen rules` command usage
  - List supported AI agents
  - Add examples for generating rules
  - _Requirements: 3.1, 3.2_

- [x] 5.2 Update docs/plugin.md
  - Document RuleTool interface
  - Add examples of implementing Rules() method
  - Explain how rules work with different agents
  - _Requirements: 1.1, 1.2, 3.3, 3.4_

- [x] 5.3 Create docs/rules-adapter.md
  - Document AgentAdapter interface
  - Provide examples of custom adapter implementation
  - Explain adapter registration process
  - _Requirements: 3.5, 5.1, 5.2, 5.3, 5.4, 5.5_
