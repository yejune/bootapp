package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install docker-bootapp as a Docker CLI plugin",
	Long: `Install docker-bootapp as a Docker CLI plugin.

This command copies the current binary to ~/.docker/cli-plugins/
and makes it executable. After installation, you can use it as:

  docker bootapp up
  docker bootapp down
  docker bootapp ls

On macOS, it also checks for docker-mac-net-connect dependency.`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Installing docker-bootapp as Docker CLI plugin...")
	fmt.Println()

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	fmt.Printf("Current executable: %s\n", exePath)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create plugins directory
	pluginDir := filepath.Join(homeDir, ".docker", "cli-plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	targetPath := filepath.Join(pluginDir, "docker-bootapp")

	// Check if already installed
	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("⚠️  docker-bootapp is already installed at: %s\n", targetPath)
		fmt.Print("Do you want to overwrite it? (y/N): ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Installation cancelled.")
			return nil
		}
	}

	// Copy binary
	fmt.Printf("📋 Copying to %s...\n", targetPath)
	if err := copyFile(exePath, targetPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	fmt.Println("✓ Binary installed successfully")
	fmt.Println()

	// Verify installation
	verifyCmd := exec.Command("docker", "bootapp", "--version")
	if output, err := verifyCmd.Output(); err == nil {
		fmt.Printf("✓ Installation verified: %s\n", string(output))
	} else {
		fmt.Println("⚠️  Warning: Unable to verify installation")
		fmt.Println("   Please ensure Docker is running and try: docker bootapp --version")
	}

	// macOS specific checks
	if runtime.GOOS == "darwin" {
		fmt.Println()
		fmt.Println("🍎 macOS detected - checking dependencies...")
		checkMacOSDependencies()
	}

	fmt.Println()
	fmt.Println("🎉 Installation complete!")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  docker bootapp up      # Start containers with auto-networking")
	fmt.Println("  docker bootapp down    # Stop containers")
	fmt.Println("  docker bootapp ls      # List managed projects")
	fmt.Println()

	return nil
}

func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Sync to ensure write is complete
	return destFile.Sync()
}

func checkMacOSDependencies() {
	// Check if docker-mac-net-connect is installed
	if _, err := exec.LookPath("docker-mac-net-connect"); err != nil {
		fmt.Println("⚠️  docker-mac-net-connect is NOT installed")
		fmt.Println()
		fmt.Println("On macOS, docker-mac-net-connect is required to access container IPs directly.")
		fmt.Println()
		fmt.Println("Install with:")
		fmt.Println("  brew install chipmk/tap/docker-mac-net-connect")
		fmt.Println("  sudo brew services start docker-mac-net-connect")
		fmt.Println()
		return
	}

	fmt.Println("✓ docker-mac-net-connect is installed")

	// Check if service is running
	cmd := exec.Command("brew", "services", "list")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	// Simple check - look for "docker-mac-net-connect" and "started" in output
	outputStr := string(output)
	if !contains(outputStr, "docker-mac-net-connect") {
		return
	}

	if contains(outputStr, "started") {
		fmt.Println("✓ docker-mac-net-connect service is running")
	} else {
		fmt.Println("⚠️  docker-mac-net-connect is installed but not running")
		fmt.Println()
		fmt.Println("Start the service with:")
		fmt.Println("  sudo brew services start docker-mac-net-connect")
		fmt.Println()
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
