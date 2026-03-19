package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			MarginLeft(2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(2)

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170")).
				Bold(true)

	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	uncheckboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1).
			MarginLeft(2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	categoryStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginTop(1).
			MarginBottom(0).
			MarginLeft(2)

	externalBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	repoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
		Italic(true).
		MarginLeft(2)

	infoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	secondaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)
)

const appName = "AI Shared Intelligence"

func renderTitle(target string, version string) string {
	title := titleStyle.Render(appName + " " + version)
	subtitle := subtitleStyle.Render("Target: " + target)
	return title + "\n" + subtitle
}

func renderTitleWithRepo(target string, repoSource string, version string) string {
	title := titleStyle.Render(appName + " " + version)
	subtitle := subtitleStyle.Render("Target: " + target)
	result := title + "\n" + subtitle
	if repoSource != "" {
		repoInfo := repoStyle.Render("📦 " + repoSource)
		result += "\n" + repoInfo
	}
	return result
}

