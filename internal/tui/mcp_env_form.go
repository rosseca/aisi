package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rosseca/aisi/internal/installer"
	"github.com/rosseca/aisi/internal/manifest"
)

// MCPEnvForm is a form for collecting MCP environment variables
type MCPEnvForm struct {
	mcp          *manifest.MCP
	inputs       []textinput.Model
	focused      int
	useEnvRef    []bool // Whether to use ${env:VAR} for each var
	varNames     []string
	descriptions []string
	required     []bool
	helpTexts    []string
	width        int
	height       int
	canceled     bool
}

// MCPEnvCompletedMsg is sent when the form is completed
type MCPEnvCompletedMsg struct {
	MCP      *manifest.MCP
	EnvVars  map[string]installer.EnvVarConfig
	Canceled bool
}

func NewMCPEnvForm(mcp *manifest.MCP) *MCPEnvForm {
	// Sort variable names to ensure consistent order
	varNames := make([]string, 0, len(mcp.Env))
	for name := range mcp.Env {
		varNames = append(varNames, name)
	}
	sort.Strings(varNames)

	descriptions := make([]string, 0, len(mcp.Env))
	required := make([]bool, 0, len(mcp.Env))
	helpTexts := make([]string, 0, len(mcp.Env))

	for _, name := range varNames {
		meta := mcp.Env[name]
		descriptions = append(descriptions, meta.Description)
		required = append(required, meta.Required)

		help := ""
		if meta.Example != "" {
			help += fmt.Sprintf("Example: %s ", meta.Example)
		}
		if meta.HelpURL != "" {
			help += fmt.Sprintf("Help: %s", meta.HelpURL)
		}
		helpTexts = append(helpTexts, help)
	}

	inputs := make([]textinput.Model, len(varNames))
	useEnvRef := make([]bool, len(varNames))

	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Placeholder = "Enter value or press tab to use ${env:VAR}"
		if i == 0 {
			inputs[i].Focus()
		}
	}

	return &MCPEnvForm{
		mcp:          mcp,
		inputs:       inputs,
		focused:      0,
		useEnvRef:    useEnvRef,
		varNames:     varNames,
		descriptions: descriptions,
		required:     required,
		helpTexts:    helpTexts,
	}
}

func (f *MCPEnvForm) Init() tea.Cmd {
	return textinput.Blink
}

