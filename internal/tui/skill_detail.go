package tui

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/rosseca/aisi/internal/registry"
)

// SkillDetail shows detailed information about a skill and asks for confirmation
type SkillDetail struct {
	skill       registry.Skill
	metadata    *SkillMetadata
	loading     bool
	err         error
	width       int
	height      int
	scrollPos   int
	content     []string // Lines of content to display
	confirmed   bool
	cancelled   bool
}

// SkillMetadata holds parsed info from SKILL.md
type SkillMetadata struct {
	Title       string
	Description string
	Content     string
}

// NewSkillDetail creates a new skill detail view
func NewSkillDetail(skill registry.Skill) *SkillDetail {
	return &SkillDetail{
		skill:     skill,
		loading:   true,
		width:     80,
		height:    24,
		scrollPos: 0,
		confirmed: false,
		cancelled: false,
	}
}

func (d *SkillDetail) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *SkillDetail) Init() tea.Cmd {
	return d.fetchSkillInfo()
}

type skillInfoMsg struct {
	metadata *SkillMetadata
	err      error
}

func (d *SkillDetail) fetchSkillInfo() tea.Cmd {
	return func() tea.Msg {
		// Try to fetch SKILL.md from GitHub raw content
		source := d.skill.Source
		if source == "" {
			parts := strings.Split(d.skill.ID, "/")
			if len(parts) >= 2 {
				source = strings.Join(parts[:2], "/")
			}
		}

		// Construct potential paths to SKILL.md
		paths := []string{
			fmt.Sprintf("https://raw.githubusercontent.com/%s/main/%s/SKILL.md", source, d.skill.Name),
			fmt.Sprintf("https://raw.githubusercontent.com/%s/main/skills/%s/SKILL.md", source, d.skill.Name),
			fmt.Sprintf("https://raw.githubusercontent.com/%s/master/%s/SKILL.md", source, d.skill.Name),
		}

		client := &http.Client{Timeout: 5 * time.Second}
		
		for _, url := range paths {
			resp, err := client.Get(url)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				content, err := io.ReadAll(resp.Body)
				if err != nil {
					continue
				}
				metadata := parseSkillContent(string(content))
				return skillInfoMsg{metadata: metadata, err: nil}
			}
		}

		// If we can't fetch, return basic info
		return skillInfoMsg{
			metadata: &SkillMetadata{
				Title:       d.skill.Name,
				Description: "No description available",
				Content:     "",
			},
			err: nil,
		}
	}
}

func parseSkillContent(content string) *SkillMetadata {
	metadata := &SkillMetadata{
		Content: content,
	}

	lines := strings.Split(content, "\n")
	inFrontmatter := false
	frontmatterStarted := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Check for frontmatter
		if line == "---" {
			if !frontmatterStarted {
				frontmatterStarted = true
				inFrontmatter = true
				continue
			} else if inFrontmatter {
				inFrontmatter = false
				continue
			}
		}

		if inFrontmatter {
			if strings.HasPrefix(line, "name:") {
				metadata.Title = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			}
			if strings.HasPrefix(line, "description:") {
				metadata.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			}
		}

		// Extract first paragraph as description if not in frontmatter
		if !inFrontmatter && metadata.Description == "" && line != "" && !strings.HasPrefix(line, "#") {
			if len(line) > 10 { // Reasonable sentence length
				metadata.Description = line
				break
			}
		}
	}

	if metadata.Title == "" {
		// Try to get from first h1
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				metadata.Title = strings.TrimPrefix(line, "# ")
				break
			}
		}
	}

	return metadata
}

func (d *SkillDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			d.cancelled = true
			return d, func() tea.Msg { return SkillDetailDoneMsg{} }

		case "q", "esc":
			d.cancelled = true
			return d, func() tea.Msg { return SkillDetailDoneMsg{} }

		case "y", "Y":
			if !d.loading {
				d.confirmed = true
				return d, func() tea.Msg {
					return SkillDetailDoneMsg{Confirmed: true, Skill: d.skill}
				}
			}

		case "n", "N":
			if !d.loading {
				d.cancelled = true
				return d, func() tea.Msg { return SkillDetailDoneMsg{} }
			}

		case "up", "k":
			if d.scrollPos > 0 {
				d.scrollPos--
			}
			return d, nil

		case "down", "j":
			maxScroll := len(d.content) - d.getContentHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if d.scrollPos < maxScroll {
				d.scrollPos++
			}
			return d, nil
		}

	case skillInfoMsg:
		d.loading = false
		d.metadata = msg.metadata
		d.err = msg.err
		d.prepareContent()
		return d, nil

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		d.prepareContent()
	}

	return d, nil
}

