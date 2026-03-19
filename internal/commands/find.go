package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rosseca/aisi/internal/config"
	"github.com/rosseca/aisi/internal/installer"
	"github.com/rosseca/aisi/internal/registry"
	"github.com/rosseca/aisi/internal/repo"
	"github.com/rosseca/aisi/internal/targets"
	"github.com/rosseca/aisi/internal/tracker"
	"github.com/spf13/cobra"
)

var findSkillCmd = &cobra.Command{
	Use:   "find skill [name]",
	Short: "Search for skills in the SkillsMP registry",
	Long: `Search SkillsMP registry and install from any repository.

Get your API key at: https://skillsmp.com/auth/login
Set it with: aisi config set-skillsmp-key <api-key>

Examples:
  aisi find skill                    # Interactive search
  aisi find skill typescript         # Search for "typescript" skills
  aisi find skill "react hooks"      # Search with spaces`,
	RunE: runFindSkill,
}

func init() {
	rootCmd.AddCommand(findSkillCmd)
}

func runFindSkill(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	// Non-interactive mode: search and display results
	if query != "" {
		return runNonInteractiveSearch(query)
	}

	// Interactive mode: fzf-style search
	return runInteractiveSearch()
}

func runNonInteractiveSearch(query string) error {
	client := registry.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	skills, err := client.Search(ctx, query, 10, "") // Use API default sort
	if err != nil {
		// Provide helpful message for API key issues
		if err.Error() == "SkillsMP API key not configured. Set it with: aisi config set-skillsmp-key <your-api-key>" {
			fmt.Println("⚠️  SkillsMP API key not configured")
			fmt.Println()
			fmt.Println("To search skills, you need an API key from https://skillsmp.com/auth/login")
			fmt.Println("Then set it with: aisi config set-skillsmp-key <your-api-key>")
			fmt.Println()
			fmt.Println("Or set the environment variable: export SKILLSMP_API_KEY=<your-api-key>")
			return nil
		}
		return fmt.Errorf("search failed: %w", err)
	}

	if len(skills) == 0 {
		fmt.Printf("No skills found for \"%s\"\n", query)
		fmt.Println()
		fmt.Println("Try a different search term or visit https://skillsmp.com")
		return nil
	}

	fmt.Printf("Found %d skill(s) for \"%s\":\n", len(skills), query)
	fmt.Println()

	for _, skill := range skills {
		installs := registry.FormatInstalls(skill.Installs)
		source := skill.Source
		if source == "" {
			source = skill.ID
		}

		fmt.Printf("  %s@%s", source, skill.Name)
		if installs != "" {
			fmt.Printf("  (%s)", installs)
		}
		fmt.Println()
		fmt.Printf("    https://skillsmp.com/s/%s\n", skill.ID)
		fmt.Println()
	}

	fmt.Println("Install with:")
	fmt.Printf("  aisi install skill --url <owner/repo@skill-name>\n")
	return nil
}

// Interactive search model
type searchModel struct {
	textInput textinput.Model
	client    *registry.Client
	query     string
	skills    []registry.Skill
	cursor    int
	loading   bool
	err       error
	selected  *registry.Skill
	sortBy    string // "stars" or "recent"
}

func initialSearchModel() searchModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search and press Enter..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return searchModel{
		textInput: ti,
		client:    registry.NewClient(),
		cursor:    0,
		sortBy:    "stars", // Default sort by stars
	}
}

type searchMsg struct {
	skills []registry.Skill
	err    error
}

func (m searchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			// If we have results, Enter selects the current skill
			if len(m.skills) > 0 && m.cursor < len(m.skills) {
				m.selected = &m.skills[m.cursor]
				return m, tea.Quit
			}
			// Otherwise, search with current query
			query := m.textInput.Value()
			if len(query) >= 2 {
				m.loading = true
				m.query = query
				return m, performSearch(m.client, query, m.sortBy)
			}
			return m, nil

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown:
			if m.cursor < len(m.skills)-1 {
				m.cursor++
			}
			return m, nil

		case tea.KeyCtrlS:
			// Toggle sort mode between stars and recent
			if m.sortBy == "stars" {
				m.sortBy = "recent"
			} else {
				m.sortBy = "stars"
			}
			// Re-search with new sort order if there's a query
			if len(m.query) >= 2 {
				m.loading = true
				return m, performSearch(m.client, m.query, m.sortBy)
			}
			return m, nil
		}

	case searchMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.skills = nil
		} else {
			m.err = nil
			m.skills = msg.skills
			m.cursor = 0
		}
		return m, nil
	}

	// Handle text input changes
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func performSearch(client *registry.Client, query string, sortBy string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Search with limit of 50 results
		skills, err := client.Search(ctx, query, 50, sortBy)
		return searchMsg{skills: skills, err: err}
	}
}

