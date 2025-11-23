package route

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsDarwin returns true if running on macOS
func IsDarwin() bool {
	return runtime.GOOS == "darwin"
}

// CheckDockerMacNetConnect checks if docker-mac-net-connect is running on macOS
func CheckDockerMacNetConnect() bool {
	if !IsDarwin() {
		return true // Not needed on Linux
	}

	cmd := exec.Command("pgrep", "-f", "docker-mac-net-connect")
	output, _ := cmd.Output()
	return len(output) > 0
}

// GetDockerVMGateway finds the Docker Desktop VM gateway IP on macOS
// This is the IP we need to route container traffic through
func GetDockerVMGateway() (string, error) {
	// Method 1: Get host.docker.internal IP from inside a container
	cmd := exec.Command("docker", "run", "--rm", "alpine", "getent", "hosts", "host.docker.internal")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		parts := strings.Fields(string(output))
		if len(parts) > 0 {
			return parts[0], nil
		}
	}

	// Method 2: Try to get the gateway from docker network
	cmd = exec.Command("docker", "network", "inspect", "bridge", "--format", "{{range .IPAM.Config}}{{.Gateway}}{{end}}")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		gateway := strings.TrimSpace(string(output))
		if gateway != "" {
			return gateway, nil
		}
	}

	// Method 3: Use default Docker Desktop gateway
	// Docker Desktop typically uses 192.168.65.1 as the VM gateway
	return "192.168.65.1", nil
}

// SetupRoute sets up routing for the subnet on macOS
func SetupRoute(subnet string) error {
	if IsLinux() {
		// Linux doesn't need routing setup - containers are directly accessible
		return nil
	}

	if !IsDarwin() {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Check if docker-mac-net-connect is available
	if CheckDockerMacNetConnect() {
		fmt.Println("✓ docker-mac-net-connect detected, routes managed automatically")
		return nil
	}

	// Get Docker VM gateway
	gateway, err := GetDockerVMGateway()
	if err != nil {
		return fmt.Errorf("cannot determine Docker gateway: %w", err)
	}

	fmt.Printf("Using Docker VM gateway: %s\n", gateway)

	// Delete existing route (ignore errors)
	deleteCmd := exec.Command("sudo", "route", "-n", "delete", "-net", subnet)
	deleteCmd.Run()

	// Add new route: route container subnet through Docker VM gateway
	addCmd := exec.Command("sudo", "route", "-n", "add", "-net", subnet, gateway)
	addCmd.Stdin = os.Stdin
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr

	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}

	fmt.Printf("✓ Route added: %s -> %s\n", subnet, gateway)
	return nil
}

// RemoveRoute removes routing for the subnet on macOS
// Always attempts removal regardless of docker-mac-net-connect status
func RemoveRoute(subnet string) error {
	if IsLinux() {
		return nil
	}

	if !IsDarwin() {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Always try to remove route (ignore errors if doesn't exist)
	cmd := exec.Command("sudo", "route", "-n", "delete", "-net", subnet)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore error

	return nil
}

// PrintRouteInfo prints information about routing setup
func PrintRouteInfo(subnet string) {
	if IsLinux() {
		fmt.Println("Linux: Direct container access (no routing needed)")
		return
	}

	if CheckDockerMacNetConnect() {
		fmt.Println("✓ docker-mac-net-connect: Routes managed automatically")
	} else {
		fmt.Printf("Route configured for subnet %s\n", subnet)
		fmt.Println("\nTIP: For easier routing, install docker-mac-net-connect:")
		fmt.Println("  brew install chipmk/tap/docker-mac-net-connect")
		fmt.Println("  sudo brew services start docker-mac-net-connect")
	}
}
