package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/docker-bootapp/internal/compose"
	"github.com/yejune/docker-bootapp/internal/hosts"
	"github.com/yejune/docker-bootapp/internal/network"
	"github.com/yejune/docker-bootapp/internal/route"
)

var (
	noBuild bool
	pull    bool
	detach  bool
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create and start containers with network setup",
	Long: `Start containers using docker-compose and automatically:
- Allocate unique subnet for the project
- Create Docker network with the subnet
- Register all container domains in /etc/hosts
- Setup routing (macOS only)
- Save configuration to .docker/network.json`,
	RunE: runUp,
}

func init() {
	upCmd.Flags().BoolVar(&noBuild, "no-build", false, "Don't build images")
	upCmd.Flags().BoolVar(&pull, "pull", false, "Pull images before starting")
	upCmd.Flags().BoolVarP(&detach, "detach", "d", true, "Run containers in background")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Find or use specified docker-compose file
	var composePath string
	var err error

	if composeFile != "" {
		// Use specified file
		composePath, err = filepath.Abs(composeFile)
		if err != nil {
			return fmt.Errorf("invalid compose file path: %w", err)
		}
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			return fmt.Errorf("compose file not found: %s", composePath)
		}
	} else {
		// Auto-detect
		composePath, err = compose.FindComposeFile()
		if err != nil {
			return err
		}
	}

	fmt.Printf("Using compose file: %s\n", composePath)

	// Parse compose file
	composeData, err := compose.ParseComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Get project info
	projectPath := filepath.Dir(composePath)
	projectName := compose.GetProjectName(composePath)
	fmt.Printf("Project: %s\n", projectName)

	// Extract domains per service from compose file
	serviceDomains := compose.ExtractServiceDomains(composeData)
	baseDomain := compose.ExtractDomain(composeData)
	if baseDomain == "" {
		baseDomain = projectName + ".local"
	}
	fmt.Printf("Base Domain: %s\n", baseDomain)
	if len(serviceDomains) > 0 {
		fmt.Println("Service Domains:")
		for svc, doms := range serviceDomains {
			fmt.Printf("  %s: %s\n", svc, strings.Join(doms, ", "))
		}
	}

	// Validate sudo credentials upfront (needed for /etc/hosts)
	fmt.Println("\nValidating sudo credentials...")
	if err := validateSudo(); err != nil {
		return fmt.Errorf("sudo authentication required: %w", err)
	}

	// Run docker-compose up
	fmt.Println("\nStarting containers...")
	if err := runDockerCompose(composePath, projectName); err != nil {
		return err
	}

	// Get container IPs and network info from default compose network
	fmt.Println("\nDiscovering containers...")
	containerIPs, networkSubnet, err := getContainerIPsAndSubnet(projectName)
	if err != nil {
		fmt.Printf("Warning: Could not get container IPs: %v\n", err)
		containerIPs = make(map[string]string)
	}
	if networkSubnet != "" {
		fmt.Printf("Network subnet: %s\n", networkSubnet)
	}

	// Build container info with domains (only for services with domain config)
	containers := buildContainerInfo(containerIPs, serviceDomains)

	// Save local config
	localConfig := network.LocalConfig{
		Project:    projectName,
		Subnet:     networkSubnet,
		Containers: containers,
	}
	if err := network.SaveLocalConfig(projectPath, &localConfig); err != nil {
		fmt.Printf("Warning: Could not save config: %v\n", err)
	}

	// Print container info
	fmt.Println("\nContainers:")
	for name, info := range containers {
		if len(info.Domains) > 0 {
			fmt.Printf("  %s: %s -> %s\n", name, strings.Join(info.Domains, ", "), info.IP)
		} else {
			fmt.Printf("  %s: %s (no domain)\n", name, info.IP)
		}
	}

	// Setup /etc/hosts for all containers
	fmt.Println("\nSetting up /etc/hosts...")
	if err := hosts.AddEntries(containers, projectName); err != nil {
		return fmt.Errorf("failed to update /etc/hosts: %w", err)
	}

	// Setup routing (macOS only) - use actual network subnet
	if networkSubnet != "" {
		fmt.Println("\nSetting up routing...")
		// Get first container IP for connectivity test
		var testIP string
		for _, info := range containers {
			if info.IP != "" {
				testIP = info.IP
				break
			}
		}
		if err := route.SetupRouteWithTest(networkSubnet, testIP); err != nil {
			fmt.Printf("Warning: Route setup failed: %v\n", err)
		}
	}

	// Print config file locations
	fmt.Println("\n📁 Configuration files:")
	fmt.Printf("  Local:  %s/.docker/network.json\n", projectPath)

	// Get main app domain
	appDomain := baseDomain
	if info, ok := containers["app"]; ok && len(info.Domains) > 0 {
		appDomain = info.Domains[0]
	}

	fmt.Printf("\n✅ Ready! Access your app at: https://%s\n", appDomain)

	return nil
}

// buildContainerInfo creates ContainerInfo map with domains
// serviceDomains: map[serviceName][]domains from compose file
// Only services with explicit domain config get domains
// Other services get IP only (no domain)
func buildContainerInfo(ips map[string]string, serviceDomains map[string][]string) map[string]network.ContainerInfo {
	containers := make(map[string]network.ContainerInfo)

	for name, ip := range ips {
		// Check if this service has domain configuration
		if domains, ok := serviceDomains[name]; ok && len(domains) > 0 {
			containers[name] = network.ContainerInfo{
				Domains: domains,
				IP:      ip,
			}
		} else {
			// No domain config - IP only
			containers[name] = network.ContainerInfo{
				IP: ip,
			}
		}
	}

	return containers
}

