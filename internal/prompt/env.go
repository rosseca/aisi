package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rosseca/aisi/internal/installer"
	"github.com/rosseca/aisi/internal/manifest"
)

// PromptForEnvVars interactively prompts the user for environment variables
func PromptForEnvVars(mcp *manifest.MCP) (map[string]installer.EnvVarConfig, error) {
	if len(mcp.Env) == 0 {
		return nil, nil
	}

	envVars := make(map[string]installer.EnvVarConfig)
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n🔧 MCP '%s' requires environment variables:\n\n", mcp.Name)

	for varName, meta := range mcp.Env {
		fmt.Printf("Variable: %s\n", varName)
		if meta.Description != "" {
			fmt.Printf("  Description: %s\n", meta.Description)
		}
		if meta.Example != "" {
			fmt.Printf("  Example: %s\n", meta.Example)
		}
		if meta.HelpURL != "" {
			fmt.Printf("  Help: %s\n", meta.HelpURL)
		}
		if meta.Required {
			fmt.Println("  Required: yes")
		}

		// Ask for input method
		fmt.Println("\n  Options:")
		fmt.Println("    1. Enter value directly (stored in config)")
		fmt.Println("    2. Use environment variable reference (${env:VAR_NAME})")
		fmt.Print("  Select (1/2) [1]: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice == "" {
			choice = "1"
		}

		var cfg installer.EnvVarConfig
		cfg.VarName = varName

		switch choice {
		case "2":
			cfg.UseEnv = true
			fmt.Printf("  Using: ${env:%s}\n", varName)
		default:
			// Prompt for value
			prompt := "  Enter value"
			if meta.Secret {
				prompt += " (input hidden)"
			}
			prompt += ": "
			fmt.Print(prompt)

			var value string
			if meta.Secret {
				// For secrets, we still show input but mask it after
				value, _ = reader.ReadString('\n')
				value = strings.TrimSpace(value)
				fmt.Println("  ✓ Value set (hidden)")
			} else {
				value, _ = reader.ReadString('\n')
				value = strings.TrimSpace(value)
			}

			if value == "" && meta.Required {
				fmt.Println("  ⚠️  Warning: This variable is required but empty")
			}

			cfg.Value = value
			cfg.UseEnv = false
		}

		envVars[varName] = cfg
		fmt.Println()
	}

	return envVars, nil
}

// PromptForEnvVarsSimple is a simplified version for non-interactive use
func PromptForEnvVarsSimple(mcp *manifest.MCP) (map[string]installer.EnvVarConfig, error) {
	if len(mcp.Env) == 0 {
		return nil, nil
	}

	envVars := make(map[string]installer.EnvVarConfig)
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n🔧 Environment variables for '%s':\n", mcp.Name)

	for varName, meta := range mcp.Env {
		prompt := fmt.Sprintf("  %s", varName)
		if meta.Required {
			prompt += " (required)"
		}
		prompt += ": "
		fmt.Print(prompt)

		value, _ := reader.ReadString('\n')
		value = strings.TrimSpace(value)

		envVars[varName] = installer.EnvVarConfig{
			VarName: varName,
			Value:   value,
			UseEnv:  false,
		}
	}

	return envVars, nil
}