func (f *MCPEnvForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			f.canceled = true
			return f, func() tea.Msg {
				return MCPEnvCompletedMsg{Canceled: true}
			}

		case "enter":
			// If on last field, submit
			if f.focused == len(f.inputs)-1 {
				return f, f.submit()
			}
			// Move to next field
			f.focused++
			for i := range f.inputs {
				if i == f.focused {
					f.inputs[i].Focus()
				} else {
					f.inputs[i].Blur()
				}
			}

		case "tab":
			// Toggle env ref mode for current field
			f.useEnvRef[f.focused] = !f.useEnvRef[f.focused]
			if f.useEnvRef[f.focused] {
				f.inputs[f.focused].SetValue(fmt.Sprintf("${env:%s}", f.varNames[f.focused]))
				f.inputs[f.focused].Blur()
			} else {
				f.inputs[f.focused].SetValue("")
				f.inputs[f.focused].Focus()
			}

		case "up":
			if f.focused > 0 {
				f.focused--
				for i := range f.inputs {
					if i == f.focused {
						f.inputs[i].Focus()
					} else {
						f.inputs[i].Blur()
					}
				}
			}

		case "down":
			if f.focused < len(f.inputs)-1 {
				f.focused++
				for i := range f.inputs {
					if i == f.focused {
						f.inputs[i].Focus()
					} else {
						f.inputs[i].Blur()
					}
				}
			}
		}
	}

	// Update focused input
	if f.focused < len(f.inputs) {
		newInput, cmd := f.inputs[f.focused].Update(msg)
		f.inputs[f.focused] = newInput
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

func (f *MCPEnvForm) submit() tea.Cmd {
	envVars := make(map[string]installer.EnvVarConfig)

	for i, varName := range f.varNames {
		value := f.inputs[i].Value()
		envVars[varName] = installer.EnvVarConfig{
			VarName: varName,
			Value:   value,
			UseEnv:  f.useEnvRef[i] || strings.HasPrefix(value, "${env:"),
		}
	}

	return func() tea.Msg {
		return MCPEnvCompletedMsg{
			MCP:      f.mcp,
			EnvVars:  envVars,
			Canceled: false,
		}
	}
}

func (f *MCPEnvForm) SetSize(width, height int) {
	f.width = width
	f.height = height
	for i := range f.inputs {
		f.inputs[i].Width = width - 20
	}
}

func (f *MCPEnvForm) View() string {
	if f.width == 0 {
		f.width = 80
	}

	title := titleStyle.Render("  🔧 MCP Configuration  ")

	// Fixed dimensions for compact view
	boxWidth := 70
	visibleFields := 4 // Show max 4 fields at a time
	fieldHeight := 6   // Approximate height per field

	var content strings.Builder
	content.WriteString(fmt.Sprintf("MCP: %s\n", f.mcp.Name))

	if f.mcp.Description != "" {
		content.WriteString(dimStyle.Render(f.mcp.Description))
		content.WriteString("\n")
	}

	content.WriteString(fmt.Sprintf("Variables [%d/%d]:\n\n", f.focused+1, len(f.varNames)))

	// Calculate which fields to show (scroll window)
	startIdx := 0
	if f.focused >= visibleFields {
		startIdx = f.focused - visibleFields + 1
	}
	endIdx := startIdx + visibleFields
	if endIdx > len(f.varNames) {
		endIdx = len(f.varNames)
	}

	// Show scroll indicator if needed
	if startIdx > 0 {
		content.WriteString(dimStyle.Render("  ▲ more above\n"))
	}

	for i := startIdx; i < endIdx; i++ {
		varName := f.varNames[i]
		desc := f.descriptions[i]
		required := f.required[i]
		helpText := f.helpTexts[i]
		isFocused := i == f.focused

		// Compact field display
		label := varName
		if required {
			label += " *"
		}

		if isFocused {
			content.WriteString(selectedItemStyle.Render("❯ " + label))
		} else {
			content.WriteString(fmt.Sprintf("  %s", dimStyle.Render(label)))
		}
		content.WriteString("\n")

		if desc != "" && isFocused {
			content.WriteString(dimStyle.Render(fmt.Sprintf("    %s", desc)))
			content.WriteString("\n")
		}

		// Input line
		if f.useEnvRef[i] {
			content.WriteString(fmt.Sprintf("    %s", successStyle.Render(fmt.Sprintf("${env:%s}", varName))))
		} else {
			inputView := f.inputs[i].View()
			if inputView == "" {
				inputView = dimStyle.Render("_")
			}
			content.WriteString(fmt.Sprintf("    %s", inputView))
		}
		content.WriteString("\n")

		if helpText != "" && isFocused {
			content.WriteString(dimStyle.Render(fmt.Sprintf("    %s", helpText)))
			content.WriteString("\n")
		}

		if isFocused {
			content.WriteString(dimStyle.Render("    tab: toggle env ref • enter: next/submit"))
			content.WriteString("\n")
		}

		content.WriteString("\n")
	}

	// Show scroll indicator if more below
	if endIdx < len(f.varNames) {
		content.WriteString(dimStyle.Render("  ▼ more below\n"))
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("esc: cancel • up/down: navigate"))

	// Fixed height box with padding
	boxContent := content.String()
	box := boxStyle.Width(boxWidth).Height(fieldHeight * visibleFields).Render(boxContent)

	return lipgloss.Place(
		f.width,
		f.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			"",
			box,
		),
	)
}
