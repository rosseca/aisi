package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rosseca/aisi/internal/config"
	"github.com/rosseca/aisi/internal/installer"
	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/registry"
	"github.com/rosseca/aisi/internal/repo"
	"github.com/rosseca/aisi/internal/targets"
	"github.com/rosseca/aisi/internal/tracker"
	"github.com/rosseca/aisi/internal/version"
)

type State int

const (
	StateWelcome State = iota // First run - show welcome and ask for repo
	StateRepoSetup            // Ask for repository URL on first run
	StateMainMenu
	StateBrowser
	StateInstalled
	StateInstalledBrowser
	StateMCPEnvForm   // Form for collecting MCP environment variables
	StateSkillURLForm // Form for installing skills from URL
	StateFindSkill    // Search for skills in the registry
	StateSkillDetail  // Show skill details before installing
	StateSettings
	StateSwitchTarget
	StateLoading
	StateInstalling
	StateError
	StateVersionError // CLI version too old - requires update
)

// VersionMismatchMsg is sent when the CLI version is below minimum required
type VersionMismatchMsg struct {
	CurrentVersion  string
	RequiredVersion string
}

type App struct {
	state            State
	mainMenu         *MainMenu
	browser          *Browser
	installedBrowser *InstalledBrowser
	mcpEnvForm       *MCPEnvForm
	skillFinder      *SkillFinder
	skillDetail      *SkillDetail
	cfg              *config.Config
	target           *targets.Target
	repoMgr          *repo.Manager
	manifest         *manifest.Manifest
	projectRoot      string
	err              error
	width            int
	height           int
	repoSource       string // Display info about repo source
	isFirstRun       bool   // True if no config file exists yet

	// Repo setup input (for first run)
	repoInput       string
	repoInputCursor int
	repoInputError  string

	// Version mismatch info
	versionCurrent  string
	versionRequired string

	// Installation progress
	spinner       spinner.Model
	installMsg    string
	installTotal  int
	installDone   int

	// Pending MCP installation
	pendingMCPItems []AssetItem // Items waiting to be installed after env form

	// Skill URL form input
	skillURLInput        string
	skillURLInputCursor  int
	skillURLError        string
	skillNameInput       string
	skillNameInputCursor int
	skillURLActiveInput  int // 0 = URL, 1 = Name
}

func NewApp(cfg *config.Config, target *targets.Target, projectRoot string, configExists bool) *App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	app := &App{
		state:       StateMainMenu,
		mainMenu:    NewMainMenu(target.DisplayName),
		cfg:         cfg,
		target:      target,
		projectRoot: projectRoot,
		spinner:     s,
		isFirstRun:  !configExists,
	}

	// Check if repo is configured
	hasRepo := cfg.Repo.URL != ""

	// If first run or no repo configured, start with welcome screen
	if !configExists || !hasRepo {
		app.isFirstRun = true
		app.state = StateWelcome
	}

	// Set repo source info for display
	if cfg.Repo.URL != "" {
		// Show shortened URL or path
		url := cfg.Repo.URL
		if len(url) > 40 {
			url = url[:37] + "..."
		}
		app.repoSource = url
	}

	return app
}

func (a *App) Init() tea.Cmd {
	// If it's the first run, show welcome screen
	if a.isFirstRun {
		return a.mainMenu.Init()
	}
	return tea.Batch(
		a.mainMenu.Init(),
		a.loadRepo,
	)
}

