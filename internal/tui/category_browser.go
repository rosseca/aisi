package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/targets"
)

const (
	CategoryAll   = ""
	CategoryOther = "__other__"
)

type CategoryItem struct {
	Name       string
	AssetCount int
}

type CategoryBrowser struct {
	categories []CategoryItem
	cursor     int
	target     *targets.Target
	width      int
	height     int
}

func NewCategoryBrowser(m *manifest.Manifest, target *targets.Target) *CategoryBrowser {
	if m == nil || target == nil {
		return &CategoryBrowser{
			categories: []CategoryItem{{Name: CategoryAll, AssetCount: 0}},
			cursor:     0,
			target:     target,
			width:      80,
			height:     24,
		}
	}

	categoryCount := make(map[string]int)
	totalAssets := 0
	uncategorizedCount := 0

	// countCategories increments count for each category the asset belongs to
	countCategories := func(categories []string) {
		totalAssets++
		if len(categories) == 0 {
			uncategorizedCount++
		} else {
			for _, cat := range categories {
				if cat != "" {
					categoryCount[cat]++
				}
			}
		}
	}

	if target.RulesDir != "" {
		for _, r := range m.Rules {
			countCategories(r.Categories)
		}
	}

	if target.SkillsDir != "" {
		for _, s := range m.Skills {
			countCategories(s.Categories)
		}
		for _, e := range m.External {
			if e.Type == "skill" {
				countCategories(e.Categories)
			}
		}
	}

	if target.AgentsDir != "" {
		for _, a := range m.Agents {
			countCategories(a.Categories)
		}
		for _, e := range m.External {
			if e.Type == "agent" {
				countCategories(e.Categories)
			}
		}
	}

	if target.HooksFile != "" {
		for _, h := range m.Hooks {
			countCategories(h.Categories)
		}
	}

	if target.MCPFile != "" {
		for _, mc := range m.MCP {
			countCategories(mc.Categories)
		}
	}

	if target.SupportsAgentsMD {
		for _, am := range m.AgentsMD {
			countCategories(am.Categories)
		}
	}

	categories := make([]CategoryItem, 0)

	categories = append(categories, CategoryItem{
		Name:       CategoryAll,
		AssetCount: totalAssets,
	})

	sortedCategories := make([]string, 0, len(categoryCount))
	for cat := range categoryCount {
		sortedCategories = append(sortedCategories, cat)
	}
	sort.Strings(sortedCategories)

	for _, cat := range sortedCategories {
		categories = append(categories, CategoryItem{
			Name:       cat,
			AssetCount: categoryCount[cat],
		})
	}

	if uncategorizedCount > 0 {
		categories = append(categories, CategoryItem{
			Name:       CategoryOther,
			AssetCount: uncategorizedCount,
		})
	}

	return &CategoryBrowser{
		categories: categories,
		cursor:     0,
		target:     target,
		width:      80,
		height:     24,
	}
}

func (c *CategoryBrowser) Init() tea.Cmd {
	return nil
}

func (c *CategoryBrowser) SetSize(width, height int) {
	c.width = width
	c.height = height
}

func (c *CategoryBrowser) HasCategories() bool {
	// Has categories if there are more than just "All" and "Other"
	// categories always contains at least "All"
	// If there are real categories defined, len will be > 1
	for _, cat := range c.categories {
		if cat.Name != CategoryAll && cat.Name != CategoryOther {
			return true
		}
	}
	return false
}

func (c *CategoryBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return c, func() tea.Msg { return CategoryBrowserDoneMsg{Cancelled: true} }
		case "up", "k":
			if c.cursor > 0 {
				c.cursor--
			}
		case "down", "j":
			if c.cursor < len(c.categories)-1 {
				c.cursor++
			}
		case "enter", " ":
			selected := c.categories[c.cursor]
			return c, func() tea.Msg {
				return CategorySelectedMsg{Category: selected.Name}
			}
		}
	}
	return c, nil
}

func (c *CategoryBrowser) View() string {
	var content strings.Builder

	content.WriteString("\n")
	content.WriteString(titleStyle.Render("Select Category"))
	content.WriteString("\n")
	content.WriteString(subtitleStyle.Render("Target: " + c.target.DisplayName))
	content.WriteString("\n\n")

	for i, cat := range c.categories {
		cursor := "  "
		style := menuItemStyle

		if c.cursor == i {
			cursor = "> "
			style = selectedItemStyle
		}

		name := cat.Name
		if name == CategoryAll {
			name = "All"
		} else if name == CategoryOther {
			name = "Other"
		}

		countStr := dimStyle.Render(fmt.Sprintf(" (%d assets)", cat.AssetCount))
		line := fmt.Sprintf("%s%s%s", cursor, style.Render(name), countStr)
		content.WriteString(line)
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Back"))

	return lipgloss.Place(
		c.width,
		c.height,
		lipgloss.Center,
		lipgloss.Center,
		content.String(),
	)
}

type CategoryBrowserDoneMsg struct {
	Cancelled bool
}

type CategorySelectedMsg struct {
	Category string
}
