package deps

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/rosseca/aisi/internal/manifest"
)

// Manager handles checking and installing command dependencies
type Manager struct {
	onProgress func(string)
}

// NewManager creates a new dependency manager
func NewManager() *Manager {
	return &Manager{}
}

// SetProgressCallback sets a callback for progress messages
func (m *Manager) SetProgressCallback(callback func(string)) {
	m.onProgress = callback
}

// reportProgress sends a progress message if callback is set
func (m *Manager) reportProgress(msg string) {
	if m.onProgress != nil {
		m.onProgress(msg)
	}
}

// CheckCommand verifies if a command exists in PATH
func (m *Manager) CheckCommand(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// Install installs a dependency based on the provided configuration
func (m *Manager) Install(config *manifest.InstallConfig) error {
	if config == nil {
		return fmt.Errorf("install configuration is nil")
	}

	var installErr error

	// Prioridad 1: npm (multiplataforma)
	if config.Npm != nil {
		installErr = m.installNpm(config.Npm)
	} else {
		// Prioridad 2: OS-specific
		switch runtime.GOOS {
		case "darwin":
			installErr = m.installMacOS(config.MacOS)
		case "linux":
			installErr = m.installLinux(config.Linux)
		default:
			return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
		}
	}

	if installErr != nil {
		return installErr
	}

	// Ejecutar post-install genérico si está definido en InstallConfig
	if config.PostInstall != "" {
		m.reportProgress(fmt.Sprintf("🔧 Running post-install: %s", config.PostInstall))
		if err := m.runPostInstall(config.PostInstall); err != nil {
			return err
		}
		m.reportProgress("✓ Post-install completed")
	}

	return nil
}

// installNpm installs packages globally using npm
func (m *Manager) installNpm(npm *manifest.NpmInstall) error {
	if npm == nil {
		return fmt.Errorf("npm install configuration is nil")
	}

	if len(npm.Packages) == 0 {
		return fmt.Errorf("no npm packages specified")
	}

	// Verificar que npm esté instalado
	if !m.CheckCommand("npm") {
		return fmt.Errorf("npm is required but not installed")
	}

	// Instalar cada paquete
	for _, pkg := range npm.Packages {
		// Construir comando: npm install -g package@version (o solo package)
		pkgSpec := pkg.Package
		if pkg.Version != "" {
			pkgSpec = pkgSpec + "@" + pkg.Version
		}

		m.reportProgress(fmt.Sprintf("📦 Installing npm package: %s", pkgSpec))
		cmd := exec.Command("npm", "install", "-g", pkgSpec)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("npm install failed for %s: %w (output: %s)", pkg.Package, err, strings.TrimSpace(string(output)))
		}
		m.reportProgress(fmt.Sprintf("✓ Package '%s' installed successfully", pkg.Package))
	}

	return nil
}

// installMacOS installs a package on macOS using brew
func (m *Manager) installMacOS(macOS *manifest.MacOSInstall) error {
	if macOS == nil {
		return fmt.Errorf("macOS install configuration is nil")
	}

	if macOS.Brew == nil {
		return fmt.Errorf("brew configuration is nil")
	}

	// Verificar que brew esté instalado
	if !m.CheckCommand("brew") {
		return fmt.Errorf("brew is required but not installed")
	}

	// Agregar tap si está especificado
	if macOS.Brew.Tap != "" {
		m.reportProgress(fmt.Sprintf("🔧 Adding brew tap: %s", macOS.Brew.Tap))
		cmd := exec.Command("brew", "tap", macOS.Brew.Tap)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("brew tap failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
		}
	}

	// Instalar paquete
	m.reportProgress(fmt.Sprintf("📦 Installing brew package: %s", macOS.Brew.Command))
	cmd := exec.Command("brew", "install", macOS.Brew.Command)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("brew install failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	m.reportProgress(fmt.Sprintf("✓ Package '%s' installed successfully", macOS.Brew.Command))

	return nil
}

// installLinux installs a package on Linux using apt
func (m *Manager) installLinux(linux *manifest.LinuxInstall) error {
	if linux == nil {
		return fmt.Errorf("linux install configuration is nil")
	}

	if linux.Apt == nil {
		return fmt.Errorf("apt configuration is nil")
	}

	// Agregar sources si están especificados
	for _, source := range linux.Apt.Sources {
		m.reportProgress(fmt.Sprintf("🔧 Adding apt source: %s", source))
		cmd := exec.Command("sudo", "add-apt-repository", "-y", source)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("add-apt-repository failed for %s: %w (output: %s)", source, err, strings.TrimSpace(string(output)))
		}
	}

	// Actualizar repositorios si se agregaron sources o por precaución
	if len(linux.Apt.Sources) > 0 {
		m.reportProgress("🔄 Running apt-get update...")
		cmd := exec.Command("sudo", "apt-get", "update")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("apt-get update failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
		}
	}

	// Instalar paquete
	m.reportProgress(fmt.Sprintf("📦 Installing apt package: %s", linux.Apt.Command))
	cmd := exec.Command("sudo", "apt-get", "install", "-y", linux.Apt.Command)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apt-get install failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	m.reportProgress(fmt.Sprintf("✓ Package '%s' installed successfully", linux.Apt.Command))

	// Ejecutar post-install específico de apt si está especificado
	if linux.Apt.PostInstall != "" {
		m.reportProgress(fmt.Sprintf("🔧 Running post-install: %s", linux.Apt.PostInstall))
		args := strings.Fields(linux.Apt.PostInstall)
		if len(args) > 0 {
			cmd := exec.Command(args[0], args[1:]...)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("post-install command failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
			}
		}
		m.reportProgress("✓ Post-install completed")
	}

	return nil
}

// runPostInstall executes a post-install command
func (m *Manager) runPostInstall(command string) error {
	args := strings.Fields(command)
	if len(args) == 0 {
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("post-install command failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}