func (a *App) loadRepo() tea.Msg {
	if !a.cfg.IsConfigured() {
		return RepoNotConfiguredMsg{}
	}

	mgr, err := repo.NewManager(a.cfg)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	if err := mgr.EnsureMainRepo(); err != nil {
		return ErrorMsg{Err: err}
	}

	manifestPath := mgr.GetManifestPath()
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	// Check CLI version compatibility
	if err := m.CheckCLIVersion(version.Version); err != nil {
		if versionErr, ok := err.(*manifest.VersionMismatchError); ok {
			return VersionMismatchMsg{
				CurrentVersion:  versionErr.CurrentVersion,
				RequiredVersion: versionErr.RequiredVersion,
			}
		}
		return ErrorMsg{Err: err}
	}

	return RepoLoadedMsg{
		Manager:  mgr,
		Manifest: m,
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.mcpEnvForm != nil {
			a.mcpEnvForm.SetSize(a.width, a.height)
		}
		// Propagate to browser if active
		if a.browser != nil && a.state == StateBrowser {
			_, cmd := a.browser.Update(msg)
			return a, cmd
		}

	case tea.KeyMsg:
		// Handle state-specific key input first
		switch a.state {
		case StateWelcome:
			// Any key on welcome screen moves to repo setup
			a.state = StateRepoSetup
			return a, nil

		case StateRepoSetup:
			return a.handleRepoSetupInput(msg)

		case StateSkillURLForm:
			return a.handleSkillURLInput(msg)

		case StateVersionError:
			// Only allow 'q' or 'ctrl+c' to quit when version is outdated
			if msg.String() == "q" || msg.Type == tea.KeyCtrlC {
				return a, tea.Quit
			}
			return a, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "esc":
			if a.state != StateMainMenu && a.state != StateInstalling && a.state != StateVersionError {
				a.state = StateMainMenu
				return a, nil
			}
		}

	// Handle spinner tick for loading animation
	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case RepoLoadedMsg:
		a.repoMgr = msg.Manager
		a.manifest = msg.Manifest
		return a, nil

	case RepoNotConfiguredMsg:
		a.err = fmt.Errorf("repository not configured. Run: aisi config set-repo <url>")
		return a, nil

	case ErrorMsg:
		a.err = msg.Err
		a.state = StateError
		return a, nil

	case VersionMismatchMsg:
		a.state = StateVersionError
		a.versionCurrent = msg.CurrentVersion
		a.versionRequired = msg.RequiredVersion
		return a, nil

	case MenuSelectedMsg:
		return a.handleMenuSelection(msg.Option)

	case BrowserDoneMsg:
		a.state = StateMainMenu
		return a, nil

	case InstalledBrowserDoneMsg:
		a.state = StateMainMenu
		return a, nil

	case SkillFinderDoneMsg:
		a.state = StateMainMenu
		a.skillFinder = nil
		return a, nil

	case SkillSelectedMsg:
		// Show skill detail view before installing
		a.skillDetail = NewSkillDetail(msg.Skill)
		a.skillDetail.SetSize(a.width, a.height)
		a.state = StateSkillDetail
		return a, a.skillDetail.Init()

	case SkillDetailDoneMsg:
		if msg.Confirmed {
			// Install the selected skill from registry
			a.state = StateInstalling
			a.installMsg = fmt.Sprintf("Installing %s...", msg.Skill.Name)
			a.installTotal = 1
			a.installDone = 0
			return a, tea.Batch(
				a.spinner.Tick,
				a.handleRegistrySkillInstall(msg.Skill),
			)
		}
		// Cancelled, go back to skill finder
		a.state = StateFindSkill
		a.skillDetail = nil
		return a, nil

	case SkillInstallErrorMsg:
		a.err = msg.Err
		a.state = StateError
		return a, nil

	case InstallRequestMsg:
		// Check if any items are MCPs that need env vars
		for _, item := range msg.Items {
			if mcp := a.manifest.GetMCP(item.Name); mcp != nil && len(mcp.Env) > 0 {
				// Found an MCP that needs env vars - show the form
				a.pendingMCPItems = msg.Items
				a.mcpEnvForm = NewMCPEnvForm(mcp)
				a.mcpEnvForm.SetSize(a.width, a.height)
				a.state = StateMCPEnvForm
				return a, a.mcpEnvForm.Init()
			}
		}

		// No MCPs with env vars - proceed with normal installation
		a.state = StateInstalling
		a.installTotal = len(msg.Items)
		a.installDone = 0
		a.installMsg = "Starting installation..."
		return a, tea.Batch(
			a.spinner.Tick,
			a.handleInstall(msg.Items),
		)

	case InstallProgressMsg:
		a.installDone = msg.Done
		a.installTotal = msg.Total
		a.installMsg = msg.Message
		return a, nil

	case InstallCompletedMsg:
		// Return to main menu after installation
		a.state = StateMainMenu
		// Could show a success message here
		return a, nil

	case UninstallRequestMsg:
		return a, a.handleUninstall(msg.Items)

	case UninstallCompletedMsg:
		a.state = StateMainMenu
		return a, nil

	case TargetSwitchedMsg:
		a.target = msg.Target
		a.mainMenu = NewMainMenu(a.target.DisplayName)
		a.state = StateMainMenu
		return a, nil

	case MCPEnvCompletedMsg:
		if msg.Canceled {
			a.state = StateBrowser
			a.mcpEnvForm = nil
			return a, nil
		}
		// Continue with installation of this MCP and remaining items
		a.state = StateInstalling
		a.installTotal = len(a.pendingMCPItems)
		a.installDone = 0
		a.installMsg = fmt.Sprintf("Installing %s...", msg.MCP.Name)
		return a, tea.Batch(
			a.spinner.Tick,
			a.handleMCPInstall(msg.MCP, msg.EnvVars, a.pendingMCPItems),
		)
	}

	switch a.state {
	case StateWelcome:
		// On welcome screen, any key press continues to main menu
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = StateMainMenu
			return a, a.loadRepo
		}

	case StateMainMenu:
		newMenu, cmd := a.mainMenu.Update(msg)
		a.mainMenu = newMenu.(*MainMenu)
		return a, cmd

	case StateBrowser:
		if a.browser != nil {
			newBrowser, cmd := a.browser.Update(msg)
			a.browser = newBrowser.(*Browser)
			return a, cmd
		}

	case StateInstalledBrowser:
		if a.installedBrowser != nil {
			newBrowser, cmd := a.installedBrowser.Update(msg)
			a.installedBrowser = newBrowser.(*InstalledBrowser)
			return a, cmd
		}

	case StateMCPEnvForm:
		if a.mcpEnvForm != nil {
			newForm, cmd := a.mcpEnvForm.Update(msg)
			a.mcpEnvForm = newForm.(*MCPEnvForm)
			return a, cmd
		}

	case StateFindSkill:
		if a.skillFinder != nil {
			newFinder, cmd := a.skillFinder.Update(msg)
			a.skillFinder = newFinder.(*SkillFinder)
			return a, cmd
		}

	case StateSkillDetail:
		if a.skillDetail != nil {
			newDetail, cmd := a.skillDetail.Update(msg)
			a.skillDetail = newDetail.(*SkillDetail)
			return a, cmd
		}

	case StateSwitchTarget:
		return a.updateTargetSwitcher(msg)
	}

	return a, nil
}

