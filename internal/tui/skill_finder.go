package tui

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rosseca/aisi/internal/registry"
)

// SearchMode represents the current search mode
type SearchMode int

const (
	ModeKeyword SearchMode = iota // Regular keyword search
	ModeAI                        // AI semantic search
)

func (m SearchMode) String() string {
	switch m {
	case ModeAI:
		return "AI Semantic"
	default:
		return "Keyword"
	}
}

// SkillFinder is a TUI component for searching skills in the registry
type SkillFinder struct {
	textInput    textinput.Model
	client       *registry.Client
	query        string
	skills       []registry.Skill
	cursor       int
	loading      bool
	err          error
	selected     *registry.Skill
	width        int
	height       int
	sortBy       string     // "stars" or "recent"
	searchMode   SearchMode // keyword or AI
	viewingDesc  bool       // Whether we're viewing skill description
	viewingMarkdown bool    // Whether we're viewing the SKILL.md content
	markdownContent string  // Content of SKILL.md
	markdownScroll  int     // Scroll position for markdown view
	loadingMarkdown bool    // Loading markdown content
}

// NewSkillFinder creates a new skill finder component
func NewSkillFinder() *SkillFinder {
	ti := textinput.New()
	ti.Placeholder = "Type to search skills and press Enter..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return &SkillFinder{
		textInput:   ti,
		client:      registry.NewClient(),
		cursor:      0,
		width:       80,
		height:      24,
		sortBy:      "stars", // Default sort by stars
		searchMode:  ModeKeyword,
		viewingDesc: false,
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
	// Handle markdown view mode separately
	if f.viewingMarkdown {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc, tea.KeyEnter:
				// Go back to detail view
				f.viewingMarkdown = false
				f.markdownScroll = 0
				return f, nil
			case tea.KeyCtrlC:
				return f, func() tea.Msg { return SkillFinderDoneMsg{} }
			case tea.KeyUp:
				if f.markdownScroll > 0 {
					f.markdownScroll--
				}
				return f, nil
			case tea.KeyDown:
				// Allow scrolling if content is long
				maxScroll := strings.Count(f.markdownContent, "\n") - f.height + 8
				if maxScroll < 0 {
					maxScroll = 0
				}
				if f.markdownScroll < maxScroll {
					f.markdownScroll++
				}
				return f, nil
			}
		case skillMarkdownMsg:
			f.loadingMarkdown = false
			if msg.err != nil {
				f.markdownContent = fmt.Sprintf("Error loading SKILL.md: %v", msg.err)
			} else {
				f.markdownContent = msg.content
			}
			return f, nil
		}
		return f, nil
	}

	// Handle description view mode separately
	if f.viewingDesc {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc, tea.KeyEnter:
				// Go back to list view
				f.viewingDesc = false
				return f, nil
			case tea.KeyCtrlC:
				return f, func() tea.Msg { return SkillFinderDoneMsg{} }
			case tea.KeyRunes:
				// Check for 'i' key to install
				if len(msg.Runes) == 1 && (msg.Runes[0] == 'i' || msg.Runes[0] == 'I') {
					if len(f.skills) > 0 && f.cursor < len(f.skills) {
						// Send install message and close finder
						skill := f.skills[f.cursor]
						return f, func() tea.Msg {
							return SkillInstallMsg{Skill: skill}
						}
					}
				}
				// Check for 'm' key to view markdown
				if len(msg.Runes) == 1 && (msg.Runes[0] == 'm' || msg.Runes[0] == 'M') {
					if len(f.skills) > 0 && f.cursor < len(f.skills) {
						skill := f.skills[f.cursor]
						if skill.GithubURL != "" {
							f.loadingMarkdown = true
							f.viewingMarkdown = true
							f.markdownScroll = 0
							return f, f.fetchSkillMarkdown(skill.GithubURL)
						}
					}
				}
			}
		}
		return f, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return f, func() tea.Msg { return SkillFinderDoneMsg{} }

		case tea.KeyEsc:
			return f, func() tea.Msg { return SkillFinderDoneMsg{} }

		case tea.KeyEnter:
			// If we have results, show description of selected skill
			if len(f.skills) > 0 && f.cursor < len(f.skills) {
				f.viewingDesc = true
				return f, nil
			}
			// Otherwise search
			query := f.textInput.Value()
			if len(query) >= 2 {
				f.loading = true
				f.query = query
				return f, f.performSearch(query)
			}
			return f, nil

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

		case tea.KeyCtrlA:
			// Toggle search mode between keyword and AI
			if f.searchMode == ModeKeyword {
				f.searchMode = ModeAI
				f.textInput.Placeholder = "Describe what you need and press Enter..."
			} else {
				f.searchMode = ModeKeyword
				f.textInput.Placeholder = "Type to search skills and press Enter..."
			}
			// Clear previous results when switching modes
			f.skills = nil
			f.cursor = 0
			return f, nil

		case tea.KeyCtrlS:
			// Toggle sort mode between stars and recent (only for keyword search)
			if f.searchMode == ModeKeyword {
				if f.sortBy == "stars" {
					f.sortBy = "recent"
				} else {
					f.sortBy = "stars"
				}
				// Re-search with new sort order if there's a query
				if len(f.query) >= 2 {
					f.loading = true
					return f, f.performSearch(f.query)
				}
			}
			return f, nil
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
				f.cursor = 0
			}
		}
		return f, nil

	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
	}

	// Handle text input changes
	var cmd tea.Cmd
	f.textInput, cmd = f.textInput.Update(msg)
	return f, cmd
}