func (m searchModel) View() string {
	var s strings.Builder

	// Title with sort indicator
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	sortStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	s.WriteString(titleStyle.Render("🔍 Find Skills on SkillsMP"))
	// Show current sort mode
	sortLabel := "⭐"
	if m.sortBy == "recent" {
		sortLabel = "🕐"
	}
	s.WriteString(" " + sortStyle.Render(sortLabel))
	s.WriteString("\n\n")

	// Search input
	s.WriteString("Search: ")
	s.WriteString(m.textInput.View())
	s.WriteString("\n\n")

	// Results
	if m.loading {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Searching..."))
		s.WriteString("\n")
	} else if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		s.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n")
	} else if len(m.skills) == 0 {
		if len(m.textInput.Value()) < 2 {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("Type at least 2 characters and press Enter to search"))
		} else {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("No skills found"))
		}
		s.WriteString("\n")
	} else {
		s.WriteString(fmt.Sprintf("Found %d result(s):\n\n", len(m.skills)))

		// Show visible results with scroll
		// Calculate visible range (show up to 15 items at a time)
		visibleCount := 15
		if visibleCount > len(m.skills) {
			visibleCount = len(m.skills)
		}

		startIdx := 0
		if m.cursor >= visibleCount {
			startIdx = m.cursor - visibleCount + 1
		}
		endIdx := startIdx + visibleCount
		if endIdx > len(m.skills) {
			endIdx = len(m.skills)
		}

		// Show scroll indicator if needed
		if startIdx > 0 {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("  ▲ more above"))
			s.WriteString("\n")
		}

		for i := startIdx; i < endIdx; i++ {
			skill := m.skills[i]
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}

			nameStyle := lipgloss.NewStyle()
			if m.cursor == i {
				nameStyle = nameStyle.Bold(true).Foreground(lipgloss.Color("6"))
			}

			source := skill.Source
			if source == "" {
				source = skill.ID
			}

			line := fmt.Sprintf("%s%s@%s", cursor, source, skill.Name)
			s.WriteString(nameStyle.Render(line))

			installs := registry.FormatInstalls(skill.Installs)
			if installs != "" {
				installsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
				s.WriteString(" " + installsStyle.Render(installs))
			}
			s.WriteString("\n")
		}

		// Show scroll indicator if needed
		if endIdx < len(m.skills) {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("  ▼ more below"))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	sortHelp := "stars"
	if m.sortBy == "recent" {
		sortHelp = "recent"
	}
	s.WriteString(helpStyle.Render(fmt.Sprintf("↑/↓: Navigate • Enter: Select/Search • Ctrl+S: Sort (%s) • Esc: Cancel", sortHelp)))
	s.WriteString("\n")

	return s.String()
}

func runInteractiveSearch() error {
	if !isTerminal() {
		fmt.Println("Interactive mode requires a terminal. Use: aisi find skill <name>")
		return nil
	}

	p := tea.NewProgram(initialSearchModel())
	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("interactive search failed: %w", err)
	}

	model := m.(searchModel)
	if model.selected == nil {
		fmt.Println("\nSearch cancelled")
		return nil
	}

	skill := model.selected
	return installFoundSkill(skill)
}

func installFoundSkill(skill *registry.Skill) error {
	source := skill.Source
	if source == "" {
		// Parse from ID (owner/repo/skill-name)
		parts := strings.Split(skill.ID, "/")
		if len(parts) >= 2 {
			source = strings.Join(parts[:2], "/")
		}
	}

	fmt.Printf("\nInstalling %s from %s...\n\n", skill.Name, source)

	// Get config and target
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	targetName := targetFlag
	if targetName == "" {
		targetName = cfg.ActiveTarget
	}
	if targetName == "" {
		targetName = "cursor"
	}

	target, err := targets.Get(targetName)
	if err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Setup repo manager and installer
	repoMgr, err := repo.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create repo manager: %w", err)
	}

	inst := installer.New(repoMgr, target, projectRoot)
	track := tracker.New(projectRoot, target)

	// Parse URL and install
	skillURL := fmt.Sprintf("%s@%s", source, skill.Name)
	parsedURL, err := repo.ParseSkillURL(skillURL)
	if err != nil {
		return fmt.Errorf("failed to parse skill URL: %w", err)
	}

	result, err := inst.InstallSkillFromURL(parsedURL, "")
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	if result.Error != nil {
		return fmt.Errorf("installation failed: %w", result.Error)
	}

	if result.Success {
		fmt.Printf("✓ Installed %s to %s\n", skill.Name, result.Path)
		fmt.Printf("\nView at: https://skillsmp.com/s/%s\n", skill.ID)

		// Record the installation with full source information
		// Don't modify project repoURL/repoCommit when installing from external source
		skillEntry := tracker.SkillEntry{
			Name:   skill.Name,
			Source: source,
			Path:   result.SourcePath, // Use the actual discovered path
		}
		_ = track.RecordSkillInstallOnly(skillEntry)
	}

	return nil
}

func isTerminal() bool {
	return os.Getenv("TERM") != "" || os.Getenv("TTY") != ""
}
