# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-03-19

### Added

- **Visual loading indicator during repository cache update**
  - Spinner animation shown while `git pull` runs on startup
  - Smooth transition from loading state to main menu when complete

## [0.4.1] - 2026-03-19

### Changed

- **Auto-update repository cache on startup** — no more stale manifests
  - TUI now runs `git pull` on the cached repository when loading
  - Ensures users always see the latest assets without manual `aisi update`
- **Lock file relocated to project root** — shared across all targets
  - `.aisi.lock` now saved at project root instead of `.cursor/` or `.kilo/`
  - Single source of truth for installed assets across all editor targets

## [0.4.0] - 2026-03-19
### Added
- Category filter for asset browser - assets can now be organized by categories
- Support for multiple categories per asset (`categories` array in manifest)
- New category selection screen before browsing assets
- `categories` field added to all asset types (rules, skills, agents, MCPs, etc.)
### Changed
- Assets without categories appear in "Other" category
- "All" category shows all available assets
- Skip category selection if no categories defined in manifest


## [0.3.0] - 2026-03-14

### Added

- **MCP Dependency Management** — because manually installing `uvx` or `npx` is so 2024
  - Automatic detection and installation of required system commands
  - Support for custom install scripts in MCP manifests
  - Progress reporting during dependency installation
- **Global MCP Installation** — install once, use everywhere
  - `InstallMCPGlobal()` for system-wide MCP configuration
  - Automatic backup of existing configs before modification
- **MCP Post-Install Hooks** — because one command is never enough
  - Run arbitrary commands after MCP installation
  - Environment variable injection support
- **Associated Skills for MCPs** — install the brain with the tool
  - Skills can reference external repositories via `SkillRef`
  - Automatic installation of companion skills alongside MCPs
- **External Skill References** — `IsLocal()` and `InstallFromURL()` for skill mobility

### Changed

- **TUI Browser now fills your terminal** like it should have from day one
  - Dynamic item count based on terminal height (no more arbitrary 15-item limit)
  - Better reserved space calculation for scroll indicators and category headers
- **Manifest v0.3** with expanded MCP schema
  - New fields: `command`, `install`, `postInstall`, `skill`

### Fixed

- Added missing `strings` import in `mcp.go` (compilation fail, classic)
- Browser scroll calculation now accounts for actual UI elements, not imaginary ones

## [0.2.0] - 2026-03-13

### Added

- Homebrew tap support — install via `brew tap rosseca/tap && brew install aisi`
- External skill installation — install skills directly from any Git repository without manifest registration
  - GitHub shorthand: `aisi install skill --url owner/repo`
  - Full URLs with path support: `aisi install skill --url https://github.com/owner/repo/tree/main/skills/foo`
  - Custom naming: `aisi install skill --url owner/repo --name my-skill`
  - SSH and GitLab support
- Improved error messages with context and suggestions

### Changed

- README restructured with clearer installation examples
- "Installing Skills from External Repositories" section reorganized following community best practices
- Homebrew promoted as recommended installation method for macOS

### Fixed

- Proper handling of skill directories with nested paths
- Lock file updates when reinstalling external skills

## [0.1.0] - 2026-03-13

### Added

- Initial release of AISI (AI Shared Intelligence CLI)
- Support for managing AI assets across multiple editors: Cursor, Kilo Code, Junie (JetBrains), Windsurf
- `install` command — install specific assets or all assets by type (rules, skills, agents, hooks, mcp, external)
- `list` command — browse available assets in the configured repository
- `status` command — check which assets are installed in the current project
- `update` command — refresh installed assets to the latest commit
- `config` command — manage repository URL, target editor, and global settings
- Interactive TUI mode (bubbletea) — run `aisi` without arguments for a guided interface
- Git-based repository syncing with local cache in `~/.aisi/`
- Lock file tracking (`.aisi.lock`) to track installed assets and their commits
- Manifest-based asset discovery with `manifest.json` format
- Support for external assets from third-party repositories
- GoReleaser configuration for cross-platform builds
- Comprehensive test suite with >80% coverage
- CI/CD pipeline with GitHub Actions

### Technical

- Clean architecture with separation between commands, installers, and infrastructure
- Modular installer system supporting multiple asset types
- Target abstraction for editor-specific directory structures
- Tracker system for lock file management
- Repository abstraction for Git operations

---

[0.5.0]: https://github.com/rosseca/aisi/releases/tag/v0.5.0
[0.4.1]: https://github.com/rosseca/aisi/releases/tag/v0.4.1
[0.4.0]: https://github.com/rosseca/aisi/releases/tag/v0.4.0
[0.3.0]: https://github.com/rosseca/aisi/releases/tag/v0.3.0
[0.2.0]: https://github.com/rosseca/aisi/releases/tag/v0.2.0
[0.1.0]: https://github.com/rosseca/aisi/releases/tag/v0.1.0
