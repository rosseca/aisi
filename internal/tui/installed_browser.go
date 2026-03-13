package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rosseca/aisi/internal/tracker"
)

type InstalledSection struct {
	Title string
	Items []InstalledItem
}

type InstalledItem struct {
	Name     string
	Type     string
	Selected bool
}

type InstalledBrowser struct {
	sections []InstalledSection
	cursor   struct {
		section int
		item    int
	}
	target  string
	tracker *tracker.Tracker
}

func NewInstalledBrowser(track *tracker.Tracker, target string) *InstalledBrowser {
	sections := make([]InstalledSection, 0)

	installed, _ := track.GetInstalled()
	if installed != nil {
		if len(installed.Rules) > 0 {
			items := make([]InstalledItem, len(installed.Rules))
			for i, r := range installed.Rules {
				items[i] = InstalledItem{Name: r, Type: "rule"}
			}
			sections = append(sections, InstalledSection{Title: "📋 RULES", Items: items})
		}

		if len(installed.Skills) > 0 {
			items := make([]InstalledItem, len(installed.Skills))
			for i, s := range installed.Skills {
				items[i] = InstalledItem{Name: s, Type: "skill"}
			}
			sections = append(sections, InstalledSection{Title: "🛠️  SKILLS", Items: items})
		}

		if len(installed.Agents) > 0 {
			items := make([]InstalledItem, len(installed.Agents))
			for i, a := range installed.Agents {
				items[i] = InstalledItem{Name: a, Type: "agent"}
			}
			sections = append(sections, InstalledSection{Title: "🤖 AGENTS", Items: items})
		}

		if len(installed.Hooks) > 0 {
			items := make([]InstalledItem, len(installed.Hooks))
			for i, h := range installed.Hooks {
				items[i] = InstalledItem{Name: h, Type: "hook"}
			}
			sections = append(sections, InstalledSection{Title: "🪝 HOOKS", Items: items})
		}

		if len(installed.MCP) > 0 {
			items := make([]InstalledItem, len(installed.MCP))
			for i, m := range installed.MCP {
				items[i] = InstalledItem{Name: m, Type: "mcp"}
			}
			sections = append(sections, InstalledSection{Title: "🔌 MCP", Items: items})
		}

		if len(installed.AgentsMD) > 0 {
			items := make([]InstalledItem, len(installed.AgentsMD))
			for i, am := range installed.AgentsMD {
				items[i] = InstalledItem{Name: am, Type: "agentsmd"}
			}
			sections = append(sections, InstalledSection{Title: "📄 AGENTS.MD", Items: items})
		}

		if len(installed.External) > 0 {
			items := make([]InstalledItem, len(installed.External))
			for i, e := range installed.External {
				items[i] = InstalledItem{Name: e, Type: "external"}
			}
			sections = append(sections, InstalledSection{Title: "🔗 EXTERNAL", Items: items})
		}
	}

	return &InstalledBrowser{
		sections: sections,
		cursor: struct {
			section int
			item    int
		}{section: 0, item: 0},
		target:  target,
		tracker: track,
	}
}

func (b *InstalledBrowser) getTotalItems() int {
	total := 0
	for _, section := range b.sections {
		total += len(section.Items)
	}
	return total
}

func (b *InstalledBrowser) getFlatIndex() int {
	idx := 0
	for s := 0; s < b.cursor.section; s++ {
		idx += len(b.sections[s].Items)
	}
	idx += b.cursor.item
	return idx
}

func (b *InstalledBrowser) setCursorFromFlatIndex(flatIdx int) {
	idx := 0
	for s, section := range b.sections {
		for i := range section.Items {
			if idx == flatIdx {
				b.cursor.section = s
				b.cursor.item = i
				return
			}
			idx++
		}
	}
}

func (b *InstalledBrowser) getCurrentItem() *InstalledItem {
	if b.cursor.section < len(b.sections) {
		section := &b.sections[b.cursor.section]
		if b.cursor.item < len(section.Items) {
			return &section.Items[b.cursor.item]
		}
	}
	return nil
}

func (b *InstalledBrowser) Init() tea.Cmd {
	return nil
}

func (b *InstalledBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return b, func() tea.Msg { return InstalledBrowserDoneMsg{} }
		case "up", "k":
			flatIdx := b.getFlatIndex()
			if flatIdx > 0 {
				b.setCursorFromFlatIndex(flatIdx - 1)
			}
		case "down", "j":
			flatIdx := b.getFlatIndex()
			if flatIdx < b.getTotalItems()-1 {
				b.setCursorFromFlatIndex(flatIdx + 1)
			}
		case " ":
			if item := b.getCurrentItem(); item != nil {
				item.Selected = !item.Selected
			}
		case "a":
			for s := range b.sections {
				for i := range b.sections[s].Items {
					b.sections[s].Items[i].Selected = true
				}
			}
		case "n":
			for s := range b.sections {
				for i := range b.sections[s].Items {
					b.sections[s].Items[i].Selected = false
				}
			}
		case "d", "delete", "backspace":
			return b, b.uninstallSelected
		case "enter":
			return b, b.uninstallSelected
		}
	}
	return b, nil
}

func (b *InstalledBrowser) uninstallSelected() tea.Msg {
	selected := make([]InstalledItem, 0)
	for _, section := range b.sections {
		for _, item := range section.Items {
			if item.Selected {
				selected = append(selected, item)
			}
		}
	}
	return UninstallRequestMsg{Items: selected}
}

func (b *InstalledBrowser) View() string {
	var s strings.Builder

	s.WriteString("\n")
	s.WriteString(titleStyle.Render("Installed Assets"))
	s.WriteString("\n")
	s.WriteString(subtitleStyle.Render("Select assets to uninstall"))
	s.WriteString("\n\n")

	if len(b.sections) == 0 {
		s.WriteString(dimStyle.Render("  No installed assets found.\n"))
		s.WriteString(dimStyle.Render("  Run 'Browse & Install Assets' to add some.\n"))
	} else {
		flatIdx := b.getFlatIndex()
		currentIdx := 0

		for _, section := range b.sections {
			// Section header
			s.WriteString(categoryStyle.Render(section.Title))
			s.WriteString("\n")

			// Items in this section
			for _, item := range section.Items {
				cursor := "  "
				if currentIdx == flatIdx {
					cursor = "> "
				}

				var checkbox string
				if item.Selected {
					checkbox = checkboxStyle.Render("[x]")
				} else {
					checkbox = uncheckboxStyle.Render("[ ]")
				}

				name := item.Name
				if currentIdx == flatIdx {
					name = selectedItemStyle.Render(name)
				}

				line := fmt.Sprintf("%s%s %s", cursor, checkbox, name)
				s.WriteString(line)
				s.WriteString("\n")

				currentIdx++
			}

			s.WriteString("\n")
		}
	}

	selectedCount := 0
	for _, section := range b.sections {
		for _, item := range section.Items {
			if item.Selected {
				selectedCount++
			}
		}
	}

	if selectedCount > 0 {
		s.WriteString(errorStyle.Render(fmt.Sprintf("  %d selected to uninstall", selectedCount)))
		s.WriteString("\n\n")
	}

	s.WriteString(helpStyle.Render("↑/↓: Navigate • Space: Toggle • d/Enter: Uninstall • a: All • n: None • Esc: Back"))
	s.WriteString("\n")

	return s.String()
}

type InstalledBrowserDoneMsg struct{}

type UninstallRequestMsg struct {
	Items []InstalledItem
}