func (f *SkillFinder) performSearch(query string) tea.Cmd {
	return func() tea.Msg {
		// AI search needs longer timeout (can take 3-5 seconds)
		timeout := 15 * time.Second
		if f.searchMode == ModeAI {
			timeout = 30 * time.Second
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		var skills []registry.Skill
		var err error

		if f.searchMode == ModeAI {
			// AI semantic search
			skills, err = f.client.AISearch(ctx, query)
		} else {
			// Keyword search with limit of 50
			skills, err = f.client.Search(ctx, query, 50, f.sortBy)
		}

		return skillSearchMsg{skills: skills, err: err, query: query}
	}
}

// skillMarkdownMsg is sent when markdown content is loaded
type skillMarkdownMsg struct {
	content string
	err     error
}

// fetchSkillMarkdown fetches the SKILL.md content from GitHub raw
func (f *SkillFinder) fetchSkillMarkdown(githubURL string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		content, err := fetchSkillMDFromGitHub(ctx, githubURL)
		return skillMarkdownMsg{content: content, err: err}
	}
}

// fetchSkillMDFromGitHub converts a GitHub URL to raw URL and fetches SKILL.md
func fetchSkillMDFromGitHub(ctx context.Context, githubURL string) (string, error) {
	// Convert GitHub URL to raw.githubusercontent.com URL
	// From: https://github.com/owner/repo/tree/main/path/to/skill
	// To:   https://raw.githubusercontent.com/owner/repo/main/path/to/skill/SKILL.md

	if githubURL == "" {
		return "", fmt.Errorf("no GitHub URL available")
	}

	// Remove protocol and domain
	url := strings.TrimPrefix(githubURL, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Split into parts
	parts := strings.Split(url, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid GitHub URL format")
	}

	// Extract owner, repo, and path
	owner := parts[1]
	repo := parts[2]

	// Find the path after /tree/ or /blob/
	pathParts := []string{}
	for i := 3; i < len(parts); i++ {
		if parts[i] == "tree" || parts[i] == "blob" {
			// Skip the tree/blob keyword and continue with the rest
			continue
		}
		if parts[i] != "" {
			pathParts = append(pathParts, parts[i])
		}
	}

	// Construct raw URL
	branch := "main"
	if len(pathParts) > 0 && (pathParts[0] == "main" || pathParts[0] == "master" || pathParts[0] == "develop") {
		branch = pathParts[0]
		pathParts = pathParts[1:]
	}

	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s/SKILL.md",
		owner, repo, branch, strings.Join(pathParts, "/"))

	// Fetch the content
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SKILL.md: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("SKILL.md not found in repository")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch SKILL.md: status %d", resp.StatusCode)
	}

	// Read content
	var builder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	lineCount := 0
	for scanner.Scan() {
		if lineCount > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(scanner.Text())
		lineCount++
		// Limit to prevent overwhelming the UI
		if lineCount > 500 {
			builder.WriteString("\n\n... (content truncated, file too long)")
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading content: %w", err)
	}

	return builder.String(), nil
}

