package targets

var WindsurfTarget = &Target{
	Name:             "windsurf",
	DisplayName:      "Windsurf",
	ConfigDir:        ".windsurf",
	RulesDir:         "rules",
	SkillsDir:        "skills",
	AgentsDir:        "agents",
	HooksFile:        "",
	HooksScriptsDir:  "",
	MCPFile:          "mcp.json",
	SupportsAgentsMD: true,
}
