package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/targets"
)

type AssetItem struct {
	Name        string
	Description string
	Type        manifest.AssetType
	IsExternal  bool
	Selected    bool
}

type Browser struct {
	items        []AssetItem
	cursor       int
	target       *targets.Target
	manifest     *manifest.Manifest
	width        int
	height       int
	visibleItems int // Calculated based on terminal height
}

func NewBrowser(m *manifest.Manifest, target *targets.Target) *Browser {
	items := make([]AssetItem, 0)

	if target.RulesDir != "" {
		for _, r := range m.Rules {
			items = append(items, AssetItem{
				Name:        r.Name,
				Description: r.Description,
				Type:        manifest.AssetTypeRule,
				IsExternal:  false,
			})
		}
	}

	if target.SkillsDir != "" {
		for _, s := range m.Skills {
			items = append(items, AssetItem{
				Name:        s.Name,
				Description: s.Description,
				Type:        manifest.AssetTypeSkill,
				IsExternal:  false,
			})
		}

		for _, e := range m.External {
			if e.Type == "skill" {
				items = append(items, AssetItem{
					Name:        e.Name,
					Description: e.Description,
					Type:        manifest.AssetTypeSkill,
					IsExternal:  true,
				})
			}
		}
	}

	if target.AgentsDir != "" {
		for _, a := range m.Agents {
			items = append(items, AssetItem{
				Name:        a.Name,
				Description: a.Description,
				Type:        manifest.AssetTypeAgent,
				IsExternal:  false,
			})
		}

		for _, e := range m.External {
			if e.Type == "agent" {
				items = append(items, AssetItem{
					Name:        e.Name,
					Description: e.Description,
					Type:        manifest.AssetTypeAgent,
					IsExternal:  true,
				})
			}
		}
	}

	if target.HooksFile != "" {
		for _, h := range m.Hooks {
			items = append(items, AssetItem{
				Name:        h.Name,
				Description: h.Description,
				Type:        manifest.AssetTypeHook,
				IsExternal:  false,
			})
		}
	}

	if target.MCPFile != "" {
		for _, mc := range m.MCP {
			items = append(items, AssetItem{
				Name:        mc.Name,
				Description: mc.Description,
				Type:        manifest.AssetTypeMCP,
				IsExternal:  false,
			})
		}
	}

	if target.SupportsAgentsMD {
		for _, am := range m.AgentsMD {
			items = append(items, AssetItem{
				Name:        am.Name,
				Description: am.Description,
				Type:        manifest.AssetTypeAgentsMD,
				IsExternal:  false,
			})
		}
	}

	return &Browser{
		items:        items,
		cursor:       0,
		target:       target,
		manifest:     m,
		width:        80,
		height:       24,
		visibleItems: 8, // Default, will be recalculated on first render
	}
}

func (b *Browser) Init() tea.Cmd {
	return nil
}

func (b *Browser) SetSize(width, height int) {
	b.width = width
	b.height = height
	// Recalculate visible items
	// Reserve space for:
	//   Before items: initial \n(1) + title(1) + \n(1) + subtitle(1) + \n\n(2) = 6 lines
	//   After items: \n(1) + selected count(1) + help(1) + \n(1) = 4 lines
	//   Plus scroll indicators (2) and category headers (variable)
	reservedLines := 11
	b.visibleItems = height - reservedLines
	if b.visibleItems < 4 {
		b.visibleItems = 4 // Minimum
	}
	// No maximum limit - show as many as fit in the terminal
}

func (b *Browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
		// Calculate visible items based on height
		// Reserve space for:
		//   Before items: initial \n(1) + title(1) + \n(1) + subtitle(1) + \n\n(2) = 6 lines
		//   After items: \n(1) + selected count(1) + help(1) + \n(1) = 4 lines
		//   Plus scroll indicators (2) and category headers (variable)
		reservedLines := 11
		b.visibleItems = b.height - reservedLines
		if b.visibleItems < 4 {
			b.visibleItems = 4 // Minimum
		}
		// No maximum limit - show as many as fit in the terminal
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return b, func() tea.Msg { return BrowserDoneMsg{} }
		case "up", "k":
			if b.cursor > 0 {
				b.cursor--
			}
		case "down", "j":
			if b.cursor < len(b.items)-1 {
				b.cursor++
			}
		case " ":
			if len(b.items) > 0 {
				b.items[b.cursor].Selected = !b.items[b.cursor].Selected
			}
		case "a":
			for i := range b.items {
				b.items[i].Selected = true
			}
		case "n":
			for i := range b.items {
				b.items[i].Selected = false
			}
		case "enter":
			return b, b.installSelected
		}
	}
	return b, nil
}