func (a *App) handleMenuSelection(option MenuOption) (tea.Model, tea.Cmd) {
	switch option {
	case MenuBrowseInstall:
		if a.manifest == nil {
			a.err = fmt.Errorf("repository not loaded")
			return a, nil
		}
		a.browser = NewBrowser(a.manifest, a.target)
		a.browser.SetSize(a.width, a.height)
		a.state = StateBrowser
		return a, a.browser.Init()

	case MenuInstallFromURL:
		a.skillURLInput = ""
		a.skillURLInputCursor = 0
		a.skillURLError = ""
		a.skillNameInput = ""
		a.skillNameInputCursor = 0
		a.skillURLActiveInput = 0
		a.state = StateSkillURLForm
		return a, nil

	case MenuFindSkill:
		a.skillFinder = NewSkillFinder()
		a.skillFinder.SetSize(a.width, a.height)
		a.state = StateFindSkill
		return a, a.skillFinder.Init()

	case MenuViewInstalled:
		track := tracker.New(a.projectRoot, a.target)
		a.installedBrowser = NewInstalledBrowser(track, a.target.DisplayName)
		a.state = StateInstalledBrowser
		return a, a.installedBrowser.Init()

	case MenuUpdateAll:
		return a, nil

	case MenuSwitchTarget:
		a.state = StateSwitchTarget
		return a, nil

	case MenuSettings:
		a.state = StateSettings
		return a, nil

	case MenuExit:
		return a, tea.Quit
	}

	return a, nil
}

func (a *App) updateTargetSwitcher(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			t, _ := targets.Get("cursor")
			return a, func() tea.Msg { return TargetSwitchedMsg{Target: t} }
		case "2":
			t, _ := targets.Get("kilo")
			return a, func() tea.Msg { return TargetSwitchedMsg{Target: t} }
		case "3":
			t, _ := targets.Get("junie")
			return a, func() tea.Msg { return TargetSwitchedMsg{Target: t} }
			case "esc", "q":
			a.state = StateMainMenu
		}
	}
	return a, nil
}

func (a *App) View() string {
	switch a.state {
	case StateWelcome:
		return a.renderWelcome()

	case StateRepoSetup:
		return a.renderRepoSetup()

	case StateMainMenu:
		return a.renderMainMenu()

	case StateBrowser:
		if a.browser != nil {
			return a.browser.View()
		}

	case StateInstalledBrowser:
		if a.installedBrowser != nil {
			return a.installedBrowser.View()
		}

	case StateMCPEnvForm:
		if a.mcpEnvForm != nil {
			return a.mcpEnvForm.View()
		}

	case StateFindSkill:
		if a.skillFinder != nil {
			return a.skillFinder.View()
		}

	case StateSkillDetail:
		if a.skillDetail != nil {
			return a.skillDetail.View()
		}

	case StateSkillURLForm:
		return a.renderSkillURLForm()

	case StateSwitchTarget:
		return a.renderTargetSwitcher()

	case StateSettings:
		return a.renderSettings()

	case StateInstalling:
		return a.renderInstalling()

	case StateError:
		return a.renderError()

	case StateVersionError:
		return a.renderVersionError()
	}

	return ""
}

