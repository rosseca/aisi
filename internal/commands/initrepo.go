package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var initRepoCmd = &cobra.Command{
	Use:   "init-repo [name]",
	Short: "Initialize a new shared intelligence repository skeleton",
	Long: `Creates the complete directory structure for a new AISI shared repository.

This command generates:
  - All required directories (rules, skills, agents, hooks, mcp, agents-md)
  - .gitkeep files to preserve empty directories
  - A base rule for manifest.yaml generation in .cursor/rules/
  - A comprehensive README.md
  - An example manifest.yaml

Examples:
  aisi init-repo my-shared-rules
  aisi init-repo .
  aisi init-repo ../company-ai-assets`,
	Args: cobra.ExactArgs(1),
	RunE: runInitRepo,
}

var repoDirs = []string{
	"rules",
	"skills",
	"agents",
	"hooks/scripts",
	"mcp",
	"agents-md",
	".cursor/rules",
}

const manifestRuleTemplate = `# AISI Manifest Generation Rule

This rule helps AI assistants generate proper manifest.yaml files for AISI repositories.

## Activation

Apply when:
- User asks to create or update a manifest
- User mentions "manifest.yaml" or "manifest.yml"
- Working with AISI/shared intelligence repositories

## Manifest Structure

The manifest.yaml must follow this schema:

` + "```yaml" + `
version: "1.0.0"                          # Required: semantic version of the manifest
minimumCliVersion: "0.5.0"               # Optional: minimum AISI CLI version

rules:                                     # Cursor/Kilo rules (.mdc files)
  - name: rule-name
    path: rules/rule-name.mdc
    description: What this rule does
    alwaysApply: false                     # Optional: auto-apply to all chats
    globs: ["**/*.go"]                     # Optional: file patterns

skills:                                    # Skill directories with SKILL.md
  - name: skill-name
    path: skills/skill-name
    description: What this skill provides

agents:                                    # Subagent definitions
  - name: agent-name
    path: agents/agent-name.md
    description: What this agent does
    model: fast                            # Optional: default model

hooks:                                     # Git hooks
  - name: hook-name
    configPath: hooks/hooks.json
    scriptsPath: hooks/scripts
    description: What these hooks do

mcp:                                       # MCP server configs
  - name: mcp-name
    path: mcp/config.json
    description: MCP server description
    env:                                   # Required environment variables
      API_KEY:
        description: API key for service
        required: true
        secret: true                       # Mark as secret

agentsMd:                                  # AGENTS.md templates
  - name: default
    path: agents-md/default.md
    description: Default AGENTS.md template

external:                                  # External repo references
  - name: external-skill
    type: skill                            # skill, agent, rule, hook, mcp
    repo: github.com/org/repo
    path: path/in/repo
    ref: main                              # Optional: branch/tag
    description: External asset description
` + "```" + `

## Rules

1. **Path Validation**: All paths must be relative to repository root
2. **Unique Names**: Asset names must be unique across their type
3. **Required Fields**: name, path, description are mandatory
4. **Valid Types**: rule, skill, agent, hook, mcp, agentsMd
5. **Version Format**: Use semantic versioning (e.g., "1.0.0")

## When Generating

1. Ask user for repository purpose if unclear
2. Suggest appropriate asset structure
3. Include at least one example of each type they need
4. Add helpful comments in YAML for complex fields
5. Validate that referenced paths exist in the repo`

const exampleManifestTemplate = `version: "1.0.0"
minimumCliVersion: "0.5.0"

# Cursor/Kilo Code rules (.mdc files)
rules:
  - name: example-rule
    path: rules/example.mdc
    description: An example rule demonstrating AISI format
    alwaysApply: false
    globs: ["**/*.go"]

# Skill directories (each contains SKILL.md)
skills:
  - name: example-skill
    path: skills/example-skill
    description: An example skill with documentation

# Subagent definitions
agents:
  - name: example-agent
    path: agents/example-agent.md
    description: An example subagent configuration
    model: fast

# Git hooks configuration
hooks:
  - name: example-hooks
    configPath: hooks/hooks.json
    scriptsPath: hooks/scripts
    description: Example git hooks setup

# MCP server configurations
mcp:
  - name: example-mcp
    path: mcp/example.json
    description: Example MCP server configuration
    env:
      EXAMPLE_KEY:
        description: Example API key
        required: true
        secret: true

# AGENTS.md templates
agentsMd:
  - name: default
    path: agents-md/default.md
    description: Default project AGENTS.md template

# External repository references
external: []
  # Example:
  # - name: shared-utils
  #   type: skill
  #   repo: github.com/your-org/shared-skills
  #   path: utils
  #   ref: main
  #   description: Shared utility skills`

