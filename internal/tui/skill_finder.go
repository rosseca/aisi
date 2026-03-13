package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rosseca/aisi/internal/registry"
)

// SkillFinder is a TUI component for searching skills in the registry
type SkillFinder struct {
	textInput  textinput.Model
	client     *registry.Client
	query      string
	skills     []registry.Skill
	cursor     int
	loading    bool
	err        error
	selected   *registry.Skill
	width      int
	height     int
	lastSearch time.Time
	debounce   time.Duration
}

// NewSkillFinder creates a new skill finder component
func NewSkillFinder() *SkillFinder {
	ti := textinput.New()
	ti.Placeholder = "Type to search skills..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40

	return &SkillFinder{
		textInput:  ti,
		client:     registry.NewClient(),
		cursor:     0,
		debounce:   200 * time.Millisecond,
		lastSearch: time.Now(),
		width:      80,
		height:     24,
	}
}

func (f *SkillFinder) SetSize(width, height int) {
	f.width = width
	f.height = height
}

func (f *SkillFinder) Init() tea.Cmd {
	return textinput.Blink
}

type skillSearchMsg struct {
	skills []registry.Skill
	err    error
	query  string
}

func (f *SkillFinder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return f, func() tea.Msg { return SkillFinderDoneMsg{} }

		case tea.KeyEsc:
			return f, func() tea.Msg { return SkillFinderDoneMsg{} }

		case tea.KeyEnter:
			if f.cursor < len(f.skills) && len(f.skills) > 0 {
				f.selected = &f.skills[f.cursor]
				return f, func() tea.Msg {
					return SkillSelectedMsg{Skill: *f.selected}
				}
			}

		case tea.KeyUp:
			if f.cursor > 0 {
				f.cursor--
			}
			return f, nil

		case tea.KeyDown:
			if f.cursor < len(f.skills)-1 {
				f.cursor++
			}
			return f, nil

		case tea.KeyF1:
			// Switch to custom URL mode with F1
			return f, func() tea.Msg { return SwitchToCustomURLMsg{} }
		}

	case skillSearchMsg:
		// Only update if this is the most recent search
		if msg.query == f.query {
			f.loading = false
			if msg.err != nil {
				f.err = msg.err
				f.skills = nil
			} else {
				f.err = nil
				f.skills = msg.skills
				if f.cursor >= len(f.skills) {
					f.cursor = 0
				}
			}
		}
		return f, nil

	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
	}

	// Handle text input changes
	oldQuery := f.textInput.Value()
	var cmd tea.Cmd
	f.textInput, cmd = f.textInput.Update(msg)
	newQuery := f.textInput.Value()

	// Trigger search on query change (with debounce)
	if newQuery != oldQuery && len(newQuery) >= 2 {
		f.loading = true
		f.query = newQuery
		return f, tea.Batch(
			cmd,
			f.debounceSearch(newQuery),
		)
	}

	// Clear results if query is too short
	if len(newQuery) < 2 && len(f.skills) > 0 {
		f.skills = nil
		f.cursor = 0
	}

	return f, cmd
}

func (f *SkillFinder) debounceSearch(query string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(f.debounce)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		skills, err := f.client.Search(ctx, query, 10)
		return skillSearchMsg{skills: skills, err: err, query: query}
	}
}

func (f *SkillFinder) View() string {
	var s strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	s.WriteString("\n")
	s.WriteString(titleStyle.Render("🔍 Find Skills Online"))
	s.WriteString("\n\n")

	// Search input
	s.WriteString("Search: ")
	s.WriteString(f.textInput.View())
	s.WriteString("\n\n")

	// Results area with fixed height for scrolling feel
	maxResultsHeight := f.height - 12 // Reserve space for title, input, help
	if maxResultsHeight < 5 {
		maxResultsHeight = 5
	}

	if f.loading {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		s.WriteString(dimStyle.Render("Searching..."))
		s.WriteString("\n")
	} else if f.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		s.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", f.err)))
		s.WriteString("\n")
	} else if len(f.skills) == 0 {
		if len(f.textInput.Value()) < 2 {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("Start typing to search (min 2 characters)"))
		} else {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("No skills found"))
		}
		s.WriteString("\n")
	} else {
		s.WriteString(fmt.Sprintf("Found %d result(s):\n\n", len(f.skills)))

		// Calculate visible range
		visibleCount := maxResultsHeight - 2
		if visibleCount > len(f.skills) {
			visibleCount = len(f.skills)
		}

		startIdx := 0
		if f.cursor >= visibleCount {
			startIdx = f.cursor - visibleCount + 1
		}
		endIdx := startIdx + visibleCount
		if endIdx > len(f.skills) {
			endIdx = len(f.skills)
		}

		// Show scroll indicator if needed
		if startIdx > 0 {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("  ▲ more above"))
			s.WriteString("\n")
		}

		for i := startIdx; i < endIdx; i++ {
			skill := f.skills[i]
			cursor := "  "
			if f.cursor == i {
				cursor = "> "
			}

			nameStyle := lipgloss.NewStyle()
			if f.cursor == i {
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
		if endIdx < len(f.skills) {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("  ▼ more below"))
			s.WriteString("\n")
		}
	}

	// Pad to maintain consistent height
	lines := strings.Count(s.String(), "\n")
	minHeight := f.height - 4
	for lines < minHeight {
		s.WriteString("\n")
		lines++
	}

	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	s.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Select • F1: Custom URL • Esc: Back"))
	s.WriteString("\n")

	return s.String()
}

// SkillFinderDoneMsg is sent when the finder is closed without selection
type SkillFinderDoneMsg struct{}

// SwitchToCustomURLMsg is sent when user wants to switch to custom URL input
type SwitchToCustomURLMsg struct{}

// SkillSelectedMsg is sent when a skill is selected
type SkillSelectedMsg struct {
	Skill registry.Skill
}
