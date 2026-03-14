package deps

import (
	"runtime"
	"testing"

	"github.com/rosseca/aisi/internal/manifest"
)

func TestManager_CheckCommand(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "existing command - sh",
			command: "sh",
			want:    true,
		},
		{
			name:    "non-existing command",
			command: "thiscommanddoesnotexist12345",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.CheckCommand(tt.command)
			if got != tt.want {
				t.Errorf("CheckCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestManager_Install_NilConfig(t *testing.T) {
	mgr := NewManager()

	err := mgr.Install(nil)
	if err == nil {
		t.Error("Install(nil) should return an error")
	}

	want := "install configuration is nil"
	if err.Error() != want {
		t.Errorf("Install(nil) error = %q, want %q", err.Error(), want)
	}
}

func TestManager_Install_UnsupportedOS(t *testing.T) {
	// Skip if running on darwin or linux since those are supported
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		t.Skip("Skipping unsupported OS test on supported OS")
	}

	mgr := NewManager()
	config := &manifest.InstallConfig{
		MacOS: &manifest.MacOSInstall{
			Brew: &manifest.BrewInstall{
				Command: "test",
			},
		},
	}

	err := mgr.Install(config)
	if err == nil {
		t.Error("Install should return error on unsupported OS")
	}
}

func TestManager_installNpm(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		name    string
		npm     *manifest.NpmInstall
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil npm config",
			npm:     nil,
			wantErr: true,
			errMsg:  "npm install configuration is nil",
		},
		{
			name: "npm not installed - will fail if npm not present",
			npm: &manifest.NpmInstall{
				Packages: []manifest.NpmPackage{
					{Package: "test-package"},
				},
			},
			wantErr: !mgr.CheckCommand("npm"), // Only expect error if npm not installed
			errMsg:  "npm is required but not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.installNpm(tt.npm)
			if (err != nil) != tt.wantErr {
				t.Errorf("installNpm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("installNpm() error message = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestManager_installMacOS_NilConfig(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-darwin OS")
	}

	mgr := NewManager()

	err := mgr.installMacOS(nil)
	if err == nil {
		t.Error("installMacOS(nil) should return an error")
	}

	want := "macOS install configuration is nil"
	if err.Error() != want {
		t.Errorf("installMacOS(nil) error = %q, want %q", err.Error(), want)
	}
}

func TestManager_installMacOS_NilBrew(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-darwin OS")
	}

	mgr := NewManager()
	config := &manifest.MacOSInstall{
		Brew: nil,
	}

	err := mgr.installMacOS(config)
	if err == nil {
		t.Error("installMacOS with nil brew should return an error")
	}

	want := "brew configuration is nil"
	if err.Error() != want {
		t.Errorf("installMacOS(nil brew) error = %q, want %q", err.Error(), want)
	}
}

func TestManager_installLinux_NilConfig(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-linux OS")
	}

	mgr := NewManager()

	err := mgr.installLinux(nil)
	if err == nil {
		t.Error("installLinux(nil) should return an error")
	}

	want := "linux install configuration is nil"
	if err.Error() != want {
		t.Errorf("installLinux(nil) error = %q, want %q", err.Error(), want)
	}
}

func TestManager_Install_PriorityNpm(t *testing.T) {
	// This test verifies that npm takes priority over OS-specific configs
	mgr := NewManager()

	// Only run if npm is available
	if !mgr.CheckCommand("npm") {
		t.Skip("npm not installed, skipping npm priority test")
	}

	config := &manifest.InstallConfig{
		Npm: &manifest.NpmInstall{
			Packages: []manifest.NpmPackage{
				{Package: "is-odd"}, // Using a small, safe package
			},
		},
		MacOS: &manifest.MacOSInstall{
			Brew: &manifest.BrewInstall{
				Command: "thiswouldfail",
			},
		},
		Linux: &manifest.LinuxInstall{
			Apt: &manifest.AptInstall{
				Command: "thiswouldfail",
			},
		},
	}

	// This should succeed using npm, ignoring the OS-specific configs
	err := mgr.Install(config)
	// We accept either success or a specific npm error (not OS errors)
	if err != nil {
		// If it fails, it should be due to npm, not OS-specific configs
		if runtime.GOOS == "darwin" && contains(err.Error(), "brew") {
			t.Errorf("npm should take priority over brew on macOS, got brew error: %v", err)
		}
		if runtime.GOOS == "linux" && contains(err.Error(), "apt") {
			t.Errorf("npm should take priority over apt on Linux, got apt error: %v", err)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
