package targets

var KiloTarget = &Target{
	Name:                 "kilo",
	DisplayName:          "Kilo Code",
	ConfigDir:            ".kilocode",
	RulesDir:             "rules",
	SkillsDir:            "skills",
	AgentsDir:            "",
	HooksFile:            "",
	HooksScriptsDir:      "",
	MCPFile:              "mcp.json",
	SupportsAgentsMD:     true,
	SupportsModeSpecific: true,
}
