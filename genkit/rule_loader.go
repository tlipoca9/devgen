package genkit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RuleFrontmatter represents the YAML frontmatter of a rule file.
type RuleFrontmatter struct {
	Description string   `yaml:"description"`
	Globs       []string `yaml:"globs"`
	AlwaysApply bool     `yaml:"alwaysApply"`
}

// LoadRulesFromDir loads all rule files from a directory.
// Rule files must be .md files with YAML frontmatter.
func LoadRulesFromDir(dir string) ([]Rule, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, nil // Directory doesn't exist, return empty
	}
	if err != nil {
		return nil, fmt.Errorf("stat directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	// Find all .md files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	var rules []Rule
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		rule, err := LoadRuleFromFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("load rule %s: %w", filePath, err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// LoadRuleFromFile loads a single rule from a markdown file with YAML frontmatter.
func LoadRuleFromFile(path string) (Rule, error) {
	file, err := os.Open(path)
	if err != nil {
		return Rule{}, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)

	// Parse frontmatter
	frontmatter, contentStart, err := parseFrontmatter(scanner)
	if err != nil {
		return Rule{}, fmt.Errorf("parse frontmatter: %w", err)
	}

	// Read remaining content
	var contentBuilder strings.Builder
	if contentStart != "" {
		contentBuilder.WriteString(contentStart)
		contentBuilder.WriteString("\n")
	}
	for scanner.Scan() {
		contentBuilder.WriteString(scanner.Text())
		contentBuilder.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return Rule{}, fmt.Errorf("read content: %w", err)
	}

	// Extract rule name from filename (without extension)
	name := strings.TrimSuffix(filepath.Base(path), ".md")

	return Rule{
		Name:        name,
		Description: frontmatter.Description,
		Globs:       frontmatter.Globs,
		AlwaysApply: frontmatter.AlwaysApply,
		Content:     strings.TrimSpace(contentBuilder.String()),
	}, nil
}

// parseFrontmatter parses YAML frontmatter from a scanner.
// Returns the parsed frontmatter and the first line of content (if any).
func parseFrontmatter(scanner *bufio.Scanner) (RuleFrontmatter, string, error) {
	var frontmatter RuleFrontmatter

	// Check for opening ---
	if !scanner.Scan() {
		return frontmatter, "", nil // Empty file
	}
	firstLine := scanner.Text()
	if strings.TrimSpace(firstLine) != "---" {
		// No frontmatter, return empty frontmatter and first line as content
		return frontmatter, firstLine, nil
	}

	// Read frontmatter lines until closing ---
	var yamlBuilder strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		yamlBuilder.WriteString(line)
		yamlBuilder.WriteString("\n")
	}

	// Parse YAML
	if yamlBuilder.Len() > 0 {
		if err := yaml.Unmarshal([]byte(yamlBuilder.String()), &frontmatter); err != nil {
			return frontmatter, "", fmt.Errorf("parse yaml: %w", err)
		}
	}

	return frontmatter, "", nil
}