func init() {
	rootCmd.AddCommand(initRepoCmd)
}

func runInitRepo(cmd *cobra.Command, args []string) error {
	repoPath := args[0]

	// Resolve path
	if repoPath == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		repoPath = cwd
	}

	// Clean and validate path
	repoPath = filepath.Clean(repoPath)

	// Check if directory exists and is not empty
	if err := validateDirectory(repoPath); err != nil {
		return err
	}

	fmt.Printf("🏗️  Initializing AISI repository at: %s\n\n", repoPath)

	// Create directory structure
	if err := createDirectoryStructure(repoPath); err != nil {
		return err
	}

	// Create .gitkeep files
	if err := createGitkeepFiles(repoPath); err != nil {
		return err
	}

	// Create manifest rule
	if err := createManifestRule(repoPath); err != nil {
		return err
	}

	// Create example manifest
	if err := createExampleManifest(repoPath); err != nil {
		return err
	}

	// Create README
	if err := createReadme(repoPath); err != nil {
		return err
	}

	fmt.Println("\n✅ Repository skeleton created successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and customize manifest.yaml")
	fmt.Println("  2. Add your rules to rules/ directory")
	fmt.Println("  3. Create skills in skills/ directories (with SKILL.md)")
	fmt.Println("  4. Initialize git and push to remote:")
	fmt.Println("     git init")
	fmt.Println("     git add .")
	fmt.Println("     git commit -m 'Initial AISI repository'")
	fmt.Println("     git remote add origin <your-repo-url>")
	fmt.Println("     git push -u origin main")
	fmt.Println("\n  5. Configure AISI to use this repo:")
	fmt.Println("     aisi config set-repo <your-repo-url>")

	return nil
}

func validateDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the directory
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			return nil
		}
		return fmt.Errorf("failed to access path %s: %w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	// Check if directory is empty
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	// Allow .git directory
	nonGitEntries := 0
	for _, entry := range entries {
		if entry.Name() != ".git" {
			nonGitEntries++
		}
	}

	if nonGitEntries > 0 {
		return fmt.Errorf("directory %s is not empty (excluding .git). Use an empty directory or '.' to initialize in current directory", path)
	}

	return nil
}

