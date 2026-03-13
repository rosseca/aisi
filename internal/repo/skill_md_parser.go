package repo

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// SkillMetadata represents the parsed frontmatter and content from SKILL.md
type SkillMetadata struct {
	Name        string
	Description string
	RawContent  string
}

// ParseSkillMD reads and parses a SKILL.md file, extracting frontmatter and content
func ParseSkillMD(path string) (*SkillMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SKILL.md: %w", err)
	}
	defer file.Close()

	metadata := &SkillMetadata{}
	scanner := bufio.NewScanner(file)
	
	var contentLines []string
	inFrontmatter := false
	frontmatterStarted := false
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check for frontmatter delimiters (---)
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
		
		// Parse frontmatter lines
		if inFrontmatter {
			if strings.HasPrefix(line, "name:") {
				metadata.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			}
			if strings.HasPrefix(line, "description:") {
				metadata.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			}
		} else {
			// Collect content lines (everything after frontmatter)
			contentLines = append(contentLines, line)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}
	
	metadata.RawContent = strings.Join(contentLines, "\n")
	
	return metadata, nil
}

// GetDescription returns a shortened description for display
func (sm *SkillMetadata) GetDescription(maxLen int) string {
	desc := sm.Description
	if desc == "" {
		// Try to extract first paragraph from content
		lines := strings.Split(sm.RawContent, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				desc = line
				break
			}
		}
	}
	
	if len(desc) > maxLen {
		return desc[:maxLen-3] + "..."
	}
	return desc
}
