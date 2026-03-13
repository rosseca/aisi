package targets

var JunieTarget = &Target{
	Name:             "junie",
	DisplayName:      "Junie (JetBrains)",
	ConfigDir:        ".junie",
	RulesDir:         "",
	SkillsDir:        "skills",
	AgentsDir:        "agents",
	HooksFile:        "",
	HooksScriptsDir:  "",
	MCPFile:          "mcp/mcp.json",
	SupportsAgentsMD: true,
}