func (b *Browser) installSelected() tea.Msg {
	selected := make([]AssetItem, 0)
	for _, item := range b.items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return InstallRequestMsg{Items: selected}
}

func (b *Browser) View() string {
	var s strings.Builder

	s.WriteString("\n")
	s.WriteString(titleStyle.Render("Select assets to install"))
	s.WriteString("\n")
	s.WriteString(subtitleStyle.Render("Target: " + b.target.DisplayName))
	s.WriteString("\n\n")

	if len(b.items) == 0 {
		s.WriteString(dimStyle.Render("  No assets available for this target"))
		s.WriteString("\n")
	} else {
		// Scroll window: dynamic based on terminal height
		startIdx := 0
		if b.cursor >= b.visibleItems {
			startIdx = b.cursor - b.visibleItems + 1
		}
		endIdx := startIdx + b.visibleItems
		if endIdx > len(b.items) {
			endIdx = len(b.items)
		}

		// Show scroll indicator if items above
		if startIdx > 0 {
			s.WriteString(dimStyle.Render("  ▲ more above"))
			s.WriteString("\n")
		}

		currentType := manifest.AssetType("")
		lastType := manifest.AssetType("")

		for i := startIdx; i < endIdx; i++ {
			item := b.items[i]

			// Show category header if changed
			if item.Type != currentType {
				currentType = item.Type
				// Add spacing between categories (except first)
				if lastType != "" {
					s.WriteString("\n")
				}
				s.WriteString(categoryStyle.Render(formatAssetType(currentType)))
				s.WriteString("\n")
				lastType = currentType
			}

			cursor := "  "
			if b.cursor == i {
				cursor = "> "
			}

			var checkbox string
			if item.Selected {
				checkbox = checkboxStyle.Render("[x]")
			} else {
				checkbox = uncheckboxStyle.Render("[ ]")
			}

			badge := ""
			if item.IsExternal {
				badge = externalBadgeStyle.Render(" [external]")
			}

			name := item.Name
			isFocused := b.cursor == i
			if isFocused {
				name = selectedItemStyle.Render(name)
			}

			// Show full description on a new line if focused, otherwise truncated inline
			if isFocused && item.Description != "" {
				// Main line with just name and badge
				line := fmt.Sprintf("%s%s %s%s", cursor, checkbox, name, badge)
				s.WriteString(line)
				s.WriteString("\n")

				// Description on next line(s), wrapped to terminal width
				wrappedDesc := wrapText(item.Description, b.width-6) // Indent by 6 spaces
				for _, descLine := range wrappedDesc {
					s.WriteString(dimStyle.Render("      " + descLine))
					s.WriteString("\n")
				}
			} else {
				// Truncated inline description
				desc := ""
				if item.Description != "" {
					maxDescLen := b.width - 50
					if maxDescLen < 20 {
						maxDescLen = 20
					}
					if maxDescLen > 60 {
						maxDescLen = 60
					}
					desc = dimStyle.Render(" - " + truncateDesc(item.Description, maxDescLen))
				}
				line := fmt.Sprintf("%s%s %s%s%s", cursor, checkbox, name, badge, desc)
				s.WriteString(line)
				s.WriteString("\n")
			}
		}

		// Show scroll indicator if items below
		if endIdx < len(b.items) {
			s.WriteString(dimStyle.Render("  ▼ more below"))
			s.WriteString("\n")
		}
	}

	selectedCount := 0
	for _, item := range b.items {
		if item.Selected {
			selectedCount++
		}
	}

	s.WriteString("\n")
	if selectedCount > 0 {
		s.WriteString(successStyle.Render(fmt.Sprintf("  %d selected", selectedCount)))
		s.WriteString("\n")
	}

	s.WriteString(helpStyle.Render("↑/↓: Navigate • Space: Toggle • a: All • n: None • Enter: Install • Esc: Back"))
	s.WriteString("\n")

	return s.String()
}

func truncateDesc(desc string, maxLen int) string {
	if len(desc) <= maxLen {
		return desc
	}
	return desc[:maxLen-3] + "..."
}

func wrapText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 > maxWidth {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Word is too long, split it
				lines = append(lines, word[:maxWidth])
				currentLine = word[maxWidth:]
			}
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

func formatAssetType(t manifest.AssetType) string {
	switch t {
	case manifest.AssetTypeRule:
		return "RULES"
	case manifest.AssetTypeSkill:
		return "SKILLS"
	case manifest.AssetTypeAgent:
		return "AGENTS"
	case manifest.AssetTypeHook:
		return "HOOKS"
	case manifest.AssetTypeMCP:
		return "MCP"
	case manifest.AssetTypeAgentsMD:
		return "AGENTS.MD"
	default:
		return string(t)
	}
}

type BrowserDoneMsg struct{}

type InstallRequestMsg struct {
	Items []AssetItem
}