func (a *App) renderWelcome() string {
	if a.width == 0 {
		a.width = 80
	}

	title := titleStyle.Render("  🧠 AI Shared Intelligence v1.0.0  ")
	boxWidth := a.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}

	content := "\n"
	content += infoStyle.Render("⚡ First Run Detected") + "\n\n"
	content += "Welcome to AI Shared Intelligence! This appears to be your first time running\n"
	content += "the CLI.\n\n"

	content += infoStyle.Render("📁 Configuration Location") + "\n"
	content += "  Global config:  " + secondaryStyle.Render("~/.aisi/config.yaml") + "\n"
	content += "  Cache directory: " + secondaryStyle.Render("~/.aisi/cache/") + "\n\n"

	content += infoStyle.Render("⚙️  Current Settings") + "\n"
	content += "  Target: " + secondaryStyle.Render(a.target.DisplayName) + "\n"
	content += "  Project: " + secondaryStyle.Render(a.projectRoot) + "\n\n"

	content += lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("Press any key to set up your repository...")

	box := boxStyle.Width(boxWidth).Render(content)

	return lipgloss.Place(
		a.width,
		a.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			"",
			box,
		),
	)
}

func (a *App) handleRepoSetupInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Validate and save repo URL
		if a.repoInput == "" {
			a.repoInputError = "Repository URL cannot be empty"
			return a, nil
		}
		// Save the repo URL
		a.cfg.SetRepo(a.repoInput, "")
		if err := a.cfg.Save(); err != nil {
			a.err = fmt.Errorf("failed to save config: %w", err)
			a.state = StateError
			return a, nil
		}
		// Update repo source display
		url := a.repoInput
		if len(url) > 40 {
			url = url[:37] + "..."
		}
		a.repoSource = url
		// Transition to main menu and load repo
		a.isFirstRun = false
		a.state = StateMainMenu
		return a, a.loadRepo

	case tea.KeyBackspace:
		if len(a.repoInput) > 0 && a.repoInputCursor > 0 {
			a.repoInput = a.repoInput[:a.repoInputCursor-1] + a.repoInput[a.repoInputCursor:]
			a.repoInputCursor--
		}
		return a, nil

	case tea.KeyDelete:
		if a.repoInputCursor < len(a.repoInput) {
			a.repoInput = a.repoInput[:a.repoInputCursor] + a.repoInput[a.repoInputCursor+1:]
		}
		return a, nil

	case tea.KeyLeft:
		if a.repoInputCursor > 0 {
			a.repoInputCursor--
		}
		return a, nil

	case tea.KeyRight:
		if a.repoInputCursor < len(a.repoInput) {
			a.repoInputCursor++
		}
		return a, nil

	case tea.KeyHome:
		a.repoInputCursor = 0
		return a, nil

	case tea.KeyEnd:
		a.repoInputCursor = len(a.repoInput)
		return a, nil

	case tea.KeyCtrlC:
		return a, tea.Quit

	default:
		// Insert character
		if msg.Type == tea.KeyRunes {
			a.repoInput = a.repoInput[:a.repoInputCursor] + string(msg.Runes) + a.repoInput[a.repoInputCursor:]
			a.repoInputCursor += len(msg.Runes)
			a.repoInputError = "" // Clear error on input
		}
		return a, nil
	}
}

func (a *App) renderRepoSetup() string {
	if a.width == 0 {
		a.width = 80
	}

	title := titleStyle.Render("  🧠 AI Shared Intelligence v1.0.0  ")
	boxWidth := a.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}

	content := "\n"
	content += infoStyle.Render("🔗 Repository Setup") + "\n\n"
	content += "Please enter the URL of your shared intelligence repository.\n"
	content += "This can be an SSH or HTTPS GitHub URL, or a local path.\n\n"
	content += "Examples:\n"
	content += "  " + dimStyle.Render("git@github.com:your-org/shared-intelligence.git") + "\n"
	content += "  " + dimStyle.Render("https://github.com/your-org/shared-intelligence") + "\n"
	content += "  " + dimStyle.Render("./local/path/to/repo") + "\n\n"

	// Input field
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 1).
		Width(boxWidth - 10)

	cursorChar := "█"
	inputText := a.repoInput
	if a.repoInputCursor <= len(inputText) {
		inputText = inputText[:a.repoInputCursor] + cursorChar + inputText[a.repoInputCursor:]
	}

	content += "Repository URL:\n"
	content += inputStyle.Render(inputText) + "\n\n"

	if a.repoInputError != "" {
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("⚠ "+a.repoInputError) + "\n\n"
	}

	content += dimStyle.Render("Press Enter to continue • Ctrl+C to quit")

	box := boxStyle.Width(boxWidth).Render(content)

	return lipgloss.Place(
		a.width,
		a.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			"",
			box,
		),
	)
}

func (a *App) renderMainMenu() string {
	// Add repo info to the main menu view
	menuView := a.mainMenu.View()
	// Replace the title section with our enhanced version
	titleWithRepo := renderTitleWithRepo(a.target.DisplayName, a.repoSource)
	// Simple string replacement - find the old title and replace
	oldTitle := renderTitle(a.target.DisplayName)
	return strings.Replace(menuView, oldTitle, titleWithRepo, 1)
}

