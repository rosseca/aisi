package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuOption int

const (
	MenuBrowseInstall MenuOption = iota
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
}

func NewMainMenu(target string) *MainMenu {
	return &MainMenu{
		choices: []string{
			"Browse & Install Assets",
			"View Installed",
			"Update All",
			"Switch Target",
			"Settings",
			"Exit",
		},
		cursor: 0,
		target: target,
	}
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
	// Left column: title + menu
	var leftCol strings.Builder

	leftCol.WriteString("\n")
	leftCol.WriteString(renderTitle(m.target))
	leftCol.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := "  "
		style := menuItemStyle

		if m.cursor == i {
			cursor = "> "
			style = selectedItemStyle
		}

		leftCol.WriteString(style.Render(cursor + choice))
		leftCol.WriteString("\n")
	}

	leftCol.WriteString("\n")
	leftCol.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Select • q: Quit"))

	// Add padding to match dinosaur height (17 lines total)
	currentLines := strings.Count(leftCol.String(), "\n")
	targetLines := 17 // Height of the dinosaur
	for i := currentLines; i < targetLines; i++ {
		leftCol.WriteString("\n")
	}

	// Right column: dinosaur
	rightCol := PixelDinoStyle().Render(PixelDinoASCII())

	// Combine columns
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftCol.String(),
		rightCol,
	) + "\n"
}

type MenuSelectedMsg struct {
	Option MenuOption
}
