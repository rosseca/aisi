package targets

var CursorTarget = &Target{
	Name:             "cursor",
	DisplayName:      "Cursor",
	ConfigDir:        ".cursor",
	RulesDir:         "rules",
	SkillsDir:        "skills",
	AgentsDir:        "agents",
	HooksFile:        "hooks.json",
	HooksScriptsDir:  "hooks",
	MCPFile:          "mcp.json",
	SupportsAgentsMD: true,
}