func (a *App) renderTargetSwitcher() string {
	return fmt.Sprintf(`
%s

  Select target:

  [1] Cursor
  [2] Kilo Code
  [3] Junie (JetBrains)

%s
`, renderTitleWithRepo(a.target.DisplayName, a.repoSource), helpStyle.Render("1-3: Select • Esc: Back"))
}


func (a *App) renderSettings() string {
	repoURL := "Not configured"
	if a.cfg.Repo.URL != "" {
		repoURL = a.cfg.Repo.URL
	}

	return fmt.Sprintf(`
%s

  Settings:

    Repository: %s
    Branch: %s
    Active Target: %s

%s
`, renderTitleWithRepo(a.target.DisplayName, a.repoSource), repoURL, a.cfg.Repo.Branch, a.cfg.ActiveTarget, helpStyle.Render("Esc: Back"))
}

func (a *App) renderInstalling() string {
	progress := ""
	if a.installTotal > 0 {
		progress = fmt.Sprintf(" (%d/%d)", a.installDone, a.installTotal)
	}

	return fmt.Sprintf(`
%s

  %s Installing assets%s...

  %s

%s
`, renderTitleWithRepo(a.target.DisplayName, a.repoSource), a.spinner.View(), progress, a.installMsg, helpStyle.Render("Please wait..."))
}

func (a *App) renderError() string {
	errMsg := "Unknown error"
	if a.err != nil {
		errMsg = a.err.Error()
	}

	// Check if this is a private repo error for better formatting
	if strings.Contains(errMsg, "private or deleted repository") {
		return fmt.Sprintf(`
|%s
|
|  %s
|
|  %s
|
|%s
|`, renderTitleWithRepo(a.target.DisplayName, a.repoSource),
			lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("⚠️  Installation Failed"),
			errMsg,
			helpStyle.Render("Esc: Back to menu • q: Quit"))
	}

	return fmt.Sprintf(`
|%s
|
|  %s
|
|%s
|`, renderTitleWithRepo(a.target.DisplayName, a.repoSource), errorStyle.Render("Error: "+errMsg), helpStyle.Render("Esc: Back • q: Quit"))
}

func (a *App) renderVersionError() string {
	if a.width == 0 {
		a.width = 80
	}

	boxWidth := a.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}

	title := titleStyle.Render("  ⚠️  Update Required  ")

	content := "\n"
	content += errorStyle.Render("Your CLI version is outdated") + "\n\n"

	content += infoStyle.Render("Current Version") + "\n"
	content += "  " + secondaryStyle.Render(a.versionCurrent) + "\n\n"

	content += lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("Required Version") + "\n"
	content += "  " + secondaryStyle.Render(a.versionRequired) + "\n\n"

	content += dimStyle.Render("Please update your CLI to continue")

	box := boxStyle.Width(boxWidth).Render(content)

	return lipgloss.Place(
		a.width,
		a.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			"",
			box,
		),
	)
}

type RepoLoadedMsg struct {
	Manager  *repo.Manager
	Manifest *manifest.Manifest
}

type RepoNotConfiguredMsg struct{}

type ErrorMsg struct {
	Err error
}

type TargetSwitchedMsg struct {
	Target *targets.Target
}

type InstallProgressMsg struct {
	Done    int
	Total   int
	Message string
}

type InstallCompletedMsg struct {
	SuccessCount int
	Errors       []string
}

type UninstallCompletedMsg struct {
	SuccessCount int
	Errors       []string
}

type SkillInstallErrorMsg struct {
	Err error
}

func (a *App) handleUninstall(items []InstalledItem) tea.Cmd {
	return func() tea.Msg {
		inst := installer.New(a.repoMgr, a.target, a.projectRoot)
		track := tracker.New(a.projectRoot, a.target)
		successCount := 0
		errors := []string{}

		for _, item := range items {
			// Convert string type to AssetType
			var assetType manifest.AssetType
			switch item.Type {
			case "rule":
				assetType = manifest.AssetTypeRule
			case "skill":
				assetType = manifest.AssetTypeSkill
			case "agent":
				assetType = manifest.AssetTypeAgent
			case "hook":
				assetType = manifest.AssetTypeHook
			case "mcp":
				assetType = manifest.AssetTypeMCP
			case "agentsmd":
				assetType = manifest.AssetTypeAgentsMD
			default:
				assetType = manifest.AssetType("external:" + item.Type)
			}

			// First remove from lock file
			if err := track.Remove(assetType, item.Name); err != nil {
				errors = append(errors, fmt.Sprintf("%s (lock): %v", item.Name, err))
				continue
			}

			// Then remove actual files
			if err := inst.Uninstall(assetType, item.Name); err != nil {
				errors = append(errors, fmt.Sprintf("%s (files): %v", item.Name, err))
			} else {
				successCount++
			}
		}

		return UninstallCompletedMsg{
			SuccessCount: successCount,
			Errors:       errors,
		}
	}
}

