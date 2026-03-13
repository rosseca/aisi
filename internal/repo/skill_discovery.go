package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillDiscovery finds skills within a repository by searching for SKILL.md files
type SkillDiscovery struct {
	rootPath string
}

// NewSkillDiscovery creates a new skill discovery for the given repository root
func NewSkillDiscovery(rootPath string) *SkillDiscovery {
	return &SkillDiscovery{rootPath: rootPath}
}

// DiscoveredSkill represents a skill found in the repository
type DiscoveredSkill struct {
	Name        string // Extracted from SKILL.md frontmatter or directory name
	Description string // Description from SKILL.md frontmatter
	Path        string // Relative path from repo root
	FullPath    string // Absolute path to the skill directory
	SKILLMdPath string // Absolute path to the SKILL.md file
}

// MultipleSkillsError is returned when multiple skills are found but none match the requested name.
// This allows the caller to present the user with a choice.
type MultipleSkillsError struct {
	RequestedName string
	FoundSkills   []*DiscoveredSkill
}

func (e *MultipleSkillsError) Error() string {
	return fmt.Sprintf("skill '%s' not found - %d skills available in repository", e.RequestedName, len(e.FoundSkills))
}

// FindSkillByName searches for a skill with the given name in the repository.
// It looks for SKILL.md files and matches either:
// 1. The directory name (last component of the path)
// 2. The name field in the SKILL.md frontmatter (if parseable)
// 3. If no match found and there's only one skill in the repo, returns that one
func (sd *SkillDiscovery) FindSkillByName(skillName string) (*DiscoveredSkill, error) {
	var dirMatches []*DiscoveredSkill
	var frontmatterMatches []*DiscoveredSkill
	var allSkills []*DiscoveredSkill

	err := filepath.Walk(sd.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if we hit errors on some paths
		}

		// Skip hidden directories and common non-skill directories
		if info.IsDir() {
			dirName := info.Name()
			if strings.HasPrefix(dirName, ".") || dirName == "node_modules" || dirName == "vendor" {
				return filepath.SkipDir
			}
		}

		// Check if this is a SKILL.md file
		if info.Name() == "SKILL.md" {
			dir := filepath.Dir(path)
			relPath, _ := filepath.Rel(sd.rootPath, dir)
			dirName := filepath.Base(dir)

			discovered := &DiscoveredSkill{
				Name:        dirName,
				Path:        relPath,
				FullPath:    dir,
				SKILLMdPath: path,
			}
			allSkills = append(allSkills, discovered)

			// Check if directory name matches the skill name
			if strings.EqualFold(dirName, skillName) {
				dirMatches = append(dirMatches, discovered)
			}

			// Also check the frontmatter name field
			if metadata, err := ParseSkillMD(path); err == nil && metadata.Name != "" {
				if strings.EqualFold(metadata.Name, skillName) {
					// Update the discovered name to match the frontmatter
					discovered.Name = metadata.Name
					frontmatterMatches = append(frontmatterMatches, discovered)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking repository: %w", err)
	}

	// Priority 1: Directory name match (exact match preferred)
	if len(dirMatches) > 0 {
		for _, match := range dirMatches {
			if match.Name == skillName {
				return match, nil
			}
		}
		return dirMatches[0], nil
	}

	// Priority 2: Frontmatter name match (exact match preferred)
	if len(frontmatterMatches) > 0 {
		for _, match := range frontmatterMatches {
			if match.Name == skillName {
				return match, nil
			}
		}
		return frontmatterMatches[0], nil
	}

	// Priority 3: If there's only one skill in the entire repo, use that
	// This handles repos where the skill name doesn't match directory name
	if len(allSkills) == 1 {
		return allSkills[0], nil
	}

	if len(allSkills) == 0 {
		return nil, fmt.Errorf("skill '%s' not found - no SKILL.md files in repository", skillName)
	}

	// Multiple skills found but none match - return error with list so caller can show selection
	return nil, &MultipleSkillsError{
		RequestedName: skillName,
		FoundSkills:   allSkills,
	}
}

// FindAllSkills discovers all skills in the repository by finding all SKILL.md files
func (sd *SkillDiscovery) FindAllSkills() ([]*DiscoveredSkill, error) {
	var skills []*DiscoveredSkill

	err := filepath.Walk(sd.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		if info.Name() == "SKILL.md" {
			dir := filepath.Dir(path)
			relPath, _ := filepath.Rel(sd.rootPath, dir)

			skills = append(skills, &DiscoveredSkill{
				Name:        filepath.Base(dir),
				Path:        relPath,
				FullPath:    dir,
				SKILLMdPath: path,
			})
		}

		return nil
	})

	return skills, err
}
