# AISI — AI Shared Intelligence

> **The problem:** You have 15 projects and copy the same Cursor rules by hand in each one like it's 1995.  
> **The solution:** One centralized repo with your rules, skills, and configs. AISI installs them where they belong with a single command.

AISI is a CLI that manages AI agent resources (rules, skills, subagents, hooks, MCP) from a shared repository to multiple editors: Cursor, Kilo Code, Junie (JetBrains), Windsurf.

---

## Table of Contents

- [Installing AISI](#installing-aisi)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Commands](#commands)
- [Asset Types](#asset-types)
- [Shared Repository Structure](#shared-repository-structure)
- [Configuration](#configuration)
- [Development](#development)

---

## Installing AISI

### Homebrew (recommended for macOS)

```bash
# Add the tap
brew tap rosseca/tap

# Install aisi
brew install aisi
```

### From Go

```bash
go install github.com/rosseca/aisi/cmd/aisi@latest
```

Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your PATH.

### From GitHub Releases

Download the binary for your platform from [Releases](../../releases) and place it in a directory in your PATH.

### Build from source

```bash
git clone https://github.com/rosseca/aisi.git
cd aisi
make build
make install  # Copies to ~/go/bin
```

---

## Quick Start

### 1. Configure your shared repository

```bash
aisi config set-repo git@github.com:your-org/shared-ai-assets.git
aisi config set-target cursor  # Or kilo, junie, windsurf
```

### 2. Explore what's available

```bash
aisi list              # Everything
aisi list rules        # Rules only
aisi list skills       # Skills only
```

### 3. Install assets

```bash
aisi install soul typescript    # Specific ones
aisi install --type=rules --all # All rules
```

### 4. Interactive mode (TUI)

```bash
aisi  # No arguments → interactive interface
```

---

## How It Works

```
┌─────────────────┐     ┌─────────────┐     ┌────────────────┐
│ Shared repo     │────▶│  AISI CLI   │────▶│  Your project  │
│  (remote Git)   │     │  (local)    │     │ (.cursor/...)  │
└─────────────────┘     └─────────────┘     └────────────────┘
       │                        │
       │                 ┌──────┴──────┐
       │                 ▼             ▼
       │            ~/.aisi/      .aisi.lock
       │            (cache)       (tracking)
       │
       ├── rules/*.mdc
       ├── skills/*/SKILL.md
       ├── agents/*.md
       ├── hooks/hooks.json
       ├── mcp/*.json
       └── manifest.json
```

1. **You configure** the remote repo in `~/.aisi/config.yaml`
2. **AISI clones/updates** the repo to local cache
3. **Reads `manifest.json`** to know what assets exist
4. **Installs** by copying files to the target directory (`.cursor/`, `.kilocode/`, etc.)
5. **Tracking**: registers what's installed in `.aisi.lock` so you can update later

---

## Commands

| Command | Description |
|---------|-------------|
| `aisi` | Interactive TUI mode |
| `aisi install <name>...` | Install specific assets |
| `aisi install --type=TYPE --all` | Install all of a type |
| `aisi list [TYPE]` | List available assets |
| `aisi status` | Show installed assets in current project |
| `aisi update` | Update installed assets to latest commit |
| `aisi remove <name>...` | Remove installed assets |
| `aisi config set-repo <url>` | Configure remote repo |
| `aisi config set-target <target>` | Default target |
| `aisi config show` | Show current configuration |
| `aisi version` | CLI version |

### Global flags

| Flag | Description |
|------|-------------|
| `--target=TARGET` | Override target (cursor, kilo, junie, windsurf) |
| `--repo=URL` | Override repo for this execution |
| `--force` | Force reinstall even if up to date |

### Installing Skills from External Repositories

You can install skills directly from any Git repository without needing to define them in your manifest first.

**From GitHub (shorthand):**

```bash
aisi install skill --url twostraws/swiftui-agent-skill
```

**From a specific path in the repo:**

```bash
aisi install skill --url https://github.com/twostraws/swiftui-agent-skill/tree/main/swiftui-pro
```

**With a custom name:**

```bash
aisi install skill --url twostraws/swiftui-agent-skill --name swiftui-pro
```

**Alternative methods:**

```bash
# Full GitHub URL
aisi install skill --url https://github.com/vercel-labs/agent-skills

# GitLab
aisi install skill --url https://gitlab.com/org/repo

# SSH
aisi install skill --url git@github.com:owner/repo.git

# Local path
aisi install skill --url ./my-local-skills
```

### Supported URL Formats

| Format | Example |
|--------|---------|
| GitHub shorthand | `owner/repo` |
| GitHub HTTPS | `https://github.com/owner/repo` |
| GitHub tree/blob | `https://github.com/owner/repo/tree/main/skills/foo` |
| GitLab HTTPS | `https://gitlab.com/org/repo` |
| SSH | `git@github.com:owner/repo.git` |
| Local path | `./path/to/skill` or `/absolute/path` |

---

## Asset Types

| Type | Description | Destination in Cursor |
|------|-------------|------------------------|
| `rules` | `.mdc` files with YAML frontmatter | `.cursor/rules/` |
| `skills` | Directories with `SKILL.md` | `.cursor/skills/` |
| `agents` | Subagents in `.md` | `.cursor/agents/` |
| `hooks` | `hooks.json` + scripts | `.cursor/hooks.json` |
| `mcp` | MCP configurations | `.cursor/mcp.json` |
| `agents-md` | Global AGENTS.md instructions | `AGENTS.md` or `.cursor/AGENTS.md` |
| `external` | Assets from external repos | Varies |

---

## Shared Repository Structure

Your assets repository should follow this convention:

```
shared-ai-assets/
├── rules/
│   ├── soul.mdc           # Rules with frontmatter
│   ├── typescript.mdc
│   └── go.mdc
├── skills/
│   ├── swift-testing/     # One folder per skill
│   │   └── SKILL.md       # Required file
│   └── gcp-docs/
│       └── SKILL.md
├── agents/
│   ├── code-reviewer.md   # Subagent
│   └── security-audit.md
├── hooks/
│   ├── hooks.json         # Hook definitions
│   └── scripts/
│       └── pre-commit.sh
├── mcp/
│   └── github.json        # MCP config
├── agents-md/
│   └── default.md         # AGENTS.md template
├── manifest.json          # Asset registry
└── README.md
```

### Manifest (`manifest.json`)

```json
{
  "version": "1.0.0",
  "minimumCliVersion": "0.5.0",
  "rules": [
    {
      "name": "soul",
      "description": "Sassy architect persona",
      "file": "rules/soul.mdc"
    }
  ],
  "skills": [
    {
      "name": "swift-testing",
      "description": "Swift Testing best practices",
      "directory": "skills/swift-testing"
    }
  ]
}
```

See [docs/manifest.md](docs/manifest.md) for the full format.

---

## Configuration

### Global (`~/.aisi/config.yaml`)

```yaml
repo:
  url: "git@github.com:your-org/shared-ai-assets.git"
  branch: "main"
activeTarget: cursor
```

### Per project (`.aisi.lock`)

Generated automatically in the target directory:

```json
{
  "installed": [
    {
      "name": "soul",
      "type": "rule",
      "commit": "abc123..."
    }
  ],
  "commit": "abc123..."
}
```

---

## Development

```bash
make test            # Unit tests
make test-coverage   # Coverage report
make build           # Local build
make build-all       # Cross-compile (goreleaser)
make lint            # golangci-lint
make clean           # Clean binaries
```

### Architecture

```
cmd/aisi/
  └── main.go              # Entry point

internal/
  ├── commands/            # Cobra commands
  ├── installer/           # Logic per asset type
  ├── manifest/            # Manifest parsing
  ├── repo/                # Git operations
  ├── targets/             # Destination definitions
  ├── tracker/             # Lock file management
  └── tui/                 # BubbleTea interface
```

---

## Supported Targets

| Target | Directory |
|--------|-----------|
| `cursor` | `.cursor/` |
| `kilo` | `.kilocode/` |
| `junie` | `.junie/` |
| `windsurf` | `.windsurf/` |

---

## License

MIT — See [LICENSE](LICENSE)

---

## FAQ

**Can I have multiple repositories?**
Yes, use `--repo` to override for specific commands.

**What happens when the manifest changes?**
`aisi update` detects the new commit and reinstalls.

**Does it work with monorepos?**
Yes, install the assets you need in each subproject.

**Can I install from external repositories?**
Yes, the manifest supports `external` assets that clone other repos.
