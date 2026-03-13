# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://github.com/rosseca/aisi/releases/tag/v0.1.0