func (f *SkillFinder) View() string {
	// If viewing description, show skill detail view
	if f.viewingDesc && len(f.skills) > 0 && f.cursor < len(f.skills) {
		return f.viewSkillDescription()
	}

	var s strings.Builder

	// Title area with mode indicator box
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1)
	activeBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 1).
		Bold(true)

	s.WriteString("\n")
	s.WriteString(titleStyle.Render("🔍 Find Skills on SkillsMP"))
	s.WriteString("\n\n")

	// Mode indicators box
	var modeBoxes []string

	// Sort mode boxes (only show in keyword mode)
	if f.searchMode == ModeKeyword {
		starsBox := boxStyle
		recentBox := boxStyle
		if f.sortBy == "stars" {
			starsBox = activeBoxStyle
		} else {
			recentBox = activeBoxStyle
		}
		modeBoxes = append(modeBoxes, starsBox.Render("⭐ Stars"))
		modeBoxes = append(modeBoxes, recentBox.Render("🕐 Recent"))
	}

	// Search mode boxes
	keywordBox := boxStyle
	aiBox := boxStyle
	if f.searchMode == ModeKeyword {
		keywordBox = activeBoxStyle
	} else {
		aiBox = activeBoxStyle
	}
	modeBoxes = append(modeBoxes, keywordBox.Render("Keyword"))
	modeBoxes = append(modeBoxes, aiBox.Render("🤖 AI"))

	// Join mode boxes
	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, modeBoxes...))
	s.WriteString("\n\n")

	// Search input
	s.WriteString("Search: ")
	s.WriteString(f.textInput.View())
	s.WriteString("\n\n")

	// Results area
	if f.loading {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		s.WriteString(dimStyle.Render("Searching..."))
		s.WriteString("\n")
	} else if f.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		errMsg := f.err.Error()
		// Show helpful message for API key issues
		if errMsg == "SkillsMP API key not configured. Set it with: aisi config set-skillsmp-key <your-api-key>" {
			s.WriteString(errStyle.Render("⚠️  API key not configured"))
			s.WriteString("\n\n")
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(helpStyle.Render("Get your API key at: https://skillsmp.com/auth/login"))
			s.WriteString("\n")
			s.WriteString(helpStyle.Render("Set it with: aisi config set-skillsmp-key <key>"))
		} else {
			s.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", f.err)))
		}
		s.WriteString("\n")
	} else if len(f.skills) == 0 {
		if len(f.textInput.Value()) < 2 {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			if f.searchMode == ModeAI {
				s.WriteString(dimStyle.Render("Describe what you need in natural language (e.g., 'web scraper with pagination')"))
			} else {
				s.WriteString(dimStyle.Render("Type at least 2 characters and press Enter to search"))
			}
		} else {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render("No skills found"))
		}
		s.WriteString("\n")
	} else {
		// Results count
		s.WriteString(fmt.Sprintf("Found %d result(s):\n\n", len(f.skills)))

		// Calculate visible range for scrolling
		maxResultsHeight := f.height - 16 // Reserve space for title, modes, input, help
		if maxResultsHeight < 5 {
			maxResultsHeight = 5
		}

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

		// Show results
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
	minHeight := f.height - 6
	for lines < minHeight {
		s.WriteString("\n")
		lines++
	}

	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	// Build help text
	helpText := "↑/↓: Navigate • Enter: Details • Ctrl+A: Mode"
	if f.searchMode == ModeKeyword {
		helpText += " • Ctrl+S: Sort"
	}
	helpText += " • Esc: Back"

	s.WriteString(helpStyle.Render(helpText))
	s.WriteString("\n")

	return s.String()
}