func (a *App) handleInstall(items []AssetItem) tea.Cmd {
	return func() tea.Msg {
		if a.repoMgr == nil || a.manifest == nil {
			return ErrorMsg{Err: fmt.Errorf("repository not loaded")}
		}

		inst := installer.New(a.repoMgr, a.target, a.projectRoot)
		track := tracker.New(a.projectRoot, a.target)
		commit, _ := a.repoMgr.GetCurrentCommit()
		repoURL := a.cfg.Repo.URL

		successCount := 0
		errors := []string{}

		for i, item := range items {
			// Update progress message - this won't show real-time in TUI but indicates activity
			a.installDone = i + 1
			a.installMsg = fmt.Sprintf("Installing %s...", item.Name)

			// Check for external requirements (just logs, doesn't block installation)
			_ = a.manifest.GetExternal(item.Name)

			result, err := inst.Install(a.manifest, item.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", item.Name, err))
				continue
			}

			if result.Success {
				_ = track.RecordInstall(result.Type, result.Name, repoURL, commit)
				successCount++
			} else {
				errors = append(errors, fmt.Sprintf("%s: %v", item.Name, result.Error))
			}
		}

		return InstallCompletedMsg{
			SuccessCount: successCount,
			Errors:       errors,
		}
	}
}

func (a *App) handleMCPInstall(mcp *manifest.MCP, envVars map[string]installer.EnvVarConfig, allItems []AssetItem) tea.Cmd {
	return func() tea.Msg {
		if a.repoMgr == nil {
			return ErrorMsg{Err: fmt.Errorf("repository not loaded")}
		}

		inst := installer.New(a.repoMgr, a.target, a.projectRoot)
		track := tracker.New(a.projectRoot, a.target)
		commit, _ := a.repoMgr.GetCurrentCommit()
		repoURL := a.cfg.Repo.URL

		errors := []string{}
		successCount := 0

		// Install the MCP with env vars first
		result, err := inst.InstallMCP(mcp, envVars)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", mcp.Name, err))
		} else if result.Success {
			_ = track.RecordInstall(result.Type, result.Name, repoURL, commit)
			successCount++
		} else {
			errors = append(errors, fmt.Sprintf("%s: %v", mcp.Name, result.Error))
		}

		// Continue with remaining items
		for _, item := range allItems {
			if item.Name == mcp.Name {
				continue // Skip the already installed MCP
			}

			a.installDone++
			a.installMsg = fmt.Sprintf("Installing %s...", item.Name)

			result, err := inst.Install(a.manifest, item.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", item.Name, err))
				continue
			}

			if result.Success {
				_ = track.RecordInstall(result.Type, result.Name, repoURL, commit)
				successCount++
			} else {
				errors = append(errors, fmt.Sprintf("%s: %v", item.Name, result.Error))
			}
		}

		return InstallCompletedMsg{
			SuccessCount: successCount,
			Errors:       errors,
		}
	}
}