func createDirectoryStructure(repoPath string) error {
	fmt.Println("📁 Creating directory structure...")

	for _, dir := range repoDirs {
		fullPath := filepath.Join(repoPath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("   ✓ %s/\n", dir)
	}

	return nil
}

func createGitkeepFiles(repoPath string) error {
	fmt.Println("\n📝 Adding .gitkeep files...")

	// Add .gitkeep to all main directories except .cursor
	keepDirs := []string{
		"rules",
		"skills",
		"agents",
		"hooks/scripts",
		"mcp",
		"agents-md",
	}

	for _, dir := range keepDirs {
		gitkeepPath := filepath.Join(repoPath, dir, ".gitkeep")
		content := fmt.Sprintf("# %s directory\n# Add your files here\n", filepath.Base(dir))
		if err := os.WriteFile(gitkeepPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create .gitkeep in %s: %w", dir, err)
		}
	}

	fmt.Println("   ✓ Added .gitkeep to all directories")
	return nil
}

func createManifestRule(repoPath string) error {
	fmt.Println("\n📋 Creating manifest generation rule...")

	rulePath := filepath.Join(repoPath, ".cursor/rules", "manifest-generation.mdc")

	if err := os.WriteFile(rulePath, []byte(manifestRuleTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create manifest rule: %w", err)
	}

	fmt.Printf("   ✓ %s\n", rulePath)
	return nil
}

func createExampleManifest(repoPath string) error {
	fmt.Println("\n📦 Creating example manifest.yaml...")

	manifestPath := filepath.Join(repoPath, "manifest.yaml")

	// Check if manifest already exists (any variant)
	variants := []string{"manifest.yaml", "manifest.yml", "manifest.json"}
	for _, variant := range variants {
		if _, err := os.Stat(filepath.Join(repoPath, variant)); err == nil {
			fmt.Printf("   ⚠️  %s already exists, skipping\n", variant)
			return nil
		}
	}

	if err := os.WriteFile(manifestPath, []byte(exampleManifestTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create manifest.yaml: %w", err)
	}

	fmt.Printf("   ✓ %s\n", manifestPath)
	return nil
}

func createReadme(repoPath string) error {
	fmt.Println("\n📖 Creating README.md...")

	readmePath := filepath.Join(repoPath, "README.md")

	// Check if README already exists
	if _, err := os.Stat(readmePath); err == nil {
		fmt.Println("   ⚠️  README.md already exists, skipping")
		return nil
	}

	// Get directory name for title
	dirName := filepath.Base(repoPath)
	title := strings.ReplaceAll(dirName, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ToUpper(title[:1]) + title[1:]

	readmeContent := fmt.Sprintf(`# %s

AISI Shared Intelligence Repository for AI coding assistants.

## Overview

This repository contains shared assets for AI coding assistants (Cursor, Kilo Code, Junie, Windsurf) managed through [AISI](https://github.com/rosseca/aisi) CLI.

## Repository Structure

`+"```"+`
.
├── rules/              # Cursor/Kilo Code rules (.mdc files)
│   └── *.mdc
├── skills/             # Skill directories with SKILL.md
│   └── skill-name/
│       └── SKILL.md
├── agents/             # Subagent definitions (.md files)
│   └── *.md
├── hooks/              # Git hooks configuration
│   ├── hooks.json
│   └── scripts/
├── mcp/                # MCP server configurations
│   └── *.json
├── agents-md/          # AGENTS.md templates
│   └── *.md
├── manifest.yaml       # Asset catalog and metadata
└── README.md          # This file
`+"```"+`

## Quick Start

### For Repository Maintainers

1. **Add Rules**: Create .mdc files in ` + "`rules/`" + `
   - Follow Cursor rule format (frontmatter + content)
   - Update ` + "`manifest.yaml`" + ` with rule metadata

2. **Add Skills**: Create directories in ` + "`skills/`" + ` with SKILL.md
   - Document the skill's purpose and usage
   - Include examples and best practices

3. **Update Manifest**: Edit ` + "`manifest.yaml`" + ` to register new assets

4. **Version Control**: Commit and push changes
   `+"```bash"+`
   git add .
   git commit -m "Add new assets"
   git push
   `+"```"+`

### For Repository Consumers

1. **Install AISI CLI**:
   `+"```bash"+`
   go install github.com/rosseca/aisi/cmd/aisi@latest
   `+"```"+`

2. **Configure Repository**:
   `+"```bash"+`
   aisi config set-repo <this-repo-url>
   aisi config set-target cursor  # or kilo, junie, windsurf
   `+"```"+`

3. **Install Assets**:
   `+"```bash"+`
   aisi list          # See available assets
   aisi install       # Interactive installation
   aisi install soul  # Install specific asset
   `+"```"+`

## Available Assets

### Rules

| Name | Description | Auto-Apply |
|------|-------------|------------|
| example-rule | An example rule | No |

### Skills

| Name | Description |
|------|-------------|
| example-skill | An example skill |

### Agents

| Name | Description | Model |
|------|-------------|-------|
| example-agent | An example agent | fast |

## Contributing

1. Follow the existing structure and naming conventions
2. Document all assets clearly in their respective files
3. Update this README when adding major new assets
4. Test assets locally before pushing

## License

[Your License Here]

---

Generated by AISI CLI on %s
`, title, time.Now().Format("2006-01-02"))

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	fmt.Printf("   ✓ %s\n", readmePath)
	return nil
}
