# Contributing to AISI

Thank you for your interest in contributing to AISI (AI Shared Intelligence CLI). This document will guide you through the contribution process.

## Development Setup

### Prerequisites

- Go 1.22 or later
- Git
- Make (optional, for convenience)

### Clone and Build

```bash
# Clone the repository
git clone git@github.com:rosseca/aisi.git
cd aisi

# Build the project
go build -o aisi ./cmd/aisi

# Or install to $GOPATH/bin
go install ./cmd/aisi
```

## Project Structure

```
.
├── cmd/aisi/           # Main application entry point
├── internal/
│   ├── commands/       # CLI command implementations
│   ├── config/         # Configuration management
│   ├── installer/      # Asset installation logic
│   ├── manifest/       # Manifest parsing and validation
│   ├── repo/           # Git repository operations
│   ├── targets/        # AI editor target definitions
│   ├── tui/            # Terminal UI (Bubbletea)
│   └── version/        # Version information
├── testdata/           # Test fixtures
└── docs/               # Documentation
```

## Making Changes

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 2. Write Code

- Follow Go conventions and idioms
- Use meaningful variable names
- Add comments for exported functions and types
- Keep functions focused and small

### 3. Add Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/manifest/...
```

### 4. Build and Test Locally

```bash
# Build
make build
# or
go build -o aisi ./cmd/aisi

# Test with a local repository
./aisi --repo=/path/to/test/repo list
./aisi --repo=/path/to/test/repo install some-asset
```

## Code Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Use `golint` for linting
- Keep lines under 100 characters when possible

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to load manifest: %w", err)
}
```

### Testing

```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid case", "input", "expected", false},
        {"invalid case", "bad", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NewFeature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewFeature() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("NewFeature() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## TUI Development

The TUI uses [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

### Key Files

- `internal/tui/tui.go` - Main app state machine
- `internal/tui/browser.go` - Asset browser
- `internal/tui/category_browser.go` - Category selection
- `internal/tui/menu.go` - Main menu
- `internal/tui/styles.go` - Shared styles

### TUI Patterns

```go
// Update handles messages
type MyModel struct {
    cursor int
}

func (m *MyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "q", "esc":
            return m, tea.Quit
        }
    }
    return m, nil
}

// View renders the UI
func (m *MyModel) View() string {
    return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "Hello")
}
```

## Submitting Changes

### 1. Commit Messages

Use clear, descriptive commit messages:

```
feat: add category filter to asset browser

- Add categories array support to manifest
- Create category browser component
- Filter assets by selected category
```

### 2. Pull Request

1. Push your branch: `git push origin feature/your-feature`
2. Open a PR against `main`
3. Fill out the PR template
4. Ensure CI passes (tests, build)
5. Request review

### PR Checklist

- [ ] Code builds without errors
- [ ] Tests pass (`go test ./...`)
- [ ] New features have tests
- [ ] Code follows project style
- [ ] Documentation updated if needed

## Release Process

Releases are handled by maintainers:

1. Version bump in `internal/version/version.go`
2. Update CHANGELOG.md
3. Tag release: `git tag v1.2.3`
4. Push tags: `git push origin v1.2.3`
5. GoReleaser creates binaries

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for questions
- Join our community chat (if available)

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.