func (a *App) handleSkillURLInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Validate and install
		if a.skillURLInput == "" {
			a.skillURLError = "Repository URL cannot be empty"
			return a, nil
		}

		// Parse URL
		skillURL, err := repo.ParseSkillURL(a.skillURLInput)
		if err != nil {
			a.skillURLError = fmt.Sprintf("Invalid URL: %v", err)
			return a, nil
		}

		// Start installation
		a.state = StateInstalling
		a.installMsg = fmt.Sprintf("Installing skill from %s...", a.skillURLInput)
		a.installTotal = 1
		a.installDone = 0

		return a, tea.Batch(
			a.spinner.Tick,
			a.handleSkillURLInstall(skillURL, a.skillNameInput),
		)

	case tea.KeyTab:
		// Switch between URL and name inputs
		a.skillURLActiveInput = (a.skillURLActiveInput + 1) % 2
		return a, nil

	case tea.KeyBackspace:
		if a.skillURLActiveInput == 0 {
			if len(a.skillURLInput) > 0 && a.skillURLInputCursor > 0 {
				a.skillURLInput = a.skillURLInput[:a.skillURLInputCursor-1] + a.skillURLInput[a.skillURLInputCursor:]
				a.skillURLInputCursor--
			}
		} else {
			if len(a.skillNameInput) > 0 && a.skillNameInputCursor > 0 {
				a.skillNameInput = a.skillNameInput[:a.skillNameInputCursor-1] + a.skillNameInput[a.skillNameInputCursor:]
				a.skillNameInputCursor--
			}
		}
		return a, nil

	case tea.KeyDelete:
		if a.skillURLActiveInput == 0 {
			if a.skillURLInputCursor < len(a.skillURLInput) {
				a.skillURLInput = a.skillURLInput[:a.skillURLInputCursor] + a.skillURLInput[a.skillURLInputCursor+1:]
			}
		} else {
			if a.skillNameInputCursor < len(a.skillNameInput) {
				a.skillNameInput = a.skillNameInput[:a.skillNameInputCursor] + a.skillNameInput[a.skillNameInputCursor+1:]
			}
		}
		return a, nil

	case tea.KeyLeft:
		if a.skillURLActiveInput == 0 {
			if a.skillURLInputCursor > 0 {
				a.skillURLInputCursor--
			}
		} else {
			if a.skillNameInputCursor > 0 {
				a.skillNameInputCursor--
			}
		}
		return a, nil

	case tea.KeyRight:
		if a.skillURLActiveInput == 0 {
			if a.skillURLInputCursor < len(a.skillURLInput) {
				a.skillURLInputCursor++
			}
		} else {
			if a.skillNameInputCursor < len(a.skillNameInput) {
				a.skillNameInputCursor++
			}
		}
		return a, nil

	case tea.KeyHome:
		if a.skillURLActiveInput == 0 {
			a.skillURLInputCursor = 0
		} else {
			a.skillNameInputCursor = 0
		}
		return a, nil

	case tea.KeyEnd:
		if a.skillURLActiveInput == 0 {
			a.skillURLInputCursor = len(a.skillURLInput)
		} else {
			a.skillNameInputCursor = len(a.skillNameInput)
		}
		return a, nil

	case tea.KeyEsc:
		a.state = StateMainMenu
		return a, nil

	case tea.KeyCtrlC:
		return a, tea.Quit

	case tea.KeyF2:
		// Switch to skill finder with F2
		a.state = StateFindSkill
		a.skillFinder = NewSkillFinder()
		a.skillFinder.SetSize(a.width, a.height)
		return a, a.skillFinder.Init()

	default:
		// Handle character input
		if msg.Type == tea.KeyRunes {
			// Regular character input
			if a.skillURLActiveInput == 0 {
				a.skillURLInput = a.skillURLInput[:a.skillURLInputCursor] + string(msg.Runes) + a.skillURLInput[a.skillURLInputCursor:]
				a.skillURLInputCursor += len(msg.Runes)
				a.skillURLError = "" // Clear error on input
			} else {
				a.skillNameInput = a.skillNameInput[:a.skillNameInputCursor] + string(msg.Runes) + a.skillNameInput[a.skillNameInputCursor:]
				a.skillNameInputCursor += len(msg.Runes)
			}
		}
		return a, nil
	}
}

func (a *App) renderSkillURLForm() string {
	if a.width == 0 {
		a.width = 80
	}

	title := titleStyle.Render("  🧠 AI Shared Intelligence  ")
	boxWidth := a.width - 8
	if boxWidth < 60 {
		boxWidth = 60
	}

	content := "\n"
	content += infoStyle.Render("🔗 Install Skill from Repository") + "\n\n"
	content += "Enter the repository URL and optional custom name.\n\n"

	// URL Input
	urlInputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 1).
		Width(boxWidth - 10)

	if a.skillURLActiveInput != 0 {
		urlInputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(boxWidth - 10)
	}

	cursorChar := "█"
	urlText := a.skillURLInput
	if a.skillURLActiveInput == 0 && a.skillURLInputCursor <= len(urlText) {
		urlText = urlText[:a.skillURLInputCursor] + cursorChar + urlText[a.skillURLInputCursor:]
	}

	content += "Repository URL:\n"
	content += urlInputStyle.Render(urlText) + "\n\n"

	// Name Input
	nameInputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(boxWidth - 10)

	if a.skillURLActiveInput == 1 {
		nameInputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1).
			Width(boxWidth - 10)
	}

	nameText := a.skillNameInput
	if a.skillURLActiveInput == 1 && a.skillNameInputCursor <= len(nameText) {
		nameText = nameText[:a.skillNameInputCursor] + cursorChar + nameText[a.skillNameInputCursor:]
	}
	if nameText == "" {
		nameText = " " // Ensure cursor is visible in empty field
	}

	content += "Custom Name (optional):\n"
	content += nameInputStyle.Render(nameText) + "\n\n"

	// Examples
	content += dimStyle.Render("Examples:") + "\n"
	content += "  " + dimStyle.Render("vercel-labs/agent-skills") + "\n"
	content += "  " + dimStyle.Render("https://github.com/user/repo/tree/main/skills/foo") + "\n"
	content += "  " + dimStyle.Render("git@github.com:org/repo.git") + "\n\n"

	if a.skillURLError != "" {
		content += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("⚠ "+a.skillURLError) + "\n\n"
	}

	content += dimStyle.Render("Tab: Switch field • Enter: Install • f: Find Skills • Esc: Back • Ctrl+C: Quit")

	box := boxStyle.Width(boxWidth).Render(content)

	return lipgloss.Place(
		a.width,
		a.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			"",
			box,
		),
	)
}