func createDockerNetwork(name, subnet string) error {
	// Check if network already exists
	checkCmd := exec.Command("docker", "network", "inspect", name)
	if checkCmd.Run() == nil {
		// Network exists, check if subnet matches
		return nil // Keep existing network
	}

	// Create network with subnet
	args := []string{
		"network", "create",
		"--driver", "bridge",
		"--subnet", subnet,
		name,
	}
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runDockerCompose(composePath, projectName string) error {
	// Use "docker compose" (V2) instead of "docker-compose"
	args := []string{"compose", "-f", composePath, "-p", projectName, "up"}

	if detach {
		args = append(args, "-d")
	}
	if !noBuild {
		args = append(args, "--build")
	}
	if pull {
		args = append(args, "--pull", "always")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = filepath.Dir(composePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// connectContainersToNetwork connects all project containers to the bootapp network
func connectContainersToNetwork(projectName, networkName string) error {
	// List containers for this project
	listCmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := listCmd.Output()
	if err != nil {
		return err
	}

	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return fmt.Errorf("no containers found for project %s", projectName)
	}

	for _, id := range containerIDs {
		// Connect container to bootapp network (ignore if already connected)
		connectCmd := exec.Command("docker", "network", "connect", networkName, id)
		if err := connectCmd.Run(); err != nil {
			// Ignore "already connected" errors
			continue
		}
	}

	return nil
}

// getContainerIPsFromNetwork gets container IPs specifically from the bootapp network
func getContainerIPsFromNetwork(projectName, networkName string) (map[string]string, error) {
	// List containers for this project
	listCmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := listCmd.Output()
	if err != nil {
		return nil, err
	}

	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return nil, fmt.Errorf("no containers found for project %s", projectName)
	}

	containers := make(map[string]string)

	for _, id := range containerIDs {
		// Get container info as JSON
		inspectCmd := exec.Command("docker", "inspect", id)
		output, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		var infos []DockerContainerInfo
		if err := json.Unmarshal(output, &infos); err != nil || len(infos) == 0 {
			continue
		}

		info := infos[0]
		serviceName := info.Config.Labels["com.docker.compose.service"]

		// Get IP from the specific bootapp network
		if netInfo, ok := info.NetworkSettings.Networks[networkName]; ok && netInfo.IPAddress != "" {
			containers[serviceName] = netInfo.IPAddress
		}
	}

	return containers, nil
}

// getContainerIPsAndSubnet returns container IPs and the network subnet
func getContainerIPsAndSubnet(projectName string) (map[string]string, string, error) {
	// List containers for this project
	listCmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := listCmd.Output()
	if err != nil {
		return nil, "", err
	}

	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return nil, "", fmt.Errorf("no containers found for project %s", projectName)
	}

	containers := make(map[string]string)
	var networkName string

	for _, id := range containerIDs {
		inspectCmd := exec.Command("docker", "inspect", id)
		output, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		var infos []DockerContainerInfo
		if err := json.Unmarshal(output, &infos); err != nil || len(infos) == 0 {
			continue
		}

		info := infos[0]
		serviceName := info.Config.Labels["com.docker.compose.service"]

		// Prefer *_default network
		for netName, net := range info.NetworkSettings.Networks {
			if net.IPAddress != "" && serviceName != "" {
				if strings.HasSuffix(netName, "_default") {
					containers[serviceName] = net.IPAddress
					networkName = netName
					break
				}
				if containers[serviceName] == "" {
					containers[serviceName] = net.IPAddress
					networkName = netName
				}
			}
		}
	}

	// Get subnet from the network
	subnet := getNetworkSubnet(networkName)

	return containers, subnet, nil
}

// getNetworkSubnet gets the subnet CIDR from a Docker network
func getNetworkSubnet(networkName string) string {
	if networkName == "" {
		return ""
	}

	cmd := exec.Command("docker", "network", "inspect", networkName, "--format", "{{range .IPAM.Config}}{{.Subnet}}{{end}}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// DockerContainerInfo for JSON parsing
type DockerContainerInfo struct {
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

func getContainerIPsJSON(projectName string) (map[string]string, error) {
	// Alternative method using JSON output
	listCmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := listCmd.Output()
	if err != nil {
		return nil, err
	}

	containerIDs := strings.Fields(string(output))
	containers := make(map[string]string)

	for _, id := range containerIDs {
		inspectCmd := exec.Command("docker", "inspect", id)
		output, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		var infos []DockerContainerInfo
		if err := json.Unmarshal(output, &infos); err != nil || len(infos) == 0 {
			continue
		}

		info := infos[0]
		serviceName := info.Config.Labels["com.docker.compose.service"]
		for _, net := range info.NetworkSettings.Networks {
			if net.IPAddress != "" && serviceName != "" {
				containers[serviceName] = net.IPAddress
				break
			}
		}
	}

	return containers, nil
}

// validateSudo prompts for sudo password and caches credentials
func validateSudo() error {
	// sudo -v prompts for password if needed and extends the timeout
	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
