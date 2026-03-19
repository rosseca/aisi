package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuOption int

const (
	MenuBrowseInstall MenuOption = iota
	MenuInstallFromURL
	MenuFindSkill
	MenuViewInstalled
	MenuUpdateAll
	MenuSwitchTarget
	MenuSettings
	MenuExit
)

type MainMenu struct {
	choices  []string
	cursor   int
	selected MenuOption
	target   string
	width    int
	height   int
	version  string
}

func NewMainMenu(target string, version string) *MainMenu {
	return &MainMenu{
		choices: []string{
			"Browse & Install Assets",
			"Install Skill from Repository",
			"Find Skills Online (skills.sh)",
			"View Installed",
			"Update All",
			"Switch Target",
			"Settings",
			"Exit",
		},
		cursor:  0,
		target:  target,
		version: version,
		width:   80,
		height:  24,
	}
}

func (m *MainMenu) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *MainMenu) Init() tea.Cmd {
	return nil
}

func (m *MainMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = MenuOption(m.cursor)
			if m.selected == MenuExit {
				return m, tea.Quit
			}
			return m, func() tea.Msg {
				return MenuSelectedMsg{Option: m.selected}
			}
		}
	}
	return m, nil
}

func (m *MainMenu) View() string {
	// Build the menu content
	var content strings.Builder

	content.WriteString("\n")
	content.WriteString(renderTitle(m.target, m.version))
	content.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := "  "
		style := menuItemStyle

		if m.cursor == i {
			cursor = "> "
			style = selectedItemStyle
		}

		content.WriteString(style.Render(cursor + choice))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Select • q: Quit"))

	menuContent := content.String()

	// Center the menu in the terminal using Place
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		menuContent,
	)
}

type MenuSelectedMsg struct {
	Option MenuOption
}