func (a *App) handleRegistrySkillInstall(skill registry.Skill) tea.Cmd {
	return func() tea.Msg {
		source := skill.Source
		if source == "" {
			// Parse from ID (owner/repo/skill-name)
			parts := strings.Split(skill.ID, "/")
			if len(parts) >= 2 {
				source = strings.Join(parts[:2], "/")
			}
		}

		// Parse URL
		skillURLStr := fmt.Sprintf("%s@%s", source, skill.Name)
		skillURL, err := repo.ParseSkillURL(skillURLStr)
		if err != nil {
			return SkillInstallErrorMsg{Err: fmt.Errorf("failed to parse skill URL: %w", err)}
		}

		var repoMgr *repo.Manager

		// Only need repo manager for git URLs (not local paths)
		if !skillURL.IsLocal {
			cfg := a.cfg
			var err error
			repoMgr, err = repo.NewManager(cfg)
			if err != nil {
				return SkillInstallErrorMsg{Err: fmt.Errorf("failed to create repository manager: %w", err)}
			}
		}

		inst := installer.New(repoMgr, a.target, a.projectRoot)
		track := tracker.New(a.projectRoot, a.target)

		// Install the skill from URL
		result, err := inst.InstallSkillFromURL(skillURL, "")
		if err != nil {
			// Check for specific error types
			errStr := err.Error()
			if strings.Contains(errStr, "repository not found") || strings.Contains(errStr, "not accessible") {
				return SkillInstallErrorMsg{
					Err: fmt.Errorf("⚠️  This skill is from a private or deleted repository (%s)\n\nThe repository may have been removed, made private, or the owner may have changed their username.\n\nTry installing from a different source or contact the skill author.", source),
				}
			}
			return SkillInstallErrorMsg{Err: err}
		}

		if result.Success {
			a.installDone = 1
			a.installMsg = fmt.Sprintf("Installed %s", result.Name)
			// Record with full source information
			// Don't modify project repoURL/repoCommit when installing from external source
			skillEntry := tracker.SkillEntry{
				Name:   result.Name,
				Source: source,
				Path:   result.SourcePath, // Use the actual discovered path
			}
			_ = track.RecordSkillInstallOnly(skillEntry)
		} else {
			// Check for specific error types
			if result.Error != nil {
				errStr := result.Error.Error()
				if strings.Contains(errStr, "repository not found") || strings.Contains(errStr, "not accessible") {
					return SkillInstallErrorMsg{
						Err: fmt.Errorf("⚠️  This skill is from a private or deleted repository (%s)\n\nThe repository may have been removed, made private, or the owner may have changed their username.\n\nTry installing from a different source or contact the skill author.", source),
					}
				}
			}
			return SkillInstallErrorMsg{Err: result.Error}
		}

		return InstallCompletedMsg{
			SuccessCount: 1,
			Errors:       []string{},
		}
	}
}

func (a *App) handleSkillURLInstall(skillURL *repo.SkillURL, overrideName string) tea.Cmd {
	return func() tea.Msg {
		var repoMgr *repo.Manager

		// Only need repo manager for git URLs (not local paths)
		if !skillURL.IsLocal {
			cfg := a.cfg
			var err error
			repoMgr, err = repo.NewManager(cfg)
			if err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to create repository manager: %w", err)}
			}
		}

		inst := installer.New(repoMgr, a.target, a.projectRoot)
		track := tracker.New(a.projectRoot, a.target)

		// Install the skill from URL
		result, err := inst.InstallSkillFromURL(skillURL, overrideName)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		if result.Success {
			a.installDone = 1
			a.installMsg = fmt.Sprintf("Installed %s", result.Name)
			// Record with full source information
			// Don't modify project repoURL/repoCommit when installing from external source
			skillEntry := tracker.SkillEntry{
				Name:   result.Name,
				Source: skillURL.RepoURL,
				Path:   result.SourcePath, // Use the actual discovered path
			}
			_ = track.RecordSkillInstallOnly(skillEntry)
		} else {
			return ErrorMsg{Err: result.Error}
		}

		return InstallCompletedMsg{
			SuccessCount: 1,
			Errors:       []string{},
		}
	}
}

func Run(cfg *config.Config, target *targets.Target, projectRoot string, configExists bool) error {
	app := NewApp(cfg, target, projectRoot, configExists)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