func (d *SkillDetail) getContentHeight() int {
	// Reserve space for: title (2) + buttons (3) + help (2) + padding (2)
	return d.height - 9
}

func (d *SkillDetail) prepareContent() {
	if d.loading || d.metadata == nil {
		return
	}

	var lines []string
	
	// Title
	title := d.skill.Name
	if d.metadata.Title != "" {
		title = d.metadata.Title
	}
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	lines = append(lines, nameStyle.Render(title))
	lines = append(lines, "")

	// Source
	source := d.skill.Source
	if source == "" {
		parts := strings.Split(d.skill.ID, "/")
		if len(parts) >= 2 {
			source = strings.Join(parts[:2], "/")
		}
	}
	lines = append(lines, fmt.Sprintf("📦 Source: %s", source))

	// Installs
	installs := registry.FormatInstalls(d.skill.Installs)
	if installs != "" {
		lines = append(lines, fmt.Sprintf("⬇️  Installs: %s", installs))
	}
	
	lines = append(lines, "")

	// Render markdown content with glamour if available
	if d.metadata.Content != "" {
		// Create glamour renderer with auto style and word wrap
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(d.width-6),
		)
		if err == nil {
			rendered, err := renderer.Render(d.metadata.Content)
			if err == nil && rendered != "" {
				// Split rendered content into lines
				renderedLines := strings.Split(rendered, "\n")
				lines = append(lines, renderedLines...)
			} else {
				// Fallback to plain text if glamour fails
				lines = append(lines, "📄 Full Content:")
				lines = append(lines, "")
				contentLines := strings.Split(d.metadata.Content, "\n")
				for _, line := range contentLines {
					wrapped := wrapText(line, d.width-4)
					lines = append(lines, wrapped...)
				}
			}
		} else {
			// Fallback to plain text if renderer creation fails
			lines = append(lines, "📄 Full Content:")
			lines = append(lines, "")
			contentLines := strings.Split(d.metadata.Content, "\n")
			for _, line := range contentLines {
				wrapped := wrapText(line, d.width-4)
				lines = append(lines, wrapped...)
			}
		}
	} else if d.metadata.Description != "" {
		// Only show description if no full content
		lines = append(lines, "📝 Description:")
		descLines := wrapText(d.metadata.Description, d.width-6)
		for _, line := range descLines {
			lines = append(lines, "  "+line)
		}
	}

	d.content = lines
}

func (d *SkillDetail) View() string {
	var s strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	s.WriteString("\n")
	s.WriteString(titleStyle.Render("📋 Skill Details"))
	s.WriteString("\n\n")

	if d.loading {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		s.WriteString(dimStyle.Render("Loading skill information..."))
		s.WriteString("\n")
	} else if d.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		s.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", d.err)))
		s.WriteString("\n")
	} else {
		// Content box with border
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1).
			Width(d.width - 2)

		contentHeight := d.getContentHeight()
		endPos := d.scrollPos + contentHeight
		if endPos > len(d.content) {
			endPos = len(d.content)
		}
		if d.scrollPos >= len(d.content) {
			d.scrollPos = 0
		}

		visibleContent := d.content[d.scrollPos:endPos]
		
		// Add scroll indicators
		contentStr := strings.Join(visibleContent, "\n")
		
		s.WriteString(boxStyle.Render(contentStr))
		s.WriteString("\n")

		// Scroll indicator
		if len(d.content) > contentHeight {
			scrollInfo := fmt.Sprintf("(%d/%d lines)", d.scrollPos+1, len(d.content))
			if d.scrollPos > 0 {
				scrollInfo = "▲ " + scrollInfo
			}
			if endPos < len(d.content) {
				scrollInfo = scrollInfo + " ▼"
			}
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			s.WriteString(dimStyle.Render(scrollInfo))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")

	// Action buttons - always visible at bottom
	buttonY := lipgloss.NewStyle().
		Background(lipgloss.Color("2")).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Padding(0, 2).
		Render("[Y] Install")
	
	buttonN := lipgloss.NewStyle().
		Background(lipgloss.Color("1")).
		Foreground(lipgloss.Color("15")).
		Bold(true).
		Padding(0, 2).
		Render("[N] Cancel")

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, buttonY, "  ", buttonN))
	s.WriteString("\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	s.WriteString(helpStyle.Render("↑/↓: Scroll • y: Install • n/q/esc: Cancel"))
	s.WriteString("\n")

	return s.String()
}

// SkillDetailDoneMsg is sent when the detail view is closed
type SkillDetailDoneMsg struct {
	Confirmed bool
	Skill     registry.Skill
}