// viewSkillDescription shows the detailed view of a selected skill
func (f *SkillFinder) viewSkillDescription() string {
	var s strings.Builder

	skill := f.skills[f.cursor]

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	s.WriteString("\n")
	s.WriteString(titleStyle.Render("📋 Skill Details"))
	s.WriteString("\n\n")

	// Skill name
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	s.WriteString(nameStyle.Render(skill.Name))
	s.WriteString("\n\n")

	// Source
	if skill.Source != "" {
		s.WriteString(fmt.Sprintf("Source: %s\n", skill.Source))
	}

	// Installs
	if skill.Installs > 0 {
		s.WriteString(fmt.Sprintf("Installs: %s\n", registry.FormatInstalls(skill.Installs)))
	}

	// Stars
	if skill.Stars > 0 {
		s.WriteString(fmt.Sprintf("⭐ Stars: %d\n", skill.Stars))
	}

	// URL
	if skill.URL != "" {
		s.WriteString(fmt.Sprintf("\nURL: %s\n", skill.URL))
	}

	// Description
	s.WriteString("\n")
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	if skill.Description != "" {
		// Wrap description to fit width
		desc := skill.Description
		maxWidth := f.width - 4
		if maxWidth < 40 {
			maxWidth = 40
		}

		// Simple word wrapping
		words := strings.Fields(desc)
		var lines []string
		currentLine := ""
		for _, word := range words {
			if len(currentLine)+len(word)+1 > maxWidth {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = word
			} else {
				if currentLine != "" {
					currentLine += " "
				}
				currentLine += word
			}
		}
		if currentLine != "" {
			lines = append(lines, currentLine)
		}

		// Show up to available height lines
		maxLines := f.height - 12
		if maxLines < 5 {
			maxLines = 5
		}
		if len(lines) > maxLines {
			lines = lines[:maxLines-1]
			lines = append(lines, "...")
		}

		s.WriteString(descStyle.Render(strings.Join(lines, "\n")))
		s.WriteString("\n")
	} else {
		s.WriteString(descStyle.Render("No description available."))
		s.WriteString("\n")
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

	// Show appropriate help based on whether we can view markdown
	if skill.GithubURL != "" {
		s.WriteString(helpStyle.Render("i: Install • m: View SKILL.md • Enter/Esc: Back • Ctrl+C: Quit"))
	} else {
		s.WriteString(helpStyle.Render("i: Install • Enter/Esc: Back • Ctrl+C: Quit"))
	}
	s.WriteString("\n")

	return s.String()
}

// viewSkillMarkdown shows the SKILL.md content
func (f *SkillFinder) viewSkillMarkdown() string {
	var s strings.Builder

	skill := f.skills[f.cursor]

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	s.WriteString("\n")
	s.WriteString(titleStyle.Render(fmt.Sprintf("📄 SKILL.md - %s", skill.Name)))
	s.WriteString("\n\n")

	if f.loadingMarkdown {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		s.WriteString(dimStyle.Render("Loading SKILL.md..."))
		s.WriteString("\n")
	} else if f.markdownContent != "" {
		// Content box style
		contentStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1).
			Width(f.width - 4)

		// Calculate visible lines based on scroll position
		contentLines := strings.Split(f.markdownContent, "\n")
		maxVisibleLines := f.height - 10 // Reserve space for title and help
		if maxVisibleLines < 5 {
			maxVisibleLines = 5
		}

		startLine := f.markdownScroll
		endLine := startLine + maxVisibleLines
		if endLine > len(contentLines) {
			endLine = len(contentLines)
		}

		visibleLines := contentLines[startLine:endLine]

		// Show scroll indicator if needed
		if startLine > 0 {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("▲ More above"))
			s.WriteString("\n")
		}

		// Show content with markdown syntax highlighting (simplified)
		var styledLines []string
		for _, line := range visibleLines {
			styledLine := styleMarkdownLine(line)
			styledLines = append(styledLines, styledLine)
		}

		s.WriteString(contentStyle.Render(strings.Join(styledLines, "\n")))

		// Show scroll indicator if needed
		if endLine < len(contentLines) {
			s.WriteString("\n")
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("▼ More below"))
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
	s.WriteString(helpStyle.Render("↑/↓: Scroll • Enter/Esc: Back to details • Ctrl+C: Quit"))
	s.WriteString("\n")

	return s.String()
}

// styleMarkdownLine applies basic styling to markdown content
func styleMarkdownLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Headers
	if strings.HasPrefix(trimmed, "# ") {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(trimmed)
	}
	if strings.HasPrefix(trimmed, "## ") {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render(trimmed)
	}
	if strings.HasPrefix(trimmed, "### ") {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("219")).Render(trimmed)
	}

	// Code blocks
	if strings.HasPrefix(trimmed, "```") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(trimmed)
	}

	// Inline code
	if strings.Contains(line, "`") && !strings.HasPrefix(trimmed, "```") {
		// Simple inline code highlighting
		parts := strings.Split(line, "`")
		var result strings.Builder
		for i, part := range parts {
			if i%2 == 1 {
				// Odd indices are code
				result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("`" + part + "`"))
			} else {
				result.WriteString(part)
			}
		}
		return result.String()
	}

	// Bullet points
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		bullet := trimmed[:2]
		rest := trimmed[2:]
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(bullet) + rest
	}

	// Links [text](url)
	if strings.Contains(line, "[") && strings.Contains(line, "](") {
		// Basic link highlighting - show in different color
		return lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render(line)
	}

	// Bold text
	if strings.Contains(line, "**") {
		return lipgloss.NewStyle().Bold(true).Render(line)
	}

	return line
}

// SkillFinderDoneMsg is sent when the finder is closed without selection
type SkillFinderDoneMsg struct{}

// SwitchToCustomURLMsg is sent when user wants to switch to custom URL input
type SwitchToCustomURLMsg struct{}

// SkillSelectedMsg is sent when a skill is selected
type SkillSelectedMsg struct {
	Skill registry.Skill
}

// SkillInstallMsg is sent when user wants to install a skill from detail view
type SkillInstallMsg struct {
	Skill registry.Skill
}
