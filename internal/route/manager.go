package route

import (
	"fmt"
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

// CheckDockerMacNetConnect checks if docker-mac-net-connect is running
func CheckDockerMacNetConnect() bool {
	cmd := exec.Command("pgrep", "-f", "docker-mac-net-connect")
	output, _ := cmd.Output()
	return len(output) > 0
}

// CheckOrbStack checks if OrbStack is the Docker runtime
func CheckOrbStack() bool {
	cmd := exec.Command("docker", "context", "show")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "orbstack")
}

// SetupRoute checks routing requirements on macOS
// On Linux, no setup needed. On macOS, docker-mac-net-connect is required.
func SetupRoute(subnet string) error {
	return SetupRouteWithTest(subnet, "")
}

// checkConnectivity tests if we can reach an IP
func checkConnectivity(testIP string) bool {
	if testIP == "" {
		return false
	}
	cmd := exec.Command("ping", "-c", "1", "-t", "2", testIP)
	return cmd.Run() == nil
}

// SetupRouteWithTest checks routing with optional connectivity test
func SetupRouteWithTest(subnet, testIP string) error {
	if IsLinux() {
		// Linux doesn't need routing setup - containers are directly accessible
		fmt.Println("✓ Linux: Direct container access")
		return nil
	}

	if !IsDarwin() {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Check OrbStack (has native container IP access)
	if CheckOrbStack() {
		if testIP != "" && checkConnectivity(testIP) {
			fmt.Println("✓ OrbStack: connected")
			return nil
		}
		if testIP != "" {
			fmt.Printf("⚠️  OrbStack detected but cannot reach %s\n", testIP)
			return fmt.Errorf("OrbStack connectivity failed")
		}
		fmt.Println("✓ OrbStack detected")
		return nil
	}

	// Docker Desktop requires docker-mac-net-connect
	if CheckDockerMacNetConnect() {
		if testIP != "" && checkConnectivity(testIP) {
			fmt.Println("✓ docker-mac-net-connect: connected")
			return nil
		}
		if testIP != "" {
			fmt.Printf("⚠️  docker-mac-net-connect running but cannot reach %s\n", testIP)
			fmt.Println("   Try: sudo brew services restart docker-mac-net-connect")
			return fmt.Errorf("docker-mac-net-connect connectivity failed")
		}
		fmt.Println("✓ docker-mac-net-connect detected")
		return nil
	}

	// Neither OrbStack nor docker-mac-net-connect found
	fmt.Println("⚠️  Docker Desktop requires docker-mac-net-connect for container IP access")
	fmt.Println("")
	fmt.Println("   Install with:")
	fmt.Println("     brew install chipmk/tap/docker-mac-net-connect")
	fmt.Println("     sudo brew services start docker-mac-net-connect")
	fmt.Println("")
	fmt.Println("   Or switch to OrbStack (supports direct container access):")
	fmt.Println("     https://orbstack.dev")
	fmt.Println("")
	return fmt.Errorf("docker-mac-net-connect not running")
}

// RemoveRoute is a no-op since docker-mac-net-connect manages routes
func RemoveRoute(subnet string) error {
	// docker-mac-net-connect manages routes automatically
	return nil
}

// PrintRouteInfo prints information about routing setup
func PrintRouteInfo(subnet string) {
	if IsLinux() {
		fmt.Println("Linux: Direct container access (no routing needed)")
		return
	}
	if CheckDockerMacNetConnect() {
		fmt.Println("macOS: docker-mac-net-connect managing routes")
	} else {
		fmt.Println("macOS: docker-mac-net-connect required")
	}
}
